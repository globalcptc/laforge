package microcloud

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// Stale Ceph images: when an instance is deleted while its RBD image is still
// held open (a QEMU that hasn't let go on some member), LXD removes the
// instance from its database but the image stays in the Ceph pool. LXD then no
// longer knows the image exists, so nothing LaForge can ask LXD to delete
// clears it -- and because instance names are deterministic, every redeploy
// of that host collides with it: the root clone fails with EEXIST ("File
// exists") and the config-drive import with "already exists on storage pool".
//
// Rather than leave the host undeployable until someone removes the image by
// hand, a deploy that hits one of those collisions -- for a name LXD has no
// instance under -- moves to a fresh variant of the name and tries again. The
// orphaned image is logged by name: it still needs removing in Ceph (`rbd rm`),
// and anything still watching it should be checked first.

// maxNameFallbacks bounds how many fresh names one deploy tries.
const maxNameFallbacks = 3

type nameSaltKey struct{}

// deployName is instanceName for a deploy: a fallback attempt carries a salt
// in its context, which gives the same readable base a different suffix.
func deployName(ctx context.Context, displayName, externalName, project string) string {
	if salt, _ := ctx.Value(nameSaltKey{}).(string); salt != "" {
		return instanceName(displayName, externalName+"\x00"+salt, project)
	}
	return instanceName(displayName, externalName, project)
}

// isStaleStorageCollision reports whether a deploy failed because the storage
// pool already holds an image under the instance's name.
func isStaleStorageCollision(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if strings.Contains(msg, "already exists on storage pool") {
		return true
	}
	return strings.Contains(msg, "rbd") && (strings.Contains(msg, "File exists") || strings.Contains(msg, "(17)"))
}

// withFreshNameOnStaleStorage runs deploy, and when it fails on a stale image
// for a name LXD holds no instance under, clears what that attempt left behind
// and runs it again under a fresh name. The name returned is the one last
// attempted, so the runner records the instance that actually exists.
func (b *Builder) withFreshNameOnStaleStorage(ctx context.Context, displayName, externalName string, deploy func(context.Context) (string, error)) (string, error) {
	name, err := deploy(ctx)
	for attempt := 1; attempt <= maxNameFallbacks && isStaleStorageCollision(err); attempt++ {
		if name == "" || b.instanceKnown(ctx, name) {
			// An instance LXD knows about is a different problem; leave it.
			break
		}
		slog.WarnContext(ctx, "MicroCloud: the storage pool holds an image LXD no longer tracks under this instance's name (likely still held open when its instance was deleted); deploying under a fresh name -- remove the orphan in Ceph (check `rbd status` for watchers first)",
			"project", b.Client.Project, "instance", name, "external_name", externalName, "error", err)
		b.clearAbandonedName(ctx, name, externalName)
		prev := name
		next, nextErr := deploy(context.WithValue(ctx, nameSaltKey{}, fmt.Sprintf("r%d", attempt)))
		if next == "" {
			next = prev
		}
		name, err = next, nextErr
		if err == nil {
			slog.WarnContext(ctx, "MicroCloud: deployed under a fresh name to step around a stale storage image",
				"project", b.Client.Project, "instance", name, "abandoned_name", prev, "external_name", externalName)
		}
	}
	if err != nil && isStaleStorageCollision(err) {
		if name != "" && b.instanceKnown(ctx, name) {
			// LXD still has an instance under this name holding the volume: a
			// rename wouldn't free its address, so the record has to go first.
			err = fmt.Errorf("%w (LXD still has an instance record %s using this volume; destroy it first -- if its delete reports \"Instance storage pool not found\", the record is broken and needs removing on the cluster)", err, name)
		} else {
			err = fmt.Errorf("%w (the storage pool holds a stale image under %s that LXD no longer tracks -- remove it in Ceph with `rbd rm` after checking `rbd status` for watchers)", err, name)
		}
	}
	return name, b.explainLeakedOVNPort(ctx, err)
}

// instanceKnown reports whether LXD has an instance by this name; an
// unreadable answer counts as known, so it never triggers a rename.
func (b *Builder) instanceKnown(ctx context.Context, name string) bool {
	_, err := b.Client.get(ctx, "/1.0/instances/"+name)
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.HTTPStatus == 404 {
		return false
	}
	return true
}

// clearAbandonedName removes what a failed attempt under name may have left
// that LXD does track -- its config-drive volume, ACLs -- and its public
// address reservation, so the fresh name starts clean and nothing leaks.
func (b *Builder) clearAbandonedName(ctx context.Context, name, externalName string) {
	if err := b.destroyInstance(ctx, name); err != nil {
		slog.WarnContext(ctx, "MicroCloud: clearing an abandoned instance name's leftovers failed", "instance", name, "error", err)
	}
	if b.PublicAddresses != nil {
		if err := b.PublicAddresses.Release(ctx, b.publicProject(), name, externalName); err != nil {
			slog.WarnContext(ctx, "MicroCloud: releasing an abandoned instance name's public address failed", "instance", name, "error", err)
		}
	}
}
