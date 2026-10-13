// AWS resource discovery for the builder wizard: regions, the chosen region's
// instance types, and AMIs (the account's own, plus a name search across common
// owners). Best-effort -- a listing that fails becomes a Discovery warning, never
// a failed connection, mirroring incus discoverResources.
package aws

import (
	"context"
	"fmt"
	"sort"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/globalcptc/laforge/internal/builder"
)

// amiSearchOwners are the owners an AMI name-search spans: the account itself
// plus Amazon's own, Canonical (Ubuntu), and Microsoft (Windows) published
// images -- the catalogs a lab actually launches from.
var amiSearchOwners = []string{"self", "amazon", "099720109477", "801119661308"}

// amiMarketplaceOwner is searched as a second, lower-priority pass: AWS
// Marketplace AMIs (Kali, pfSense, appliance images, ...) live under the
// aws-marketplace alias, not the official owners above. Searched only with a
// name term and capped like everything else, so the catalog doesn't flood the
// list. Note: launching a Marketplace AMI may require accepting its terms /
// subscribing in the account first -- discovery lists it either way.
const amiMarketplaceOwner = "aws-marketplace"

// maxDiscoveredImages caps how many AMIs a single search returns, so a broad term
// ("a") can't drag back thousands or spin pagination.
const maxDiscoveredImages = 100

// Rediscover lists what the AWS account has, scoped by conn.Region (and, for AMI
// search, conn.ImageQuery). Regions are always returned; instance types and
// owned AMIs only once a region is chosen; searched AMIs only when a query is set.
func (onboarder) Rediscover(ctx context.Context, conn builder.Connection) (*builder.Discovery, error) {
	region := conn.Region
	if region == "" {
		region = "us-east-1" // any valid region serves the DescribeRegions call
	}
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
		awsconfig.WithRetryMode(awssdk.RetryModeAdaptive), // throttle-safe, like aws.New
		awsconfig.WithRetryMaxAttempts(8),
	}
	if conn.AWSAccessKeyID != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(conn.AWSAccessKeyID, conn.AWSSecretAccessKey, "")))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading AWS credentials: %w", err)
	}
	c := ec2.NewFromConfig(cfg)

	d := &builder.Discovery{
		Regions: []string{}, InstanceTypes: []string{},
		Images: []builder.ImageInfo{}, Warnings: []string{},
	}

	if out, err := c.DescribeRegions(ctx, &ec2.DescribeRegionsInput{}); err != nil {
		d.Warnings = append(d.Warnings, "couldn't list regions: "+err.Error())
	} else {
		for _, r := range out.Regions {
			d.Regions = append(d.Regions, awssdk.ToString(r.RegionName))
		}
		sort.Strings(d.Regions)
	}

	if conn.Region == "" {
		return d, nil // nothing region-scoped to list yet
	}

	d.InstanceTypes = discoverInstanceTypes(ctx, c, d)
	d.RecommendedComposeAMI = recommendedComposeAMI(ctx, c, d)
	appendImages(ctx, c, d, []string{"self"}, "")
	if q := strings.TrimSpace(conn.ImageQuery); q != "" {
		// Official owners first (they fill the cap before Marketplace noise),
		// then a Marketplace pass for whatever slots remain.
		appendImages(ctx, c, d, amiSearchOwners, q)
		appendImages(ctx, c, d, []string{amiMarketplaceOwner}, q)
	}
	return d, nil
}

// recommendedComposeAMI resolves the latest Amazon Linux 2023 x86_64 AMI in the
// region -- a sensible Docker-capable default for a compose-host (Docker and the
// compose plugin are installed on first boot, so a stock AL2023 image works).
// Reuses the already-granted ec2:DescribeImages (no SSM dependency, no extra
// IAM). Best-effort: a failure just leaves it empty and the operator picks one.
func recommendedComposeAMI(ctx context.Context, c *ec2.Client, d *builder.Discovery) string {
	out, err := c.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners: []string{"amazon"},
		Filters: []ec2types.Filter{
			// Standard AL2023 (not -minimal), kernel-tagged, x86_64.
			{Name: awssdk.String("name"), Values: []string{"al2023-ami-2023.*-kernel-*-x86_64"}},
			{Name: awssdk.String("state"), Values: []string{"available"}},
			{Name: awssdk.String("architecture"), Values: []string{"x86_64"}},
		},
	})
	if err != nil {
		d.Warnings = append(d.Warnings, "couldn't resolve a recommended compose-host AMI: "+err.Error())
		return ""
	}
	// CreationDate is ISO-8601, so a lexicographic max is the newest release.
	var newestID, newestDate string
	for _, img := range out.Images {
		if date := awssdk.ToString(img.CreationDate); date > newestDate {
			newestDate, newestID = date, awssdk.ToString(img.ImageId)
		}
	}
	return newestID
}

func discoverInstanceTypes(ctx context.Context, c *ec2.Client, d *builder.Discovery) []string {
	var types []string
	p := ec2.NewDescribeInstanceTypesPaginator(c, &ec2.DescribeInstanceTypesInput{MaxResults: awssdk.Int32(100)})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			d.Warnings = append(d.Warnings, "couldn't list instance types: "+err.Error())
			break
		}
		for _, it := range page.InstanceTypes {
			types = append(types, string(it.InstanceType))
		}
	}
	sort.Strings(types)
	return types
}

// appendImages lists AMIs for the given owners (optionally name-filtered) and
// merges them into d.Images, de-duplicated by image id and capped.
func appendImages(ctx context.Context, c *ec2.Client, d *builder.Discovery, owners []string, nameQuery string) {
	in := &ec2.DescribeImagesInput{Owners: owners}
	if nameQuery != "" {
		in.Filters = []ec2types.Filter{{Name: awssdk.String("name"), Values: nameFilterValues(nameQuery)}}
		// Bound the server response -- a Marketplace name match can be enormous;
		// we only keep up to maxDiscoveredImages anyway. (No MaxResults on the
		// owned, unfiltered call: an account's own AMIs are few.)
		in.MaxResults = awssdk.Int32(maxDiscoveredImages)
	}
	out, err := c.DescribeImages(ctx, in)
	if err != nil {
		d.Warnings = append(d.Warnings, "couldn't list images: "+err.Error())
		return
	}
	seen := make(map[string]bool, len(d.Images))
	for _, img := range d.Images {
		seen[img.Fingerprint] = true
	}
	for _, img := range out.Images {
		id := awssdk.ToString(img.ImageId)
		if id == "" || seen[id] {
			continue
		}
		if len(d.Images) >= maxDiscoveredImages {
			break
		}
		seen[id] = true
		d.Images = append(d.Images, toImageInfo(img))
	}
}

// toImageInfo maps an EC2 AMI onto the builder-neutral ImageInfo the wizard's
// picker renders (fingerprint=ami id, the name as its one alias). Pure, so it is
// unit-tested offline.
func toImageInfo(img ec2types.Image) builder.ImageInfo {
	info := builder.ImageInfo{
		Fingerprint:  awssdk.ToString(img.ImageId),
		Aliases:      []string{awssdk.ToString(img.Name)},
		Architecture: string(img.Architecture),
		Type:         "virtual-machine",
	}
	info.Properties.OS = awssdk.ToString(img.PlatformDetails)
	info.Properties.Description = awssdk.ToString(img.Description)
	info.Properties.Created = awssdk.ToString(img.CreationDate)
	info.Properties.Owner = ownerLabel(awssdk.ToString(img.OwnerId))
	return info
}

// ownerLabel turns a bare AWS owner account id into a readable publisher name for
// the well-known catalogs the search spans, so the picker shows "Canonical"
// rather than "099720109477"; any other id is shown as-is.
func ownerLabel(ownerID string) string {
	switch ownerID {
	case "099720109477":
		return "Canonical"
	case "801119661308":
		return "Microsoft"
	case "137112412989", "amazon":
		return "Amazon"
	}
	return ownerID
}

// nameFilterValues turns a search term into the set of case-variant globs the
// EC2 DescribeImages `name` filter needs: that filter is case-SENSITIVE, so a
// lowercase "windows" never matches Microsoft's "Windows_Server-2022-..." names.
// Matching the raw term plus its lower/Title/UPPER forms covers the real catalog
// naming (Canonical lowercases, Microsoft Title-cases) without a client-side scan.
func nameFilterValues(q string) []string {
	variants := []string{
		"*" + q + "*",
		"*" + strings.ToLower(q) + "*",
		"*" + strings.ToUpper(q) + "*",
		"*" + strings.Title(strings.ToLower(q)) + "*", //nolint:staticcheck // ASCII title-case is exactly what AMI names use
	}
	seen := make(map[string]bool, len(variants))
	out := make([]string, 0, len(variants))
	for _, v := range variants {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
