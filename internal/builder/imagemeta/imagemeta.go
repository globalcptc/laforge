// Package imagemeta resolves an OCI image's own ENTRYPOINT+CMD from its
// registry. Container builders that run the LaForge agent as the container's
// supervising entrypoint need to know what real command the agent should
// supervise; a platform that hands back the image's entrypoint itself (Incus)
// doesn't need this, but ones that don't (AWS Fargate, OpenStack Zun) do, and
// they need it identically -- so the Docker Hub registry dance lives here once
// rather than duplicated per builder.
//
// DRAFT scope, same as its callers: anonymous Docker Hub pulls only. A private
// registry, or any registry other than docker.io, needs auth not modeled here;
// setting `command:` on the container avoids this path entirely (the caller
// supervises that instead and never calls Entrypoint).
package imagemeta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Entrypoint fetches imageRef's ENTRYPOINT+CMD from its registry and joins them
// into a single shell command string -- what a container's supervising agent
// should run. Returns an error for a reference this draft can't resolve (a
// non-docker.io registry) or an image that declares no entrypoint/command.
func Entrypoint(ctx context.Context, imageRef string) (string, error) {
	repo, tag := ParseDockerRef(imageRef)
	if repo == "" {
		return "", fmt.Errorf("image %q is not a Docker Hub reference this draft can resolve -- set command:", imageRef)
	}
	token, err := dockerAnonToken(ctx, repo)
	if err != nil {
		return "", err
	}
	const acceptManifests = "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json"
	manURL := fmt.Sprintf("https://registry-1.docker.io/v2/%s/manifests/%s", repo, tag)
	raw, err := registryGet(ctx, manURL, token, acceptManifests)
	if err != nil {
		return "", err
	}
	// A multi-arch index lists per-platform manifests; pick linux/amd64 (the
	// agent's arch) and fetch that manifest instead.
	var idx struct {
		Manifests []struct {
			Digest   string `json:"digest"`
			Platform struct {
				OS   string `json:"os"`
				Arch string `json:"architecture"`
			} `json:"platform"`
		} `json:"manifests"`
	}
	if json.Unmarshal(raw, &idx); len(idx.Manifests) > 0 {
		digest := ""
		for _, m := range idx.Manifests {
			if m.Platform.OS == "linux" && m.Platform.Arch == "amd64" {
				digest = m.Digest
				break
			}
		}
		if digest == "" {
			return "", fmt.Errorf("image %q has no linux/amd64 manifest", imageRef)
		}
		raw, err = registryGet(ctx, fmt.Sprintf("https://registry-1.docker.io/v2/%s/manifests/%s", repo, digest), token, acceptManifests)
		if err != nil {
			return "", err
		}
	}
	var man struct {
		Config struct {
			Digest string `json:"digest"`
		} `json:"config"`
	}
	if err := json.Unmarshal(raw, &man); err != nil || man.Config.Digest == "" {
		return "", fmt.Errorf("image %q: could not read config from manifest", imageRef)
	}
	blob, err := registryGet(ctx, fmt.Sprintf("https://registry-1.docker.io/v2/%s/blobs/%s", repo, man.Config.Digest), token, "application/vnd.oci.image.config.v1+json, application/vnd.docker.container.image.v1+json, */*")
	if err != nil {
		return "", err
	}
	var cfg struct {
		Config struct {
			Entrypoint []string `json:"Entrypoint"`
			Cmd        []string `json:"Cmd"`
		} `json:"config"`
	}
	if err := json.Unmarshal(blob, &cfg); err != nil {
		return "", err
	}
	parts := append(append([]string{}, cfg.Config.Entrypoint...), cfg.Config.Cmd...)
	if len(parts) == 0 {
		return "", fmt.Errorf("image %q declares no entrypoint or command -- set command: on the container", imageRef)
	}
	return strings.Join(parts, " "), nil
}

// ParseDockerRef splits a Docker Hub image ref into repo (library/-qualified for
// official images) and tag. Returns "" repo for a ref that names another registry
// (has a host before the first "/"), which this draft can't resolve.
func ParseDockerRef(ref string) (repo, tag string) {
	tag = "latest"
	if i := strings.LastIndex(ref, ":"); i >= 0 && i > strings.LastIndex(ref, "/") {
		tag = ref[i+1:]
		ref = ref[:i]
	}
	if first, _, ok := strings.Cut(ref, "/"); ok && strings.ContainsAny(first, ".:") {
		return "", tag // a registry host -- not docker.io
	}
	if !strings.Contains(ref, "/") {
		ref = "library/" + ref
	}
	return ref, tag
}

func dockerAnonToken(ctx context.Context, repo string) (string, error) {
	body, err := registryGet(ctx, fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", repo), "", "application/json")
	if err != nil {
		return "", err
	}
	var t struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &t); err != nil {
		return "", err
	}
	return t.Token, nil
}

func registryGet(ctx context.Context, url, token, accept string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	return data, nil
}
