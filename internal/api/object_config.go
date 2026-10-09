package api

import (
	"errors"
	"net/http"

	"github.com/globalcptc/laforge/internal/loader"
)

// The read side of "show everything we can from a host's config." The deployed
// object records runtime state (status, power, external ref); its authored
// intent -- open ports, env, vars, dependencies, findings, size/disk -- lives in
// the build's content revision. This resolves that content and returns the one
// object's full declarative config for the host info panel.

type objectConfig struct {
	PublicAddress string            `json:"public_address"`
	Kind          string            `json:"kind"`
	Name          string            `json:"name"`
	OS            string            `json:"os,omitempty"`    // host
	Image         string            `json:"image,omitempty"` // container
	Size          string            `json:"size,omitempty"`
	Disk          int               `json:"disk,omitempty"`    // host
	Command       []string          `json:"command,omitempty"` // container
	TCPPorts      []string          `json:"tcp_ports,omitempty"`
	UDPPorts      []string          `json:"udp_ports,omitempty"`
	Env           map[string]string `json:"env,omitempty"` // container
	Vars          map[string]string `json:"vars,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	DependsOn     []string          `json:"depends_on,omitempty"`
	Findings      []loader.Finding  `json:"findings,omitempty"`
}

// handleObjectConfig returns one host/container's full authored config from the
// build's content revision. levelRead -- it's the same content the Hosts page
// already shows, just complete.
func (s *Server) handleObjectConfig(w http.ResponseWriter, r *http.Request) {
	build, repo, err := s.buildAndOwningRepository(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if _, err := s.requireLevel(r.Context(), r, repo, levelRead); err != nil {
		writeAuthError(w, err)
		return
	}
	objectID, err := s.objectInBuild(r, build.ID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	obj, err := s.Queries.GetDeployedObject(r.Context(), objectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if obj.Kind == "network" {
		writeError(w, http.StatusBadRequest, errors.New("a network has no host config"))
		return
	}
	content, err := s.loadBuildContent(r, build)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	cfg := objectConfig{Kind: obj.Kind, Name: obj.ObjectName, PublicAddress: obj.PublicAddress}
	switch obj.Kind {
	case "host":
		for i := range content.Hosts {
			if content.Hosts[i].Name == obj.ObjectName {
				h := content.Hosts[i]
				cfg.OS, cfg.Size, cfg.Disk = h.OS, h.Size, h.Disk
				cfg.TCPPorts, cfg.UDPPorts = h.Ports.TCP, h.Ports.UDP
				cfg.Vars, cfg.Tags, cfg.DependsOn, cfg.Findings = h.Vars, h.Tags, h.DependsOn, h.Findings
				break
			}
		}
	case "container":
		for i := range content.Containers {
			if content.Containers[i].Name == obj.ObjectName {
				c := content.Containers[i]
				cfg.Image, cfg.Size, cfg.Command = c.Image, c.Size, c.Command
				cfg.TCPPorts, cfg.UDPPorts = c.Ports.TCP, c.Ports.UDP
				cfg.Env, cfg.Vars, cfg.Tags = c.Env, c.Vars, c.Tags
				cfg.DependsOn, cfg.Findings = c.DependsOn, c.Findings
				break
			}
		}
	}
	writeJSON(w, http.StatusOK, cfg)
}
