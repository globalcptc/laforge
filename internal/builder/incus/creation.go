package incus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// A /wait timeout is only the end of a long poll. Keep following a known
// creation operation while Incus reports it is running, even when a large
// image takes many polling windows to unpack. The runner's context still
// cancels the wait; transport errors and terminal failures remain errors.
func (c *Client) waitForCreation(ctx context.Context, opPath string) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, err := c.waitForOperation(ctx, opPath)
		var pending *operationPendingError
		if !errors.As(err, &pending) || pending.Cause != nil {
			return err
		}
		slog.InfoContext(ctx, "Incus creation still running; continuing to wait", "project", c.Project, "operation", opPath, "status", pending.Status)
		// Normally /wait blocks for OperationTimeout. Avoid a hot loop if
		// the server returns a pending state immediately.
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// Instance power state is not creation state: Incus exposes a stopped instance
// while its image is still being unpacked. Rediscover creation operations when
// a deploy is retried or a runner restarts, before touching that instance or
// its config drive. The list is project-scoped and resources match exactly.
func (c *Client) waitForInstanceCreation(ctx context.Context, name string) error {
	raw, err := c.get(ctx, "/1.0/operations?recursion=1")
	if err != nil {
		return fmt.Errorf("listing creation operations for %s: %w", name, err)
	}
	var groups map[string][]struct {
		ID          string              `json:"id"`
		Description string              `json:"description"`
		StatusCode  int                 `json:"status_code"`
		Resources   map[string][]string `json:"resources"`
	}
	if err := json.Unmarshal(raw, &groups); err != nil {
		return fmt.Errorf("decoding creation operations for %s: %w", name, err)
	}
	for _, ops := range groups {
		for _, op := range ops {
			if op.StatusCode < 100 || op.StatusCode >= 200 ||
				(op.Description != "Creating instance" && op.Description != "Copying instance") {
				continue
			}
			for _, resource := range op.Resources["instances"] {
				u, err := url.Parse(resource)
				if err != nil || u.Path != "/1.0/instances/"+name {
					continue
				}
				project := c.Project
				if project == "" {
					project = "default"
				}
				if p := u.Query().Get("project"); p != "" && p != project {
					continue
				}
				if op.ID == "" {
					return fmt.Errorf("creation operation for %s has no ID", name)
				}
				slog.InfoContext(ctx, "Incus waiting for existing instance creation", "project", project, "instance", name, "operation", op.ID)
				if err := c.waitForCreation(ctx, "/1.0/operations/"+url.PathEscape(op.ID)); err != nil {
					// Completed operations expire from Incus's operation list.
					// A missing operation no longer holds the creation lock.
					var apiErr *APIError
					if !errors.As(err, &apiErr) || apiErr.HTTPStatus != http.StatusNotFound {
						return fmt.Errorf("waiting for creation of %s: %w", name, err)
					}
				}
				break
			}
		}
	}
	return nil
}

func (c *Client) createInstance(ctx context.Context, name string, body interface{}) error {
	_, err := c.post(ctx, "/1.0/instances", body)
	var pending *operationPendingError
	if errors.As(err, &pending) {
		return c.waitForCreation(ctx, pending.Path)
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.AlreadyExists() {
		if err := c.waitForInstanceCreation(ctx, name); err != nil {
			return err
		}
		// A failed create can disappear while we wait. Do not adopt an
		// instance merely because its name existed at the start of the poll.
		_, err = c.get(ctx, "/1.0/instances/"+url.PathEscape(name))
	}
	return err
}
