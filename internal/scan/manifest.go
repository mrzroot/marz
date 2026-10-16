package scan

import (
	"encoding/json"
	"path"
	"regexp"
	"strings"
)

// Dep is a dependency (or an implied host) declared in a manifest.
type Dep struct {
	Eco  string // npm, pypi, go, pub, maven, pod, composer, gem
	Name string
	Line int
	Host string // set for implied network use, e.g. Docker Hub for "FROM node"
	Kind string
}

var (
	reReqLine   = regexp.MustCompile(`^\s*([A-Za-z0-9][A-Za-z0-9._\-]*)\s*(?:\[[^\]]*\])?\s*(?:[<>=!~;@ ]|$)`)
	reQuoted    = regexp.MustCompile(`^\s*["']([A-Za-z0-9][A-Za-z0-9._\-]*)\s*(?:\[[^\]]*\])?\s*[<>=!~;@ "']`)
	reTomlKey   = regexp.MustCompile(`^\s*["']?([A-Za-z0-9][A-Za-z0-9._\-]*)["']?\s*=`)
	reGoReq     = regexp.MustCompile(`^\s*(?:require\s+)?([a-zA-Z0-9][^\s]*\.[^\s]*/[^\s]*|[a-z0-9.\-]+\.[a-z]+/[^\s]*)\s+v[0-9]`)
	reYamlKey   = regexp.MustCompile(`^(\s+)([A-Za-z0-9_]+)\s*:`)
	reGradle    = regexp.MustCompile(`["']([A-Za-z0-9_.\-]+:[A-Za-z0-9_.\-]+)(?::[^"']*)?["']`)
	reTomlMod   = regexp.MustCompile(`module\s*=\s*"([A-Za-z0-9_.\-]+:[A-Za-z0-9_.\-]+)"`)
	reTomlGA    = regexp.MustCompile(`group\s*=\s*"([A-Za-z0-9_.\-]+)"\s*,\s*name\s*=\s*"([A-Za-z0-9_.\-]+)"`)
	rePomGroup  = regexp.MustCompile(`<groupId>\s*([^<\s]+)\s*</groupId>`)
	rePomArt    = regexp.MustCompile(`<artifactId>\s*([^<\s]+)\s*</artifactId>`)
	rePod       = regexp.MustCompile(`^\s*pod\s+['"]([^'"]+)['"]`)
	reGem       = regexp.MustCompile(`^\s*gem\s+['"]([^'"]+)['"]`)
	reFrom      = regexp.MustCompile(`(?i)^\s*FROM\s+(?:--[a-z]+=\S+\s+)*(\S+)`)
	reImage     = regexp.MustCompile(`^\s*-?\s*(?:image|container)\s*:\s*["']?([^"'\s#]+)`)
	reSectionTo = regexp.MustCompile(`^\s*\[([^\]]+)\]`)
)

// Manifest extracts dependencies from known manifest files.
func Manifest(rel string, lines []string) []Dep {
	base := path.Base(rel)
	lower := strings.ToLower(base)
	switch {
	case base == "package.json":
		return jsonDeps("npm", lines, "dependencies", "devDependencies", "peerDependencies", "optionalDependencies")
	case base == "composer.json":
		return jsonDeps("composer", lines, "require", "require-dev")
	case isRequirements(base):
		return lineDeps("pypi", lines, reReqLine)
	case base == "pyproject.toml":
		return pyproject(lines)
	case base == "Pipfile":
		return tomlSectionKeys("pypi", lines, map[string]bool{"packages": true, "dev-packages": true})
	case base == "go.mod":
		return goMod(lines)
	case base == "pubspec.yaml":
		d := yamlBlockKeys("pub", lines, "dependencies", "dev_dependencies")
		for i, l := range lines {
			if strings.HasPrefix(l, "dependencies:") {
				d = append(d, Dep{Host: "pub.dev", Name: "pub.dev (flutter pub get)", Line: i + 1, Kind: "registry"})
				break
			}
		}
		return d
	case strings.HasSuffix(base, ".gradle") || strings.HasSuffix(base, ".gradle.kts"):
		return gradle(lines)
	case strings.HasSuffix(base, ".versions.toml"):
		return versionsToml(lines)
	case base == "pom.xml":
		return pom(lines)
	case base == "Podfile":
		return lineDeps("pod", lines, rePod)
	case base == "Gemfile":
		return lineDeps("gem", lines, reGem)
	case strings.HasPrefix(lower, "dockerfile") || strings.HasSuffix(lower, ".dockerfile") || lower == "containerfile":
		return images(lines, reFrom)
	case (strings.Contains(lower, "compose") || strings.HasPrefix(rel, ".github/workflows/") || lower == ".gitlab-ci.yml") &&
		(strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".yaml")):
		return images(lines, reImage)
	}
	return nil
}

func jsonDeps(eco string, lines []string, keys ...string) []Dep {
	var m map[string]json.RawMessage
	if json.Unmarshal([]byte(strings.Join(lines, "\n")), &m) != nil {
		return nil
	}
	var out []Dep
	for _, k := range keys {
		var deps map[string]any
		if json.Unmarshal(m[k], &deps) != nil {
			continue
		}
		for name := range deps {
			out = append(out, Dep{Eco: eco, Name: name, Line: findLine(lines, `"`+name+`"`)})
		}
	}
	return out
}

func findLine(lines []string, needle string) int {
	for i, l := range lines {
		if strings.Contains(l, needle) {
			return i + 1
		}
	}
	return 1
}

func lineDeps(eco string, lines []string, re *regexp.Regexp) []Dep {
	var out []Dep
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "-") {
			continue
		}
		if m := re.FindStringSubmatch(l); m != nil {
			out = append(out, Dep{Eco: eco, Name: m[1], Line: i + 1})
		}
	}
	return out
}

func pyproject(lines []string) []Dep {
	var out []Dep
	section := ""
	inArray := false
	for i, l := range lines {
		if m := reSectionTo.FindStringSubmatch(l); m != nil && !inArray {
			section = m[1]
			continue
		}
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "#") {
			continue
		}
		if strings.HasSuffix(section, "dependencies") && strings.HasPrefix(section, "tool.poetry") {
			if m := reTomlKey.FindStringSubmatch(l); m != nil && m[1] != "python" {
				out = append(out, Dep{Eco: "pypi", Name: m[1], Line: i + 1})
			}
			continue
		}
		if !inArray && (strings.HasPrefix(t, "dependencies") || section == "project.optional-dependencies" || section == "dependency-groups") && strings.Contains(t, "[") {
			inArray = true
			t = t[strings.Index(t, "[")+1:]
		}
		if inArray {
			for _, part := range strings.Split(t, ",") {
				if m := reQuoted.FindStringSubmatch(part + " "); m != nil {
					out = append(out, Dep{Eco: "pypi", Name: m[1], Line: i + 1})
				}
			}
			if strings.Contains(t, "]") && !strings.Contains(t, "[") {
				inArray = false
			}
		}
	}
	return out
}

func tomlSectionKeys(eco string, lines []string, sections map[string]bool) []Dep {
	var out []Dep
	cur := ""
	for i, l := range lines {
		if m := reSectionTo.FindStringSubmatch(l); m != nil {
			cur = m[1]
			continue
		}
		if sections[cur] {
			if m := reTomlKey.FindStringSubmatch(l); m != nil {
				out = append(out, Dep{Eco: eco, Name: m[1], Line: i + 1})
			}
		}
	}
	return out
}

func goMod(lines []string) []Dep {
	var out []Dep
	in := false
	for i, l := range lines {
		t := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(t, "require ("):
			in = true
			continue
		case in && t == ")":
			in = false
			continue
		}
		if in || strings.HasPrefix(t, "require ") {
			if m := reGoReq.FindStringSubmatch(t); m != nil {
				out = append(out, Dep{Eco: "go", Name: m[1], Line: i + 1})
			}
		}
	}
	return out
}

func yamlBlockKeys(eco string, lines []string, blocks ...string) []Dep {
	want := map[string]bool{}
	for _, b := range blocks {
		want[b] = true
	}
	var out []Dep
	in := false
	indent := -1
	for i, l := range lines {
		if l == "" || strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}
		if l[0] != ' ' && l[0] != '\t' {
			k := strings.TrimSuffix(strings.TrimSpace(strings.SplitN(l, ":", 2)[0]), ":")
			in = want[k]
			indent = -1
			continue
		}
		if !in {
			continue
		}
		if m := reYamlKey.FindStringSubmatch(l); m != nil {
			if indent < 0 {
				indent = len(m[1])
			}
			if len(m[1]) == indent {
				out = append(out, Dep{Eco: eco, Name: m[2], Line: i + 1})
			}
		}
	}
	return out
}

func gradle(lines []string) []Dep {
	var out []Dep
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "//") {
			continue
		}
		for _, m := range reGradle.FindAllStringSubmatch(l, -1) {
			out = append(out, Dep{Eco: "maven", Name: m[1], Line: i + 1})
		}
	}
	return out
}

func versionsToml(lines []string) []Dep {
	var out []Dep
	for i, l := range lines {
		if m := reTomlMod.FindStringSubmatch(l); m != nil {
			out = append(out, Dep{Eco: "maven", Name: m[1], Line: i + 1})
		} else if m := reTomlGA.FindStringSubmatch(l); m != nil {
			out = append(out, Dep{Eco: "maven", Name: m[1] + ":" + m[2], Line: i + 1})
		}
	}
	return out
}

func pom(lines []string) []Dep {
	var out []Dep
	group := ""
	for i, l := range lines {
		if m := rePomGroup.FindStringSubmatch(l); m != nil {
			group = m[1]
		}
		if m := rePomArt.FindStringSubmatch(l); m != nil && group != "" {
			out = append(out, Dep{Eco: "maven", Name: group + ":" + m[1], Line: i + 1})
			group = ""
		}
	}
	return out
}

// images reports container images that will be pulled from Docker Hub
// (no registry host in the reference).
func images(lines []string, re *regexp.Regexp) []Dep {
	var out []Dep
	for i, l := range lines {
		m := re.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		img := m[1]
		if img == "scratch" || strings.ContainsAny(img, "${") {
			continue
		}
		img = strings.TrimPrefix(img, "docker://")
		first := strings.SplitN(img, "/", 2)[0]
		if strings.Contains(img, "/") && (strings.ContainsAny(first, ".:") || first == "localhost") {
			if first != "docker.io" && first != "index.docker.io" && first != "registry-1.docker.io" {
				continue
			}
		}
		// A bare name like "build" in "FROM build AS final" refers to an earlier stage.
		if re == reFrom && !strings.ContainsAny(img, ":/@") && stageNames(lines)[strings.ToLower(img)] {
			continue
		}
		out = append(out, Dep{Host: "registry-1.docker.io", Name: "docker.io/" + img, Line: i + 1, Kind: "image"})
	}
	return out
}

var reAs = regexp.MustCompile(`(?i)^\s*FROM\s+.*\s+AS\s+(\S+)`)

func stageNames(lines []string) map[string]bool {
	m := map[string]bool{}
	for _, l := range lines {
		if s := reAs.FindStringSubmatch(l); s != nil {
			m[strings.ToLower(s[1])] = true
		}
	}
	return m
}
