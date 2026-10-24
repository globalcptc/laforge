package microcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/globalcptc/laforge/internal/builder"
)

// externalNameKey records, on every instance LaForge creates, the deployment
// it was created for (the runner's deterministic external name). An instance
// found already existing at create time -- an earlier attempt that was
// interrupted (a runner restart, a lost connection) after the cluster
// accepted it -- is adopted only when it carries this deployment's name.
const externalNameKey = "user.laforge_external_name"

// instanceOwner is who an instance being created is for.
type instanceOwner struct {
	externalName string
	team         string
}

// adoptExisting takes over an instance that already exists at the name being
// created. The cluster keeps running a create it accepted even when the
// client that asked is gone, so first any creation or copy still running
// for it is waited out; then the instance must belong to this deployment
// (externalNameKey, or for instances made before that key existed the
// public NIC's recorded deployment, or the team key) or the deploy stops
// with a permanent error rather than take over someone else's box.
func (b *Builder) adoptExisting(ctx context.Context, name string, owner instanceOwner) error {
	if err := b.Client.waitForInstanceCreation(ctx, name); err != nil {
		return err
	}
	full, _, err := b.getInstancePut(ctx, name)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.HTTPStatus == http.StatusNotFound {
			// The earlier create failed and its record is gone: the caller's
			// next attempt creates it afresh.
			return fmt.Errorf("instance %s vanished while its earlier creation finished; creating it again: %w", name, err)
		}
		return err
	}
	switch {
	case full.Config[externalNameKey] != "":
		if full.Config[externalNameKey] != owner.externalName {
			return fmt.Errorf("%w: instance %s already exists and belongs to deployment %s, not %s; refusing to adopt it",
				builder.ErrPermanentConfig, name, full.Config[externalNameKey], owner.externalName)
		}
	case full.Config[publicNICKey] != "":
		var plan publicNICPlan
		if err := json.Unmarshal([]byte(full.Config[publicNICKey]), &plan); err != nil {
			return err
		}
		if plan.ExternalName != owner.externalName {
			return fmt.Errorf("%w: instance %s already exists and belongs to deployment %s; refusing to adopt it",
				builder.ErrPermanentConfig, name, plan.ExternalName)
		}
	case full.Config[teamConfigKey] != owner.team:
		return fmt.Errorf("%w: instance %s already exists and isn't recorded as this deployment's (team %q, want %q); refusing to adopt it -- remove it at the hoster or rebuild",
			builder.ErrPermanentConfig, name, full.Config[teamConfigKey], owner.team)
	}
	slog.InfoContext(ctx, "MicroCloud instance already existed from an earlier attempt; adopting it", "project", b.Client.Project, "instance", name, "external_name", owner.externalName)
	return nil
}

// waitForInstanceCreation waits out any running operation creating or
// copying the named instance. Completed operations drop out of the list, so
// nothing running means nothing to wait for.
func (c *Client) waitForInstanceCreation(ctx context.Context, name string) error {
	raw, err := c.get(ctx, "/1.0/operations?recursion=1")
	if err != nil {
		return fmt.Errorf("listing operations for %s: %w", name, err)
	}
	var groups map[string][]struct {
		ID          string              `json:"id"`
		Description string              `json:"description"`
		StatusCode  int                 `json:"status_code"`
		Resources   map[string][]string `json:"resources"`
	}
	if err := json.Unmarshal(raw, &groups); err != nil {
		return fmt.Errorf("decoding operations for %s: %w", name, err)
	}
	project := c.Project
	if project == "" {
		project = "default"
	}
	for _, ops := range groups {
		for _, op := range ops {
			if op.StatusCode < 100 || op.StatusCode >= 200 || (op.Description != "Creating instance" && op.Description != "Copying instance") {
				continue
			}
			for _, resource := range op.Resources["instances"] {
				u, err := url.Parse(resource)
				if err != nil || u.Path != "/1.0/instances/"+name {
					continue
				}
				if p := u.Query().Get("project"); p != "" && p != project {
					continue
				}
				slog.InfoContext(ctx, "MicroCloud waiting for an earlier creation of this instance to finish", "project", project, "instance", name, "operation", op.ID)
				if err := c.waitForOperationDone(ctx, "/1.0/operations/"+url.PathEscape(op.ID)); err != nil {
					var apiErr *APIError
					if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusNotFound {
						return fmt.Errorf("waiting for the earlier creation of %s: %w", name, err)
					}
				}
			}
		}
	}
	return nil
}

// waitForOperationDone waits on an operation until it ends, however many of
// the server's long-poll windows that takes. Its outcome isn't returned as
// an error: a failed earlier create is the caller's to discover by reading
// the instance.
func (c *Client) waitForOperationDone(ctx context.Context, opPath string) error {
	for {
		_, err := c.waitForOperation(ctx, opPath)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var pending *pendingOperationError
		if !errors.As(err, &pending) || !errors.Is(pending.Err, errOperationRunning) {
			var apiErr *APIError
			if errors.As(err, &apiErr) && apiErr.HTTPStatus == http.StatusNotFound {
				return err
			}
			return nil // ended, one way or the other
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
