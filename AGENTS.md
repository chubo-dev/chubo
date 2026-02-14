# Repository Guidelines

This repo is the Chubo-OS fork of Talos. The product goal is a Talos-like OS API/UX for the Hashi stack (openwonton/opengyoza/openbao) with no Kubernetes/etcd product surface.

## Project Structure & Module Organization

- Go module: `github.com/chubo-dev/chubo`
- OS services:
  - `internal/app/machined/` (boot/runtime controllers, config, services)
  - `internal/app/apid/` (OS API server)
  - `internal/app/trustd/` (trust/PKI for OS and workload APIs)
- CLI:
  - `cmd/chuboctl/` (primary)
  - `cmd/talosctl/` (legacy compatibility alias/shim during rename wave)
- Provisioning/E2E tooling:
  - `pkg/provision/` (qemu/docker providers, networking, DHCP)
  - `hack/qemu/` and `hack/chubo/` (dev loop + E2E scripts)

## Build, Test, and Development Commands

- `make help` and `make unit-tests` for local sanity.
- `make chuboctl` to build the primary CLI into `_out/`.
- Fast inner-loop VM: `./hack/qemu/chubo-qemu.sh` (optionally `VMNET_ENABLE=1` for bridged runtime mTLS).
- Strict QEMU E2E lanes (require root/QEMU on macOS):
  - `./hack/chubo/e2e-core-qemu.sh`
  - `./hack/chubo/e2e-opengyoza-quorum-qemu.sh`
  - `./hack/chubo/e2e-helper-bundles-qemu.sh`

macOS + Colima notes:
- Recommended sizing for amd64 boot artifacts: `colima start --cpu 6 --memory 8 --disk 80`.
- Buildx: the default `docker` driver can hang on large `--output=type=local` builds; QEMU E2E scripts force a `docker-container` builder (`BUILDX_BUILDER=local`) for boot artifacts.
- If a root-run fixture appears stuck in `make initramfs` with ~0% CPU and the initramfs file already exists, force-cancel by restarting Colima (`colima stop && colima start ...`) and rerun the fixture.

In this workspace, the canonical operator docs and execution checklist live in the sibling repo:
- `../chubo/docs/dev/chubo-os-qemu-devloop.md` (quick iteration, NOPASSWD sudoers, troubleshooting)
- `../chubo/docs/talos/plan.md` (execution checklist with commit references)

## Coding Style & Naming Conventions

- Go defaults (`gofmt`, standard layout, no shelling out in control paths).
- Prefer deterministic/idempotent controllers and explicit state transitions.
- Build tags:
  - `chubo` is the k8s-less product build.
  - Keep `talos*` names only when required for compatibility during rename waves.

## Commit & Pull Request Guidelines

- Small, imperative commits (one concern per commit).
- When completing checklist items, reference the commit hashes in `../chubo/docs/talos/plan.md` and/or `../chubo/docs/talos/chubo-product-source-clean-plan.md`.
