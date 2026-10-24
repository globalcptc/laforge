package incus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/globalcptc/laforge/internal/builder"
)

func publicACLName(ref string) string {
	return "lfpacl" + strings.TrimPrefix(hostACLName(ref), "lfhacl")
}

// NAT preserves the client's source address. Public ports need NIC allow rules
// in addition to the private visible_from rules.
func (b *Builder) ensurePublicPortACL(ctx context.Context, h builder.ExternalHost) error {
	name := publicACLName(h.ExternalRef)
	rules := nicIngressRules([]string{"0.0.0.0/0"}, builder.HostAccess{TCPPorts: h.TCPPorts, UDPPorts: h.UDPPorts})
	if err := b.ensureACLObject(ctx, name, "LaForge public ports for "+h.ExternalRef, rules); err != nil {
		return err
	}
	return b.attachNICACL(ctx, h.ExternalRef, name)
}

// Private and public access tasks may run concurrently. Merge the current ACL
// list with an ETag precondition so neither task can detach the other's rules.
func (b *Builder) attachNICACL(ctx context.Context, ref, name string) error {
	path := "/1.0/instances/" + url.PathEscape(ref)
	return retryOnBusy(ctx, func() error {
		for attempt := 0; attempt < 5; attempt++ {
			var headers http.Header
			raw, err := b.Client.doWithHeaders(ctx, http.MethodGet, path, nil, nil, &headers)
			if err != nil {
				return err
			}
			var instance struct {
				Devices map[string]map[string]string `json:"devices"`
			}
			if err := json.Unmarshal(raw, &instance); err != nil {
				return err
			}
			nic := instance.Devices["eth0"]
			if nic == nil || nic["type"] != "nic" {
				return fmt.Errorf("instance %s has no active eth0 NIC", ref)
			}
			names := []string{}
			for _, acl := range strings.Split(nic["security.acls"], ",") {
				if acl = strings.TrimSpace(acl); acl != "" && acl != name {
					names = append(names, acl)
				}
			}
			nic["security.acls"] = strings.Join(append(names, name), ",")
			nic["security.acls.default.ingress.action"] = "drop"
			nic["security.acls.default.egress.action"] = "allow"
			etag := headers.Get("ETag")
			if etag == "" {
				return fmt.Errorf("instance %s response has no ETag; cannot safely merge NIC ACLs", ref)
			}
			requestHeaders := http.Header{"If-Match": []string{etag}}
			_, err = b.Client.doWithHeaders(ctx, http.MethodPatch, path, map[string]interface{}{
				"devices": map[string]interface{}{"eth0": nic},
			}, requestHeaders, nil)
			var apiErr *APIError
			if errors.As(err, &apiErr) && apiErr.HTTPStatus == http.StatusPreconditionFailed {
				continue
			}
			return err
		}
		return fmt.Errorf("instance %s kept changing while attaching ACL %s; retry the task", ref, name)
	})
}
