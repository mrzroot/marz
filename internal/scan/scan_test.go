package scan

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/mrzroot/marz/internal/kb"
)

func rules(hits []Hit) []string {
	m := map[string]bool{}
	for _, h := range hits {
		m[h.Rule] = true
	}
	var out []string
	for r := range m {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

func TestDemoProject(t *testing.T) {
	s := New(kb.MustLoad(), Options{})
	hits, files, err := s.Walk("../../testdata/demo")
	if err != nil {
		t.Fatal(err)
	}
	if files != 7 {
		t.Errorf("files = %d", files)
	}
	got := strings.Join(rules(hits), ",")
	want := "docker-hub,firebase,google-analytics,google-fonts,google-maps,google-recaptcha,jsdelivr,openai-api,sentry,stripe,youtube-embed"
	if got != want {
		t.Fatalf("rules\n got %s\nwant %s", got, want)
	}
	for _, h := range hits {
		if h.Rule == "firebase" && h.File == "src/firebase.ts" {
			if h.Line != 6 || h.Match != "shop-demo.firebaseapp.com" || h.Where != "unknown" {
				t.Errorf("firebase hit %+v", h)
			}
		}
		if h.Rule == "docker-hub" && h.Match != "docker.io/node:22-alpine" && h.Match != "docker.io/python:3.12-slim" {
			t.Errorf("docker hit %+v", h)
		}
		if h.File == "server/ai.py" && h.Where != "server" {
			t.Errorf("where %+v", h)
		}
	}
}

func TestURLEdgeCases(t *testing.T) {
	s := New(kb.MustLoad(), Options{})
	cases := map[string]string{
		`fetch("https://api.openai.com/v1/chat")`:                "openai-api",
		`src="//www.google.com/recaptcha/api.js"`:                "google-recaptcha",
		`see https://www.google.com/search?q=1`:                  "",
		`url: "mongodb+srv://u:p@cluster0.x1.mongodb.net/db"`:    "mongodb-atlas",
		`const host = "notfonts.googleapis.com.evil.com"`:        "",
		`dropbox.com and box.com`:                                "",
		`<a href="https://t.me/joinchat">`:                       "telegram",
		`"https://x.com/i/status"`:                               "twitter-embed",
		`foo.x.community`:                                        "",
		`authDomain: "a-b.firebaseapp.com" // marz-ignore`:       "",
		`https://storage.googleapis.com/pub-packages/packages/x`: "pub-dev",
	}
	for in, want := range cases {
		got := strings.Join(rules(s.File("a.js", []byte(in))), ",")
		if got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestManifests(t *testing.T) {
	cases := []struct {
		file, body string
		want       []string
	}{
		{"requirements.txt", "# c\nopenai==1.0\nsentry_sdk[flask]>=2\n-r other.txt\ngoogle-cloud-storage\n", []string{"pypi:openai", "pypi:sentry_sdk", "pypi:google-cloud-storage"}},
		{"pyproject.toml", "[project]\nname='x'\ndependencies = [\n  \"anthropic>=0.30\",\n  \"httpx\",\n]\n[tool.poetry.dependencies]\npython = \"^3.11\"\nstripe = \"*\"\n", []string{"pypi:anthropic", "pypi:httpx", "pypi:stripe"}},
		{"go.mod", "module x\n\nrequire (\n\tcloud.google.com/go/storage v1.40.0\n\tgithub.com/getsentry/sentry-go v0.27.0 // indirect\n)\nrequire github.com/stripe/stripe-go/v79 v79.1.0\n", []string{"go:cloud.google.com/go/storage", "go:github.com/getsentry/sentry-go", "go:github.com/stripe/stripe-go/v79"}},
		{"pubspec.yaml", "name: app\ndependencies:\n  flutter:\n    sdk: flutter\n  firebase_core: ^3.0.0\n  google_maps_flutter: any\ndev_dependencies:\n  lints: ^4.0.0\nflutter:\n  uses-material-design: true\n", []string{"pub:flutter", "pub:firebase_core", "pub:google_maps_flutter", "pub:lints", ":pub.dev (flutter pub get)"}},
		{"app/build.gradle.kts", "dependencies {\n  implementation(\"com.google.firebase:firebase-auth:23.0.0\")\n  // implementation 'com.google.android.gms:play-services-ads:1'\n  implementation(platform(\"com.google.firebase:firebase-bom:33.1.0\"))\n}\n", []string{"maven:com.google.firebase:firebase-auth", "maven:com.google.firebase:firebase-bom"}},
		{"gradle/libs.versions.toml", "[libraries]\nmaps = { module = \"com.google.android.gms:play-services-maps\", version = \"19.0.0\" }\nx = { group = \"io.sentry\", name = \"sentry-android\", version.ref = \"s\" }\n", []string{"maven:com.google.android.gms:play-services-maps", "maven:io.sentry:sentry-android"}},
		{"pom.xml", "<dependency>\n<groupId>com.stripe</groupId>\n<artifactId>stripe-java</artifactId>\n</dependency>\n", []string{"maven:com.stripe:stripe-java"}},
		{"ios/Podfile", "target 'App' do\n  pod 'Firebase/Analytics'\n  pod 'GoogleMaps', '~> 9'\nend\n", []string{"pod:Firebase/Analytics", "pod:GoogleMaps"}},
		{"Gemfile", "source 'https://rubygems.org'\ngem 'stripe'\ngem \"rails\", \"~> 7\"\n", []string{"gem:stripe", "gem:rails"}},
		{"Dockerfile", "FROM --platform=$BUILDPLATFORM golang:1.23 AS build\nFROM build AS test\nFROM gcr.io/distroless/static\nFROM docker.io/library/alpine:3.20\nFROM scratch\nFROM ${BASE}\n", []string{":docker.io/golang:1.23", ":docker.io/docker.io/library/alpine:3.20"}},
		{"docker-compose.yml", "services:\n  db:\n    image: postgres:16\n  app:\n    image: ghcr.io/me/app:1\n", []string{":docker.io/postgres:16"}},
	}
	for _, c := range cases {
		var got []string
		for _, d := range Manifest(c.file, strings.Split(c.body, "\n")) {
			got = append(got, d.Eco+":"+d.Name)
		}
		sort.Strings(got)
		want := append([]string(nil), c.want...)
		sort.Strings(want)
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("%s:\n got %v\nwant %v", c.file, got, want)
		}
	}
}

func TestIgnoreAndDocs(t *testing.T) {
	root := t.TempDir()
	w := func(p, s string) {
		os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755)
		os.WriteFile(filepath.Join(root, p), []byte(s), 0o644)
	}
	w("README.md", "Uses https://api.openai.com")
	w("node_modules/x/index.js", "https://api.openai.com")
	w("legacy/old.js", "https://api.anthropic.com")
	w("src/a.js", "https://api.stripe.com")
	w("img.png", "https://api.openai.com")
	w("package-lock.json", `"resolved": "https://registry.npmjs.org/firebase"`)
	w(".marzignore", "# comment\nlegacy/\n")
	s := New(kb.MustLoad(), Options{})
	hits, files, _ := s.Walk(root)
	if strings.Join(rules(hits), ",") != "stripe" || files != 1 {
		t.Fatalf("hits %v files %d", rules(hits), files)
	}
	s = New(kb.MustLoad(), Options{Docs: true})
	hits, _, _ = s.Walk(root)
	if strings.Join(rules(hits), ",") != "openai-api,stripe" {
		t.Fatalf("docs hits %v", rules(hits))
	}
	s = New(kb.MustLoad(), Options{Exclude: []string{"src/*.js"}})
	hits, _, _ = s.Walk(root)
	if len(hits) != 0 {
		t.Fatalf("exclude hits %v", hits)
	}
}

func TestClassify(t *testing.T) {
	cases := map[string]string{
		"public/index.html":        "client",
		"server/main.py":           "server",
		"app/api/route.ts":         "server",
		".github/workflows/ci.yml": "build",
		"Dockerfile":               "build",
		"src/components/Map.tsx":   "client",
		"lib/main.dart":            "client",
		"scripts/thing.js":         "unknown",
	}
	for in, want := range cases {
		if got := Classify(in); got != want {
			t.Errorf("%s = %s want %s", in, got, want)
		}
	}
}
