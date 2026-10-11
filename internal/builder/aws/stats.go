// Stats reports the AWS stack's own backend health to the builder page via
// builder.MetricProvider: CloudWatch time-series aggregated across the running
// LaForge instances (CPU %, network throughput, EBS disk R/W ops), the API
// response time, and quota usage-vs-limit gauges (the limits that actually break
// a deploy -- VPCs/region, Elastic IPs). Memory and disk-USAGE aren't reachable
// without an on-instance agent, so they're simply not published. See
// docs/plans/builder-monitoring.md.
package aws

import (
	"context"
	"fmt"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"

	"github.com/globalcptc/laforge/internal/builder"
)

// Compile-time proof the builder publishes stats (the collector probes for this).
var _ builder.MetricProvider = (*Builder)(nil)

// cwPeriod is the CloudWatch aggregation window. Basic EC2 monitoring publishes
// at 5-minute granularity, so a shorter period would mostly return gaps.
const cwPeriod = 300

func (b *Builder) Stats(ctx context.Context) (builder.Stats, error) {
	var stats builder.Stats

	// API response time: time DescribeInstances, which also gives us the running
	// instances to query CloudWatch for.
	start := time.Now()
	insts, err := b.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{managedFilter, filter("instance-state-name", "running")},
	})
	latency := float64(time.Since(start).Milliseconds())
	if err != nil {
		return stats, fmt.Errorf("describing instances: %w", err)
	}
	stats.Metrics = append(stats.Metrics, builder.Metric{
		Key: "api_latency", Label: "API response time", Kind: builder.MetricTimeseries, Unit: "ms", Value: latency,
	})

	var ids []string
	for _, r := range insts.Reservations {
		for _, i := range r.Instances {
			ids = append(ids, awssdk.ToString(i.InstanceId))
		}
	}
	if len(ids) > 0 {
		b.appendCloudWatch(ctx, ids, &stats)
	}
	b.appendQuotaGauges(ctx, &stats)
	return stats, nil
}

// appendCloudWatch batches one GetMetricData across all running instances (five
// per-instance EC2 metrics) and appends the aggregated CPU/network/disk-ops
// time-series. Best-effort: a CloudWatch failure warns and omits those charts.
func (b *Builder) appendCloudWatch(ctx context.Context, ids []string, stats *builder.Stats) {
	end := time.Now()
	queries := buildMetricQueries(ids)
	latest := map[string]float64{}
	var next *string
	for {
		out, err := b.cw.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
			StartTime:         awssdk.Time(end.Add(-15 * time.Minute)),
			EndTime:           awssdk.Time(end),
			MetricDataQueries: queries,
			NextToken:         next,
		})
		if err != nil {
			stats.Warnings = append(stats.Warnings, "cloudwatch: "+err.Error())
			return
		}
		for _, r := range out.MetricDataResults {
			if len(r.Values) > 0 { // results are newest-first (ScanBy default)
				if _, seen := latest[awssdk.ToString(r.Id)]; !seen {
					latest[awssdk.ToString(r.Id)] = r.Values[0]
				}
			}
		}
		if out.NextToken == nil {
			break
		}
		next = out.NextToken
	}
	cpu, netBps, opsPerS := aggregateInstanceMetrics(ids, latest)
	if cpu != nil {
		stats.Metrics = append(stats.Metrics, builder.Metric{
			Key: "cpu", Label: "CPU (avg)", Kind: builder.MetricTimeseries, Unit: "%", Value: *cpu,
		})
	}
	stats.Metrics = append(stats.Metrics,
		builder.Metric{Key: "net", Label: "Network", Kind: builder.MetricTimeseries, Unit: "bytes/s", Value: netBps},
		builder.Metric{Key: "disk_ops", Label: "Disk R/W ops", Kind: builder.MetricTimeseries, Unit: "ops/s", Value: opsPerS},
	)
}

// buildMetricQueries makes five per-instance AWS/EC2 queries (CPU avg, net
// in/out sum, EBS read/write ops sum). Ids encode the metric + instance index so
// aggregateInstanceMetrics can fold them.
func buildMetricQueries(ids []string) []cwtypes.MetricDataQuery {
	spec := []struct{ prefix, name, stat string }{
		{"cpu", "CPUUtilization", "Average"},
		{"nin", "NetworkIn", "Sum"},
		{"nout", "NetworkOut", "Sum"},
		{"rops", "EBSReadOps", "Sum"},
		{"wops", "EBSWriteOps", "Sum"},
	}
	var queries []cwtypes.MetricDataQuery
	for i, id := range ids {
		instanceID := id
		for _, s := range spec {
			queries = append(queries, cwtypes.MetricDataQuery{
				Id: awssdk.String(fmt.Sprintf("%s_%d", s.prefix, i)),
				MetricStat: &cwtypes.MetricStat{
					Metric: &cwtypes.Metric{
						Namespace:  awssdk.String("AWS/EC2"),
						MetricName: awssdk.String(s.name),
						Dimensions: []cwtypes.Dimension{{Name: awssdk.String("InstanceId"), Value: awssdk.String(instanceID)}},
					},
					Period: awssdk.Int32(cwPeriod),
					Stat:   awssdk.String(s.stat),
				},
			})
		}
	}
	return queries
}

// aggregateInstanceMetrics folds the per-instance GetMetricData results into the
// stack totals: CPU is the mean across instances (nil when none reported),
// network and disk ops are summed across instances and converted to per-second
// rates. Pure, so it's unit-testable without CloudWatch.
func aggregateInstanceMetrics(ids []string, latest map[string]float64) (cpu *float64, netBytesPerSec, opsPerSec float64) {
	var cpuSum float64
	var cpuN int
	var netBytes, ops float64
	for i := range ids {
		if v, ok := latest[fmt.Sprintf("cpu_%d", i)]; ok {
			cpuSum += v
			cpuN++
		}
		netBytes += latest[fmt.Sprintf("nin_%d", i)] + latest[fmt.Sprintf("nout_%d", i)]
		ops += latest[fmt.Sprintf("rops_%d", i)] + latest[fmt.Sprintf("wops_%d", i)]
	}
	if cpuN > 0 {
		avg := cpuSum / float64(cpuN)
		cpu = &avg
	}
	return cpu, netBytes / cwPeriod, ops / cwPeriod
}

// appendQuotaGauges reports usage-vs-limit for the account quotas that actually
// break a deploy. Usage is counted account-wide (the quota is account-wide), the
// limit from Service Quotas. Best-effort per quota: a failed read warns and
// omits that gauge.
func (b *Builder) appendQuotaGauges(ctx context.Context, stats *builder.Stats) {
	// VPCs per region (default 5; LaForge makes one per team).
	if vpcs, err := b.ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{}); err != nil {
		stats.Warnings = append(stats.Warnings, "vpc count: "+err.Error())
	} else if limit, err := b.quotaLimit(ctx, "vpc", "L-F678F1CE"); err != nil {
		stats.Warnings = append(stats.Warnings, "vpc quota: "+err.Error())
	} else {
		stats.Metrics = append(stats.Metrics, quotaGauge("vpcs", "VPCs (region)", float64(len(vpcs.Vpcs)), limit))
	}
	// Elastic IPs (default 5; one per public: host).
	if addrs, err := b.ec2.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{}); err != nil {
		stats.Warnings = append(stats.Warnings, "eip count: "+err.Error())
	} else if limit, err := b.quotaLimit(ctx, "ec2", "L-0263D0A3"); err != nil {
		stats.Warnings = append(stats.Warnings, "eip quota: "+err.Error())
	} else {
		stats.Metrics = append(stats.Metrics, quotaGauge("eips", "Elastic IPs", float64(len(addrs.Addresses)), limit))
	}
}

func (b *Builder) quotaLimit(ctx context.Context, service, code string) (float64, error) {
	out, err := b.quotas.GetServiceQuota(ctx, &servicequotas.GetServiceQuotaInput{
		ServiceCode: awssdk.String(service), QuotaCode: awssdk.String(code),
	})
	if err != nil {
		return 0, err
	}
	if out.Quota == nil || out.Quota.Value == nil {
		return 0, fmt.Errorf("quota %s/%s has no value", service, code)
	}
	return *out.Quota.Value, nil
}

func quotaGauge(key, label string, used, limit float64) builder.Metric {
	l := limit
	return builder.Metric{Key: "quota:" + key, Label: label, Kind: builder.MetricGauge, Unit: "count", Value: used, Max: &l}
}
