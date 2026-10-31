package microcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/globalcptc/laforge/internal/builder"
)

// Cluster leftovers: when an LXD create or delete dies partway (a cluster
// member that drops out, a Ceph clone that hits an orphaned image), LXD can
// leave pieces behind that its own API no longer cleans up. Retrying a deploy
// against them never succeeds, so these are reported as permanent errors that
// say exactly what is left and how to remove it on the cluster.

// ovnDuplicateIP matches OVN refusing a new port because another logical
// switch port already holds the address.
var ovnDuplicateIP = regexp.MustCompile(`duplicate IPv4 address '([^']+)' found on logical switch port '([^']+)'`)

// portInstanceUUID pulls the LXD instance UUID out of an OVN port name
// (lxd-net<N>-instance-<uuid>-<nic>).
var portInstanceUUID = regexp.MustCompile(`-instance-([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})-`)

// explainLeakedOVNPort turns OVN's duplicate-address refusal into a permanent
// error. If an instance in this project still owns the port, the address is
// really in use and that instance is named; otherwise the port is a leftover
// and the error gives the command that removes it.
func (b *Builder) explainLeakedOVNPort(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	m := ovnDuplicateIP.FindStringSubmatch(err.Error())
	if m == nil {
		return err
	}
	address, port := m[1], m[2]
	uuid := ""
	if u := portInstanceUUID.FindStringSubmatch(port); u != nil {
		uuid = u[1]
	}
	if uuid != "" {
		if owner, ok := b.instanceByUUID(ctx, uuid); ok {
			return fmt.Errorf("%w: address %s is still held by instance %s (OVN port %s); destroy or readdress it before retrying: %v",
				builder.ErrPermanentConfig, address, owner, port, err)
		}
	}
	return fmt.Errorf("%w: OVN port %s still holds address %s but no instance in project %s owns it (LXD left it behind after a create or delete failed partway). "+
		"On a cluster member, confirm nothing owns it with `lxc query \"/1.0/instances?recursion=1&all-projects=true\" | jq -r '.[] | select(.config[\"volatile.uuid\"]==\"%s\") | .name'`, "+
		"then remove it with `microovn.ovn-nbctl lsp-del %s` and retry: %v",
		builder.ErrPermanentConfig, port, address, b.Client.Project, uuid, port, err)
}

// instanceByUUID finds the instance in this project whose volatile.uuid is
// uuid. An unreadable listing reports not found: the caller only uses it to
// make an error message more precise.
func (b *Builder) instanceByUUID(ctx context.Context, uuid string) (string, bool) {
	raw, err := b.Client.get(ctx, "/1.0/instances?recursion=1")
	if err != nil {
		return "", false
	}
	var insts []struct {
		Name   string            `json:"name"`
		Config map[string]string `json:"config"`
	}
	if err := json.Unmarshal(raw, &insts); err != nil {
		return "", false
	}
	for _, in := range insts {
		if in.Config["volatile.uuid"] == uuid {
			return in.Name, true
		}
	}
	return "", false
}

// instancePoolMissing matches LXD failing to delete an instance whose record
// has no root volume behind it -- a create that died before the volume was
// recorded. LXD can't delete such a record through its API.
func instancePoolMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Instance storage pool not found") || strings.Contains(msg, "Failed getting instance pool")
}

// brokenInstanceRecordError is the permanent error for such a record, with the
// cleanup it needs on the cluster.
func (b *Builder) brokenInstanceRecordError(name string, err error) error {
	return fmt.Errorf("%w: LXD has an instance record %s in project %s with no root volume behind it (a create that died partway), and can't delete it through its API. "+
		"On a cluster member remove the record with `lxd sql global \"DELETE FROM instances WHERE name='%s' AND project_id=(SELECT id FROM projects WHERE name='%s')\"`, "+
		"then delete its config drive with `lxc storage volume delete %s custom/cidata-%s --project %s`; a redeploy then reports any OVN port it left: %v",
		builder.ErrPermanentConfig, name, b.Client.Project, name, b.Client.Project, b.Config.storagePoolOrDefault(), name, b.Client.Project, err)
}
