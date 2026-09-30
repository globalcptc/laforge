package render

import "fmt"

// ContextView is a presentation-only shape for a resolved Context --
// separate from Context itself so the output reads cleanly (lowercase
// keys, only what's useful to a human) without constraining Context's
// own Go-side shape to double as a serialization format. Originally
// cmd/laforge's own runContext (`laforge context`); moved here so
// internal/lsp's "Show everything available" command
// is the exact same code,
// not a second implementation that could drift from what the CLI prints.
type ContextView struct {
	Environment string `yaml:"environment" json:"environment"`
	Team        int    `yaml:"team" json:"team"`
	Teams       int    `yaml:"teams_total" json:"teams_total"`
	Start       string `yaml:"start,omitempty" json:"start,omitempty"`
	Stop        string `yaml:"stop,omitempty" json:"stop,omitempty"`

	Host struct {
		As      string `yaml:"as" json:"as"`
		Kind    string `yaml:"kind" json:"kind"`
		OS      string `yaml:"os,omitempty" json:"os,omitempty"`
		Image   string `yaml:"image,omitempty" json:"image,omitempty"`
		Size    string `yaml:"size" json:"size"`
		Address string `yaml:"address" json:"address"`
	} `yaml:"host" json:"host"`

	Network struct {
		Name  string   `yaml:"name" json:"name"`
		CIDR  string   `yaml:"cidr" json:"cidr"`
		Peers []string `yaml:"other_hosts_on_this_network" json:"other_hosts_on_this_network"`
	} `yaml:"network" json:"network"`

	Vars []VarView `yaml:"vars" json:"vars"`

	PeopleSources []PeopleView `yaml:"people_sources" json:"people_sources"`
}

type VarView struct {
	Key      string   `yaml:"key" json:"key"`
	Value    string   `yaml:"value" json:"value"`
	Source   string   `yaml:"source" json:"source"`
	Shadowed []string `yaml:"shadowed,omitempty" json:"shadowed,omitempty"`
}

type PeopleView struct {
	Name    string   `yaml:"name" json:"name"`
	Rows    int      `yaml:"rows" json:"rows"`
	Columns []string `yaml:"columns" json:"columns"`
}

// NewContextView builds the presentation view of a resolved Context --
// the same data `laforge context` prints as YAML.
func NewContextView(ctx *Context) ContextView {
	var v ContextView
	v.Environment = ctx.EnvironmentName
	v.Team = ctx.Team
	v.Teams = ctx.TeamsTotal
	v.Start = ctx.Start
	v.Stop = ctx.Stop
	v.Host.As = ctx.As
	v.Host.Kind = ctx.ObjectKind
	v.Host.OS = ctx.OS
	v.Host.Image = ctx.Image
	v.Host.Size = ctx.Size
	v.Host.Address = ctx.Address
	v.Network.Name = ctx.NetworkName
	v.Network.CIDR = ctx.NetworkCIDR
	for _, p := range ctx.Peers {
		v.Network.Peers = append(v.Network.Peers, fmt.Sprintf("%s (%s)", p.As, p.Kind))
	}
	for _, ve := range ctx.Vars {
		vv := VarView{Key: ve.Key, Value: ve.Value, Source: ve.Source}
		for _, s := range ve.Shadowed {
			vv.Shadowed = append(vv.Shadowed, fmt.Sprintf("%s=%s", s.Source, s.Value))
		}
		v.Vars = append(v.Vars, vv)
	}
	for _, ps := range ctx.People {
		v.PeopleSources = append(v.PeopleSources, PeopleView{Name: ps.Name, Rows: ps.Rows, Columns: ps.Columns})
	}
	return v
}
