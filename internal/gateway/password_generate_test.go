package gateway

import (
	"testing"

	"github.com/globalcptc/laforge/internal/agentproto"
	"github.com/globalcptc/laforge/internal/loader"
)

// TestSetPasswordGenerateMatchesTemplate proves set_password's `generate:`
// produces the build's stable generated password -- the same value a
// `{{ passphrase 3 }}` template gives on the same host, so a script can write
// out what the step set -- and that the build secret, not the preview one, is
// what a real expansion uses.
func TestSetPasswordGenerateMatchesTemplate(t *testing.T) {
	c, err := loader.Load("../../examples/lm-test")
	if err != nil {
		t.Fatalf("loader.Load: %v", err)
	}
	for i := range c.Hosts {
		if c.Hosts[i].Name == "database" {
			c.Hosts[i].Steps = append(c.Hosts[i].Steps,
				loader.Step{"set_password": map[string]interface{}{
					"name": "svc", "generate": map[string]interface{}{"type": "passphrase", "size": 3},
				}},
				loader.Step{"set_password": map[string]interface{}{
					"name": "svc2", "password": "{{ passphrase 3 }}",
				}},
				loader.Step{"create_user": map[string]interface{}{
					"name": "ops", "generate": map[string]interface{}{"type": "random", "size": 20.0, "label": "ops"},
				}},
			)
		}
	}
	expand := func(secret []byte) (setGen, setTmpl, created string) {
		t.Helper()
		cmds, _, err := ExpandSteps("../../examples/lm-test", c, "lm-test", "db01", 1, WithRenderSecret(secret))
		if err != nil {
			t.Fatalf("ExpandSteps: %v", err)
		}
		for _, cmd := range cmds {
			switch p := cmd.Payload.(type) {
			case agentproto.SetPasswordPayload:
				switch p.Username {
				case "svc":
					setGen = p.Password
				case "svc2":
					setTmpl = p.Password
				}
			case agentproto.CreateUserPayload:
				if p.Username == "ops" {
					created = p.Password
				}
			}
		}
		return
	}
	gen, tmpl, created := expand([]byte("build-a"))
	if gen == "" || gen != tmpl {
		t.Fatalf("generate: %q vs {{ passphrase 3 }}: %q -- want the same non-empty password", gen, tmpl)
	}
	if len(created) != 20 {
		t.Errorf("create_user random 20 = %q, want 20 characters", created)
	}
	if again, _, _ := expand([]byte("build-a")); again != gen {
		t.Errorf("same build expanded twice gave %q then %q", gen, again)
	}
	if other, _, _ := expand([]byte("build-b")); other == gen {
		t.Errorf("a different build secret gave the same password %q", gen)
	}
	if preview, _, _ := expand(nil); preview == gen {
		t.Errorf("an expansion without the build secret matched the real password %q", gen)
	}
}
