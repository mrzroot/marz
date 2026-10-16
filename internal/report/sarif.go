package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// SARIF renders a SARIF 2.1.0 log for GitHub code scanning.
func SARIF(w io.Writer, r *Result) error {
	type msg struct {
		Text string `json:"text"`
	}
	type rule struct {
		ID               string         `json:"id"`
		Name             string         `json:"name"`
		ShortDescription msg            `json:"shortDescription"`
		FullDescription  msg            `json:"fullDescription"`
		Help             msg            `json:"help"`
		Properties       map[string]any `json:"properties"`
	}
	type region struct {
		StartLine   int `json:"startLine"`
		StartColumn int `json:"startColumn"`
	}
	type loc struct {
		PhysicalLocation struct {
			ArtifactLocation struct {
				URI string `json:"uri"`
			} `json:"artifactLocation"`
			Region region `json:"region"`
		} `json:"physicalLocation"`
	}
	type result struct {
		RuleID    string `json:"ruleId"`
		Level     string `json:"level"`
		Message   msg    `json:"message"`
		Locations []loc  `json:"locations"`
	}
	var rules []rule
	var results []result
	for _, f := range r.Findings {
		if f.Worst == "" || f.Worst == "info" {
			continue
		}
		var parts, help []string
		for _, v := range f.Verdicts {
			if v.Severity == "" || v.Severity == "info" {
				continue
			}
			parts = append(parts, v.Country+" "+v.Status)
			h := v.Country + ": " + v.Status
			if len(v.Alternatives) > 0 {
				h += " → " + strings.Join(v.Alternatives, ", ")
			}
			for _, s := range v.Sources {
				h += " " + s.URL
			}
			help = append(help, h)
		}
		if f.Fix != "" {
			help = append(help, "Fix: "+f.Fix)
		}
		rules = append(rules, rule{
			ID: "marz/" + f.Rule, Name: f.Name,
			ShortDescription: msg{f.Name + " is " + strings.Join(parts, ", ")},
			FullDescription:  msg{f.Why},
			Help:             msg{strings.Join(help, "\n")},
			Properties:       map[string]any{"tags": []string{"region", f.Category}, "security-severity": secSev(f.Worst)},
		})
		for _, h := range f.Hits {
			var l loc
			l.PhysicalLocation.ArtifactLocation.URI = h.File
			l.PhysicalLocation.Region = region{h.Line, max(h.Col, 1)}
			results = append(results, result{
				RuleID: "marz/" + f.Rule, Level: level(f.Worst),
				Message:   msg{fmt.Sprintf("%s (%s) will not work in: %s.", f.Name, h.Match, strings.Join(parts, ", "))},
				Locations: []loc{l},
			})
		}
	}
	doc := map[string]any{
		"$schema": "https://json.schemastore.org/sarif-2.1.0.json",
		"version": "2.1.0",
		"runs": []any{map[string]any{
			"tool": map[string]any{"driver": map[string]any{
				"name": "marz", "version": r.Version, "informationUri": "https://github.com/mrzroot/marz", "rules": nonNil(rules),
			}},
			"results": nonNil(results),
		}},
	}
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(doc)
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func level(s string) string {
	switch s {
	case "critical", "high":
		return "error"
	case "medium":
		return "warning"
	}
	return "note"
}

func secSev(s string) string {
	return map[string]string{"critical": "9.0", "high": "7.0", "medium": "5.0", "low": "2.0"}[s]
}
