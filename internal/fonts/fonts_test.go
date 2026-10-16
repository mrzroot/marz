package fonts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFix(t *testing.T) {
	var uas []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uas = append(uas, r.UserAgent())
		switch {
		case r.URL.Path == "/css2":
			if r.URL.Query().Get("display") != "swap" {
				t.Errorf("query lost: %s", r.URL.RawQuery)
			}
			w.Write([]byte("/* latin */\n@font-face {\n  font-family: 'Inter';\n  src: url(https://fonts.gstatic.com/s/inter/v18/abc.woff2) format('woff2');\n}\n@font-face {\n  font-family: 'Vazirmatn';\n  src: url(https://fonts.gstatic.com/s/vazirmatn/v13/def.woff2) format('woff2');\n}\n"))
		case strings.HasSuffix(r.URL.Path, ".woff2"):
			w.Write([]byte("wOF2" + r.URL.Path))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	Base = srv.URL
	defer func() { Base = "" }()

	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "src", "styles"), 0o755)
	html := `<head>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;700&family=Vazirmatn&amp;display=swap" rel="stylesheet">
</head>`
	css := `@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;700&family=Vazirmatn&display=swap');
body { font-family: Inter; }`
	os.WriteFile(filepath.Join(root, "index.html"), []byte(html), 0o644)
	os.WriteFile(filepath.Join(root, "src", "styles", "main.css"), []byte(css), 0o644)

	// Dry run changes nothing.
	plan, err := Fix(context.Background(), Options{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Changes) != 2 || len(plan.Fonts) != 0 {
		t.Fatalf("dry run plan %+v", plan)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "index.html")); string(b) != html {
		t.Fatal("dry run modified file")
	}

	plan, err = Fix(context.Background(), Options{Root: root, Write: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Fonts) != 2 {
		t.Fatalf("fonts %v", plan.Fonts)
	}
	b, _ := os.ReadFile(filepath.Join(root, "index.html"))
	got := string(b)
	if strings.Contains(got, "googleapis") || strings.Contains(got, "gstatic") {
		t.Fatalf("html still references google:\n%s", got)
	}
	if !strings.Contains(got, `href="./fonts/inter-`) {
		t.Fatalf("html not rewritten:\n%s", got)
	}
	b, _ = os.ReadFile(filepath.Join(root, "src", "styles", "main.css"))
	if !strings.Contains(string(b), "@import url('../../fonts/inter-") {
		t.Fatalf("css not rewritten:\n%s", b)
	}
	entries, _ := os.ReadDir(filepath.Join(root, "fonts"))
	var cssFile string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".css") {
			cssFile = e.Name()
		}
	}
	lc, _ := os.ReadFile(filepath.Join(root, "fonts", cssFile))
	if strings.Contains(string(lc), "https://fonts.gstatic.com") || !strings.Contains(string(lc), "url(./vazirmatn-") {
		t.Fatalf("local css:\n%s", lc)
	}
	for _, ua := range uas {
		if !strings.Contains(ua, "Chrome/") {
			t.Fatalf("ua %q", ua)
		}
	}
	// Idempotent: a second run finds nothing to do.
	plan, err = Fix(context.Background(), Options{Root: root, Write: true})
	if err != nil || len(plan.Changes) != 0 {
		t.Fatalf("second run: %+v %v", plan, err)
	}
}
