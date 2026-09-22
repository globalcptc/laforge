package incus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/gen0cide/laforge/ent"
	incusclient "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

const (
	gatewayApplyPath    = "/usr/local/sbin/laforge-gateway-apply"
	gatewayTransitIf    = "eth0"
	ingressDevicePrefix = "ingress-"
	networkDevicePrefix = "net-"
)

// gatewayNetwork is one team network attached to the team gateway.
type gatewayNetwork struct {
	Bridge    string
	Interface string
	// Address is the gateway address with prefix length, e.g. 10.0.1.254/24.
	Address string
	// Subnet is the network base address and dotted netmask for DHCP.
	Subnet string
	Mask   string
}

// gatewayReservation pins a VM NIC MAC address to its LaForge SubnetIP.
type gatewayReservation struct {
	MAC string
	IP  string
}

// gatewayIngress publishes one host port on the team gateway.
type gatewayIngress struct {
	Protocol   string
	HostPort   int
	TargetIP   string
	TargetPort int
}

type gatewaySpec struct {
	Networks     []gatewayNetwork
	Reservations []gatewayReservation
	Ingress      []gatewayIngress
	DNSServers   []string
}

// ingressPorts maps a host's exposed ports onto the per-team port window:
// listen port = base + teamNumber*stride + index. Ports are ordered TCP
// first, then UDP, each sorted numerically, so the mapping is stable.
func ingressPorts(
	host HostConfig,
	teamNumber int,
	targetIP string,
	tcpPorts []string,
	udpPorts []string,
) ([]gatewayIngress, error) {
	type exposed struct {
		protocol string
		port     int
	}

	var ports []exposed
	for _, set := range []struct {
		protocol string
		values   []string
	}{{"tcp", tcpPorts}, {"udp", udpPorts}} {
		parsed := make([]int, 0, len(set.values))
		for _, value := range set.values {
			port, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || port < 1 || port > 65535 {
				return nil, fmt.Errorf("exposed %s port %q must be a single port number", set.protocol, value)
			}
			parsed = append(parsed, port)
		}
		sort.Ints(parsed)
		for _, port := range parsed {
			ports = append(ports, exposed{protocol: set.protocol, port: port})
		}
	}

	if host.IngressListenAddress == "" {
		return nil, fmt.Errorf("Incus host %q has no ingress_listen_address configured", host.Name)
	}
	if len(ports) > host.IngressPortStride {
		return nil, fmt.Errorf(
			"ingress host exposes %d ports, more than ingress_port_stride %d on Incus host %q",
			len(ports),
			host.IngressPortStride,
			host.Name,
		)
	}

	result := make([]gatewayIngress, 0, len(ports))
	for index, port := range ports {
		hostPort := host.IngressPortBase + teamNumber*host.IngressPortStride + index
		if hostPort > 65535 {
			return nil, fmt.Errorf(
				"team %d ingress port %d exceeds 65535; lower ingress_port_base or ingress_port_stride on Incus host %q",
				teamNumber,
				hostPort,
				host.Name,
			)
		}
		result = append(result, gatewayIngress{
			Protocol:   port.protocol,
			HostPort:   hostPort,
			TargetIP:   targetIP,
			TargetPort: port.port,
		})
	}

	return result, nil
}

// gatewayDevices is the complete desired device set for a team gateway.
func gatewayDevices(spec gatewaySpec, host HostConfig) map[string]map[string]string {
	devices := map[string]map[string]string{
		"root": {
			"type": "disk",
			"pool": host.StoragePool,
			"path": "/",
		},
		gatewayTransitIf: {
			"type":    "nic",
			"network": host.TransitNetwork,
			"name":    gatewayTransitIf,
		},
	}

	for _, network := range spec.Networks {
		devices[networkDevicePrefix+network.Bridge] = map[string]string{
			"type":    "nic",
			"network": network.Bridge,
			"name":    network.Interface,
			"hwaddr":  macAddress("gateway-" + network.Bridge),
		}
	}

	for _, ingress := range spec.Ingress {
		devices[fmt.Sprintf("%s%s-%d", ingressDevicePrefix, ingress.Protocol, ingress.HostPort)] = map[string]string{
			"type": "proxy",
			"bind": "host",
			"listen": fmt.Sprintf(
				"%s:%s:%d",
				ingress.Protocol,
				host.IngressListenAddress,
				ingress.HostPort,
			),
			"connect": fmt.Sprintf(
				"%s:%s:%d",
				ingress.Protocol,
				ingress.TargetIP,
				ingress.TargetPort,
			),
		}
	}

	return devices
}

// renderGatewayScript renders the idempotent script that configures the
// gateway: addresses, forwarding, NAT and DHCP reservations.
func renderGatewayScript(spec gatewaySpec) string {
	networks := append([]gatewayNetwork(nil), spec.Networks...)
	sort.Slice(networks, func(i, j int) bool { return networks[i].Bridge < networks[j].Bridge })
	reservations := append([]gatewayReservation(nil), spec.Reservations...)
	sort.Slice(reservations, func(i, j int) bool { return reservations[i].MAC < reservations[j].MAC })

	var script bytes.Buffer
	script.WriteString("#!/bin/sh\nset -eu\n")
	script.WriteString("cloud-init status --wait >/dev/null 2>&1 || true\n")
	script.WriteString("sysctl -w net.ipv4.ip_forward=1 >/dev/null\n")
	script.WriteString("mkdir -p /etc/sysctl.d\n")
	script.WriteString("echo 'net.ipv4.ip_forward=1' > /etc/sysctl.d/90-laforge.conf\n")

	for _, network := range networks {
		fmt.Fprintf(&script, "ip addr replace %s dev %s\n", network.Address, network.Interface)
		fmt.Fprintf(&script, "ip link set %s up\n", network.Interface)
	}

	script.WriteString("nft -f - <<'LAFORGE_NFT'\n")
	script.WriteString("table ip laforge\n")
	script.WriteString("delete table ip laforge\n")
	script.WriteString("table ip laforge {\n")
	script.WriteString("\tchain postrouting {\n")
	script.WriteString("\t\ttype nat hook postrouting priority srcnat; policy accept;\n")
	fmt.Fprintf(&script, "\t\toifname %q masquerade\n", gatewayTransitIf)
	script.WriteString("\t}\n}\nLAFORGE_NFT\n")

	script.WriteString("mkdir -p /etc/dnsmasq.d\n")
	script.WriteString("cat > /etc/dnsmasq.d/laforge.conf <<'LAFORGE_DNSMASQ'\n")
	script.WriteString("port=0\ndhcp-authoritative\n")
	dns := strings.Join(spec.DNSServers, ",")
	for _, network := range networks {
		fmt.Fprintf(&script, "interface=%s\n", network.Interface)
		fmt.Fprintf(&script, "dhcp-range=set:%s,%s,static,%s,12h\n", network.Interface, network.Subnet, network.Mask)
		fmt.Fprintf(&script, "dhcp-option=tag:%s,option:dns-server,%s\n", network.Interface, dns)
	}
	for _, reservation := range reservations {
		fmt.Fprintf(&script, "dhcp-host=%s,%s\n", reservation.MAC, reservation.IP)
	}
	script.WriteString("LAFORGE_DNSMASQ\n")
	script.WriteString("systemctl restart dnsmasq\n")

	return script.String()
}

// gatewayNetworkFor derives the gateway attachment for a provisioned network.
func gatewayNetworkFor(bridge string, cidr string) (gatewayNetwork, error) {
	address, err := networkGateway(cidr)
	if err != nil {
		return gatewayNetwork{}, err
	}
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return gatewayNetwork{}, fmt.Errorf("invalid network CIDR %q: %w", cidr, err)
	}

	return gatewayNetwork{
		Bridge:    bridge,
		Interface: gatewayInterface(bridge),
		Address:   address,
		Subnet:    ipNet.IP.String(),
		Mask:      net.IP(ipNet.Mask).String(),
	}, nil
}

func networkGateway(cidr string) (string, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid network CIDR %q: %w", cidr, err)
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		return "", fmt.Errorf("Incus builder only supports IPv4 networks, got %q", cidr)
	}

	ones, bits := ipNet.Mask.Size()
	if bits != 32 || ones > 24 {
		return "", fmt.Errorf(
			"network CIDR %q cannot use the required .254 gateway",
			cidr,
		)
	}

	gateway := append(net.IP(nil), ipv4...)
	gateway[3] = 254
	if !ipNet.Contains(gateway) {
		return "", fmt.Errorf("gateway %s is outside network CIDR %q", gateway, cidr)
	}

	return fmt.Sprintf("%s/%d", gateway.String(), ones), nil
}

// reconcileExclusions names rows that are being torn down and must be left
// out of the desired gateway state even though their ent rows still exist.
type reconcileExclusions struct {
	networks map[string]struct{}
	hosts    map[string]struct{}
}

// buildGatewaySpec reads the team's current networks and hosts from ent.
func (builder *IncusBuilder) buildGatewaySpec(
	ctx context.Context,
	team *ent.Team,
	host HostConfig,
	exclude reconcileExclusions,
) (gatewaySpec, error) {
	spec := gatewaySpec{DNSServers: builder.Config.dnsServers()}

	provisionedNetworks, err := team.QueryProvisionedNetworks().All(ctx)
	if err != nil {
		return spec, fmt.Errorf("failed to query networks for team %d: %w", team.TeamNumber, err)
	}

	var ingressHost string
	for _, provisionedNetwork := range provisionedNetworks {
		bridge := provisionedNetwork.Vars[varNetwork]
		if bridge == "" {
			continue
		}
		if _, skip := exclude.networks[bridge]; skip {
			continue
		}

		network, err := gatewayNetworkFor(bridge, provisionedNetwork.Cidr)
		if err != nil {
			return spec, err
		}
		spec.Networks = append(spec.Networks, network)

		provisionedHosts, err := provisionedNetwork.QueryProvisionedHosts().All(ctx)
		if err != nil {
			return spec, fmt.Errorf("failed to query hosts for network %q: %w", provisionedNetwork.Name, err)
		}
		for _, provisionedHost := range provisionedHosts {
			if provisionedHost.SubnetIP == "" {
				continue
			}
			if _, skip := exclude.hosts[provisionedHost.ID.String()]; skip {
				continue
			}
			spec.Reservations = append(spec.Reservations, gatewayReservation{
				MAC: macAddress(provisionedHost.ID.String()),
				IP:  provisionedHost.SubnetIP,
			})

			laforgeHost, err := provisionedHost.QueryHost().Only(ctx)
			if err != nil {
				return spec, fmt.Errorf("failed to query host for provisioned host %s: %w", provisionedHost.ID, err)
			}
			if laforgeHost.Vars["public_ingress"] != "true" {
				continue
			}
			if ingressHost != "" {
				return spec, fmt.Errorf(
					"team %d already has public ingress for %q; only one ingress host per team is supported",
					team.TeamNumber,
					ingressHost,
				)
			}
			ingressHost = laforgeHost.Hostname

			ingress, err := ingressPorts(
				host,
				team.TeamNumber,
				provisionedHost.SubnetIP,
				laforgeHost.ExposedTCPPorts,
				laforgeHost.ExposedUDPPorts,
			)
			if err != nil {
				return spec, err
			}
			if len(ingress) == 0 {
				return spec, fmt.Errorf("public ingress host %q has no exposed ports", laforgeHost.Hostname)
			}
			spec.Ingress = append(spec.Ingress, ingress...)
		}
	}

	return spec, nil
}

// reconcileGateway makes the team gateway match the team's current networks,
// hosts and ingress. It is the single place that mutates the gateway.
func (builder *IncusBuilder) reconcileGateway(
	ctx context.Context,
	team *ent.Team,
	exclude reconcileExclusions,
) error {
	lock := builder.teamLock(team)
	lock.Lock()
	defer lock.Unlock()

	team, err := refreshTeam(ctx, team)
	if err != nil {
		return err
	}
	client, host, project, err := builder.teamTarget(team)
	if err != nil {
		return err
	}
	client = client.UseProject(project)

	spec, err := builder.buildGatewaySpec(ctx, team, host, exclude)
	if err != nil {
		return err
	}

	gateway := team.Vars[varGateway]
	instance, etag, err := client.GetInstance(gateway)
	if err != nil {
		return fmt.Errorf("failed to get Incus gateway %q: %w", gateway, err)
	}

	desired := gatewayDevices(spec, host)
	if !reflect.DeepEqual(instance.Devices, desired) {
		put := instance.Writable()
		put.Devices = desired
		op, err := client.UpdateInstance(gateway, put, etag)
		if err != nil {
			return fmt.Errorf("failed to update devices on Incus gateway %q: %w", gateway, err)
		}
		if err := op.Wait(); err != nil {
			return fmt.Errorf("failed waiting for Incus gateway %q devices: %w", gateway, err)
		}
	}

	if err := applyGatewayScript(client, gateway, renderGatewayScript(spec)); err != nil {
		return err
	}

	return builder.setTeamIngress(ctx, team, host, spec)
}

func (builder *IncusBuilder) setTeamIngress(
	ctx context.Context,
	team *ent.Team,
	host HostConfig,
	spec gatewaySpec,
) error {
	vars := cloneVars(team.Vars)
	if len(spec.Ingress) == 0 {
		delete(vars, varIngressAddr)
		delete(vars, varIngressPorts)
	} else {
		ports := make(map[string]int, len(spec.Ingress))
		for _, ingress := range spec.Ingress {
			ports[fmt.Sprintf("%s/%d", ingress.Protocol, ingress.TargetPort)] = ingress.HostPort
		}
		encoded, err := json.Marshal(ports)
		if err != nil {
			return fmt.Errorf("failed to encode ingress ports for team %d: %w", team.TeamNumber, err)
		}
		vars[varIngressAddr] = host.IngressListenAddress
		vars[varIngressPorts] = string(encoded)
	}

	if reflect.DeepEqual(vars, team.Vars) {
		return nil
	}
	if err := team.Update().SetVars(vars).Exec(ctx); err != nil {
		return fmt.Errorf("failed to update ingress vars for team %d: %w", team.TeamNumber, err)
	}
	team.Vars = vars

	return nil
}

func applyGatewayScript(client incusclient.InstanceServer, gateway string, script string) error {
	if err := client.CreateInstanceFile(gateway, gatewayApplyPath, incusclient.InstanceFileArgs{
		Content:   strings.NewReader(script),
		UID:       0,
		GID:       0,
		Mode:      0o755,
		Type:      "file",
		WriteMode: "overwrite",
	}); err != nil {
		return fmt.Errorf("failed to write apply script to Incus gateway %q: %w", gateway, err)
	}

	var output bytes.Buffer
	dataDone := make(chan bool)
	op, err := client.ExecInstance(gateway, api.InstanceExecPost{
		Command:      []string{gatewayApplyPath},
		WaitForWS:    true,
		Interactive:  false,
		RecordOutput: false,
	}, &incusclient.InstanceExecArgs{
		Stdin:    nil,
		Stdout:   nopWriteCloser{&output},
		Stderr:   nopWriteCloser{&output},
		DataDone: dataDone,
	})
	if err != nil {
		return fmt.Errorf("failed to run apply script on Incus gateway %q: %w", gateway, err)
	}
	if err := op.Wait(); err != nil {
		return fmt.Errorf("apply script failed on Incus gateway %q: %w", gateway, err)
	}
	<-dataDone

	metadata := op.Get().Metadata
	if code, ok := metadata["return"].(float64); ok && code != 0 {
		return fmt.Errorf(
			"apply script on Incus gateway %q exited %d: %s",
			gateway,
			int(code),
			strings.TrimSpace(output.String()),
		)
	}

	return nil
}

type nopWriteCloser struct {
	*bytes.Buffer
}

func (nopWriteCloser) Close() error { return nil }
