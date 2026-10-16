// Command marz tells you which parts of your app will break in Iran, China
// or Russia (blocked, sanctioned or unavailable services), with sources and
// alternatives, and can probe reachability from inside the country.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mrzroot/marz/internal/fonts"
	"github.com/mrzroot/marz/internal/kb"
	"github.com/mrzroot/marz/internal/ooni"
	"github.com/mrzroot/marz/internal/probe"
	"github.com/mrzroot/marz/internal/report"
	"github.com/mrzroot/marz/internal/scan"
)

var version = "dev"

const usage = `marz: will your app work in Iran, China and Russia?

Usage:
  marz [scan] [path]          scan a project (default: current directory)
  marz probe <host|url>...    test reachability from this network, step by step
  marz probe --from [path]    probe every service marz finds in a project
  marz check <domain>...      live OONI measurements from inside each country
  marz fix fonts [path]       self-host Google Fonts (dry run; add --write)
  marz rules                  list the knowledge base
  marz version

Scan flags:
  --country IR,CN,RU   countries to check (default all)
  --format text|json|md|sarif
  --fail-on critical|high|medium|low|never   exit 1 at or above this severity
  --ooni               attach live OONI evidence (network)
  --docs               also scan Markdown and other prose
  --exclude GLOB       skip paths (repeatable; also reads .marzignore)
  -o FILE              write the report to FILE
  -v                   every location and source link

Docs: https://mrzroot.github.io/marz/
`

type flags struct {
	pos     []string
	country []string
	format  string
	failOn  string
	out     string
	exclude []string
	docs    bool
	verbose bool
	ooni    bool
	write   bool
	jsonOut bool
	from    string
	fromSet bool
	dir     string
	days    int
	timeout time.Duration
	noColor bool
}

func parse(args []string) (*flags, error) {
	f := &flags{format: "text", failOn: "never", days: 30, timeout: 10 * time.Second}
	need := func(i *int, name string) (string, error) {
		if *i+1 >= len(args) {
			return "", fmt.Errorf("%s needs a value", name)
		}
		*i++
		return args[*i], nil
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, val, hasVal := strings.Cut(a, "=")
		get := func() (string, error) {
			if hasVal {
				return val, nil
			}
			return need(&i, name)
		}
		var err error
		var v string
		switch name {
		case "--country", "--countries", "-c":
			if v, err = get(); err == nil {
				f.country, err = countries(v)
			}
		case "--format", "-f":
			f.format, err = get()
		case "--fail-on":
			f.failOn, err = get()
		case "-o", "--output":
			f.out, err = get()
		case "--exclude":
			if v, err = get(); err == nil {
				f.exclude = append(f.exclude, v)
			}
		case "--dir":
			f.dir, err = get()
		case "--days":
			if v, err = get(); err == nil {
				_, err = fmt.Sscan(v, &f.days)
			}
		case "--timeout":
			if v, err = get(); err == nil {
				f.timeout, err = time.ParseDuration(v)
			}
		case "--from":
			f.fromSet = true
			if hasVal {
				f.from = val
			} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				f.from = args[i]
			} else {
				f.from = "."
			}
		case "--docs":
			f.docs = true
		case "--ooni":
			f.ooni = true
		case "--write", "-w":
			f.write = true
		case "--json":
			f.jsonOut = true
		case "--no-color":
			f.noColor = true
		case "-v", "--verbose":
			f.verbose = true
		case "-h", "--help":
			return nil, errHelp
		default:
			if strings.HasPrefix(a, "-") && a != "-" {
				return nil, fmt.Errorf("unknown flag %s", a)
			}
			f.pos = append(f.pos, a)
		}
		if err != nil {
			return nil, err
		}
	}
	if len(f.country) == 0 {
		f.country = kb.Countries
	}
	switch f.format {
	case "text", "json", "md", "markdown", "sarif":
	default:
		return nil, fmt.Errorf("unknown format %q (text, json, md, sarif)", f.format)
	}
	switch f.failOn {
	case "critical", "high", "medium", "low", "never", "none":
	default:
		return nil, fmt.Errorf("unknown --fail-on %q", f.failOn)
	}
	return f, nil
}

var errHelp = errors.New("help")

func countries(v string) ([]string, error) {
	if strings.EqualFold(v, "all") {
		return kb.Countries, nil
	}
	var out []string
	for _, c := range strings.Split(v, ",") {
		c = strings.ToUpper(strings.TrimSpace(c))
		if c == "" {
			continue
		}
		if kb.CountryNames[c] == "" {
			return nil, fmt.Errorf("unknown country %q (supported: IR, CN, RU)", c)
		}
		out = append(out, c)
	}
	return out, nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	cmd := "scan"
	if len(args) > 0 {
		switch args[0] {
		case "scan", "probe", "check", "fix", "rules", "version", "help":
			cmd, args = args[0], args[1:]
		case "--version":
			cmd, args = "version", args[1:]
		}
	}
	if cmd == "help" {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if cmd == "version" {
		fmt.Fprintf(stdout, "marz %s (knowledge base %s)\n", version, kb.MustLoad().Updated)
		return 0
	}
	f, err := parse(args)
	if errors.Is(err, errHelp) {
		fmt.Fprint(stdout, usage)
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "marz: %v\n", err)
		return 2
	}
	if b := os.Getenv("MARZ_FONTS_BASE"); b != "" {
		fonts.Base = b
	}
	k := kb.MustLoad()
	switch cmd {
	case "scan":
		return cmdScan(ctx, k, f, stdout, stderr)
	case "probe":
		return cmdProbe(ctx, k, f, stdout, stderr)
	case "check":
		return cmdCheck(ctx, f, stdout, stderr)
	case "fix":
		return cmdFix(ctx, f, stdout, stderr)
	case "rules":
		return cmdRules(k, f, stdout)
	}
	return 2
}

func color(w io.Writer, f *flags) bool {
	if f.noColor || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := file.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}

func doScan(k *kb.KB, f *flags, target string) (*report.Result, error) {
	s := scan.New(k, scan.Options{Docs: f.docs, Exclude: f.exclude})
	hits, files, err := s.Walk(target)
	if err != nil {
		return nil, err
	}
	abs, _ := filepath.Abs(target)
	name := filepath.Base(abs)
	return report.Build(k, name, files, hits, f.country, version), nil
}

func cmdScan(ctx context.Context, k *kb.KB, f *flags, stdout, stderr io.Writer) int {
	target := "."
	if len(f.pos) > 0 {
		target = f.pos[0]
	}
	res, err := doScan(k, f, target)
	if err != nil {
		fmt.Fprintf(stderr, "marz: %v\n", err)
		return 2
	}
	if f.ooni {
		attachOONI(ctx, k, res, stderr)
	}
	w := stdout
	if f.out != "" {
		file, err := os.Create(f.out)
		if err != nil {
			fmt.Fprintf(stderr, "marz: %v\n", err)
			return 2
		}
		defer file.Close()
		w = file
	}
	switch f.format {
	case "json":
		report.JSON(w, res)
	case "md", "markdown":
		report.Markdown(w, res)
	case "sarif":
		report.SARIF(w, res)
	default:
		report.Text(w, res, report.Style{Color: color(w, f)}, f.verbose)
	}
	if res.Fails(f.failOn) {
		fmt.Fprintf(stderr, "marz: found services at or above %q severity\n", f.failOn)
		return 1
	}
	return 0
}

func hostOf(match string) string {
	h := match
	if i := strings.Index(h, "/"); i >= 0 {
		h = h[:i]
	}
	return h
}

func attachOONI(ctx context.Context, k *kb.KB, res *report.Result, stderr io.Writer) {
	c := &ooni.Client{Base: os.Getenv("MARZ_OONI_BASE")}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for fi := range res.Findings {
		fd := &res.Findings[fi]
		host := ""
		for _, h := range fd.Hits {
			if h.Kind == "url" {
				host = hostOf(h.Match)
				// OONI measures well-known domains, not "my-app.firebaseapp.com".
				if _, pat := k.MatchURL(host, ""); pat != "" && !strings.Contains(pat, "/") {
					host = pat
				}
				break
			}
		}
		if r := k.Rule(fd.Rule); r != nil && r.OONI != "" {
			host = r.OONI
		}
		if host == "" {
			continue
		}
		for vi := range fd.Verdicts {
			v := &fd.Verdicts[vi]
			wg.Add(1)
			go func(host string, v *report.Verdict) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				s, err := c.Aggregate(ctx, v.Country, host, 30, time.Now())
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					fmt.Fprintf(stderr, "marz: OONI %s/%s: %v\n", v.Country, host, err)
					return
				}
				v.Live = &report.LiveEvidence{Source: "OONI " + host, Summary: s.Summary(), URL: s.URL}
			}(host, v)
		}
	}
	wg.Wait()
}

func cmdProbe(ctx context.Context, k *kb.KB, f *flags, stdout, stderr io.Writer) int {
	targets := f.pos
	if f.fromSet {
		res, err := doScan(k, f, f.from)
		if err != nil {
			fmt.Fprintf(stderr, "marz: %v\n", err)
			return 2
		}
		seen := map[string]bool{}
		for _, fd := range res.Findings {
			for _, h := range fd.Hits {
				host := hostOf(h.Match)
				if h.Kind == "package" {
					continue
				}
				if h.Kind == "image" {
					host = "registry-1.docker.io"
				}
				if !seen[host] {
					seen[host] = true
					targets = append(targets, host)
				}
			}
		}
	}
	if len(targets) == 0 {
		fmt.Fprintln(stderr, "marz probe: give hosts or URLs, or --from <path>")
		return 2
	}
	st := report.Style{Color: color(stdout, f)}
	where, ip, werr := probe.Where(ctx)
	p := &probe.Prober{Timeout: f.timeout}
	results := make([]probe.Result, len(targets))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for i, t := range targets {
		wg.Add(1)
		go func(i int, t string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = p.Probe(ctx, t)
		}(i, t)
	}
	wg.Wait()
	if f.jsonOut {
		e := json.NewEncoder(stdout)
		e.SetIndent("", "  ")
		e.Encode(map[string]any{"network_country": where, "results": results})
		return 0
	}
	if werr == nil && where != "" {
		name := kb.CountryNames[where]
		if name == "" {
			name = where
		}
		fmt.Fprintf(stdout, "%s probing from %s (%s, %s); direct connections, no proxy\n\n", st.Bold("marz"), name, where, ip)
	} else {
		fmt.Fprintf(stdout, "%s probing from this network (country unknown); direct connections, no proxy\n\n", st.Bold("marz"))
	}
	bad := 0
	for _, r := range results {
		icon, col := "✓", "32"
		if r.Verdict != probe.OK {
			icon, col, bad = "✖", "31", bad+1
		}
		fmt.Fprintf(stdout, "%s %-34s %s %s\n", st.Paint(col, icon), r.Host, st.Paint(col, fmt.Sprintf("%-12s", r.Verdict)), r.Detail)
		if f.verbose {
			for _, s := range r.Steps {
				mark := "ok "
				if !s.OK {
					mark = "ERR"
				}
				fmt.Fprintf(stdout, "    %s %-4s %6s  %s\n", mark, s.Name, s.Took.Round(time.Millisecond), s.Info)
			}
		}
	}
	fmt.Fprintf(stdout, "\n%d/%d reachable.", len(results)-bad, len(results))
	if where != "" && kb.CountryNames[where] == "" {
		fmt.Fprintf(stdout, " Note: this network exits from %s, so results describe %s, not IR/CN/RU.", where, where)
	}
	fmt.Fprintln(stdout)
	return 0
}

func cmdCheck(ctx context.Context, f *flags, stdout, stderr io.Writer) int {
	if len(f.pos) == 0 {
		fmt.Fprintln(stderr, "marz check: give one or more domains, e.g. marz check fonts.googleapis.com")
		return 2
	}
	c := &ooni.Client{Base: os.Getenv("MARZ_OONI_BASE")}
	var all []ooni.Stats
	code := 0
	for _, d := range f.pos {
		d = hostOf(strings.TrimPrefix(strings.TrimPrefix(d, "https://"), "http://"))
		for _, cc := range f.country {
			s, err := c.Aggregate(ctx, cc, d, f.days, time.Now())
			if err != nil {
				fmt.Fprintf(stderr, "marz: OONI %s %s: %v\n", cc, d, err)
				code = 1
				continue
			}
			all = append(all, s)
			if !f.jsonOut {
				fmt.Fprintf(stdout, "%s %-28s %s\n", cc, d, s.Summary())
				if f.verbose && s.Measurements > 0 {
					fmt.Fprintf(stdout, "   %s\n", s.URL)
				}
			}
		}
	}
	if f.jsonOut {
		e := json.NewEncoder(stdout)
		e.SetIndent("", "  ")
		e.Encode(all)
	} else {
		fmt.Fprintln(stdout, "\nSource: OONI web_connectivity measurements (https://ooni.org). Domains outside the OONI test lists have no data; use `marz probe` from inside the country.")
	}
	return code
}

func cmdFix(ctx context.Context, f *flags, stdout, stderr io.Writer) int {
	if len(f.pos) == 0 || f.pos[0] != "fonts" {
		fmt.Fprintln(stderr, "marz fix: only `marz fix fonts [path] [--write]` is supported")
		return 2
	}
	root := "."
	if len(f.pos) > 1 {
		root = f.pos[1]
	}
	plan, err := fonts.Fix(ctx, fonts.Options{Root: root, Dir: f.dir, Write: f.write, Log: func(format string, a ...any) {
		if f.verbose {
			fmt.Fprintf(stderr, format+"\n", a...)
		}
	}})
	if err != nil {
		fmt.Fprintf(stderr, "marz: %v\n", err)
		return 1
	}
	if len(plan.Changes) == 0 {
		fmt.Fprintln(stdout, "No Google Fonts stylesheets found.")
		return 0
	}
	for _, c := range plan.Changes {
		rel, _ := filepath.Rel(root, c.File)
		for i, u := range c.URLs {
			fmt.Fprintf(stdout, "%s\n  - %s\n  + %s\n", rel, u, c.Local[i])
		}
	}
	if !f.write {
		dir := f.dir
		if dir == "" {
			dir = fonts.DefaultDir(root)
		}
		fmt.Fprintf(stdout, "\nDry run: %d file(s) would change. Re-run with --write to download the fonts into %s/ and rewrite them.\n", len(plan.Changes), dir)
		return 0
	}
	if f.dir == "" {
		f.dir = fonts.DefaultDir(root)
	}
	fmt.Fprintf(stdout, "\nSelf-hosted %d font file(s) (%d KB) in %s/ and rewrote %d file(s). Preconnect hints to Google were removed.\n", len(plan.Fonts), plan.Bytes/1024, f.dir, len(plan.Changes))
	return 0
}

func cmdRules(k *kb.KB, f *flags, stdout io.Writer) int {
	if f.jsonOut {
		e := json.NewEncoder(stdout)
		e.SetIndent("", "  ")
		e.Encode(k)
		return 0
	}
	rs := append([]kb.Rule(nil), k.Rules...)
	sort.Slice(rs, func(i, j int) bool { return rs[i].Category+rs[i].Name < rs[j].Category+rs[j].Name })
	fmt.Fprintf(stdout, "%-40s %-11s", "service", "category")
	for _, c := range f.country {
		fmt.Fprintf(stdout, " %-12s", c)
	}
	fmt.Fprintln(stdout)
	for _, r := range rs {
		fmt.Fprintf(stdout, "%-40s %-11s", r.Name, r.Category)
		for _, c := range f.country {
			fmt.Fprintf(stdout, " %-12s", r.Countries[c].Status)
		}
		fmt.Fprintln(stdout)
	}
	fmt.Fprintf(stdout, "\n%d services · updated %s · `marz rules --json` for sources and alternatives\n", len(rs), k.Updated)
	return 0
}
