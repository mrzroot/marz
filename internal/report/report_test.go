package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mrzroot/marz/internal/kb"
	"github.com/mrzroot/marz/internal/scan"
)

func TestBuildScoresAndOrder(t *testing.T) {
	k := kb.MustLoad()
	hits := []scan.Hit{
		{Rule: "jsdelivr", File: "a.html", Line: 1, Match: "cdn.jsdelivr.net"},
		{Rule: "firebase", File: "a.js", Line: 2, Match: "npm:firebase"},
		{Rule: "firebase", File: "b.js", Line: 3, Match: "x.firebaseio.com"},
	}
	r := Build(k, "t", 2, hits, []string{"IR", "CN", "RU"}, "test")
	if r.Findings[0].Rule != "firebase" || len(r.Findings[0].Hits) != 2 || r.Findings[0].Worst != "critical" {
		t.Fatalf("%+v", r.Findings[0])
	}
	scores := map[string]int{}
	for _, s := range r.Summary {
		scores[s.Country] = s.Score
	}
	// IR: one critical (30) -> 100/(1+0.6) = 62.5 -> 63; RU: firebase unknown (info), jsdelivr ok -> 100
	if scores["IR"] != 63 || scores["RU"] != 100 || scores["CN"] >= scores["IR"] {
		t.Fatalf("scores %v", scores)
	}
	if !r.Fails("critical") || r.Fails("never") {
		t.Fatal("Fails")
	}
	clean := Build(k, "t", 1, []scan.Hit{{Rule: "docker-hub", File: "Dockerfile", Line: 1}}, []string{"IR"}, "test")
	if clean.Fails("low") {
		t.Fatal("ok status must not fail")
	}
	var b bytes.Buffer
	Text(&b, r, Style{}, true)
	for _, want := range []string{"CRITICAL", "IR ⊘ sanctioned", "https://cloud.google.com/terms", "63/100"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("text missing %q", want)
		}
	}
	b.Reset()
	if err := SARIF(&b, r); err != nil || !strings.Contains(b.String(), `"security-severity": "9.0"`) {
		t.Fatalf("sarif %v %s", err, b.String())
	}
}
