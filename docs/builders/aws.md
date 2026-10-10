# AWS builder

Deploys an environment onto AWS using EC2/VPC for networks and hosts and
ECS/Fargate for containers. LaForge talks to the AWS APIs directly (no Terraform).

> **Status: implemented, pending real-account verification.** The builder is
> written to the full `builder.Builder` contract, but its firewall/access paths
> have not yet been proven against a live account with a packet-level probe (see
> [builder-authoring.md](../builder-authoring.md) §8). Treat it as Draft until that
> passes.

## What it maps to

| LaForge          | AWS                                                                 |
| ---------------- | ------------------------------------------------------------------- |
| team             | one **VPC** (the isolation boundary) + internet gateway + a public route table + a custom Network ACL |
| network          | a **subnet** in the team VPC (its CIDR added as a VPC CIDR block)    |
| host             | an **EC2 instance** in that subnet, with a per-object security group |
| container        | an **ECS/Fargate task** with the same per-object security group     |
| `visible_from:` + `ports:` | per-object security-group ingress rules (same VPC routes for free) |
| access schedule (open/close) | a deny-all ingress entry on the team NACL (stateless → drops established flows) |
| `public:` ports  | an **Elastic IP** per host + a security-group rule from `0.0.0.0/0` |

Everything is tagged `laforge:managed=true`, `laforge:team=<n>`, and
`laforge:external-name=<name>`, which is how adopt-or-create and Inspect find it.

## What you provision (one account)

Everything below lives in **one AWS account**. Two pieces:

1. **An access key** — an IAM user (or role) LaForge authenticates *as*, with the
   EC2/ECS/STS permissions below plus `iam:PassRole`. You enter it **in the builder
   wizard**; LaForge validates it (an STS identity check) and stores it the same
   way it stores an Incus client credential — the wizard keeps only the credential
   id, never the secret. (Alternatively, leave the key blank and give the runner
   host an IAM instance-profile role; see "Credential" below.)
2. **An ECS task execution role** — *not* a second login. It is an IAM role that
   AWS's Fargate API requires you to hand to each task so ECS can pull the image
   and ship logs. You create it once and name its ARN in the wizard; LaForge only
   references the ARN (via `iam:PassRole`). Hosts-only environments don't need it.

## Fastest path: the CloudFormation template

[`aws-iam.yaml`](aws-iam.yaml) (next to this doc) provisions the runner
IAM user (with the policy below) **and** the ECS task execution role in one stack.
Deploy it, create an access key for the runner user, and paste the key + the role
ARN (a stack Output) into the wizard:

```bash
aws cloudformation deploy --stack-name laforge-iam \
  --template-file docs/builders/aws-iam.yaml \
  --capabilities CAPABILITY_NAMED_IAM --region <your-region>
aws cloudformation describe-stacks --stack-name laforge-iam \
  --query 'Stacks[0].Outputs' --region <your-region>
```

The stack deliberately creates **no** access key (a CloudFormation output would
expose the secret) — make one for the runner user in the IAM console. The policy
below is the same set, for reference and for least-privilege tightening by hand.

## Permissions for the runner credential

Attach this policy to the IAM user or role whose credentials the runner uses.
EC2 networking actions generally can't be resource-scoped, so they are granted
account-wide but confined to the builder's region with a condition; destructive
instance/SG actions can be further narrowed to `laforge:managed` resources if you
want least privilege (see the note after the policy).

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "LaForgeEC2",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeVpcs", "ec2:CreateVpc", "ec2:DeleteVpc", "ec2:AssociateVpcCidrBlock",
        "ec2:DescribeSubnets", "ec2:CreateSubnet", "ec2:DeleteSubnet", "ec2:ModifySubnetAttribute",
        "ec2:DescribeInternetGateways", "ec2:CreateInternetGateway", "ec2:AttachInternetGateway",
        "ec2:DetachInternetGateway", "ec2:DeleteInternetGateway",
        "ec2:DescribeRouteTables", "ec2:CreateRouteTable", "ec2:DeleteRouteTable",
        "ec2:CreateRoute", "ec2:AssociateRouteTable", "ec2:DisassociateRouteTable",
        "ec2:DescribeNetworkAcls", "ec2:CreateNetworkAcl", "ec2:DeleteNetworkAcl",
        "ec2:CreateNetworkAclEntry", "ec2:DeleteNetworkAclEntry", "ec2:ReplaceNetworkAclAssociation",
        "ec2:DescribeSecurityGroups", "ec2:CreateSecurityGroup", "ec2:DeleteSecurityGroup",
        "ec2:AuthorizeSecurityGroupIngress", "ec2:RevokeSecurityGroupIngress",
        "ec2:RunInstances", "ec2:TerminateInstances", "ec2:DescribeInstances",
        "ec2:StartInstances", "ec2:StopInstances", "ec2:RebootInstances",
        "ec2:CreateTags",
        "ec2:AllocateAddress", "ec2:ReleaseAddress", "ec2:AssociateAddress",
        "ec2:DisassociateAddress", "ec2:DescribeAddresses"
      ],
      "Resource": "*",
      "Condition": { "StringEquals": { "aws:RequestedRegion": "us-east-1" } }
    },
    {
      "Sid": "LaForgeECS",
      "Effect": "Allow",
      "Action": [
        "ecs:RegisterTaskDefinition", "ecs:RunTask", "ecs:StopTask"
      ],
      "Resource": "*",
      "Condition": { "StringEquals": { "aws:RequestedRegion": "us-east-1" } }
    },
    {
      "Sid": "LaForgePassExecutionRole",
      "Effect": "Allow",
      "Action": "iam:PassRole",
      "Resource": "arn:aws:iam::<ACCOUNT_ID>:role/<ECS_TASK_EXECUTION_ROLE>",
      "Condition": { "StringEquals": { "iam:PassedToService": "ecs-tasks.amazonaws.com" } }
    },
    {
      "Sid": "LaForgeIdentityCheck",
      "Effect": "Allow",
      "Action": "sts:GetCallerIdentity",
      "Resource": "*"
    }
  ]
}
```

Replace `us-east-1` with your region and the `iam:PassRole` resource with your
task execution role ARN. Drop the `LaForgeECS` / `LaForgePassExecutionRole`
statements if you deploy no `container:` objects (hosts-only).

**Tighter, optional:** the destructive instance, security-group, EIP and
delete-\* actions can carry `"Condition": {"StringEquals": {"aws:ResourceTag/laforge:managed": "true"}}`
so the credential can only touch resources LaForge itself created. `Describe*`
and the `Create*` actions can't be tag-scoped and stay broad.

## The ECS task execution role (containers only)

Create a role named in `execution_role_arn` with:

- **Trust policy** — assumable by `ecs-tasks.amazonaws.com`:
  ```json
  { "Version": "2012-10-17", "Statement": [{ "Effect": "Allow",
    "Principal": { "Service": "ecs-tasks.amazonaws.com" },
    "Action": "sts:AssumeRole" }] }
  ```
- **Permissions** — the AWS-managed `AmazonECSTaskExecutionRolePolicy` (ECR image
  pull + CloudWatch Logs). That's the whole requirement when task logs go to
  CloudWatch (`log_group`). If `container_logs` sends elsewhere (e.g. Splunk HEC),
  the role still needs ECR pull but not CloudWatch; the task's own subnet just
  needs egress to the collector.

AWS creates this role for you the first time you use the ECS console, named
`ecsTaskExecutionRole`; otherwise create it as above.

## Credential

Enter the access key in **Admin → Infrastructure → Add builder → AWS → AWS
Credentials** and click **Connect**. LaForge runs `sts:GetCallerIdentity` to
confirm it works (so a bad key fails at connect, not first deploy) and stores it
server-side; the UI then shows "connected" and holds only the credential id. The
secret is never returned to the browser again.

> Prefer not to store a long-lived key? Give the **runner host an IAM
> instance-profile role** with the policy below and leave the wizard's key fields
> blank — LaForge then uses the ambient role. That's AWS's recommended practice,
> but it means the credential isn't managed in the wizard.

## Settings — what each one is and how to get it

The wizard walks these in order: **Connect → Region → Images → Sizes → Review**.
If you deployed `aws-iam.yaml`, two of these are stack Outputs you paste in; the
rest are discovered for you or have sane defaults.

### Connect step

- **Access Key ID / Secret Access Key** — an IAM access key LaForge authenticates
  as (the runner identity). *Required.* Create one for the runner user:
  ```bash
  aws iam create-access-key --user-name laforge-runner
  ```
  (the `laforge-runner` user is created by `aws-iam.yaml`). Paste the
  `AccessKeyId` and `SecretAccessKey` and click **Connect** — LaForge validates it
  with `sts:GetCallerIdentity` and stores it; you won't see the secret again. No
  region here — this step is global.

### Region step

- **Region** — where this builder deploys (VPC, instances, AMIs are all here).
  *Required.* **Autocompleted**: the dropdown is populated by `DescribeRegions`
  from your account; just pick one (e.g. `us-east-1`). Everything in the next two
  steps is discovered for the region you choose.
- **ECS Cluster** — the ECS cluster Fargate containers run in. *Optional, default
  `default`.* Every account has a cluster named `default`; leave it unless you
  made a dedicated one (`aws ecs list-clusters` shows their names). Only matters if
  you deploy `container:` objects.
- **Task Execution Role ARN** — the IAM role Fargate assumes to pull images and
  ship logs. *Required only if you deploy containers.* This is the
  **`ExecutionRoleArn` Output** of the `aws-iam.yaml` stack — copy it verbatim:
  ```bash
  aws cloudformation describe-stacks --stack-name laforge-iam \
    --query "Stacks[0].Outputs[?OutputKey=='ExecutionRoleArn'].OutputValue" --output text
  ```
  (It looks like `arn:aws:iam::<account>:role/laforge-runner-ecs-exec`.)
- **CloudWatch Log Group** — where Fargate task logs go when an environment
  doesn't set its own `container_logs`. *Optional.* Leave blank for no default log
  forwarding, or create one (`aws logs create-log-group --log-group-name
  /laforge/containers`) and enter its name.
- **Agent Init Image** — a small public image a Fargate container runs once to
  download the LaForge agent. *Optional, default `alpine:3.20`.* Leave it unless
  your account can't pull from Docker Hub (then point it at an ECR mirror).

### Images step

- **Images** — map each LaForge `os` name (as your content uses it, e.g.
  `ubuntu22`, `windows2022`) to an **AMI**. *At least one required.* **Autocompleted**:
  your own AMIs are listed; type a name (e.g. "ubuntu", "Windows Server 2022",
  "kali") to search the Amazon/Canonical/Microsoft **and AWS Marketplace** catalogs
  for the chosen region. If search doesn't surface the exact image you want, **paste
  its AMI id** and pick "Use …" — the field accepts any id. (A Marketplace AMI may
  require accepting its terms / subscribing in the account before it will launch;
  do that in the AWS console once.) For compose containers, also map `compose-host`
  to a Docker-capable Linux AMI.

### Sizes step

- **Sizes** — map each LaForge `size` name (e.g. `small`, `large`) to an EC2
  **instance type**. *At least one required.* **Autocompleted** from the region's
  `DescribeInstanceTypes` (e.g. `t3.medium`). A Fargate `container:` instead reads
  a `"cpu/memory"` pair (e.g. `"512/1024"`) from this map.

## Limits and notes

- **VPC CIDR-block quota.** Each team network adds its CIDR to the team VPC as a
  secondary block; AWS allows **5 per VPC** by default. For teams with more than 5
  networks, raise the quota (Service Quotas → VPC → "IPv4 CIDR blocks per VPC").
- **Compose containers** run on an EC2 instance booted from the `compose-host`
  image (the agent runs `docker compose up` on it, same as the other builders) —
  so map `compose-host` to a Docker-capable AMI in the Images step. This is one
  EC2 instance per compose container; single-image containers stay on Fargate.
  (Packing compose projects onto shared EC2-backed ECS capacity is a possible
  future density optimization, not built.)
- **Close = full isolation.** Closing a team's access adds a deny-all ingress NACL
  entry; because NACLs are stateless this also severs the in-guest agent's return
  traffic, i.e. a closed team is fully dark, matching the OVN builders.
- Lab AMIs should have Secure Boot off (Ubuntu cloud images otherwise loop) and
  cloud-init/cloudbase-init installed (that is how the agent installs on first boot).
