package microcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// exec runs a command in a RUNNING instance via POST /1.0/instances/{name}/exec
// and returns its exit code -- the same call `incus exec` / `lxc exec` makes.
// record-output is on (so a later enhancement can retrieve stdout/stderr) but
// this only reads the return code, matching the Incus builder's own exec: a
// nonzero code is a real command failure the caller surfaces. c.post waits for
// the background operation to finish (waitForOperation), so Metadata.Return is
// the command's real exit status, not a pending placeholder.
func (c *Client) exec(ctx context.Context, instance string, command ...string) (int, error) {
	raw, err := c.post(ctx, "/1.0/instances/"+url.PathEscape(instance)+"/exec", map[string]interface{}{
		"command":            command,
		"wait-for-websocket": false,
		"record-output":      true,
		"interactive":        false,
	})
	if err != nil {
		return -1, err
	}
	var op struct {
		Metadata struct {
			Return int `json:"return"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &op); err != nil {
		return -1, fmt.Errorf("decoding exec result: %w (body: %s)", err, raw)
	}
	return op.Metadata.Return, nil
}

// pushFile writes content into an instance at path with the given octal mode,
// owned by root -- POST /1.0/instances/{name}/files?path=..., the same call
// `incus file push` makes. Works on a stopped instance (the server mounts its
// rootfs), which is how the builder plants the LaForge agent on the Docker host
// before running the nested container. Both the Incus and LXD header spellings
// are sent so it works against either server, exactly like importISOVolume.
func (c *Client) pushFile(ctx context.Context, instance, path string, mode int, content []byte) error {
	u := c.BaseURL + "/1.0/instances/" + url.PathEscape(instance) + "/files?path=" + url.QueryEscape(path)
	if c.Project != "" {
		u += "&project=" + url.QueryEscape(c.Project)
	}
	reqCtx, cancel := context.WithTimeout(ctx, c.OperationTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, u, bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	m := fmt.Sprintf("%04o", mode)
	for _, prefix := range []string{"X-Incus", "X-LXD"} {
		req.Header.Set(prefix+"-type", "file")
		req.Header.Set(prefix+"-uid", "0")
		req.Header.Set(prefix+"-gid", "0")
		req.Header.Set(prefix+"-mode", m)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("pushing file %s to %s: %w", path, instance, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if len(data) == 0 {
		return nil // 200 with empty body -- file written
	}
	var ar apiResponse
	if err := json.Unmarshal(data, &ar); err != nil {
		return fmt.Errorf("decoding file-push response: %w (body: %s)", err, data)
	}
	if ar.Type == "error" {
		return &APIError{HTTPStatus: resp.StatusCode, ErrorCode: ar.ErrorCode, Message: ar.Error}
	}
	return nil
}
