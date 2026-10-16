// Package fonts self-hosts Google Fonts: it finds stylesheet links in a
// project, downloads the CSS and woff2 files, and rewrites the references.
package fonts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// A Chrome user agent makes Google serve woff2 with unicode-range subsets.
const chromeUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"

var (
	reCSSURL  = regexp.MustCompile(`(?:https?:)?//fonts\.googleapis\.com/(?:css2?|icon)\?[^"'()\s<>]+`)
	rePrecon  = regexp.MustCompile(`(?i)[ \t]*<link[^>]+rel=["']?(?:preconnect|dns-prefetch)["']?[^>]+fonts\.(?:googleapis|gstatic)\.com[^>]*>[ \t]*\r?\n?|[ \t]*<link[^>]+fonts\.(?:googleapis|gstatic)\.com[^>]+rel=["']?(?:preconnect|dns-prefetch)["']?[^>]*>[ \t]*\r?\n?`)
	reFontURL = regexp.MustCompile(`url\((https://fonts\.gstatic\.com/[^)]+)\)`)
	reFamily  = regexp.MustCompile(`font-family:\s*'([^']+)'`)
)

var exts = map[string]bool{".html": true, ".htm": true, ".css": true, ".scss": true, ".sass": true, ".less": true, ".vue": true, ".svelte": true, ".astro": true, ".jsx": true, ".tsx": true, ".js": true, ".ts": true, ".php": true, ".erb": true, ".ejs": true, ".hbs": true, ".twig": true, ".liquid": true}

var skip = map[string]bool{".git": true, "node_modules": true, "vendor": true, ".next": true, "dist": true, "build": true, ".venv": true}

// Options for Fix.
type Options struct {
	Root   string // project root
	Dir    string // output directory relative to root (default "fonts")
	Write  bool   // apply changes; otherwise dry run
	Client *http.Client
	Log    func(format string, a ...any)
}

// Change describes an edit to one file.
type Change struct {
	File  string   `json:"file"`
	URLs  []string `json:"urls"`
	Local []string `json:"local"`
}

// Plan is the outcome of Fix.
type Plan struct {
	Changes []Change `json:"changes"`
	Fonts   []string `json:"fonts"` // downloaded files
	Bytes   int64    `json:"bytes"`
}

// Fix finds Google Fonts stylesheets and replaces them with local copies.
func Fix(ctx context.Context, o Options) (*Plan, error) {
	if o.Dir == "" {
		o.Dir = DefaultDir(o.Root)
	}
	if o.Client == nil {
		o.Client = &http.Client{Timeout: 60 * time.Second}
	}
	logf := o.Log
	if logf == nil {
		logf = func(string, ...any) {}
	}
	type fileHit struct {
		path string
		urls []string
	}
	var files []fileHit
	urlSet := map[string]bool{}
	err := filepath.WalkDir(o.Root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != o.Root && (skip[d.Name()] || filepath.Join(o.Root, o.Dir) == p) {
				return filepath.SkipDir
			}
			return nil
		}
		if !exts[strings.ToLower(filepath.Ext(p))] {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		us := uniq(reCSSURL.FindAllString(string(b), -1))
		if len(us) > 0 {
			files = append(files, fileHit{p, us})
			for _, u := range us {
				urlSet[u] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	plan := &Plan{}
	if len(files) == 0 {
		return plan, nil
	}
	outDir := filepath.Join(o.Root, o.Dir)
	local := map[string]string{} // css url -> local css path (absolute)
	urls := make([]string, 0, len(urlSet))
	for u := range urlSet {
		urls = append(urls, u)
	}
	sort.Strings(urls)
	for _, u := range urls {
		name := cssName(u)
		local[u] = filepath.Join(outDir, name)
		if !o.Write {
			continue
		}
		logf("fetch %s", u)
		css, err := get(ctx, o.Client, normalize(u))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", u, err)
		}
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return nil, err
		}
		out := string(css)
		for _, m := range uniq(reFontURLAll(out)) {
			fname := fontName(m, out)
			dst := filepath.Join(outDir, fname)
			if _, err := os.Stat(dst); err != nil {
				data, err := get(ctx, o.Client, m)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", m, err)
				}
				if err := os.WriteFile(dst, data, 0o644); err != nil {
					return nil, err
				}
				plan.Bytes += int64(len(data))
				plan.Fonts = append(plan.Fonts, dst)
				logf("  saved %s (%d KB)", fname, len(data)/1024)
			}
			out = strings.ReplaceAll(out, "url("+m+")", "url(./"+fname+")")
		}
		header := "/* Self-hosted by marz from " + u + " (marz-ignore) */\n"
		if err := os.WriteFile(local[u], []byte(header+out), 0o644); err != nil {
			return nil, err
		}
	}
	for _, f := range files {
		ch := Change{File: f.path, URLs: f.urls}
		b, err := os.ReadFile(f.path)
		if err != nil {
			return nil, err
		}
		s := string(b)
		for _, u := range f.urls {
			rel, err := filepath.Rel(filepath.Dir(f.path), local[u])
			if err != nil {
				return nil, err
			}
			rel = filepath.ToSlash(rel)
			if !strings.HasPrefix(rel, ".") {
				rel = "./" + rel
			}
			ch.Local = append(ch.Local, rel)
			s = strings.ReplaceAll(s, u, rel)
		}
		s = rePrecon.ReplaceAllString(s, "")
		if o.Write {
			if err := os.WriteFile(f.path, []byte(s), 0o644); err != nil {
				return nil, err
			}
		}
		plan.Changes = append(plan.Changes, ch)
	}
	return plan, nil
}

// DefaultDir picks where fonts go: the project's web root (public/, static/,
// www/) if it has one, so the files are actually served, else fonts/.
func DefaultDir(root string) string {
	for _, d := range []string{"public", "static", "www", "wwwroot"} {
		if st, err := os.Stat(filepath.Join(root, d)); err == nil && st.IsDir() {
			return filepath.Join(d, "fonts")
		}
	}
	return "fonts"
}

func reFontURLAll(css string) []string {
	var out []string
	for _, m := range reFontURL.FindAllStringSubmatch(css, -1) {
		out = append(out, m[1])
	}
	return out
}

func normalize(u string) string {
	if strings.HasPrefix(u, "//") {
		u = "https:" + u
	}
	u = strings.Replace(u, "http://", "https://", 1)
	return strings.ReplaceAll(u, "&amp;", "&")
}

// Base lets tests point at a fake Google Fonts server.
var Base = ""

func get(ctx context.Context, c *http.Client, u string) ([]byte, error) {
	if Base != "" {
		u = strings.Replace(strings.Replace(u, "https://fonts.googleapis.com", Base, 1), "https://fonts.gstatic.com", Base, 1)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", chromeUA)
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}

func cssName(u string) string {
	fam := "fonts"
	if i := strings.Index(u, "family="); i >= 0 {
		f := u[i+7:]
		if j := strings.IndexAny(f, ":&|"); j >= 0 {
			f = f[:j]
		}
		fam = slug(f)
	}
	return fam + "-" + hash(u)[:8] + ".css"
}

func fontName(u, css string) string {
	fam := "font"
	if i := strings.Index(css, u); i >= 0 {
		before := css[:i]
		if ms := reFamily.FindAllStringSubmatch(before, -1); len(ms) > 0 {
			fam = slug(ms[len(ms)-1][1])
		}
	}
	ext := filepath.Ext(strings.SplitN(u, "?", 2)[0])
	if ext == "" {
		ext = ".woff2"
	}
	return fam + "-" + hash(u)[:10] + ext
}

func slug(s string) string {
	s = strings.ToLower(strings.NewReplacer("+", "-", " ", "-", "%20", "-").Replace(s))
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "font"
	}
	return b.String()
}

func hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func uniq(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
