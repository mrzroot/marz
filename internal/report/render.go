package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Style controls terminal colours.
type Style struct{ Color bool }

func (s Style) c(code, t string) string {
	if !s.Color {
		return t
	}
	return "\x1b[" + code + "m" + t + "\x1b[0m"
}

var statusColor = map[string]string{"blocked": "1;31", "sanctioned": "1;35", "unavailable": "1;33", "degraded": "33", "unknown": "2", "ok": "32"}

var statusIcon = map[string]string{"blocked": "✖", "sanctioned": "⊘", "unavailable": "✖", "degraded": "◐", "unknown": "?", "ok": "✓"}

// Text renders the human report.
func Text(w io.Writer, r *Result, st Style, verbose bool) {
	fmt.Fprintf(w, "%s %s · will it work in %s?\n", st.c("1", "marz"), r.Version, strings.Join(r.Countries, ", "))
	fmt.Fprintf(w, "%s %s · %d files · knowledge base %s\n\n", st.c("2", "scanned"), r.Target, r.Files, r.KBUpdated)
	if len(r.Findings) == 0 {
		fmt.Fprintln(w, st.c("32", "✓ No known region-sensitive services found."))
		fmt.Fprintln(w, st.c("2", "  marz only knows the services in its knowledge base; run `marz probe <host>` for anything else."))
		return
	}
	for _, f := range r.Findings {
		sev := f.Worst
		if sev == "" {
			sev = "ok"
		}
		fmt.Fprintf(w, "%s %s %s\n", sevBadge(st, sev), st.c("1", f.Name), st.c("2", "("+f.Category+")"))
		var cells []string
		for _, v := range f.Verdicts {
			cells = append(cells, st.c(statusColor[v.Status], v.Country+" "+statusIcon[v.Status]+" "+v.Status))
		}
		fmt.Fprintf(w, "   %s\n", strings.Join(cells, "   "))
		fmt.Fprintf(w, "   %s\n", f.Why)
		n := len(f.Hits)
		show := f.Hits
		if !verbose && n > 3 {
			show = f.Hits[:3]
		}
		for _, h := range show {
			tag := ""
			if h.Where != "unknown" {
				tag = " [" + h.Where + "]"
			}
			fmt.Fprintf(w, "   %s %s\n", st.c("36", fmt.Sprintf("%s:%d", h.File, h.Line)), st.c("2", h.Match+tag))
		}
		if len(show) < n {
			fmt.Fprintf(w, "   %s\n", st.c("2", fmt.Sprintf("… and %d more (use -v)", n-len(show))))
		}
		for _, v := range f.Verdicts {
			if (v.Severity == "" || v.Severity == "info" && !verbose) && v.Live == nil {
				continue
			}
			line := v.Country + ": "
			if v.Note != "" {
				line += v.Note + " "
			}
			if len(v.Alternatives) > 0 {
				line += "→ " + strings.Join(v.Alternatives, " · ")
			}
			fmt.Fprintf(w, "   %s\n", line)
			if verbose {
				for _, s := range v.Sources {
					fmt.Fprintf(w, "      %s\n", st.c("2", s.URL))
				}
			}
			if v.Live != nil {
				fmt.Fprintf(w, "      %s %s\n", st.c("34", v.Live.Source+":"), v.Live.Summary)
			}
		}
		if f.Fix != "" {
			fmt.Fprintf(w, "   %s %s\n", st.c("32", "fix:"), f.Fix)
		}
		fmt.Fprintln(w)
	}
	for _, s := range r.Summary {
		bar := scoreBar(s.Score)
		col := "32"
		switch {
		case s.Score < 50:
			col = "31"
		case s.Score < 80:
			col = "33"
		}
		brk := "nothing known to break"
		if len(s.Breaks) > 0 {
			brk = "breaks: " + strings.Join(s.Breaks, ", ")
		}
		fmt.Fprintf(w, "%s %-7s %s %s  %s\n", st.c("1", s.Country), s.Name, st.c(col, bar), st.c(col, fmt.Sprintf("%3d/100", s.Score)), st.c("2", brk))
	}
	fmt.Fprintln(w, st.c("2", "\nStatuses come from public sources (run with -v for links). Confirm from inside the country with `marz probe`."))
}

func sevBadge(st Style, s string) string {
	col := map[string]string{"critical": "1;41;97", "high": "1;31", "medium": "1;33", "low": "36", "info": "2", "ok": "32"}[s]
	return st.c(col, fmt.Sprintf("%-8s", strings.ToUpper(s)))
}

func scoreBar(score int) string {
	n := score / 10
	return strings.Repeat("█", n) + strings.Repeat("░", 10-n)
}

// JSON renders machine-readable output.
func JSON(w io.Writer, r *Result) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(r)
}

// Markdown renders a report for PR comments and job summaries.
func Markdown(w io.Writer, r *Result) {
	fmt.Fprintf(w, "## marz: will it work in %s?\n\n", strings.Join(r.Countries, ", "))
	fmt.Fprintf(w, "| Country | Score | What breaks |\n|---|---|---|\n")
	for _, s := range r.Summary {
		b := "nothing known"
		if len(s.Breaks) > 0 {
			b = strings.Join(s.Breaks, ", ")
		}
		fmt.Fprintf(w, "| %s %s | **%d**/100 | %s |\n", flag(s.Country), s.Name, s.Score, b)
	}
	if len(r.Findings) == 0 {
		fmt.Fprintf(w, "\nNo known region-sensitive services found in %d files.\n", r.Files)
		return
	}
	fmt.Fprintf(w, "\n| Service | %s | Where |\n|---|%s---|\n", strings.Join(r.Countries, " | "), strings.Repeat("---|", len(r.Countries)))
	for _, f := range r.Findings {
		var cells []string
		for _, v := range f.Verdicts {
			cells = append(cells, statusIcon[v.Status]+" "+v.Status)
		}
		h := f.Hits[0]
		where := fmt.Sprintf("`%s:%d`", h.File, h.Line)
		if len(f.Hits) > 1 {
			where += fmt.Sprintf(" +%d", len(f.Hits)-1)
		}
		fmt.Fprintf(w, "| **%s** | %s | %s |\n", f.Name, strings.Join(cells, " | "), where)
	}
	fmt.Fprintf(w, "\n<details><summary>Fixes and alternatives</summary>\n\n")
	for _, f := range r.Findings {
		if f.Worst == "" || f.Worst == "info" {
			continue
		}
		fmt.Fprintf(w, "**%s**: %s\n", f.Name, f.Why)
		for _, v := range f.Verdicts {
			if v.Severity == "" || v.Severity == "info" {
				continue
			}
			var src []string
			for _, s := range v.Sources {
				src = append(src, "["+shortTitle(s.Title)+"]("+s.URL+")")
			}
			fmt.Fprintf(w, "- %s %s: %s %s", v.Country, v.Status, v.Note, strings.Join(v.Alternatives, ", "))
			if len(src) > 0 {
				fmt.Fprintf(w, " (sources: %s)", strings.Join(src, ", "))
			}
			fmt.Fprintln(w)
		}
		if f.Fix != "" {
			fmt.Fprintf(w, "- Fix: %s\n", f.Fix)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "</details>\n\n<sub>marz %s · knowledge base %s · %d files scanned</sub>\n", r.Version, r.KBUpdated, r.Files)
}

func shortTitle(t string) string {
	if i := strings.Index(t, ":"); i > 0 && i < 30 {
		return t[:i]
	}
	if i := strings.Index(t, " ("); i > 0 {
		return t[:i]
	}
	return t
}

func flag(c string) string {
	if len(c) != 2 {
		return ""
	}
	return string(rune(0x1F1E6+int(c[0]-'A'))) + string(rune(0x1F1E6+int(c[1]-'A')))
}

// Bold and Paint are small helpers for other commands.
func (s Style) Bold(t string) string        { return s.c("1", t) }
func (s Style) Paint(code, t string) string { return s.c(code, t) }
