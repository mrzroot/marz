// Package kb holds marz's embedded knowledge base: which third-party services
// break in which country, why, with sources and alternatives.
package kb

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed rules.json
var raw []byte

// Countries marz knows about, in display order.
var Countries = []string{"IR", "CN", "RU"}

// CountryNames maps codes to English names.
var CountryNames = map[string]string{"IR": "Iran", "CN": "China", "RU": "Russia"}

// Status values, from worst to best.
const (
	Blocked     = "blocked"     // filtered by the state censor
	Sanctioned  = "sanctioned"  // the provider refuses the country
	Unavailable = "unavailable" // the provider does not operate there
	Degraded    = "degraded"    // throttled, slow or intermittently broken
	Unknown     = "unknown"     // no reliable public data; probe it
	OK          = "ok"
)

// Impact / severity levels, from worst to best.
var Levels = []string{"critical", "high", "medium", "low", "info"}

// LevelRank returns 0 for critical .. 4 for info, 5 for anything else.
func LevelRank(l string) int {
	for i, v := range Levels {
		if v == l {
			return i
		}
	}
	return len(Levels)
}

type Source struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type Verdict struct {
	Status       string   `json:"status"`
	Note         string   `json:"note,omitempty"`
	Sources      []string `json:"sources,omitempty"`
	Alternatives []string `json:"alternatives,omitempty"`
}

type Rule struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Category  string              `json:"category"`
	Impact    string              `json:"impact"`
	Side      string              `json:"side"`
	Hosts     []string            `json:"hosts,omitempty"`
	Paths     []string            `json:"paths,omitempty"`
	Packages  map[string][]string `json:"packages,omitempty"`
	Why       string              `json:"why"`
	FA        string              `json:"fa"`
	Fix       string              `json:"fix,omitempty"`
	OONI      string              `json:"ooni,omitempty"` // canonical domain to query OONI with
	Countries map[string]Verdict  `json:"countries"`
}

type KB struct {
	Updated string            `json:"updated"`
	Sources map[string]Source `json:"sources"`
	Rules   []Rule            `json:"rules"`

	byID     map[string]*Rule
	patterns []pattern
}

type pattern struct {
	text string // lower-case host or host/path
	path bool
	rule *Rule
}

// Load parses the embedded knowledge base.
func Load() (*KB, error) { return Parse(raw) }

// MustLoad is Load that panics; the embedded file is validated by tests.
func MustLoad() *KB {
	k, err := Load()
	if err != nil {
		panic(err)
	}
	return k
}

// Parse parses and validates a knowledge base document.
func Parse(b []byte) (*KB, error) {
	var k KB
	if err := json.Unmarshal(b, &k); err != nil {
		return nil, err
	}
	k.byID = map[string]*Rule{}
	for i := range k.Rules {
		r := &k.Rules[i]
		if r.ID == "" || r.Name == "" {
			return nil, fmt.Errorf("rule %d: missing id or name", i)
		}
		if _, dup := k.byID[r.ID]; dup {
			return nil, fmt.Errorf("duplicate rule id %q", r.ID)
		}
		if LevelRank(r.Impact) >= 4 {
			return nil, fmt.Errorf("rule %s: bad impact %q", r.ID, r.Impact)
		}
		k.byID[r.ID] = r
		for c, v := range r.Countries {
			if CountryNames[c] == "" {
				return nil, fmt.Errorf("rule %s: unknown country %q", r.ID, c)
			}
			switch v.Status {
			case Blocked, Sanctioned, Unavailable, Degraded, Unknown, OK:
			default:
				return nil, fmt.Errorf("rule %s/%s: bad status %q", r.ID, c, v.Status)
			}
			if v.Status != OK && v.Status != Unknown && len(v.Sources) == 0 {
				return nil, fmt.Errorf("rule %s/%s: status %q needs a source", r.ID, c, v.Status)
			}
			for _, s := range v.Sources {
				if _, ok := k.Sources[s]; !ok {
					return nil, fmt.Errorf("rule %s/%s: unknown source %q", r.ID, c, s)
				}
			}
		}
		for _, h := range r.Hosts {
			h = strings.ToLower(h)
			k.patterns = append(k.patterns, pattern{text: h, path: strings.Contains(h, "/"), rule: r})
		}
		for _, p := range r.Paths {
			k.patterns = append(k.patterns, pattern{text: strings.ToLower(p), path: true, rule: r})
		}
	}
	// Longest pattern first, so the most specific rule wins.
	sort.SliceStable(k.patterns, func(i, j int) bool { return len(k.patterns[i].text) > len(k.patterns[j].text) })
	return &k, nil
}

// Rule returns a rule by id.
func (k *KB) Rule(id string) *Rule { return k.byID[id] }

// HostPatterns returns all host-only patterns (for building text matchers).
func (k *KB) HostPatterns() []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range k.patterns {
		t := p.text
		if p.path {
			t = t[:strings.Index(t, "/")]
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

// MatchURL finds the rule for a host and path ("" if none). Host matching
// covers subdomains; path patterns need the host and a path prefix.
func (k *KB) MatchURL(host, path string) (*Rule, string) {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	hp := host + strings.ToLower(path)
	for _, p := range k.patterns {
		if p.path {
			ph := p.text[:strings.Index(p.text, "/")]
			pp := p.text[len(ph):]
			if (host == ph || strings.HasSuffix(host, "."+ph)) && strings.HasPrefix(strings.ToLower(path), pp) {
				return p.rule, p.text
			}
			_ = hp
			continue
		}
		if host == p.text || strings.HasSuffix(host, "."+p.text) {
			return p.rule, p.text
		}
	}
	return nil, ""
}

// MatchPackage finds the rule for a package in an ecosystem
// (npm, pypi, go, pub, maven, pod, composer, gem).
func (k *KB) MatchPackage(eco, name string) *Rule {
	n := strings.ToLower(name)
	if eco == "pypi" {
		n = strings.ReplaceAll(n, "_", "-")
	}
	var best *Rule
	bestLen := -1
	for i := range k.Rules {
		r := &k.Rules[i]
		for _, p := range r.Packages[eco] {
			p = strings.ToLower(p)
			if eco == "pypi" {
				p = strings.ReplaceAll(p, "_", "-")
			}
			ok := false
			if strings.HasSuffix(p, "*") {
				ok = strings.HasPrefix(n, strings.TrimSuffix(p, "*"))
			} else {
				ok = n == p
			}
			if ok && len(p) > bestLen {
				best, bestLen = r, len(p)
			}
		}
	}
	return best
}

// Severity for a rule in a country: the rule's impact for hard failures,
// one level lower for degraded, "info" for unknown, "" when fine.
func (r *Rule) Severity(country string) string {
	v, ok := r.Countries[country]
	if !ok {
		return ""
	}
	switch v.Status {
	case Blocked, Sanctioned, Unavailable:
		return r.Impact
	case Degraded:
		i := LevelRank(r.Impact) + 1
		if i > 3 {
			i = 3
		}
		return Levels[i]
	case Unknown:
		return "info"
	}
	return ""
}
