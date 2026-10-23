// External access (content `public:` ports) on AWS: an Elastic IP per host that
// declares public ports, associated to the instance, plus a security-group
// ingress rule for those ports from 0.0.0.0/0. This is the simple public-IP case
// the per-builder design names -- the endpoint is just <public-ip>:<port>, no
// remap (ExternalPort == InternalPort). Idempotent: the EIP is adopted by its
// laforge:external-ref tag, so re-asserting a host reuses the same address.
package aws

import (
	"context"
	"fmt"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/globalcptc/laforge/internal/builder"
)

func (b *Builder) ConfigureExternalAccess(ctx context.Context, team string, hosts []builder.ExternalHost) ([]builder.ExternalEndpoint, error) {
	var out []builder.ExternalEndpoint
	for _, h := range hosts {
		if len(h.TCPPorts) == 0 && len(h.UDPPorts) == 0 {
			continue
		}
		// EIP association addresses an EC2 instance by id. Fargate tasks get a
		// public IP at launch (AssignPublicIp) but no stable EIP; exposing a
		// container's public ports is left as a documented gap rather than
		// silently returning a volatile address.
		if !strings.HasPrefix(h.ExternalRef, "i-") {
			continue
		}
		publicIP, err := b.ensureElasticIP(ctx, team, h.ExternalRef)
		if err != nil {
			return nil, err
		}
		// Open the public ports on the host's own SG from anywhere.
		if sgID, err := b.sgForRef(ctx, h.ExternalRef); err == nil && sgID != "" {
			perms := portPermissions([]string{"0.0.0.0/0"}, h.TCPPorts, h.UDPPorts)
			if len(perms) > 0 {
				if _, err := b.ec2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
					GroupId: awssdk.String(sgID), IpPermissions: perms,
				}); err != nil && !isDuplicate(err) {
					return nil, fmt.Errorf("authorizing public ports on %s: %w", sgID, err)
				}
			}
		}
		for _, p := range h.TCPPorts {
			out = append(out, externalEndpoint(h.ExternalRef, "tcp", p, publicIP))
		}
		for _, p := range h.UDPPorts {
			out = append(out, externalEndpoint(h.ExternalRef, "udp", p, publicIP))
		}
	}
	return out, nil
}

// externalEndpoint builds the endpoint record. On a public-IP builder the
// external port equals the internal one (no remap) and the address is the host's
// own public IP -- per builder.ExternalEndpoint's own doc.
func externalEndpoint(ref, proto, port, publicIP string) builder.ExternalEndpoint {
	return builder.ExternalEndpoint{
		ExternalRef: ref, Protocol: proto, InternalPort: port, ExternalPort: port,
		PublicAddress: publicIP + ":" + port,
	}
}

// ensureElasticIP adopts (by ref tag) or allocates an EIP, associates it with the
// instance, and returns its public IP.
func (b *Builder) ensureElasticIP(ctx context.Context, team, ref string) (string, error) {
	allocID, publicIP, err := b.findElasticIP(ctx, ref)
	if err != nil {
		return "", err
	}
	if allocID == "" {
		alloc, err := b.ec2.AllocateAddress(ctx, &ec2.AllocateAddressInput{
			Domain:            ec2types.DomainTypeVpc,
			TagSpecifications: []ec2types.TagSpecification{b.tagSpec(ec2types.ResourceTypeElasticIp, "", team, "")},
		})
		if err != nil {
			return "", fmt.Errorf("allocating elastic IP for %s: %w", ref, err)
		}
		allocID = awssdk.ToString(alloc.AllocationId)
		publicIP = awssdk.ToString(alloc.PublicIp)
		// Tag with the ref after allocation (AllocateAddress's tag spec can't
		// reference the instance id we're keying on independently).
		if _, err := b.ec2.CreateTags(ctx, &ec2.CreateTagsInput{Resources: []string{allocID}, Tags: []ec2types.Tag{tag(laforgeExternalRefTag, ref)}}); err != nil {
			return "", fmt.Errorf("tagging elastic IP %s: %w", allocID, err)
		}
	}
	if _, err := b.ec2.AssociateAddress(ctx, &ec2.AssociateAddressInput{
		AllocationId: awssdk.String(allocID), InstanceId: awssdk.String(ref),
	}); err != nil {
		return "", fmt.Errorf("associating elastic IP %s with %s: %w", allocID, ref, err)
	}
	return publicIP, nil
}

func (b *Builder) findElasticIP(ctx context.Context, ref string) (allocID, publicIP string, err error) {
	out, err := b.ec2.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{Filters: []ec2types.Filter{
		filter("tag:"+laforgeExternalRefTag, ref), managedFilter,
	}})
	if err != nil {
		return "", "", fmt.Errorf("looking up elastic IP for %s: %w", ref, err)
	}
	if len(out.Addresses) == 0 {
		return "", "", nil
	}
	return awssdk.ToString(out.Addresses[0].AllocationId), awssdk.ToString(out.Addresses[0].PublicIp), nil
}

// releaseEIPForRef disassociates and releases any EIP we allocated for an
// instance, so DestroyHost doesn't leak an allocation. Best-effort.
func (b *Builder) releaseEIPForRef(ctx context.Context, ref string) {
	out, err := b.ec2.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{Filters: []ec2types.Filter{
		filter("tag:"+laforgeExternalRefTag, ref), managedFilter,
	}})
	if err != nil {
		return
	}
	for _, a := range out.Addresses {
		if a.AssociationId != nil {
			_, _ = b.ec2.DisassociateAddress(ctx, &ec2.DisassociateAddressInput{AssociationId: a.AssociationId})
		}
		if a.AllocationId != nil {
			_, _ = b.ec2.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{AllocationId: a.AllocationId})
		}
	}
}

// HostPublicAddress reports an instance's public IP (its Elastic IP if we
// attached one, else the subnet-assigned public IP), so the runner can record it
// in deployed_object.public_address. The runner probes for this optional method.
func (b *Builder) HostPublicAddress(ctx context.Context, externalRef string) (string, error) {
	out, err := b.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{externalRef}})
	if err != nil {
		return "", fmt.Errorf("describing instance %s: %w", externalRef, err)
	}
	for _, r := range out.Reservations {
		for _, inst := range r.Instances {
			return awssdk.ToString(inst.PublicIpAddress), nil
		}
	}
	return "", nil
}
