// Package compose is the content side of a container that runs a Docker
// Compose project instead of a single image (`compose: app/compose.yaml`). The
// compose file's directory -- the file plus whatever it bind-mounts or reads,
// e.g. `.env`, `nginx/nginx.conf` -- is the project, and it runs with real
// `docker compose` on one machine.
//
// The whole directory travels to that machine as one archive, so anything the
// compose file refers to by relative path is there, unchanged, on every
// builder. The compose FILE itself gets LaForge's template pass at deploy
// (`{{ vars.x }}`, `{{ .Team }}`, ...), so it can vary per team/object; the
// caller supplies the renderer (see Load's renderCompose). Compose's own
// `${VAR}`/`$host` interpolation is left untouched -- it isn't `{{ }}` -- so it
// still resolves at `docker compose` runtime as usual. The other files in the
// directory travel verbatim.
//
// Images are never built there. A service keeps its `build:` for local
// development, but must also name the `image:` LaForge pulls.
package compose

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// MaxArchiveBytes bounds a project's compressed directory. It rides to the agent
// inside one command payload, so this is for configuration files, not
// application source or data -- those belong in the image.
const MaxArchiveBytes = 1 << 20

// ProjectName turns a container's name into a Docker Compose project name
// (lowercase letters, digits, - and _, starting with a letter or digit), which
// is also safe as a directory name and an unquoted shell word.
func ProjectName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0:
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "stack"
	}
	return b.String()
}

// Bundle is one project, read and checked, ready to ship.
type Bundle struct {
	// File is the compose file's name inside the project directory.
	File string
	// Archive is the project directory as a gzipped tar, byte-identical for
	// identical content.
	Archive []byte
	// Images is every service's image ref, sorted and de-duplicated. A ref
	// still carrying a `${VAR}` is left out -- its registry isn't known here.
	Images []string
}

// Load reads the project whose compose file is at file (slash-separated,
// relative to repoRoot) and checks everything that can be known before a
// machine exists: the compose file parses, every service names an image, every
// relative bind mount is inside the directory, and the directory is small
// enough to ship.
// renderTemplate, when non-nil, is the LaForge template pass (`{{ vars.x }}`,
// `{{ .Team }}`, ...). It is applied to the compose file, and to any supporting
// file whose name ends in `.tmpl` (which then ships with the `.tmpl` stripped --
// `nginx.conf.tmpl` -> `nginx.conf`). The caller supplies it with the object's
// render context; compose itself stays render-agnostic. nil (the
// load/check/fingerprint paths, which have no per-object context) ships every
// file verbatim, `.tmpl` names and all.
func Load(repoRoot, file string, renderTemplate func([]byte) ([]byte, error)) (*Bundle, error) {
	full := filepath.Join(repoRoot, filepath.FromSlash(file))
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("compose file %s: not found in the content repo", file)
	}
	if renderTemplate != nil {
		if data, err = renderTemplate(data); err != nil {
			return nil, fmt.Errorf("compose file %s: rendering templates: %w", file, err)
		}
	}
	dir := filepath.Dir(full)
	b := &Bundle{File: filepath.Base(full)}
	if b.Images, err = checkFile(dir, data); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	// Ship the (possibly rendered) compose file, not the raw on-disk one; `.tmpl`
	// supporting files are rendered and renamed by archive.
	if b.Archive, err = archive(dir, renderTemplate, map[string][]byte{b.File: data}); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	if len(b.Archive) > MaxArchiveBytes {
		return nil, fmt.Errorf("%s: its directory is %d KiB compressed, over the %d KiB limit -- keep only the compose file and its configuration there; application code belongs in the image", file, len(b.Archive)/1024, MaxArchiveBytes/1024)
	}
	return b, nil
}

// checkFile validates the parts of a compose file LaForge depends on and
// returns the images it will pull.
func checkFile(dir string, data []byte) ([]string, error) {
	var doc struct {
		Services map[string]struct {
			Image   string        `yaml:"image"`
			Build   interface{}   `yaml:"build"`
			Volumes []interface{} `yaml:"volumes"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	if len(doc.Services) == 0 {
		return nil, fmt.Errorf("no services defined")
	}
	names := make([]string, 0, len(doc.Services))
	for n := range doc.Services {
		names = append(names, n)
	}
	sort.Strings(names)

	seen := map[string]bool{}
	var images []string
	for _, n := range names {
		svc := doc.Services[n]
		if svc.Image == "" {
			if svc.Build != nil {
				return nil, fmt.Errorf("service %q has build: but no image: -- LaForge does not build images; push it to a registry and name it with image: (build: can stay for local use)", n)
			}
			return nil, fmt.Errorf("service %q has no image:", n)
		}
		// Skip refs/paths that aren't resolvable at load time: compose's own
		// `${VAR}` interpolation, and LaForge `{{ ... }}` templates (resolved
		// per object at deploy, not here).
		if !unresolved(svc.Image) && !seen[svc.Image] {
			seen[svc.Image] = true
			images = append(images, svc.Image)
		}
		for _, v := range svc.Volumes {
			src := bindSource(v)
			if src == "" || unresolved(src) {
				continue
			}
			if rel := filepath.Clean(src); rel == ".." || strings.HasPrefix(rel, "../") {
				return nil, fmt.Errorf("service %q mounts %q, which is outside the compose file's directory and won't be shipped", n, src)
			}
			// The mounted file may be produced by a `.tmpl` at render time, so a
			// `foo.conf` mount is satisfied by `foo.conf` OR `foo.conf.tmpl`.
			if _, err := os.Stat(filepath.Join(dir, src)); err != nil {
				if _, err2 := os.Stat(filepath.Join(dir, src+".tmpl")); err2 != nil {
					return nil, fmt.Errorf("service %q mounts %q, which does not exist next to the compose file", n, src)
				}
			}
		}
	}
	sort.Strings(images)
	return images, nil
}

// bindSource returns a volume entry's host path when it is a relative bind
// mount (`./conf:/etc/app:ro`, or the long form with type: bind), else "".
// Named volumes, absolute host paths and interpolated paths are left alone.
func bindSource(v interface{}) string {
	var src string
	switch x := v.(type) {
	case string:
		src, _, _ = strings.Cut(x, ":")
	default:
		m, ok := asMap(v)
		if !ok || m["type"] != "bind" {
			return ""
		}
		src, _ = m["source"].(string)
	}
	if strings.Contains(src, "$") || !(src == "." || src == ".." || strings.HasPrefix(src, "./") || strings.HasPrefix(src, "../")) {
		return ""
	}
	return src
}

// archive packs dir as a gzipped tar with sorted entries and no timestamps or
// owners, so the same content always produces the same bytes.
// unresolved reports whether a value still carries syntax that isn't resolvable
// at load/check time: compose's own `${VAR}`/`$VAR` interpolation, or a LaForge
// `{{ ... }}` template (resolved per object at deploy).
func unresolved(s string) bool {
	return strings.Contains(s, "$") || strings.Contains(s, "{{")
}

func archive(dir string, renderTemplate func([]byte) ([]byte, error), override map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	// WalkDir visits entries in lexical order, which is what makes this stable.
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil || rel == "." {
			return err
		}
		rel = filepath.ToSlash(rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			return tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: rel + "/", Mode: 0o755})
		case !info.Mode().IsRegular():
			return fmt.Errorf("%s is not a regular file (symlinks and devices can't be shipped)", rel)
		}
		mode := int64(0o644)
		if info.Mode()&0o111 != 0 {
			mode = 0o755
		}
		// The compose file was rendered by the caller and passed in override; a
		// `.tmpl` supporting file is rendered here and ships with the suffix
		// stripped; everything else is verbatim.
		name := rel
		data, ok := override[rel]
		if !ok {
			if data, err = os.ReadFile(path); err != nil {
				return err
			}
			if renderTemplate != nil && strings.HasSuffix(rel, ".tmpl") {
				if data, err = renderTemplate(data); err != nil {
					return fmt.Errorf("rendering %s: %w", rel, err)
				}
				name = strings.TrimSuffix(rel, ".tmpl")
			}
		}
		if err := tw.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: name, Mode: mode, Size: int64(len(data))}); err != nil {
			return err
		}
		_, err = tw.Write(data)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RegistryHost extracts the registry host from an image ref, or "" for a
// Docker Hub ref. A first path segment counts as a host only if it looks like
// one (has a "." or ":", or is "localhost"); otherwise it's a Docker Hub
// namespace ("library/nginx", "myuser/app").
func RegistryHost(image string) string {
	slash := strings.IndexByte(image, '/')
	if slash < 0 {
		return "" // "nginx", "nginx:alpine"
	}
	first := image[:slash]
	if strings.ContainsAny(first, ".:") || first == "localhost" {
		return first
	}
	return ""
}

func asMap(v interface{}) (map[string]interface{}, bool) {
	m, ok := v.(map[string]interface{})
	return m, ok
}
