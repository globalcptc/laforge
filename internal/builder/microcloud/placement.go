// Cluster-group placement: LaForge does not pick or balance cluster members. It
// targets the configured LXD cluster group (Config.ClusterGroup, default
// "default" = the whole cluster) and lets LXD/MicroCloud schedule each instance
// within it. This replaced an earlier host-blocklist + per-instance member
// selection; LXD is far better at placement than a name-hash ever was.
package microcloud

import "context"

// clusterTarget is the `target` query-param for creating an instance: the
// configured cluster group as "@<group>", or "" (no explicit target) when none
// is set. It takes ctx only to match the create call site's shape; the value is
// pure config, so there's no cluster lookup and nothing to fail on.
func (b *Builder) clusterTarget(_ context.Context) string {
	return b.Config.clusterTargetParam()
}
