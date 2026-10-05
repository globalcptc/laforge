package incus

import (
	"context"
	"strings"
	"testing"

	"github.com/globalcptc/laforge/internal/builder"
)

// A compose container boots from the builder's docker base image; until that
// has been built (and with no compose-host override) the deploy says so
// instead of falling through to the single-image path.
func TestDeployComposeContainerNeedsDockerBaseImage(t *testing.T) {
	b := &Builder{Config: Config{Images: map[string]ImageRef{"ubuntu22": {}}}}
	_, err := b.DeployContainer(context.Background(), builder.ContainerSpec{ComposeHost: true, Size: "small"})
	if err == nil || !strings.Contains(err.Error(), "no docker base image yet") {
		t.Fatalf("err = %v, want a missing docker base image error", err)
	}
}
