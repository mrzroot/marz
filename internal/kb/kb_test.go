package kb

import "testing"

func TestEmbeddedValid(t *testing.T) {
	k, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(k.Rules) < 25 {
		t.Fatalf("only %d rules", len(k.Rules))
	}
	for _, r := range k.Rules {
		for _, c := range Countries {
			if _, ok := r.Countries[c]; !ok {
				t.Errorf("%s: no verdict for %s", r.ID, c)
			}
		}
		if r.FA == "" || r.Why == "" {
			t.Errorf("%s: missing text", r.ID)
		}
		if len(r.Hosts)+len(r.Paths)+len(r.Packages) == 0 {
			t.Errorf("%s: nothing to match", r.ID)
		}
	}
}

func TestMatchURL(t *testing.T) {
	k := MustLoad()
	cases := map[string]string{
		"fonts.googleapis.com|/css2?family=Inter": "google-fonts",
		"my-app.firebaseio.com|/":                 "firebase",
		"www.google.com|/recaptcha/api.js":        "google-recaptcha",
		"www.google.com|/search":                  "",
		"storage.googleapis.com|/pub-packages/x":  "pub-dev",
		"storage.googleapis.com|/bucket/x":        "google-cloud",
		"cluster0.ab12c.mongodb.net|":             "mongodb-atlas",
		"box.com|/":                               "",
		"x.com|/foo":                              "twitter-embed",
		"API.OPENAI.COM.|/v1":                     "openai-api",
	}
	for in, want := range cases {
		var h, p string
		for i := range in {
			if in[i] == '|' {
				h, p = in[:i], in[i+1:]
			}
		}
		r, _ := k.MatchURL(h, p)
		got := ""
		if r != nil {
			got = r.ID
		}
		if got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestMatchPackage(t *testing.T) {
	k := MustLoad()
	cases := []struct{ eco, name, want string }{
		{"npm", "firebase", "firebase"},
		{"npm", "@firebase/auth", "firebase"},
		{"npm", "@sentry/react", "sentry"},
		{"npm", "react", ""},
		{"pypi", "Google_Cloud_Storage", "google-cloud"},
		{"maven", "com.google.android.gms:play-services-maps", "google-maps"},
		{"maven", "com.google.android.gms:play-services-location", "play-services"},
		{"go", "cloud.google.com/go/storage", "google-cloud"},
		{"pub", "google_mobile_ads", "google-ads"},
	}
	for _, c := range cases {
		r := k.MatchPackage(c.eco, c.name)
		got := ""
		if r != nil {
			got = r.ID
		}
		if got != c.want {
			t.Errorf("%s %s: got %q want %q", c.eco, c.name, got, c.want)
		}
	}
}

func TestSeverity(t *testing.T) {
	k := MustLoad()
	if s := k.Rule("firebase").Severity("IR"); s != "critical" {
		t.Errorf("firebase IR = %q", s)
	}
	if s := k.Rule("jsdelivr").Severity("CN"); s != "low" {
		t.Errorf("jsdelivr CN = %q", s)
	}
	if s := k.Rule("google-fonts").Severity("IR"); s != "" {
		t.Errorf("fonts IR = %q", s)
	}
	if s := k.Rule("google-recaptcha").Severity("IR"); s != "info" {
		t.Errorf("recaptcha IR = %q", s)
	}
}

func TestParseRejectsUnsourced(t *testing.T) {
	_, err := Parse([]byte(`{"sources":{},"rules":[{"id":"a","name":"A","impact":"high","hosts":["a.com"],"countries":{"IR":{"status":"blocked"}}}]}`))
	if err == nil {
		t.Fatal("expected error for blocked without source")
	}
}
