# Quickstart

This repo is still early-stage. The fastest honest way to try it today is:

1. build `chuboctl`
2. inspect the CLI locally
3. use the root-run QEMU lane if you want an authoritative boot test

## 1. Build The CLI

From the repo root:

```sh
make chuboctl
```

The host binary is written into `_out/`.

If you already have the host binary from a local build, these commands work:

```sh
./_out/chuboctl-host --help
./_out/chuboctl-host cluster create --help
./_out/chuboctl-host gen --help
```

This is the quickest way to understand the public surface before running any VMs.

## 2. Understand The Runtime Model

Chubo OS is API-managed:

- `chuboctl` talks to the OS API over mTLS
- there is no SSH or shell-based operations model
- OpenWonton, OpenGyoza, and OpenBao access are exposed through helper bundles downloaded via `chuboctl`
- those bundles are meant to be used with the native CLIs: `wonton`, `gyoza`, and `bao`/`vault`

For this alpha phase, the main reference path is local CLI help from the built binary:

```sh
./_out/chuboctl-host --help
./_out/chuboctl-host <command> --help
```

The generated CLI reference also lives in [`reference/cli.md`](reference/cli.md).
The helper-bundle workflow is explained in [`workload-access.md`](workload-access.md).

## 3. Run The Authoritative Local Lane

If you want to boot the OS locally, use the QEMU lane:

```sh
sudo -n ./hack/chubo/e2e-core-qemu.sh
```

This is the main local validation path used in the repo.

To watch a fixture while it is running:

```sh
sudo tail -f /tmp/chubo-*-e2e-state/*/*.log
```

## 4. Run The Helper-Bundle Smoke Lane

To validate the workload access flow:

```sh
sudo -n ./hack/chubo/e2e-helper-bundles-qemu.sh
```

This exercises the path behind:

- `chuboctl nomadconfig`
- `chuboctl consulconfig`
- `chuboctl openbaoconfig`

The default operator path after extraction is:

- use `wonton` with `nomadconfig`
- use `gyoza` with `consulconfig`
- use `bao` or `vault` with `openbaoconfig`

## 5. Know The Platform Caveats

- macOS: QEMU lanes are the authoritative path and may require root, vmnet, and local Docker/Colima setup
- Docker fallback: useful for narrow checks, but not authoritative on macOS
- if you are narrowing a bug, prefer the smallest loop that proves or disproves the specific path you changed

## 6. What To Read Next

- [README.md](../README.md): project overview and repo map
- [README.md](README.md): docs index
- [examples/README.md](examples/README.md): runnable alpha examples
- [reference/cli.md](reference/cli.md): generated CLI reference
- [guides/README.md](guides/README.md): capability guides
- [workload-access.md](workload-access.md): helper bundles and native CLI usage
- [talos/plan.md](talos/plan.md): internal execution checklist
