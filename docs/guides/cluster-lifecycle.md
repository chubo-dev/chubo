# Cluster Lifecycle

The current codebase supports a local-cluster operator loop centered on `chuboctl cluster`.

## Create

Main local create paths:

- `chuboctl cluster create dev`
- `chuboctl cluster create qemu`
- `chuboctl cluster create docker`

For alpha, the practical recommendations are:

- use `dev` or the dedicated QEMU scripts for serious local validation
- treat Docker as a narrower convenience path, especially on macOS

Inspect the create surface with:

```sh
./_out/chuboctl-host cluster create --help
./_out/chuboctl-host cluster create dev --help
./_out/chuboctl-host cluster create qemu --help
./_out/chuboctl-host cluster create docker --help
```

### Practical Alpha Defaults

For the current repo state:

- macOS: prefer `sudo -n ./hack/chubo/e2e-core-qemu.sh`
- Linux: prefer the same QEMU lane first, then experiment with lower-cost create paths if needed
- use direct `cluster create` flows mainly when you are narrowing provisioning behavior, not when you need the most authoritative proof

## Inspect

Local cluster inspection commands:

- `chuboctl cluster show`
- `chuboctl dashboard`

The dashboard gives a cluster/node-oriented operational view. `cluster show` gives provisioned-cluster metadata.

## Destroy

Destroy local state with:

```sh
./_out/chuboctl-host cluster destroy --help
```

Use the matching provisioner and cluster state path when cleaning up partial or failed local runs.

## Upgrade And Rollback

Current lifecycle actions on running nodes include:

- `upgrade`
- `rollback`
- `reboot`
- `shutdown`
- `reset`

These are OS lifecycle operations, not workload-scheduler operations. In alpha, prefer validating them through the local QEMU lanes rather than treating them as broadly portable host workflows.

## Current Reality

What is clearly supported in the repo today:

- local cluster provisioning
- QEMU-based validation
- install, reboot, upgrade, rollback, and support-bundle flows in local fixtures
- helper-bundle extraction for workload APIs

What is still rough:

- polished newcomer UX
- non-QEMU platform parity
- concise, opinionated examples for each create mode
