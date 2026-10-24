package runner

import "testing"

func TestRegistryHost(t *testing.T) {
	cases := map[string]string{
		"nginx":                              "",
		"nginx:alpine":                       "",
		"library/nginx":                      "",
		"myuser/app:1.2":                     "",
		"registry.internal/team/app:tag":     "registry.internal",
		"registry.internal:5000/app":         "registry.internal:5000",
		"localhost:5000/app":                 "localhost:5000",
		"ghcr.io/globalcptc/scoreboard:main": "ghcr.io",
	}
	for image, want := range cases {
		if got := registryHost(image); got != want {
			t.Errorf("registryHost(%q) = %q, want %q", image, got, want)
		}
	}
}
