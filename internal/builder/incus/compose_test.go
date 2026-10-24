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

func TestComposeHostNesting(t *testing.T) {
	for _, test := range []struct {
		name        string
		customImage bool
		vm          bool
		ordinary    bool
		wantNesting string
	}{
		{name: "built-in Docker base", wantNesting: "true"},
		{name: "custom system container", customImage: true, wantNesting: "true"},
		{name: "custom VM", customImage: true, vm: true},
		{name: "ordinary container host", ordinary: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, client := newFakeSnapshotServer(t)
			config := Config{
				Images: map[string]ImageRef{}, Sizes: map[string]SizeSpec{"small": {}},
				DockerBaseFingerprint: "ready",
			}
			if test.customImage {
				config.Images[builder.ComposeHostImage] = ImageRef{Alias: "custom-docker-image", VM: test.vm}
			}
			b := New(client, config)
			var err error
			if test.ordinary {
				config.Images["ubuntu"] = ImageRef{Alias: "ubuntu"}
				_, err = b.DeployHost(context.Background(), builder.HostSpec{ExternalName: "ordinary", OS: "ubuntu", Size: "small"})
			} else {
				_, err = b.DeployContainer(context.Background(), builder.ContainerSpec{ExternalName: "compose", ComposeHost: true, Size: "small"})
			}
			if err != nil {
				t.Fatal(err)
			}
			instanceConfig := server.createReq["config"].(map[string]interface{})
			nesting, _ := instanceConfig["security.nesting"].(string)
			if nesting != test.wantNesting {
				t.Fatalf("security.nesting = %q, want %q", nesting, test.wantNesting)
			}
		})
	}
}
