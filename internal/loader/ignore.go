package loader

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/globalcptc/laforge/internal/schema"
)

// IgnoreFile is an optional file at the content repo's root listing paths that
// are not LaForge content: a Docker Compose project a container runs, vendored
// configuration, anything else whose YAML isn't LaForge's. Everything that
// reads a content repo -- `laforge check`, the server, the editor -- goes
// through Load, so one file keeps them all in agreement.
//
// The syntax is the familiar subset of .gitignore: one pattern per line, `#`
// comments, `*`/`?`/`[...]` within a path segment, `**` for any number of
// directories, a trailing `/` to match directories only, and a leading `/` (or
// any `/` inside the pattern) to anchor it to the repo root; a bare name
// matches at any depth. There is no `!` negation.
const IgnoreFile = ".laforgeignore"

type ignorePattern struct {
	segments []string // the pattern split on "/"
	anchored bool     // matches from the repo root only
	dirOnly  bool
}

type ignoreRules []ignorePattern

// loadIgnore reads root's IgnoreFile. A repo without one ignores nothing.
func loadIgnore(root string) (ignoreRules, []schema.FieldError) {
	f, err := os.Open(filepath.Join(root, IgnoreFile))
	if err != nil {
		return nil, nil
	}
	defer f.Close()

	var rules ignoreRules
	var errs []schema.FieldError
	sc := bufio.NewScanner(f)
	for line := 1; sc.Scan(); line++ {
		p := strings.TrimSpace(sc.Text())
		if p == "" || strings.HasPrefix(p, "#") {
			continue
		}
		if strings.HasPrefix(p, "!") {
			errs = append(errs, schema.FieldError{File: IgnoreFile, Line: line, Message: fmt.Sprintf("%q: negated patterns (!) are not supported", p)})
			continue
		}
		rule := ignorePattern{dirOnly: strings.HasSuffix(p, "/")}
		p = strings.TrimSuffix(p, "/")
		rule.anchored = strings.Contains(p, "/")
		rule.segments = strings.Split(strings.TrimPrefix(p, "/"), "/")
		bad := p == ""
		for _, seg := range rule.segments {
			if _, err := path.Match(seg, ""); err != nil {
				bad = true
			}
		}
		if bad {
			errs = append(errs, schema.FieldError{File: IgnoreFile, Line: line, Message: fmt.Sprintf("%q is not a valid pattern", sc.Text())})
			continue
		}
		rules = append(rules, rule)
	}
	return rules, errs
}

// match reports whether rel (slash-separated, relative to the repo root) is
// ignored. Load prunes an ignored directory, so files beneath it are never
// asked about.
func (r ignoreRules) match(rel string, isDir bool) bool {
	parts := strings.Split(rel, "/")
	for _, rule := range r {
		if rule.dirOnly && !isDir {
			continue
		}
		if rule.anchored {
			if matchSegments(rule.segments, parts) {
				return true
			}
			continue
		}
		if ok, _ := path.Match(rule.segments[0], parts[len(parts)-1]); ok {
			return true
		}
	}
	return false
}

// matchSegments matches a pattern against a whole path, segment by segment,
// with "**" standing for zero or more segments.
func matchSegments(pattern, parts []string) bool {
	if len(pattern) == 0 {
		return len(parts) == 0
	}
	if pattern[0] == "**" {
		for i := 0; i <= len(parts); i++ {
			if matchSegments(pattern[1:], parts[i:]) {
				return true
			}
		}
		return false
	}
	if len(parts) == 0 {
		return false
	}
	if ok, _ := path.Match(pattern[0], parts[0]); !ok {
		return false
	}
	return matchSegments(pattern[1:], parts[1:])
}
