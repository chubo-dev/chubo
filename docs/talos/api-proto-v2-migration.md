# API/Proto v2 Migration Plan (Talos Namespace -> Chubo Namespace)

This document defines the Phase 3 migration model referenced by `docs/talos/talos-reference-elimination-plan.md`.

## Goal

Move API/proto namespaces from `talos.*` to `chubo.*` without breaking:

- mixed-version node upgrades/rollbacks,
- existing clients still using v1 (`talos.*`) descriptors,
- support bundle / COSI resource decoding across old and new artifacts.

## Scope

- Resource definition protos under `api/resource/definitions/*`.
- Generated stubs under `pkg/machinery/api/resource/definitions/*`.
- Runtime type registration/decoding paths that depend on proto full names.

Out of scope for this slice:

- Removing compatibility aliases.
- Renaming every legacy on-disk filename/path immediately.

## Version Model

- **v1 (compat):** existing `talos.resource.definitions.*` proto package namespace.
- **v2 (target):** `chubo.resource.definitions.*` proto package namespace.
- Transition status (February 23, 2026): source/stubs now emit v2 package names by default; compatibility behavior is partially landed for runtime event TypeURLs (`talos/runtime/*` + `chubo/runtime/*`), with mixed-version core and control-plane cluster fixture coverage now validated locally; remaining mixed worker/helper coverage is still pending.

## Execution Phases

1. **Dual generation scaffold**
   - Introduce v2 proto package variants and generate v2 stubs.
   - Keep v1 stubs unchanged.
   - Status (February 21, 2026): phase-1 scaffold is partially landed via `talos` commit `23a6bb5c9` by adding `tools/structprotogen --resource-namespace` (default remains `talos.resource.definitions`).
2. **Dual registration + dual decode**
   - Register both v1 and v2 type URLs/full names in runtime decode paths.
   - Prefer v2 for new registrations where safe.
3. **Default emit switch**
   - Switch default emitted/advertised type URLs to v2.
   - Keep v1 decode and compatibility readers enabled.
   - Status (February 21, 2026): v2 package emission is landed via `talos` commit `aefddf03a`; compatibility readers are still open work.
4. **Validation gates**
   - Unit tests for v1<->v2 decoding symmetry.
   - Local QEMU upgrade/rollback between v1-emitting and v2-emitting builds.
5. **Sunset (post cutoff)**
   - Remove v1 emission and eventually v1 decode per compatibility sunset policy.

## Safety Rules

- No mixed functional changes with namespace migration commits.
- Regenerate stubs in dedicated commits.
- Keep guardrails green (`make chubo-guardrails`) on each slice.
- Record each migration commit in `docs/talos/plan.md`.

## Current Scope Snapshot (February 21, 2026)

Audit artifacts: `docs/talos/audits/2026-02-21/`

- `proto_package_files=16` (all `api/resource/definitions/*/*.proto` still declare `package talos.resource.definitions.*`)
- `generated_stub_files_with_talos_namespace=16` (`pkg/machinery/api/resource/definitions/*/*.pb.go`)
- `other_non_generated_talos_namespace_lines=13` (all in `tools/structprotogen/*`)

Pre-slice implication (from the audit above):

1. Extend the landed `--resource-namespace` scaffold to the exact dual-generation mode we want for v1/v2 coexistence.
2. Apply package/import namespace migration to the 16 proto definition files in one mechanical slice.
3. Regenerate stubs immediately after the proto package change.

Post-slice update (February 21, 2026):

- `api/resource/definitions/*/*.proto` package declarations are migrated to `chubo.resource.definitions.*` (`talos` commit `aefddf03a`).
- `pkg/machinery/api/resource/definitions/*/*.pb.go` stubs are regenerated and now emit the chubo package namespace (`talos` commit `aefddf03a`).
- Runtime event transition behavior is now dual-read + chubo-emit:
  - `talos` commit `f6bd91314`: client accepts both `talos/runtime/*` and `chubo/runtime/*` TypeURLs.
  - `talos` commit `b52c82e04`: runtime event emission switched to `chubo/runtime/*`, with runtime unit tests.
- Post-slice mixed-version validation update (February 23, 2026):
  - mixed-version core upgrade/rollback fixture PASS with old fork installer image `localhost:5001/chubo/installer:old-1070f9e22` (old -> new -> old observed, artifacts under `/tmp/chubo-core-work-25601`).
  - mixed-version cluster fixture PASS for control-plane coverage (`WORKER_COUNT=0`) with the same old image and explicit HTTP mirror override (`REGISTRY_MIRROR_NODE=10.<net>.1:5001=http://10.<net>.1:5001`), artifacts under `/tmp/chubo-cluster-work-31614`.
  - mixed worker coverage remains open: old installer `v1.13.0-alpha.1-398-g1070f9e22` rejects `spec.modules.chubo.nomad.networkInterface` in current machineconfig during install.
  - mixed helper-bundles coverage remains open until an old arm64 installer artifact is available (current old image artifact is amd64-only).
  - gate decision for this migration slice: do not block Phase 3/4 progression on the two deferred items above; keep them tracked as follow-up compatibility work.
