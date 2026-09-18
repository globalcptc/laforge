# LaForge replacement options

Working notes for replacing (or succeeding) LaForge as CPTC's competition
infrastructure engine. This is a shortlist with kill criteria, not a vendor
bake-off. Dig into survivors in follow-up docs.

Last updated: 2026-09-17

## Why this exists

LaForge works. It has run CPTC at >4000 nodes across AWS, OpenStack, vSphere /
NSX-T, GCP (Terraform era), and now Canonical MicroCloud. The reasons to even
talk about a replacement are operational, not conceptual:

- The *engine* is a volunteer tax. Content authors write bash/pwsh + HCL.
  Platform work is Go + Ent + GraphQL + Angular + per-cloud builders.
- Hosts keep moving. A new builder is a multi-month specialist project.
- The current stack (server, agent, UI, planner, scheduler, file middleware)
  is one tightly coupled product. Hard to evolve one piece without the rest.

The architecture is still the right one for a pentest competition. Most
"just use $INDUSTRY_TOOL" answers fail at least two of the constraints below.

## What LaForge actually does

Ignore the README marketing. The product is four layers:

```text
environment repo (HCL + scripts + files)
        │  overlay / include / git
        ▼
planner (build → N teams → networks → hosts → ordered steps)
        │
        ├─ builder plugin  (cloud/hypervisor API)
        │     deploy/teardown team, network, host
        │
        └─ agent (Go, gRPC callback, mTLS)
              execute, download, extract, users, ansible-local,
              scheduled/cron steps, heartbeats
```

### Game model

Anything we pick has to encode these as first-class objects, not as "write a
module":

| Concept | What it means |
| --- | --- |
| Environment | Named game: `team_count`, builder, admin CIDRs, VDI ports |
| Competition | Timing, root password, DNS |
| Network | CIDR, VDI visibility, per-team instantiation |
| Host | hostname, OS image, last octet, size, disk, exposed ports, vars |
| IncludedNetwork | which hosts sit on which network |
| HostDependency | boot/provision order across hosts/networks |
| Identity | in-game users (name, email, password, avatar) |
| Script / Command | bash, pwsh, whatever; args, timeout, ignore_errors, templated vars |
| FileDownload / Extract / Delete | agent pulls files from LaForge, not from a jump box |
| Ansible | **local** playbooks on the box via the agent, not SSH from a control node |
| ProvisioningStep | ordered graph per host |
| ScheduledStep | cron or run-once in-game injects |
| Finding | metadata hung off scripts (severity/difficulty) |
| Build / Team / Provisioned\* | clone the environment N times with isolated state |
| AgentStatus | heartbeat: OS, load, mem, uptime |

Team clone is not "repeat a Terraform module." It is: same host graph, same
scripts, unique addressing, unique credentials, isolated L2/L3, N copies,
independent provision state, independent teardown.

### Agent model

The agent is a custom Go service. It **calls out** over gRPC/mTLS, heartbeats,
pulls tasks, reports status. Primitives:

`EXECUTE`, `DOWNLOAD`, `EXTRACT`, `DELETE`, `REBOOT`, `CREATEUSER`,
`CREATEUSERPASS`, `ADDTOGROUP`, `CHANGEPERMS`, `APPENDFILE`, `VALIDATE`,
`ANSIBLE`

No inbound SSH/WinRM from a control plane into student networks for
provisioning. Students do not get Ansible, Salt, Puppet, SSM, or a k8s API
as a well-known target. They get an obscure binary with a private protocol.

That is not accidental. CPTC competitors *will* attack the management plane
if you put a famous one in reach.

### Builder model

`Builder` is a six-method interface: deploy/teardown of host, network, team.
Current slugs: `aws`, `openstack`, `vsphere-nsxt`, `microcloud`, `generic`.

Content (HCL + scripts) is supposed to be builder-agnostic. Reality: image
names, instance sizes, and network features leak. But the *intent* is that
volunteers do not rewrite the game when the hoster changes.

MicroCloud today: one LXD project per team, OVN networks, no inter-team
peering, one public ingress host via network forward. That is the current
isolation primitive.

### Collaboration model

Git repo of `.laforge` files + scripts. `include { path = ... }` overlays
(Docker-layer analogy). Maintainers on blocks. UI imports repos, builds,
watches host/step/agent status, ad-hoc tasks, teardown.

Volunteers mostly never touch Go. They touch:

- a host/network/env HCL file
- a bash or PowerShell script
- maybe a templated file download

**That** is the "everyone can understand it" bar. Preserve it.

## Hard constraints

Must-haves. Fail two and the option is dead.

1. **Volunteer content = shell.** New people drop in for a weekend, write
   bash/pwsh (and maybe a host block), drop out. No Ansible-role career path,
   no Terraform-module priesthood, no charm/operator SDK.
2. **N-way isolated clone.** One environment definition → N team copies
   including networks, not just "flag VMs." Scale historically >4000 nodes.
3. **Callback agent, not push config-mgmt.** Management plane stays off the
   competition networks. No well-known orchestration stack living where
   students can eat it.
4. **Host portability.** We have been in AWS, OpenStack, GCP, vSphere, and
   MicroCloud. We will move again. Hypervisor-locked tools are a trap.
5. **Thin ops surface.** Postgres + a server + an agent is fine. K8s +
   Keycloak + StackStorm + Helm + three UIs is not a volunteer platform.
6. **Live ops.** Heartbeats, per-step status, ad-hoc execute, scheduled
   injects, rebuild a single host, tear down a single team. Image-bake-only
   is not enough.
7. **Windows + Linux.** CPTC is an enterprise pentest. AD, Windows agents,
   mixed provision steps. Linux-only range toys are insufficient.

## Evaluation rubric

Score 1–5. 5 looks like the column on the right.

| Axis | 5 looks like |
| --- | --- |
| Volunteer content | bash/pwsh + a small DSL; new author productive in <1 day |
| Team clone | first-class; not `for_each` spaghetti |
| Attack surface | custom/obscure callback agent; mgmt out of band |
| Host portability | new builder without rewriting the game |
| Ops thinness | one server process + DB, not a platform mesh |
| Live control | status, injects, rebuild, teardown at host/team grain |
| Windows | first-class, not "community role maybe" |
| Scale | thousands of VMs, tens of teams, parallel provision |
| Exit cost | we can leave without rewriting 5 years of scripts |

## Options

### 0. Keep LaForge, surgically replace the painful bits

**What:** Do not replace the product. Split and modernize in place.

Candidates to carve out over time:

- Keep the HCL game model, planner, agent, UI.
- Make builders a real plugin boundary (the interface already exists; the
  implementations are in-tree snowflakes).
- Replace Ent/GraphQL/Angular only if they are the actual volunteer blocker
  (they are not — content authors do not use them).
- Treat MicroCloud as the primary builder; keep others on life support.
- Freeze Ansible-local as optional; scripts stay the default.

**Fits:** All seven constraints. We already run the game this way.

**Fails:** Does not reduce the specialist bus-factor on the engine. Next host
migration is still "write another Go builder." UI/planner debt remains.

**When this wins:** If the real pain is *builders* and *reliability*, not the
game model. Given MicroCloud just landed, this is the lowest-risk path for
the next 1–2 events.

**Kill if:** We cannot staff Go work at all, or the HCL/Ent model is blocking
content (no evidence of that — content is scripts).

### 1. Successor: same architecture, new implementation ("LaForge 2")

**What:** Rebuild the four layers as separate pieces with the same contracts.

```text
DSL (HCL or CUE/Pkl) + git overlays
        ▼
planner / state (team clone, DAG, status)
        ▼
builder interface  ← OpenTofu, Incus API, AWS, whatever
        ▼
thin callback agent  ← keep roughly the current proto
```

Keep:

- bash/pwsh as the guest language
- callback agent (rewrite ok, protocol/capability set stays)
- team/network/host as native types
- pluggable builders

Change:

- Overlay: HCL `include` is cute and under-specified. CUE or Pkl are actual
  overlay languages. YAML is a trap at this complexity.
- State: something boring (Postgres + a journal) instead of Ent graph soup
  if that is the maintenance sink.
- UI: status + actions. Do not rebuild Metronic.
- Builders: Incus/MicroCloud first, OpenTofu as an escape hatch for clouds
  we do not want to write SDKs for.

**Fits:** All seven constraints if we do not get bored and add a mesh.

**Fails:** We still own a product. Year-1 cost is real. Migration of existing
`.laforge` repos needs a compiler (HCL → new DSL) or we keep HCL.

> Speculation: keeping HCL is the correct migration move even if CUE is nicer.
> Volunteers already know the blocks. Overlay can be a loader feature without
> changing author-facing syntax.

**When this wins:** We want a 10-year tool and we can staff a rewrite without
skipping an event. Best long-term answer if Option 0 is politically dead.

**Kill if:** Rewrite starts pulling in k8s, "real" config mgmt, or a new
guest language.

### 2. Compose: OpenTofu/Terraform + cloud-init + keep the LaForge agent

**What:** Stop writing builders. Infra = OpenTofu (`for_each` teams, provider
per hoster). Guest config = cloud-init to drop the agent, then the existing
(or a slimmer) callback agent runs scripts.

Providers exist for AWS, GCP, Azure, OpenStack, vSphere, and Incus/LXD.
That is the actual portability story of the industry.

**Fits:** Host portability (3–4), volunteer *scripts* (5) if the DSL stays
in front of Tofu, live control if we keep the agent (4).

**Fails:**

- Team clone and DAG are *our* code on top of Tofu, or they become unreadable
  modules. Tofu does not know what a CPTC team is.
- State files / backends at this scale are an ops job. Volunteer-hostile.
- Per-hoster modules still leak (OVN vs VPC vs NSX). Content stays portable
  only if we keep a frontend DSL that renders Tofu.
- Tofu in-guest is irrelevant; Tofu on the control box is fine. Attack
  surface is OK *if* we do not also SSH in with Ansible.

Honest version of this option: **it is a builder backend, not a replacement.**
It is how Option 0/1 should implement AWS/GCP next time, not how volunteers
author a game.

**Kill if:** Someone proposes volunteers write `.tf` + `for_each` for each
host. That fails constraint 1 immediately.

### 3. Ludus (CISA / Bad Sector Labs)

Proxmox + YAML range configs + Packer templates + Ansible roles. CLI-driven.
Genuinely good at "spin a range." Volunteer-friendly *if* you already live
in Ansible and Proxmox.

**Fits:** YAML is approachable. Range clone exists. Community momentum.

**Fails hard:**

- **Proxmox only.** Documented: no other hypervisor will be supported. We
  are on MicroCloud. We have been on AWS/OpenStack/GCP. Constraint 4 is a
  kill.
- **Ansible is the provisioner.** Push/role model. Famous, hackable, and a
  different volunteer skill than bash/pwsh.
- Scale target is home-lab / class range (~150 ranges, tens of VMs), not
  CPTC-sized enterprise clones.
- Windows is via templates/roles, not our identity/script graph.
- No callback-agent live ops story comparable to ours.

**Verdict:** Excellent for a satellite lab (GOAD-style research ranges for
volunteers). Dead as the CPTC engine.

**Kill:** already killed on host lock-in + Ansible + scale.

### 4. Crucible (CMU SEI)

Modular DoD/education stack: TopoMojo (form labs), Caster (OpenTofu modules
for VMware/Proxmox/Azure/AWS), Steamfitter (tasks via StackStorm + Ansible),
Player, Gameboard, Alloy, Keycloak, Helm on Kubernetes.

**Fits:** Cross-cloud-ish (not Incus/OpenStack). Exercise/MSEL thinking is
closer to a competition than Ludus. OpenTofu under Caster.

**Fails:**

- This is the bulky stack constraint 5 exists to forbid. k8s + Keycloak +
  Redis + MinIO + 6 apps.
- Steamfitter's execution path is StackStorm/Ansible — well-known, often
  in-band.
- Volunteer onboarding is "learn Crucible," not "write a shell script."
- No MicroCloud/Incus builder. Next hoster = hope Caster grows it, or we
  write Tofu modules *and* operate Crucible.
- CPTC is not a training-lab + MSEL product. We need clone-an-enterprise,
  not a student portal.

**Verdict:** Study Caster's Tofu module pattern. Do not adopt the platform.

**Kill:** ops thinness + volunteer model + missing Incus.

### 5. KYPO CRP / CyberRangeCZ (Masaryk)

OpenStack sandboxes + Kubernetes platform services + Terraform + Ansible +
JSON/YAML definitions. KYPO CRP development has ended; CyberRangeCZ is the
successor.

**Fits:** Academic range, sandbox isolation, some clone semantics.

**Fails:** OpenStack+K8s is exactly the stack we left. Ansible guest config.
Training/CTF oriented. Volunteer tax is the platform, not the scripts.
Host portability is "do you have OpenStack."

**Kill:** hoster assumption + bulky platform. Do not revisit unless we are
back on OpenStack and want a student-facing training UI, which we do not.

### 6. Ansible / Salt / Puppet / Chef as the platform

**Verdict:** No.

Push config-mgmt is the opposite of the agent model. Control node credentials
and Python/WinRM/SSH become the competition. Salt is catnip. Puppet/Chef are
dead weight. Ansible-pull is slightly less dumb and still a famous target.

LaForge already runs Ansible **locally through the agent** when someone
insists. That is the only acceptable Ansible: on-box, no control plane, no
SSH from mgmt into the range.

**Kill:** constraints 1, 3, 5.

### 7. Image bake only (Packer / image-builder / golden clones)

Bake everything into templates, clone VMs, cloud-init for hostname/IP/users.
Fast boot. Common CCDC-style approach.

**Fits:** Attack surface (little runtime mgmt), speed at T0, hosters that
can clone volumes.

**Fails:** Live provision, scheduled injects, last-minute content, per-team
identity injection, rebuild-with-new-script, agent status. CPTC content
changes until the week of, sometimes during. Baking is a **layer** (and
Ludus/Packer do this well for bases) not an engine.

Use as: builder optimization. Golden images + agent for the delta.

### 8. Incus/MicroCloud-native orchestrator (thin, host-coupled)

Write a small controller against the LXD API we already use: projects = teams,
OVN = networks, cloud-init = agent bootstrap, keep the agent.

**Fits:** Current hoster perfectly. Ops can be very thin. Volunteers still
write scripts if we keep a DSL.

**Fails:** Constraint 4 unless the DSL/planner stays host-agnostic and this
is *just* the MicroCloud builder (i.e. Option 0/1). If the "replacement"
*is* the Incus controller, the next move to AWS rewrites everything.

**Verdict:** Do this as the MicroCloud builder (already happening). Do not
confuse it with a LaForge replacement.

Canonical Juju/charms are the "official" answer and fail constraint 1
(charms ≠ bash). Ignore Juju.

### 9. Pulumi / CDK / Crossplane / Nomad

Pulumi/CDK: real programming languages. Worse volunteer story than HCL.
Crossplane: k8s to do VMs. Nomad/Consul: another famous cluster in range
reach. All fail 1 and/or 3 and/or 5.

Mentioned so we do not rediscover them in six months.

### 10. Nix / NixOS

Reproducible Linux. Hostile to Windows enterprise forests, hostile to
weekend volunteers, irrelevant to OVN team isolation. No.

## Matrix

Scores are directional, not scientific. 1 = hostile, 5 = native fit.

| Option | Volunteer | Clone | Agent/attack | Portability | Thin ops | Live ops | Windows | Scale | Exit |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 0 Keep LaForge | 5 | 5 | 5 | 4 | 3 | 5 | 4 | 5 | 3 |
| 1 Successor | 5 | 5 | 5 | 5 | 4 | 5 | 4 | 5 | 4 |
| 2 Tofu + agent | 3 | 2 | 5 | 5 | 3 | 4 | 4 | 4 | 4 |
| 3 Ludus | 3 | 3 | 2 | 1 | 3 | 2 | 3 | 2 | 2 |
| 4 Crucible | 2 | 3 | 2 | 3 | 1 | 4 | 3 | 3 | 2 |
| 5 KYPO/CRCZ | 2 | 3 | 2 | 2 | 1 | 3 | 3 | 3 | 2 |
| 6 Ansible-as-platform | 2 | 2 | 1 | 3 | 2 | 3 | 3 | 3 | 2 |
| 7 Bake-only | 3 | 4 | 4 | 4 | 5 | 1 | 3 | 4 | 3 |
| 8 Incus-only controller | 4 | 4 | 5 | 1 | 5 | 4 | 3 | 4 | 2 |

Windows on MicroCloud is currently weaker than Linux (builder rejects
Windows until the image/agent contract is real). That is a current-host
gap, not a reason to pick Ludus.

## Recommendation

Do not buy a cyber range platform. Ludus/Crucible/KYPO optimize for training
labs and famous toolchains. CPTC optimizes for "clone an enterprise N times,
configure it with volunteer shell, do not hand students a Salt master."

**Near term (next event):** Option 0. Keep LaForge. Invest in the MicroCloud
builder, agent TLS/bootstrap, Windows-on-Incus, and reliability. Optionally
implement the next *cloud* builder as OpenTofu behind the existing Builder
interface (Option 2 as a backend).

**Medium term:** Decide if the engine is maintainable. If yes, stay on 0 and
publish a builder SDK so a volunteer can add a hoster without touching
planner/UI. If no, Option 1 with a hard rule: HCL+scripts stay, agent stays
callback, no k8s.

**Never:** replace the guest language with Ansible roles, bind the product to
Proxmox, or put a well-known config-mgmt control plane on a network students
can route to.

## What a successor must keep

Copy these into any future design doc. If a prototype cannot do them, it is
not a replacement.

1. Author a host as HCL (or equivalent) + a bash script + a pwsh script.
   No other toolchain required.
2. `team_count = 10` produces 10 isolated L2/L3 copies from one definition.
3. Provisioning is a DAG (host deps, step order). Failures are per-step and
   visible.
4. Guests pull work over an authenticated callback channel. No SSH from
   mgmt into team nets for the happy path.
5. Swap builder in the environment file, not by rewriting hosts/scripts.
6. Scheduled inject at T+N minutes across all teams.
7. Rebuild one host on one team without touching the others.
8. Tear down one team.
9. Heartbeat/status for every provisioned host.
10. Windows and Linux in the same environment.

## Follow-ups

Dig in this order:

1. **Capability gap list vs current MicroCloud builder** — Windows, DNS,
   public ingress, identity creation, scheduled steps. Ground truth for
   Option 0.
2. **Agent contract freeze** — treat `grpc/proto` as the stability boundary.
   Any successor keeps this even if the server dies.
3. **HCL inventory of real CPTC repos** — how much overlay, ansible-local,
   templated files, host deps we actually use. Design against reality.
4. **Builder-as-Tofu spike** — one team on Incus via OpenTofu, still driven
   by LaForge planner/agent. Tests Option 2 as backend.
5. **Volunteer UX study** — time-to-first-script for a new author on current
   HCL vs a YAML mock. Do not assume YAML is easier; our blocks are small.
6. **Attack-surface review of current agent + gin file middleware** — if we
   keep this model, make it the thing we harden instead of replacing it with
   Ansible.

## Rejects, parked

- Crossplane, Nomad, Consul, k8s-as-range, Foreman, MAAS, Juju
- Nix
- "Just Terraform" with volunteers writing modules
- Commercial ranges (RangeForce, SimSpace, Cyberbit) — cost, lock-in,
  not CPTC-shaped

## References

- [Ludus docs](https://docs.ludus.cloud/docs/intro)
- [cisagov/Ludus](https://github.com/cisagov/Ludus)
- [KYPO Cyber Range Platform](https://crp.kypo.muni.cz/en)
- [Crucible documentation](https://cmu-sei.github.io/crucible/)
- [cmu-sei/crucible](https://github.com/cmu-sei/crucible)
