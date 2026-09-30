package microcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/kdomanski/iso9660"
)

// buildCidataISO builds a NoCloud config-drive ISO (volume label CIDATA)
// carrying meta-data and user-data -- the channel a Windows guest's
// cloudbase-init reads, since it can't read the /dev/lxd/sock datasource
// the Linux cloud-init path uses. instanceID must be unique per deploy so
// cloudbase-init treats each boot as a fresh instance and re-runs.
// cidataVolName is the storage-volume name for an instance's config
// drive -- stable for a given instance so a retried deploy reuses/replaces
// it rather than piling up volumes.
func cidataVolName(instanceName string) string {
	return "cidata-" + instanceName
}

func buildCidataISO(instanceID, userData string) ([]byte, error) {
	w, err := iso9660.NewWriter()
	if err != nil {
		return nil, fmt.Errorf("creating ISO writer: %w", err)
	}
	defer w.Cleanup()

	metaData := fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", instanceID, shortName(instanceID))
	if err := w.AddFile(strings.NewReader(metaData), "meta-data"); err != nil {
		return nil, fmt.Errorf("adding meta-data: %w", err)
	}
	if err := w.AddFile(strings.NewReader(userData), "user-data"); err != nil {
		return nil, fmt.Errorf("adding user-data: %w", err)
	}

	var buf bytes.Buffer
	if err := w.WriteTo(&buf, "CIDATA"); err != nil {
		return nil, fmt.Errorf("writing ISO: %w", err)
	}
	return buf.Bytes(), nil
}

// importISOVolume uploads an ISO as a custom storage volume of content
// type "iso" (POST /1.0/storage-pools/{pool}/volumes/custom, the raw ISO
// as the body, name and type on X-Incus-* headers -- the same call
// `incus storage volume import --type iso` makes). Idempotent: an existing
// volume of the same name is replaced, so a retried deploy re-imports
// cleanly.
func (c *Client) importISOVolume(ctx context.Context, pool, name string, iso []byte) error {
	// Replace any stale volume from a previous attempt first.
	_ = c.deleteVolume(ctx, pool, name)

	u := c.BaseURL + "/1.0/storage-pools/" + url.PathEscape(pool) + "/volumes/custom"
	if c.Project != "" {
		u += "?project=" + url.QueryEscape(c.Project)
	}
	reqCtx, cancel := context.WithTimeout(ctx, c.OperationTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, u, bytes.NewReader(iso))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	// Send both the Incus and the LXD header spellings. MicroCloud runs
	// LXD, which reads X-LXD-name/X-LXD-type and ignores the X-Incus-*
	// ones -- so with only the Incus headers the volume type came through
	// empty and LXD treated the raw ISO as a compressed backup, failing
	// with "Unsupported compression". A native Incus server is the mirror
	// image (reads X-Incus-*, ignores X-LXD-*), so setting both makes the
	// ISO import work against either server. Found live on a MicroCloud
	// (LXD) Ceph pool.
	req.Header.Set("X-Incus-name", name)
	req.Header.Set("X-Incus-type", "iso")
	req.Header.Set("X-LXD-name", name)
	req.Header.Set("X-LXD-type", "iso")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("importing ISO volume %s: %w", name, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var ar apiResponse
	if err := json.Unmarshal(data, &ar); err != nil {
		return fmt.Errorf("decoding ISO import response: %w (body: %s)", err, data)
	}
	switch ar.Type {
	case "error":
		return &APIError{HTTPStatus: resp.StatusCode, ErrorCode: ar.ErrorCode, Message: ar.Error}
	case "async":
		_, err := c.waitForOperation(ctx, ar.Operation)
		return err
	default:
		return nil
	}
}

// deleteVolume removes a custom storage volume (used to clean up a config
// drive once its instance is gone, and before re-importing). A missing
// volume is not an error.
func (c *Client) deleteVolume(ctx context.Context, pool, name string) error {
	_, err := c.delete(ctx, "/1.0/storage-pools/"+url.PathEscape(pool)+"/volumes/custom/"+url.PathEscape(name))
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.NotFound() {
			return nil
		}
	}
	return err
}
