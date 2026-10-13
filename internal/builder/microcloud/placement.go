// Host-blocklist placement: when Config.HostBlocklist names cluster members
// LaForge must not use, the builder explicitly targets an ALLOWED member for each
// instance instead of letting LXD auto-schedule (which could land on a blocked
// one). With no blocklist this is a no-op and LXD schedules as before.
package microcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
)

// targetMember returns the cluster member to place the named instance on, or ""
// to let LXD auto-schedule (the blocklist is empty). With a blocklist it picks,
// DETERMINISTICALLY by instance name, one of the allowed online members -- so a
// given instance always targets the same member (stable across create retries)
// while instances spread evenly across the allowed set.
func (b *Builder) targetMember(ctx context.Context, name string) (string, error) {
	if len(b.Config.HostBlocklist) == 0 {
		return "", nil
	}
	members, err := b.allowedMembers(ctx)
	if err != nil {
		return "", err
	}
	if len(members) == 0 {
		return "", fmt.Errorf("every online cluster member is on the host blocklist -- nowhere to place %q", name)
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return members[int(h.Sum32())%len(members)], nil
}

// allowedMembers is the online cluster members not on Config.HostBlocklist,
// sorted and cached for the builder's life (a build's cluster membership is
// stable enough, and this avoids a members list per instance).
func (b *Builder) allowedMembers(ctx context.Context) ([]string, error) {
	b.membersOnce.Do(func() { b.membersCached, b.membersErr = b.resolveAllowedMembers(ctx) })
	return b.membersCached, b.membersErr
}

func (b *Builder) resolveAllowedMembers(ctx context.Context) ([]string, error) {
	raw, err := b.Client.get(ctx, "/1.0/cluster/members?recursion=1")
	if err != nil {
		return nil, fmt.Errorf("listing cluster members: %w", err)
	}
	var members []struct {
		ServerName string `json:"server_name"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal(raw, &members); err != nil {
		return nil, fmt.Errorf("decoding cluster members: %w", err)
	}
	blocked := map[string]bool{}
	for _, m := range b.Config.HostBlocklist {
		blocked[m] = true
	}
	var allowed []string
	for _, m := range members {
		if strings.EqualFold(m.Status, "online") && !blocked[m.ServerName] {
			allowed = append(allowed, m.ServerName)
		}
	}
	sort.Strings(allowed)
	return allowed, nil
}
