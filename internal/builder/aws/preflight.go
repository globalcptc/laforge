// Pre-flight IAM permission check. The AWS builder calls many EC2/ECS actions
// across a deploy, and a missing permission otherwise only surfaces as a raw 400
// mid-deploy (e.g. ecs:TagResource on RunTask) -- discovered one at a time. At
// credential-enroll time (onboard) this simulates the required actions against
// the caller and reports any that are denied, up front, as connection warnings.
package aws

import (
	"context"
	"fmt"
	"sort"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
)

// requiredActions is the EC2+ECS action set the runner credential must hold to
// deploy and tear down on AWS -- the single source of truth the pre-flight check
// simulates. Keep it in sync with the API calls the builder makes (and with
// docs/builders/aws.md + aws-iam.yaml). iam:PassRole is deliberately excluded:
// it is resource-scoped to the task execution role, whose ARN isn't known until
// a later wizard step, so simulating it against "*" would false-positive.
var requiredActions = []string{
	"ec2:CreateVpc", "ec2:DeleteVpc", "ec2:DescribeVpcs", "ec2:AssociateVpcCidrBlock",
	"ec2:CreateSubnet", "ec2:DeleteSubnet", "ec2:DescribeSubnets", "ec2:ModifySubnetAttribute",
	"ec2:CreateInternetGateway", "ec2:DeleteInternetGateway", "ec2:AttachInternetGateway",
	"ec2:DetachInternetGateway", "ec2:DescribeInternetGateways",
	"ec2:CreateRouteTable", "ec2:DeleteRouteTable", "ec2:CreateRoute", "ec2:AssociateRouteTable", "ec2:DescribeRouteTables",
	"ec2:CreateNetworkAcl", "ec2:DeleteNetworkAcl", "ec2:CreateNetworkAclEntry", "ec2:DeleteNetworkAclEntry",
	"ec2:DescribeNetworkAcls", "ec2:ReplaceNetworkAclAssociation",
	"ec2:CreateSecurityGroup", "ec2:DeleteSecurityGroup", "ec2:DescribeSecurityGroups",
	"ec2:AuthorizeSecurityGroupIngress", "ec2:RevokeSecurityGroupIngress",
	"ec2:RunInstances", "ec2:TerminateInstances", "ec2:DescribeInstances", "ec2:CreateTags",
	"ec2:AllocateAddress", "ec2:AssociateAddress", "ec2:DisassociateAddress", "ec2:ReleaseAddress", "ec2:DescribeAddresses",
	"ec2:DescribeRegions", "ec2:DescribeInstanceTypes", "ec2:DescribeImages",
	"ec2:DescribeNetworkInterfaces", // builder-page quota gauge (ENIs per region / SGs per NIC)
	"ecs:RegisterTaskDefinition", "ecs:DeregisterTaskDefinition", "ecs:ListTaskDefinitions",
	"ecs:RunTask", "ecs:StopTask", "ecs:ListTasks", "ecs:DescribeClusters", "ecs:CreateCluster", "ecs:TagResource",
	"logs:CreateLogGroup", "logs:TagResource", // ensure (and tag) the Fargate awslogs group
	"cloudwatch:GetMetricData",      // builder-page health time-series
	"servicequotas:GetServiceQuota", // builder-page quota gauges
}

// checkPermissions simulates requiredActions against the caller and returns a
// warning naming any that are denied -- so an operator sees every missing
// permission when enrolling the credential, not one 400 per deploy. Best-effort:
// if the credential can't call iam:SimulatePrincipalPolicy itself (or the call
// errors), it returns one advisory line and never blocks enrollment -- a
// credential missing, say, only ECS perms can still deploy hosts. region is
// supplied as the aws:RequestedRegion context so a region-scoped policy
// condition (as in docs/builders/aws-iam.yaml) evaluates correctly rather than
// false-positiving. nil result = all required actions are allowed.
func checkPermissions(ctx context.Context, cfg awssdk.Config, callerARN, region string) []string {
	src := policySourceARN(callerARN)
	if src == "" {
		return nil // an unexpected principal shape (e.g. root) -- skip rather than guess
	}
	in := &iam.SimulatePrincipalPolicyInput{
		PolicySourceArn: awssdk.String(src),
		ActionNames:     requiredActions,
	}
	if region != "" {
		in.ContextEntries = []iamtypes.ContextEntry{{
			ContextKeyName:   awssdk.String("aws:RequestedRegion"),
			ContextKeyType:   iamtypes.ContextKeyTypeEnumString,
			ContextKeyValues: []string{region},
		}}
	}
	client := iam.NewFromConfig(cfg)
	var denied []string
	for {
		out, err := client.SimulatePrincipalPolicy(ctx, in)
		if err != nil {
			return []string{fmt.Sprintf("couldn't pre-verify the runner credential's IAM permissions (%s) -- grant iam:SimulatePrincipalPolicy to enable this check, and confirm the policy in docs/builders/aws.md is attached", errSummary(err))}
		}
		for _, r := range out.EvaluationResults {
			if r.EvalDecision != iamtypes.PolicyEvaluationDecisionTypeAllowed {
				denied = append(denied, awssdk.ToString(r.EvalActionName))
			}
		}
		if !out.IsTruncated || out.Marker == nil {
			break
		}
		in.Marker = out.Marker
	}
	if len(denied) == 0 {
		return nil
	}
	sort.Strings(denied)
	return []string{"the runner credential is missing IAM permissions it needs to deploy -- grant these (see docs/builders/aws.md): " + strings.Join(denied, ", ")}
}

// policySourceARN normalizes the caller ARN for SimulatePrincipalPolicy. A user
// or role ARN works directly; an assumed-role session ARN
// (arn:aws:sts::<acct>:assumed-role/<Role>/<session>) must be rewritten to its
// role ARN (arn:aws:iam::<acct>:role/<Role>). Anything else yields "" (skip).
func policySourceARN(arn string) string {
	switch {
	case strings.Contains(arn, ":user/"), strings.Contains(arn, ":role/"):
		return arn
	case strings.Contains(arn, ":assumed-role/"):
		acct := arnAccount(arn)
		rest := arn[strings.Index(arn, ":assumed-role/")+len(":assumed-role/"):]
		role := rest
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			role = rest[:i]
		}
		if acct != "" && role != "" {
			return "arn:aws:iam::" + acct + ":role/" + role
		}
	}
	return ""
}

func arnAccount(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) >= 5 {
		return parts[4]
	}
	return ""
}

// errSummary reduces an AWS error to its API code when it has one (e.g.
// AccessDenied), so the advisory warning stays short.
func errSummary(err error) string {
	if code := apiErrorCode(err); code != "" {
		return code
	}
	return err.Error()
}
