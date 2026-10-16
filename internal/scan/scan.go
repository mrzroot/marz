// Package scan finds third-party services in a code base: URLs and hostnames
// in any text file, plus dependencies in package manifests.
package scan

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/mrzroot/marz/internal/kb"
)

// Hit is one place where a service is referenced.
type Hit struct {
	Rule  string `json:"-"`
	File  string `json:"file"`
	Line  int    `json:"line"`
	Col   int    `json:"col"`
	Kind  string `json:"kind"` // url, package, image
	Match string `json:"match"`
	Where string `json:"where"` // client, server, build, unknown
}

// Options control which files are scanned.
type Options struct {
	Docs     bool     // also scan Markdown and other prose files
	Exclude  []string // extra ignore patterns (.marzignore syntax)
	MaxBytes int64    // skip files larger than this (default 2 MiB)
}

// Scanner matches content against a knowledge base.
type Scanner struct {
	KB   *kb.KB
	Opts Options
	host *regexp.Regexp
}

// New builds a scanner.
func New(k *kb.KB, o Options) *Scanner {
	hosts := k.HostPatterns()
	sort.Slice(hosts, func(i, j int) bool { return len(hosts[i]) > len(hosts[j]) })
	alt := make([]string, len(hosts))
	for i, h := range hosts {
		alt[i] = regexp.QuoteMeta(h)
	}
	re := regexp.MustCompile(`(?i)(?:^|[^a-z0-9.\-_])((?:[a-z0-9-]+\.)*(?:` + strings.Join(alt, "|") + `))(/[^\s"'<>` + "`" + `)\]\\,;|{}]*)?`)
	if o.MaxBytes == 0 {
		o.MaxBytes = 2 << 20
	}
	return &Scanner{KB: k, Opts: o, host: re}
}

var skipDirs = map[string]bool{
	".git": true, ".hg": true, ".svn": true, "node_modules": true, "vendor": true, ".venv": true, "venv": true,
	"__pycache__": true, ".mypy_cache": true, ".pytest_cache": true, ".tox": true, "target": true, "Pods": true,
	".gradle": true, ".dart_tool": true, ".idea": true, ".vscode": true, ".next": true, ".nuxt": true, ".svelte-kit": true,
	".turbo": true, ".cache": true, "coverage": true, ".terraform": true, "bower_components": true,
}

var lockfiles = map[string]bool{
	"package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true, "bun.lockb": true, "bun.lock": true,
	"poetry.lock": true, "Pipfile.lock": true, "uv.lock": true, "go.sum": true, "composer.lock": true, "Gemfile.lock": true,
	"pubspec.lock": true, "Podfile.lock": true, "Cargo.lock": true, "gradle.lockfile": true, ".marzignore": true,
}

var binaryExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".ico": true, ".avif": true, ".bmp": true,
	".woff": true, ".woff2": true, ".ttf": true, ".otf": true, ".eot": true, ".mp4": true, ".webm": true, ".mp3": true,
	".wav": true, ".ogg": true, ".pdf": true, ".zip": true, ".gz": true, ".tgz": true, ".xz": true, ".7z": true, ".rar": true,
	".jar": true, ".class": true, ".so": true, ".dylib": true, ".dll": true, ".exe": true, ".wasm": true, ".map": true,
	".psd": true, ".sketch": true, ".fig": true, ".db": true, ".sqlite": true, ".pyc": true, ".o": true, ".a": true,
}

var docExt = map[string]bool{".md": true, ".mdx": true, ".rst": true, ".adoc": true, ".txt": true, ".org": true}

// Wanted reports whether a file (relative slash path) should be scanned.
func (s *Scanner) Wanted(rel string) bool {
	base := path.Base(rel)
	if lockfiles[base] {
		return false
	}
	ext := strings.ToLower(path.Ext(base))
	if binaryExt[ext] {
		return false
	}
	if docExt[ext] && !s.Opts.Docs && !isRequirements(base) {
		return false
	}
	up := strings.ToUpper(base)
	if strings.HasPrefix(up, "LICENSE") || strings.HasPrefix(up, "LICENCE") || strings.HasPrefix(up, "CHANGELOG") || strings.HasPrefix(up, "NOTICE") {
		return false
	}
	return true
}

func isRequirements(base string) bool {
	return strings.HasPrefix(base, "requirements") && strings.HasSuffix(base, ".txt") || base == "constraints.txt"
}

// Walk scans every wanted file under root.
func (s *Scanner) Walk(root string) ([]Hit, int, error) {
	ign := append(readIgnore(filepath.Join(root, ".marzignore")), s.Opts.Exclude...)
	var hits []Hit
	files := 0
	info, err := os.Stat(root)
	if err != nil {
		return nil, 0, err
	}
	if !info.IsDir() {
		b, err := os.ReadFile(root)
		if err != nil {
			return nil, 0, err
		}
		return s.File(filepath.Base(root), b), 1, nil
	}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if p != root && (skipDirs[d.Name()] || ignored(ign, rel+"/")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || !s.Wanted(rel) || ignored(ign, rel) {
			return nil
		}
		fi, err := d.Info()
		if err != nil || fi.Size() > s.Opts.MaxBytes {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		files++
		hits = append(hits, s.File(rel, b)...)
		return nil
	})
	return hits, files, err
}

func readIgnore(p string) []string {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "#") {
			out = append(out, l)
		}
	}
	return out
}

// ignored implements a small subset of gitignore: "dir/", "*.ext", "path/glob".
func ignored(pats []string, rel string) bool {
	for _, p := range pats {
		p = strings.TrimPrefix(p, "/")
		if strings.HasSuffix(p, "/") {
			d := strings.TrimSuffix(p, "/")
			if strings.HasPrefix(rel, d+"/") || strings.Contains(rel, "/"+d+"/") {
				return true
			}
			continue
		}
		r := strings.TrimSuffix(rel, "/")
		if ok, _ := path.Match(p, r); ok {
			return true
		}
		if !strings.Contains(p, "/") {
			if ok, _ := path.Match(p, path.Base(r)); ok {
				return true
			}
		}
		if strings.HasPrefix(r, p+"/") {
			return true
		}
	}
	return false
}

// File scans one file's content. rel is used for reporting and to pick a
// manifest parser.
func (s *Scanner) File(rel string, b []byte) []Hit {
	if bytes.IndexByte(b[:min(len(b), 8000)], 0) >= 0 {
		return nil
	}
	where := Classify(rel)
	var hits []Hit
	seen := map[string]bool{}
	add := func(h Hit) {
		k := h.Rule + "\x00" + itoa(h.Line)
		if seen[k] {
			return
		}
		seen[k] = true
		h.File, h.Where = rel, where
		hits = append(hits, h)
	}
	lines := strings.Split(string(b), "\n")
	for i, line := range lines {
		if strings.Contains(line, "marz-ignore") {
			continue
		}
		for _, m := range s.host.FindAllStringSubmatchIndex(line, -1) {
			host := strings.ToLower(line[m[2]:m[3]])
			end := m[3]
			if end < len(line) {
				c := line[end]
				if c == '-' || c == '_' || isAlnum(c) {
					continue
				}
			}
			p := ""
			if m[4] >= 0 {
				p = line[m[4]:m[5]]
			}
			if r, _ := s.KB.MatchURL(host, p); r != nil {
				add(Hit{Rule: r.ID, Line: i + 1, Col: m[2] + 1, Kind: "url", Match: host + trimPath(p)})
			}
		}
	}
	for _, d := range Manifest(rel, lines) {
		if d.Host != "" {
			if r, _ := s.KB.MatchURL(d.Host, ""); r != nil {
				add(Hit{Rule: r.ID, Line: d.Line, Col: 1, Kind: d.Kind, Match: d.Name})
			}
			continue
		}
		if r := s.KB.MatchPackage(d.Eco, d.Name); r != nil {
			add(Hit{Rule: r.ID, Line: d.Line, Col: 1, Kind: "package", Match: d.Eco + ":" + d.Name})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Line < hits[j].Line })
	return hits
}

func trimPath(p string) string {
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	if len(p) > 48 {
		p = p[:48] + "…"
	}
	return p
}

func isAlnum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// Classify guesses whether a file runs in the browser/app (client), on a
// server, or in the build/CI pipeline.
func Classify(rel string) string {
	l := strings.ToLower(rel)
	base := path.Base(l)
	ext := path.Ext(base)
	switch {
	case strings.HasPrefix(l, ".github/") || strings.HasPrefix(l, ".gitlab") || strings.HasPrefix(base, "dockerfile") ||
		strings.HasSuffix(base, ".dockerfile") || strings.Contains(base, "compose") && (ext == ".yml" || ext == ".yaml") ||
		base == "pubspec.yaml" || base == "jenkinsfile" || base == ".gitlab-ci.yml" || base == "makefile":
		return "build"
	}
	for _, seg := range []string{"server/", "backend/", "api/", "functions/", "worker/", "workers/", "cmd/", "internal/", "lambda/", "cron/", "jobs/"} {
		if strings.HasPrefix(l, seg) || strings.Contains(l, "/"+seg) {
			return "server"
		}
	}
	switch ext {
	case ".html", ".htm", ".css", ".scss", ".sass", ".less", ".vue", ".svelte", ".jsx", ".tsx", ".astro", ".dart", ".swift", ".m", ".xml", ".plist", ".twig", ".blade.php", ".erb", ".ejs", ".hbs", ".liquid":
		return "client"
	case ".go", ".py", ".rb", ".php", ".rs", ".cs", ".java", ".ex", ".exs", ".scala", ".sql", ".tf":
		if strings.Contains(l, "android/") && ext == ".java" {
			return "client"
		}
		return "server"
	case ".kt", ".kts", ".gradle":
		return "client"
	}
	for _, seg := range []string{"public/", "static/", "src/components/", "src/pages/", "app/", "pages/", "components/", "web/", "frontend/", "client/", "ios/", "android/", "lib/"} {
		if strings.HasPrefix(l, seg) || strings.Contains(l, "/"+seg) {
			return "client"
		}
	}
	return "unknown"
}
