// Package report turns scan hits into per-country findings and renders them
// as text, JSON, Markdown or SARIF.
package report

import (
	"sort"

	"github.com/mrzroot/marz/internal/kb"
	"github.com/mrzroot/marz/internal/scan"
)

// Verdict is a rule's status in one country, resolved for output.
type Verdict struct {
	Country      string        `json:"country"`
	Status       string        `json:"status"`
	Severity     string        `json:"severity,omitempty"`
	Note         string        `json:"note,omitempty"`
	Alternatives []string      `json:"alternatives,omitempty"`
	Sources      []kb.Source   `json:"sources,omitempty"`
	Live         *LiveEvidence `json:"live,omitempty"`
}

// LiveEvidence is optional data from OONI or a local probe.
type LiveEvidence struct {
	Source  string `json:"source"`
	Summary string `json:"summary"`
	URL     string `json:"url,omitempty"`
}

// Finding groups every hit of one rule.
type Finding struct {
	Rule     string     `json:"rule"`
	Name     string     `json:"name"`
	Category string     `json:"category"`
	Impact   string     `json:"impact"`
	Why      string     `json:"why"`
	WhyFA    string     `json:"why_fa"`
	Fix      string     `json:"fix,omitempty"`
	Worst    string     `json:"worst_severity,omitempty"`
	Verdicts []Verdict  `json:"verdicts"`
	Hits     []scan.Hit `json:"hits"`
}

// CountrySummary is the headline for one country.
type CountrySummary struct {
	Country string         `json:"country"`
	Name    string         `json:"name"`
	Score   int            `json:"score"`
	Counts  map[string]int `json:"counts"`
	Breaks  []string       `json:"breaks"`
}

// Result is a whole scan.
type Result struct {
	Tool      string           `json:"tool"`
	Version   string           `json:"version"`
	KBUpdated string           `json:"kb_updated"`
	Target    string           `json:"target"`
	Files     int              `json:"files_scanned"`
	Countries []string         `json:"countries"`
	Summary   []CountrySummary `json:"summary"`
	Findings  []Finding        `json:"findings"`
}

var weight = map[string]int{"critical": 30, "high": 15, "medium": 7, "low": 3}

// Build assembles a Result.
func Build(k *kb.KB, target string, files int, hits []scan.Hit, countries []string, version string) *Result {
	res := &Result{Tool: "marz", Version: version, KBUpdated: k.Updated, Target: target, Files: files, Countries: countries}
	by := map[string]*Finding{}
	var order []string
	for _, h := range hits {
		f := by[h.Rule]
		if f == nil {
			r := k.Rule(h.Rule)
			f = &Finding{Rule: r.ID, Name: r.Name, Category: r.Category, Impact: r.Impact, Why: r.Why, WhyFA: r.FA, Fix: r.Fix}
			for _, c := range countries {
				v, ok := r.Countries[c]
				if !ok {
					v = kb.Verdict{Status: kb.Unknown}
				}
				out := Verdict{Country: c, Status: v.Status, Severity: r.Severity(c), Note: v.Note, Alternatives: v.Alternatives}
				for _, s := range v.Sources {
					out.Sources = append(out.Sources, k.Sources[s])
				}
				f.Verdicts = append(f.Verdicts, out)
			}
			by[h.Rule] = f
			order = append(order, h.Rule)
		}
		f.Hits = append(f.Hits, h)
	}
	for _, id := range order {
		f := by[id]
		f.Worst = worst(f.Verdicts)
		res.Findings = append(res.Findings, *f)
	}
	sort.SliceStable(res.Findings, func(i, j int) bool {
		a, b := kb.LevelRank(res.Findings[i].Worst), kb.LevelRank(res.Findings[j].Worst)
		if a != b {
			return a < b
		}
		return res.Findings[i].Name < res.Findings[j].Name
	})
	for _, c := range countries {
		s := CountrySummary{Country: c, Name: kb.CountryNames[c], Counts: map[string]int{}}
		penalty := 0
		for _, f := range res.Findings {
			for _, v := range f.Verdicts {
				if v.Country != c || v.Severity == "" {
					continue
				}
				s.Counts[v.Severity]++
				penalty += weight[v.Severity]
				if v.Severity != "info" {
					s.Breaks = append(s.Breaks, f.Name)
				}
			}
		}
		// 100 when nothing breaks; one critical service ≈ 62, three ≈ 36.
		s.Score = int(100/(1+float64(penalty)/50) + 0.5)
		res.Summary = append(res.Summary, s)
	}
	return res
}

func worst(vs []Verdict) string {
	best := ""
	for _, v := range vs {
		if v.Severity != "" && (best == "" || kb.LevelRank(v.Severity) < kb.LevelRank(best)) {
			best = v.Severity
		}
	}
	return best
}

// Fails reports whether any finding is at or above the threshold
// (critical, high, medium, low; "never" disables).
func (r *Result) Fails(threshold string) bool {
	if threshold == "" || threshold == "never" || threshold == "none" {
		return false
	}
	t := kb.LevelRank(threshold)
	for _, f := range r.Findings {
		for _, v := range f.Verdicts {
			if v.Severity != "" && v.Severity != "info" && kb.LevelRank(v.Severity) <= t {
				return true
			}
		}
	}
	return false
}
