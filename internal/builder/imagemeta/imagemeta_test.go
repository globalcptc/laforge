package imagemeta

import "testing"

func TestParseDockerRef(t *testing.T) {
	cases := []struct {
		ref, repo, tag string
	}{
		{"nginx", "library/nginx", "latest"},
		{"nginx:alpine", "library/nginx", "alpine"},
		{"laforge/scoreboard:1.2", "laforge/scoreboard", "1.2"},
		{"library/alpine", "library/alpine", "latest"},
		{"registry.internal/team/app:1.0", "", "1.0"}, // another registry -- not docker.io
		{"ghcr.io/org/app", "", "latest"},
	}
	for _, c := range cases {
		repo, tag := ParseDockerRef(c.ref)
		if repo != c.repo || tag != c.tag {
			t.Errorf("ParseDockerRef(%q) = (%q, %q), want (%q, %q)", c.ref, repo, tag, c.repo, c.tag)
		}
	}
}
