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
	b.appendCloudWatch(ctx, ids, &stats)
	b.appendQuotaGauges(ctx, &stats)
	return stats, nil
}

// appendCloudWatch batches one GetMetricData across all running instances (five
// per-instance EC2 metrics) and appends the aggregated CPU/network/disk-ops
// time-series. The three charts are ALWAYS published -- at 0 when no instances
// are running -- so an idle region still shows its CPU/Network/Disk blocks
// (nothing-in-use is itself useful) rather than collapsing to just API latency.
// Best-effort: a CloudWatch failure warns and the charts read 0 for this sample.
func (b *Builder) appendCloudWatch(ctx context.Context, ids []string, stats *builder.Stats) {
	latest := map[string]float64{}
	if len(ids) > 0 {
		end := time.Now()
		queries := buildMetricQueries(ids)
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
				break
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
	}
	cpu, netBps, opsPerS := aggregateInstanceMetrics(ids, latest)
	cpuVal := 0.0
	if cpu != nil {
		cpuVal = *cpu
	}
	stats.Metrics = append(stats.Metrics,
		builder.Metric{Key: "cpu", Label: "CPU (avg)", Kind: builder.MetricTimeseries, Unit: "%", Value: cpuVal},
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

// appendQuotaGauges reports usage-vs-limit for every quota in the registry
// (quotaGaugeSpecs). The limit comes from Service Quotas -- which returns the
// account's APPLIED value, so a quota the operator raised above the default is
// reflected automatically. Best-effort per quota: a failed usage count, an
// unknown quota code, or a missing permission warns and omits just that one
// gauge, so adding a speculative quota can't break the others.
func (b *Builder) appendQuotaGauges(ctx context.Context, stats *builder.Stats) {
	for _, s := range b.quotaGaugeSpecs() {
		used, err := s.usage(ctx)
		if err != nil {
			stats.Warnings = append(stats.Warnings, fmt.Sprintf("%s usage: %s", s.key, err.Error()))
			continue
		}
		limit, err := b.quotaLimit(ctx, s.service, s.code)
		if err != nil {
			stats.Warnings = append(stats.Warnings, fmt.Sprintf("%s quota (%s/%s): %s", s.key, s.service, s.code, err.Error()))
			continue
		}
		stats.Metrics = append(stats.Metrics, quotaGauge(s.key, s.label, used, limit))
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
