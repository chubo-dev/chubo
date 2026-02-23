# Naming Convergence Map (Wave B Freeze)

This document freezes the naming targets for Wave B mechanical renames. No ad-hoc renames outside this map.

## Goal

Keep Talos-like API behavior while converging product naming to Chubo.

## Frozen Rename Map

| Scope | Current | Target | Notes |
| --- | --- | --- | --- |
| CLI binary | `talosctl` | `chuboctl` | Keep `talosctl` alias for one transition cycle. |
| Client config file | `talosconfig` | `chuboconfig` | Accept both names during transition; write `chuboconfig` by default. |
| Build variant tag | `chuboos` | `chubo` | Applies to build tags, target names, artifact directories. |
| Make targets | `chuboos-*` | `chubo-*` | Example: `chuboos-e2e-qemu` -> `chubo-e2e-qemu`. |
| Script paths | `hack/chuboos/*` | `hack/chubo/*` | Keep temporary wrappers for old paths until docs are updated. |
| Temp/output prefixes | `/tmp/chuboos-*`, `_out/chuboos` | `/tmp/chubo-*`, `_out/chubo` | Avoid mixed prefixes in new scripts. |
| Env var (config path) | `TALOSCONFIG` | `CHUBOCONFIG` | Accept both during transition; prefer `CHUBOCONFIG`. |
| Env var (state dir) | `TALOS_HOME` | `CHUBO_HOME` | Accept both during transition; prefer `CHUBO_HOME`. |
| Env var (editor) | `TALOS_EDITOR` | `CHUBO_EDITOR` | Accept both during transition; prefer `CHUBO_EDITOR`. |
| Go module path | `github.com/siderolabs/talos` | `github.com/chubo-dev/chubo` | Single mechanical import rewrite in one pass. |
| Proto go_package base | `github.com/siderolabs/talos/...` | `github.com/chubo-dev/chubo/...` | Regenerate protobufs in same commit as module rewrite. |
| Java proto package | `dev.talos.api.*` | `dev.chubo.api.*` | Update only where generated artifacts are maintained. |

## Sidero/Siderolabs Policy

- Keep third-party dependencies hosted under `github.com/siderolabs/*` unless we fork them.
- Keep SideroLink internals untouched in Wave B unless they block Chubo naming externally.
- Rename only product ownership surfaces first (CLI, config, module path, docs, generated API package names).

## Explicitly Not Renamed in Wave B

- `MachineService` and `MachineConfig` API object names stay as-is.
- Core daemon identifiers (`machined`, `apid`, `trustd`) stay as-is.
- Existing COSI resource type names stay as-is unless they include Kubernetes semantics.

Reason: keep API shape close to Talos and reduce breakage risk during first rename pass.

## Mechanical Pass Rules

1. No behavioral changes mixed with renames.
2. One commit per layer: docs/scripts, CLI UX, build paths, module/proto imports.
3. Keep compatibility aliases for one release cycle, then remove.
4. Wave B is complete only after full QEMU E2E (`install -> mTLS -> upgrade -> rollback -> support`) passes on renamed surface.
