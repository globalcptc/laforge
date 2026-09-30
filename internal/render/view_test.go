package render_test

import (
	"testing"

	"github.com/globalcptc/laforge/internal/loader"
	"github.com/globalcptc/laforge/internal/render"
)

// TestNewContextViewMatchesRealResolvedContext is the moved-from-
// cmd/laforge presentation layer (see internal/render/view.go's own doc
// comment on why it lives here now), checked against real content so
// `laforge context` and internal/lsp's "Show everything available"
// command stay provably identical rather than merely both compiling.
func TestNewContextViewMatchesRealResolvedContext(t *testing.T) {
	c, err := loader.Load("../../examples/lm-test")
	if err != nil {
		t.Fatalf("loader.Load: %v", err)
	}
	if len(c.Errors) != 0 {
		t.Fatalf("content.Errors = %+v, want none", c.Errors)
	}
	ctx, err := render.Resolve(c, "lm-test", "web01", 1)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	v := render.NewContextView(ctx)
	if v.Environment != "lm-test" || v.Team != 1 || v.Host.As != "web01" {
		t.Fatalf("view = %+v, want environment=lm-test team=1 host.as=web01", v)
	}
	if v.Host.Kind != "host" {
		t.Fatalf("v.Host.Kind = %q, want host", v.Host.Kind)
	}
	var sawCompany bool
	for _, vv := range v.Vars {
		if vv.Key == "company" {
			sawCompany = true
			if vv.Value != "Allports" || vv.Source != "environment" {
				t.Fatalf("company var = %+v, want value=Allports source=environment", vv)
			}
		}
	}
	if !sawCompany {
		t.Fatalf("no \"company\" var in view.Vars = %+v", v.Vars)
	}
}
