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

func TestShellJoin(t *testing.T) {
	// The nginx regression: ["nginx","-g","daemon off;"] must survive `sh -c`
	// re-splitting as three args, not split "daemon off;" into two.
	got := ShellJoin([]string{"/docker-entrypoint.sh", "nginx", "-g", "daemon off;"})
	want := `'/docker-entrypoint.sh' 'nginx' '-g' 'daemon off;'`
	if got != want {
		t.Errorf("ShellJoin nginx = %q, want %q", got, want)
	}
	// An embedded single quote is escaped so the quoting can't be broken out of.
	if got := ShellJoin([]string{"echo", "it's"}); got != `'echo' 'it'\''s'` {
		t.Errorf("ShellJoin quote-escape = %q", got)
	}
	if got := ShellJoin(nil); got != "" {
		t.Errorf("ShellJoin(nil) = %q, want empty", got)
	}
}
