package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runT runs the CLI in-process and returns the exit code, stdout and stderr.
func runT(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

// demo is a small project with known findings (Firebase, Neshan, ...).
const demo = "../../testdata/demo"

func TestScanText(t *testing.T) {
	code, out, _ := runT(t, demo)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"CRITICAL Firebase", "IR ⊘ sanctioned", "Neshan", "src/firebase.ts:6", "IR Iran", "/100"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(out, "\x1b[") {
		t.Error("colour codes written to a non-terminal")
	}
}

func TestScanJSONForOneCountry(t *testing.T) {
	code, out, _ := runT(t, "scan", demo, "--format", "json", "--country", "ru")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	var res struct {
		Countries []string
		Summary   []struct{ Score int }
		Findings  []struct {
			Rule     string
			Verdicts []struct{ Country, Status string }
		}
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Countries) != 1 || res.Countries[0] != "RU" || len(res.Findings) < 8 {
		t.Fatalf("%+v", res)
	}
}

func TestFailOnCriticalExitsOne(t *testing.T) {
	code, _, stderr := runT(t, demo, "--fail-on", "critical", "-f", "json")
	if code != 1 || !strings.Contains(stderr, "critical") {
		t.Fatalf("fail-on exit %d %q", code, stderr)
	}
}

// Informational findings (a plain Dockerfile) must not trip --fail-on low on an otherwise clean directory.
func TestFailOnLowIgnoresInfoFindings(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := runT(t, dir, "--fail-on", "low"); code != 0 {
		t.Fatalf("info finding failed the build: %d", code)
	}
}

func TestSARIFAndMarkdown(t *testing.T) {
	out := filepath.Join(t.TempDir(), "r.sarif")
	if code, _, _ := runT(t, demo, "--format", "sarif", "-o", out); code != 0 {
		t.Fatal(code)
	}
	b, _ := os.ReadFile(out)
	var doc struct {
		Version string
		Runs    []struct {
			Results []struct {
				RuleID string
				Level  string
			}
		}
	}
	if err := json.Unmarshal(b, &doc); err != nil || doc.Version != "2.1.0" || len(doc.Runs[0].Results) < 10 {
		t.Fatalf("sarif %v %+v", err, doc)
	}
	_, md, _ := runT(t, demo, "--format", "md", "--country", "IR")
	if !strings.Contains(md, "| 🇮🇷 Iran |") || !strings.Contains(md, "<details>") {
		t.Fatalf("md:\n%s", md)
	}
}

func TestClean(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<script src="/app.js"></script>`), 0o644)
	_, out, _ := runT(t, dir)
	if !strings.Contains(out, "No known region-sensitive services") {
		t.Fatal(out)
	}
}

func TestBadInput(t *testing.T) {
	for _, args := range [][]string{{"--country", "FR"}, {"--format", "xml"}, {"--fail-on", "sometimes"}, {"--nope"}, {"scan", "/does/not/exist"}} {
		if code, _, _ := runT(t, args...); code != 2 {
			t.Errorf("%v: exit %d", args, code)
		}
	}
}

func TestRulesAndVersion(t *testing.T) {
	_, out, _ := runT(t, "rules", "--country", "IR")
	if !strings.Contains(out, "Google Maps Platform") || !strings.Contains(out, "sanctioned") {
		t.Fatal(out)
	}
	_, out, _ = runT(t, "version")
	if !strings.HasPrefix(out, "marz ") {
		t.Fatal(out)
	}
	_, out, _ = runT(t, "--help")
	if !strings.Contains(out, "marz probe") {
		t.Fatal(out)
	}
}

func TestCheckWithFakeOONI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"result":{"anomaly_count":10,"confirmed_count":80,"failure_count":0,"ok_count":10,"measurement_count":100}}`))
	}))
	defer srv.Close()
	t.Setenv("MARZ_OONI_BASE", srv.URL)
	code, out, _ := runT(t, "check", "https://www.youtube.com/watch", "--country", "IR")
	if code != 0 || !strings.Contains(out, "IR www.youtube.com") || !strings.Contains(out, "blocked: 100 measurements, 90% anomalous") {
		t.Fatalf("%d\n%s", code, out)
	}
	_, out, _ = runT(t, demo, "--ooni", "--country", "IR")
	if !strings.Contains(out, "OONI fonts.googleapis.com:") {
		t.Fatalf("no live evidence:\n%s", out)
	}
}

func TestFixFontsCLI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/css2" {
			w.Write([]byte("@font-face { font-family: 'Vazirmatn'; src: url(https://fonts.gstatic.com/s/v/1.woff2) format('woff2'); }"))
			return
		}
		w.Write([]byte("wOF2"))
	}))
	defer srv.Close()
	t.Setenv("MARZ_FONTS_BASE", srv.URL)
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "public"), 0o755)
	os.WriteFile(filepath.Join(dir, "public", "index.html"), []byte(`<link href="https://fonts.googleapis.com/css2?family=Vazirmatn&display=swap" rel="stylesheet">`), 0o644)
	code, out, _ := runT(t, "fix", "fonts", dir, "--write")
	if code != 0 || !strings.Contains(out, "public/fonts/") {
		t.Fatalf("%d %s", code, out)
	}
	_, out, _ = runT(t, dir, "--country", "CN")
	if !strings.Contains(out, "No known region-sensitive services") {
		t.Fatalf("still flagged after fix:\n%s", out)
	}
}
