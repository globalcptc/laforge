package incus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"net/url"

	"github.com/globalcptc/laforge/internal/builder"
)

func (b *Builder) DeployNetwork(ctx context.Context, spec builder.NetworkSpec) (string, error) {
	mode := b.Config.NetworkMode()
	if mode != "bridge" && mode != "ovn" {
		return "", fmt.Errorf("unsupported Incus network type %q", mode)
	}
	if mode == "ovn" && b.Config.OVNUplinkNetwork == "" {
		return "", fmt.Errorf("OVN networking requires an uplink network")
	}
	name := networkName(spec.DisplayName, spec.ExternalName)
	gateway, err := bridgeAddressCIDR(spec.CIDR)
	if err != nil {
		return "", fmt.Errorf("computing gateway address for %s: %w", spec.CIDR, err)
	}
	config := map[string]string{"ipv4.address": gateway, "ipv4.nat": "true", "ipv6.address": "none"}
	if mode == "bridge" {
		if err := b.checkBridgeNetwork(ctx, name, spec.CIDR); err != nil {
			return "", err
		}
		config["bridge.driver"] = "native"
		config["bridge.mtu"] = "1500"
		config["ipv4.dhcp"] = "true"
	} else {
		config["network"] = b.Config.OVNUplinkNetwork
	}
	_, err = b.Client.post(ctx, "/1.0/networks", map[string]interface{}{"name": name, "type": mode, "config": config})
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.AlreadyExists() {
			raw, err := b.Client.get(ctx, "/1.0/networks/"+url.PathEscape(name))
			if err != nil {
				return "", err
			}
			var existing struct {
				Type   string
				Config map[string]string
			}
			if err := json.Unmarshal(raw, &existing); err != nil {
				return "", err
			}
			if existing.Type != mode || existing.Config["ipv4.address"] != gateway || (mode == "bridge" && existing.Config["bridge.driver"] != "" && existing.Config["bridge.driver"] != "native") {
				return "", fmt.Errorf("existing network %s does not match requested %s network at %s; rebuild before changing network mode", name, mode, gateway)
			}
			return name, nil
		}
		return "", fmt.Errorf("creating %s network %s: %w", mode, name, err)
	}
	return name, nil
}

// Bridges share the host routing table. Check before creating anything, including
// overlap with another build or a non-LaForge network on the same server.
func (b *Builder) checkBridgeNetwork(ctx context.Context, name, cidr string) error {
	supported, err := b.Client.bridgeACLSupported(ctx)
	if err != nil {
		return err
	}
	if !supported {
		return fmt.Errorf("native bridge networking requires Incus network_bridge_acl_devices support and the nftables firewall")
	}
	wanted, err := netip.ParsePrefix(cidr)
	if err != nil {
		return err
	}
	raw, err := b.Client.get(ctx, "/1.0/networks?recursion=1")
	if err != nil {
		return fmt.Errorf("checking bridge subnets: %w", err)
	}
	var networks []struct {
		Name, Type string
		Managed    bool
		Config     map[string]string
	}
	if err := json.Unmarshal(raw, &networks); err != nil {
		return err
	}
	for _, n := range networks {
		if n.Name == name || !n.Managed || n.Type != "bridge" {
			continue
		}
		prefix, err := netip.ParsePrefix(n.Config["ipv4.address"])
		if err == nil && wanted.Overlaps(prefix) {
			return fmt.Errorf("bridge subnet %s overlaps existing network %s (%s); use a free host or tear down its previous deployment", cidr, n.Name, prefix)
		}
	}
	return nil
}

func (c *Client) bridgeACLSupported(ctx context.Context) (bool, error) {
	raw, err := c.get(ctx, "/1.0")
	if err != nil {
		return false, fmt.Errorf("detecting bridge networking support: %w", err)
	}
	var server struct {
		APIExtensions []string `json:"api_extensions"`
		Environment   struct {
			Firewall string `json:"firewall"`
		} `json:"environment"`
	}
	if err := json.Unmarshal(raw, &server); err != nil {
		return false, err
	}
	for _, extension := range server.APIExtensions {
		if extension == "network_bridge_acl_devices" {
			return server.Environment.Firewall == "nftables", nil
		}
	}
	return false, nil
}
