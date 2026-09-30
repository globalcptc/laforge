package orchestrator

import (
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/loader"
)

func lmTestEnv(t *testing.T) (*loader.Content, *loader.Environment) {
	t.Helper()
	c, err := loader.Load("../../examples/lm-test")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(c.Errors) != 0 {
		t.Fatalf("example repo should load clean: %+v", c.Errors)
	}
	env := findEnvironment(c, "lm-test")
	if env == nil {
		t.Fatal("lm-test environment not found")
	}
	return c, env
}

func containerNamesOf(c *loader.Content) map[string]bool {
	m := make(map[string]bool, len(c.Containers))
	for _, ct := range c.Containers {
		m[ct.Name] = true
	}
	return m
}

// There is no capability gating anymore -- content is builder-agnostic -- so the
// only pre-build check is image-map completeness: every os a host uses must be
// mapped in the builder config.

func TestCheckEnvironmentImagesRejectsMissingImageMapping(t *testing.T) {
	c, env := lmTestEnv(t)
	// The exact "windows-server-2022 not listed" scenario:
	// an images map missing an os this environment actually uses.
	images := map[string]bool{"ubuntu22": true} // lm-test's hosts use more than this
	err := checkEnvironmentImages(env, c, containerNamesOf(c), images)
	if err == nil {
		t.Fatal("expected an error for an os/image the builder has no image configured for")
	}
	if !strings.Contains(err.Error(), "no image configured for") {
		t.Errorf("error should name the missing-image reason, got: %v", err)
	}
}

func TestCheckEnvironmentImagesSkipsWhenNilImages(t *testing.T) {
	// nil images means "this builder kind doesn't expose an inspectable image
	// map" (e.g. the fake builder) -- not "definitely wrong."
	c, env := lmTestEnv(t)
	if err := checkEnvironmentImages(env, c, containerNamesOf(c), nil); err != nil {
		t.Errorf("expected no error when images is nil, got: %v", err)
	}
}
