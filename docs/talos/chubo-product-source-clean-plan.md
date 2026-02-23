# Chubo Product/Source Clean Plan

This plan answers a key distinction:

- Product goal: Chubo runtime/API must be Kubernetes/etcd free.
- Repository goal: legacy Kubernetes/etcd code can be removed in staged passes.

Do not mix these tracks. Deliver Product Clean first, then Source Clean.

## Tracking Rules

- [ ] When a task is complete, check it and append commit hash(es) in parentheses.
- [ ] Keep commits small and scoped (one concern per commit).
- [ ] Every phase must end with explicit verification commands and artifacts.
- [ ] Keep the local QEMU fixture loop runnable non-interactively (`sudo -n`), per `docs/dev/chubo-os-qemu-devloop.md`.

## Phase 0: Baseline and Scope Freeze

- [x] Freeze scope in `docs/talos/rename-map.md` and this plan (no ad-hoc renames). (`chubo`: `600295a`)
- [x] Capture current baseline inventories (use `docs/talos/audits/2026-02-13/collect.sh`). (`chubo`: `9f11e67`)
  - raw refs: `rg -n -i '(kubernetes|k8s|kube|etcd|\\bcri\\b)' .`
  - active refs (exclude docs/tests/changelogs/pb/vendor)
  - chubo-tag refs only
  - chubo dependency graph (`go list -deps` for chubo targets)
- [x] Save inventories under `docs/talos/audits/` with date-stamped files. (`chubo`: `9f11e67`)

## Phase 1: Product Clean (Hard Requirement)

### 1.1 Build/Dependency Gates

- [x] Add CI gate: chubo-tagged builds must not import `k8s.io/*` or `go.etcd.io/etcd/*` (allowlist only if strictly required and documented). (`talos`: `3dcb90062`, baseline + regression check via `hack/chubo/check-go-deps.sh` in `chubo-guardrails`)
- [x] Apply first `machined` dependency-reduction slices for chubo-tagged builds (cluster health split, etcd service + kubeconfig condition stubs, diagnostics/runtime/secrets build-tag splits, VIP no-op operator; then server method/tag splits and cgroup decoupling), reducing forbidden deps for `internal/app/machined` from `341 -> 96 -> 0` (`talos`: `8eb95d80d`, `3fc42892d`).
- [x] Gate key targets with failing checks on forbidden deps (`machined` and `cmd/chuboctl` via `hack/chubo/check-go-deps.sh`) and keep CI/runtime guardrails + QEMU E2E lanes enabled for regression catch (`talos`: `3dcb90062`, `3fc42892d`, `05c9a24ba`, `48344b8a8`, `fa73194ec`).
- [x] Keep rootfs/runtime guardrails mandatory in CI (guardrails lane runs rootfs + deps checks; QEMU E2E lane runs runtime surface checks). (`talos`: `05c9a24ba`, `48344b8a8`, `f2752431f`, `736a33e63` local fast-rerun knobs for guardrails: `CHUBO_GUARDRAILS_SKIP_BUILD=1` and `CHUBO_GUARDRAILS_BUILD_TARGETS`; `696e65271` enforces chubo-tagged CLI build + wording scan in guardrails)

### 1.2 Runtime Surface

- [x] Verify chubo runtime services never include kubelet/etcd/cri service processes (enforced by `hack/chubo/check-runtime-surface.sh` in core QEMU E2E). (`talos`: `b83d317ee`)
- [x] Keep only installer-required CRI config controllers, with explicit TODO to remove once installer path no longer needs them. (`talos`: `7b95e1787`; validated 2026-02-16 via `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1`, `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build ./internal/app/machined`, `sudo -n ./hack/chubo/e2e-core-qemu.sh --skip-build` -> `EXIT:0`)
- [x] Ensure support bundle collection has no kube/etcd hard dependencies. (`talos`: `ca8659b5e`, `c305481f4`)

### 1.3 CLI/API Surface

- [x] Hide or remove Kubernetes/etcd CLI commands from chubo builds (and guard in `make chubo-guardrails`). (`talos`: `fb68e6ee3`)
- [x] Exclude kube/etcd-heavy `talosctl` command files from `chubo`-tagged builds (`etcd`, `bootstrap`, `kubeconfig`, `upgrade-k8s`, `rotate-ca`, `conformance`, `health`, `edit`) and drop `jsonpath` output mode in chubo-tagged CLI (`talos`: `30f28d269`; `cmd/talosctl` forbidden deps `561 -> 268`, `go.etcd.io/etcd` deps `71 -> 0`).
- [x] Split `cluster create` kube-specific paths behind build tags (kubeconfig merge and readiness checks now `!chubo` code paths) to prepare full chubo-native provisioning surface (`talos`: `df510d19f`).
- [x] Replace chubo-tagged `support` with a Kubernetes-free implementation and add chubo build-tag variants for `pkg/cluster` + `pkg/provision/access`, reducing `cmd/talosctl` forbidden deps `268 -> 46` (`talos`: `ca8659b5e`; remaining deps are k8s API/type packages still referenced by shared machinery config surfaces).
- [x] Remove `inject` command from chubo-tagged mgmt root and enforce strict `cmd/chuboctl == 0 forbidden deps` in `hack/chubo/check-go-deps.sh` (`talos`: `5d2032f6e`, `fa73194ec`; strict CLI target moved from `cmd/talosctl` to `cmd/chuboctl` in `fa73194ec`).
- [x] Ensure OS API exposes only Chubo-relevant workflows for Nomad/Consul/OpenBao bootstrap and operations. (`talos`: `b73e6c9ba`; validated 2026-02-16 via `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1`)
- [x] Remove kube-centric wording from chubo-facing CLI help and docs. (`talos`: `794d68a43`, `751f46af9`, `2a6deb387`, `318d2b9b2`, `d303aa0f1`, `696e65271`; validated 2026-02-16 by generating CLI docs from chubo-tagged binary and scanning with `rg -n -i 'kubernetes|kubectl|kube-system|etcd' /tmp/chuboctl-docs` -> no matches; scan now enforced by `make chubo-guardrails`)

### 1.4 Chubo Control-Plane Machinery (Replacement, Not Just Deletion)

- [x] Replace placeholder/mocked openwonton/opengyoza paths with real service lifecycle integration (real binaries/config/startup/health semantics). (`talos`: `b5e8dd316`, `7beff5fc1`, `7f2097ec7`, `3a2091b89`, `cd4a0d1f8`; validated 2026-02-13 via `sudo -n ./hack/chubo/e2e-opengyoza-quorum-qemu.sh`)
- [x] Fix OpenWonton artifact execution: run the real `wonton` agent (glibc-linked) by extracting a minimal glibc runtime bundle from the upstream OCI image tar and executing via the bundled loader (no container runtime involved). (`talos`: `1b68c1ee0`, `c791c893a`)
- [x] Fix OpenGyoza (Consul) multi-IP startup fatal by binding/advertising the default-route IP at runtime. (`talos`: `80ae77569`)
- [x] Expand Nomad/Consul config surface: `bootstrapExpect` + `join` and render into openwonton/opengyoza configs. (`talos`: `0298a6680`)
- [x] Add multi-node QEMU fixture to validate real openwonton/opengyoza cluster formation via `bootstrapExpect` + `join`, and run it in CI. (`talos`: `1663f2485`, `d179f9b1c`, `ced971fc8`)
- [x] Add Talos-grade bootstrap controllers for openwonton/opengyoza. (`talos`: `a8c57d00c`, `72deac52b`)
  - [x] initial cluster bootstrap and idempotent re-apply (`talos`: `a8c57d00c`)
  - [x] join/leave behavior for node lifecycle (`talos`: `72deac52b`)
  - [x] role-aware behavior (server/client) and safe defaults (`talos`: `a8c57d00c`)
- [x] Add Talos-grade trust/identity management for openwonton/opengyoza. (`talos`: `46143ae38`, `3aa2e12a0`, `01b8bf530`)
  - [x] CA and leaf cert issuance for workload API access (`talos`: `46143ae38`, `feada7a5e`)
  - [x] Make workload mTLS robust to skewed/broken clocks by backdating the OS CA `NotBefore` and aligning service leaf cert validity to the CA window. (`talos`: `3aa2e12a0`; validated 2026-02-14 via `./hack/chubo/e2e-helper-bundles-qemu.sh` -> `EXIT:0`)
  - [x] Add loopback/localhost SANs to OS API certs for slirp/hostfwd dev loops. (`talos`: `c5bc09a95`)
  - [x] Cert rotation policy + triggers for workload API access (rotate on CA/SAN/key drift; revisit leaf validity once upstream time bugs are fixed). (`talos`: `03fdd88a2`, `3aa2e12a0`)
  - [x] ACL/bootstrap token lifecycle and persistence model. (`talos`: `01b8bf530`; `chubo`: `6d07796`)
  - [x] clear recovery behavior after reboot/upgrade/reset. (`talos`: `72deac52b`, `ecd9598db`; `chubo`: `4ff22f4`)
- [x] Add upgrade-safe orchestration hooks. (`talos`: `1fa73d1b3`, `42666de2a`, `ecd9598db`)
  - [x] openwonton drain + stop/start sequencing across OS upgrade (best-effort client-role drain hook in chubo sequencer before graceful reboot/shutdown/upgrade, plus existing stop/start from sequencer lifecycle; `talos`: `1fa73d1b3`, `2e43b5fd0`, `ecd9598db`; validated 2026-02-12 via helper unit tests + chubo `machined` build + `chubo-e2e-qemu --skip-build` `EXIT:0`)
  - [x] opengyoza quorum-safe stop/start checks for server nodes (server-role pre-stop check queries `/v1/status/peers` and blocks graceful lifecycle only when stopping this server would break quorum, while keeping transient API failures best-effort; `talos`: `42666de2a`; validated 2026-02-12 via `go test ./internal/app/machined/pkg/runtime/v1alpha1/internal/opengyozaquorum`, chubo `machined` build, and `sudo -n ./hack/chubo/e2e-core-qemu.sh --skip-build >/tmp/chubo-e2e-run-opengyoza-quorum.log 2>&1; echo EXIT:$?` -> `EXIT:0`)
- [x] Add operational diagnostics parity for Chubo stack. (`talos`: `a8c57d00c`, `23cd91499`; `chubo`: `622d5af`)
  - [x] COSI status resources include detailed states and last errors. (`talos`: `e1d27c717`, `5c9e3602f`, `a8c57d00c`, `01b8bf530`)
  - [x] support bundle collectors include openwonton/opengyoza configs, logs, health snapshots (`talos`: `23cd91499`, `70ef9db15`)
  - [x] failure modes documented with deterministic remediation steps. (`chubo`: `4ff22f4`, `622d5af`)

## Phase 2: Naming Convergence to Chubo

### 2.1 User-Facing Names

- [x] Make `chuboctl` primary; keep `talosctl` compatibility shim for one transition cycle. (`talos`: `a6b8f0339`, `f3275b299`, `fa73194ec`; `chubo`: `24e8ca8`)
- [x] Make `CHUBOCONFIG`/`CHUBO_HOME`/`CHUBO_EDITOR` primary; keep TALOS equivalents as compatibility aliases. (`talos`: `704de5939`, `9cc94bb62`; `chubo`: `531e390`)
- [x] Normalize `chubo-*` targets, script paths, and tmp/output prefixes. (`talos`: `2151c973c`, `d2b99c04b`)

### 2.2 Code and Generated Surfaces

- [x] Complete import/module rename to `github.com/chubo-dev/chubo`. (`talos`: `bf49edf62`, `d38be74bd`)
- [x] Update proto `go_package` (and Java package where applicable), regenerate code, and verify downstream compile. (`talos`: `931d97628`, `42df3bd14`, `cd6d279a2`)
- [x] Remove remaining `chuboos` naming except intentional compatibility wrappers. (`talos`: `2151c973c`, `d2b99c04b`)
- [x] Debrand remaining runtime/config user-facing messaging and low-risk wording in shared constants/imager/meta helpers. (`talos`: `f31aa6387`, `cb45c6b10`, `dd4fed54b`, `e9cdb1313`, `67dae1141`, `e8a21a96b`, `43d2616f2`, `4899e25a1`, `f643f6418`, `cb406188c`, `0f72f1938`, `8112e74cb`, `f6e212211`, `c4c383c10`, `6c12535d5`, `8e8b93d49`, `76e462cbe`, `19ab500e9`, `8f8965093`, `9da8c7d88`)

## Phase 3: Source Clean (Repository Purge)

Goal: delete Kubernetes/etcd from the *repository*, not just from the `chubo` build output.

Rules:
- Do this after Product Clean is stable (guardrails + QEMU fixtures green).
- Prefer small, mechanical commits; keep the tree buildable at every step.
- If we keep a useful subsystem which happens to have `kube*` naming (e.g. KubeSpan), rename it as part of this phase or explicitly quarantine it.

### 3.1 CI and Test Surface

- [x] Remove Kubernetes-focused GitHub workflows (cron/integration suites) from `.github/workflows/`, keeping only Chubo-relevant lanes (unit tests, guardrails, QEMU E2E). (`talos`: `94135a7d0`, `4d5249595`)
- [x] Remove or quarantine Kubernetes-only `hack/test/e2e-*` scripts and terraform fixtures not used by Chubo. (`talos`: `cb2f79c2a`)
- [x] Update `README.md`/docs to reflect the new CI surface and supported test commands. (`talos`: `ed72a8dde`)

### 3.2 Runtime Controllers/Services

- [x] Delete Kubernetes/etcd services (kubelet, etcd, CRI runtime service) from `internal/app/machined/pkg/system/services/` and ensure they are not registered anywhere. (`talos`: `026095c53`)
- [x] Delete Kubernetes/etcd controllers from `internal/app/machined/pkg/controllers/` (k8s, etcd, kubeaccess). (`talos`: `5fe694890`)
- [x] Delete remaining Kubernetes-backed cluster discovery/affiliate registry code under `internal/app/machined/pkg/controllers/cluster` and `internal/pkg/discovery/registry`. (`talos`: `261e3607e`)
- [x] Decide Kubespan fate. (decision: remove) (`talos`: `22c36ef85`, `cec66fbd6`, `a829cd2ef`, `64490e567`)
  - [x] Remove it completely. (`talos`: `22c36ef85`)
- [x] Remove Kubernetes/etcd-specific sequencer tasks and health/precheck logic so the OS lifecycle is OS+Chubo only. (`talos`: `1392ddfa6`, `026095c53`)
- [x] Purge remaining kubelet-centric diagnostics and readiness wiring (e.g. kubelet CSR warnings, static-pod/node readiness checks) from the runtime controller set. (`talos`: `661bb8a7b`)
- [x] Remove legacy CRI inspector/client codepaths (`internal/pkg/cri`, `internal/pkg/containers/cri/cri.go`), move default runtime inspection to containerd-only, and rename the remaining registry helper package from `internal/pkg/containers/cri/containerd` to `internal/pkg/containers/runtimecfg`. (`talos`: `5104f0cee`, `ef57c2356`; validated 2026-02-17 via `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build ./internal/app/machined`, `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./internal/app/machined`, `go test ./internal/pkg/containers/runtimecfg ./cmd/talosctl/cmd/talos`, `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1`)
- [x] Re-run `make unit-tests` and `make chubo-guardrails` after each deletion slice. (validated 2026-02-14 on `talos`: `5fe694890` via `make chubo-guardrails`)
- [x] Delete dead legacy Kubernetes helper package (`pkg/kubernetes`) and remove the local `hack/third_party/go-kubernetes` fork wiring from `go.mod` (integration-k8s suite now imports upstream `github.com/siderolabs/go-kubernetes` directly). (`talos`: `767e234c0`, `70b0cf8e9`; validated 2026-02-16 via `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1`)

### 3.3 API/Resources/Config Types

- [x] Move worker trustd endpoint discovery off `k8s.Endpoint` resources to a cluster-scoped control-plane endpoints resource. (`talos`: `21877bc37`)
- [x] Remove remaining Kubernetes resource wiring from core runtime paths (node name, VIP operator, API cert SANs). (`talos`: `49c1799d3`, `abce78314`, `c372e9c82`)
- [x] Remove Kubernetes resource definitions under `api/resource/definitions/k8s` (and generated stubs/resources). (`talos`: `cfba94817`)
- [x] Remove `kubeaccess` resource definitions under `api/resource/definitions/kubeaccess` (and generated stubs/resources). (`talos`: `ff138fbf5`)
- [x] Remove `kubespan` resource definitions under `api/resource/definitions/kubespan` (and generated stubs/resources). (`talos`: `22c36ef85`, `cec66fbd6`, `a829cd2ef`, `64490e567`)
- [x] Make config containers nil-safe when `v1alpha1.Config` is absent (avoid typed-nil + nil-DeepCopy panics in configloader reflection fuzz). (`talos`: `c9a48e9cf`)
- [x] Drop `k8s.io/code-generator` deepcopy-gen usage from `pkg/machinery/config/types/v1alpha1` (switch to `github.com/siderolabs/deep-copy` and keep doc generation working). (`talos`: `f826fa34a`)
- [x] Remove Kubernetes/etcd config types/schemas/docs under `pkg/machinery/config/types/v1alpha1/*` that are no longer reachable in Chubo, and ensure schema generation still works. (`talos`: `a76b43a0c`, `0d9622c9c`, `b2c9e1592`; validated 2026-02-15 via `make unit-tests`, `make chubo-guardrails`, and core QEMU E2E `./hack/chubo/e2e-core-qemu.sh` -> `EXIT:0`; validated 2026-02-16 via `go test ./pkg/machinery/config/types/v1alpha1 ./pkg/machinery/config ./pkg/machinery/config/configdiff ./pkg/machinery/config/configloader/internal/decoder ./pkg/machinery/config/configloader ./pkg/machinery/config/configpatcher`)
- [x] Remove kubeconfig generator package and kubeconfig RPC surface. (`talos`: `b4358552c`)
- [x] Remove Kubernetes/etcd secrets resources/controllers which are now dead code. (`talos`: `17dc459b8`)
- [x] Remove Kubernetes CA rotation machinery which depended on Kubernetes secrets. (`talos`: `8721e0dfa`)
- [x] Regenerate protobuf/machinery stubs and ensure `go test ./...` remains green. (`talos`: `4ab35b159`; validated 2026-02-15 via `make unit-tests` and `make chubo-guardrails`)

### 3.4 CLI and UX

- [x] Remove Kubernetes/etcd-only `talosctl` commands under `cmd/talosctl/cmd/talos/` (bootstrap, kubeconfig, upgrade-k8s, etcd, rotate-ca, conformance, health). (`talos`: `e6f6951bd`)
- [x] Remove Kubernetes-only `talosctl` management `inject` command under `cmd/talosctl/cmd/mgmt/inject`. (`talos`: `b57254f27`)
- [x] Rename local provisioning surface from `KubernetesEndpoint` to a generic control-plane endpoint (fixes QEMU fixture output and removes k8s-centric naming from dev tooling). (`talos`: `b4de59166`)
- [x] Remove remaining Kubernetes-only `talosctl cluster *` UX under `cmd/talosctl/cmd/mgmt/cluster/` (kubeconfig generation, k8s readiness gates, k8s-only flags/help). Result: `cluster create` is a dev fixture only (apply-config optional + wait for runtime OS API); no Kubernetes bootstrap or kubeconfig merge. (`talos`: `b9bea2772`, `30d7d0c40`, `3ac26b8f9`, `6a4f9c983`; validated 2026-02-15 via `make unit-tests` + `make chubo-guardrails`)
- [x] Ensure the remaining CLI/API verbs align with the Chubo OS model (OS API is the only remote control plane; workload APIs are accessed via helper bundles). (`talos`: `318d2b9b2`, `d303aa0f1`, `696e65271`; validated 2026-02-16 via `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1` and generated-CLI-doc scan with no kube/etcd wording)

### 3.5 Compatibility Shims Sunset

- [x] Define an explicit sunset date/version for legacy `talos*`/`TALOS*`/`chuboos*` wrappers (docs only). (`chubo`: `897125b`; see `docs/talos/migration-notes.md`)
- [ ] Delete wrappers once the sunset window closes and migration notes are published.

## Phase 4: Validation and Release Readiness

- [x] Full chubo QEMU flow: install -> mTLS -> helper bundles -> upgrade -> rollback -> support. (validated 2026-02-13 via `sudo -n ./hack/chubo/e2e-core-qemu.sh --skip-build --with-helpers >/private/tmp/chubo-e2e-core.latest.log 2>&1; echo EXIT:$?` -> `EXIT:0`, workdir `/tmp/chubo-core-work-795`; `talos`: `34b75b5ee`)
- [x] Helper bundles QEMU flow: install -> runtime mTLS -> openwonton/opengyoza/openbao -> download helper bundles. (`talos`: `1d9bbc605`, `dd2a50b6a`; validated 2026-02-15 via `sudo -n ./hack/chubo/e2e-helper-bundles-qemu.sh >/tmp/chubo-e2e-helper-bundles.latest.log 2>&1; echo EXIT:$?` -> `EXIT:0`, and re-validated 2026-02-17 via `INSTALLER_TAG=helperfix1771321136 SKIP_BUILD=1 VMNET_ENABLE=0 ./hack/chubo/e2e-helper-bundles-qemu.sh` -> `helper bundle smoke complete`)
- [x] Dedicated opengyoza quorum fixture lane exists for explicit unsafe/safe graceful-upgrade assertions (`make chubo-e2e-opengyoza-quorum-qemu`, legacy alias `make chuboos-e2e-opengyoza-quorum-qemu`; `talos`: `67f42d9a2`, `71cd476d6`, `9fce9674e` [colima DOCKER_HOST autodetect with isolated DOCKER_CONFIG]; validated 2026-02-13 via `DOCKER_HOST=unix:///Users/$USER/.colima/default/docker.sock sudo -n ./hack/chubo/e2e-opengyoza-quorum-qemu.sh --skip-build >/private/tmp/chubo-e2e-opengyoza-quorum.latest.log 2>&1; echo EXIT:$?` -> `EXIT:0`).
- [x] Guardrails + helper-bundles paths use `chuboctl` as the primary CLI binary (with legacy `TALOSCTL*` env fallback only). (`talos`: `ef3507b51`; validated 2026-02-17 via `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1` and `sudo -n ./hack/chubo/e2e-helper-bundles-qemu.sh` -> `helper bundle smoke complete`)
- [x] Validate the multi-node cluster fixture with the new real Nomad workload assertion (`hack/chubo/e2e-cluster-qemu.sh` now submits and waits for a Nomad batch job, then purges it). (`talos`: `5b271c97a`, `e7806ddf2`; validated 2026-02-18 via `sudo -n SKIP_BUILD=1 CONTROLPLANE_COUNT=2 SKIP_NOMAD_JOB_PROBE=0 ./hack/chubo/e2e-cluster-qemu.sh` -> `chubo cluster E2E passed`; re-validated 2026-02-20 via `sudo -n SKIP_BUILD=1 CONTROLPLANE_COUNT=2 SKIP_NOMAD_JOB_PROBE=0 RUN_ID=57511 ./hack/chubo/e2e-cluster-qemu.sh` -> `chubo cluster E2E passed`)
- [x] Add the multi-node cluster fixture as a CI lane with Nomad probe enabled (`CONTROLPLANE_COUNT=2`, `SKIP_NOMAD_JOB_PROBE=0`). (`talos`: `7279bd61b`)
- [x] Make `chubo-guardrails` runnable on macOS+colima by avoiding `/tmp` bind-mount paths (extract to a docker-shared temp dir). (`talos`: `89922ef1d`)
- [x] Repeat on Lima bridged workflow and document exact operator steps. (`chubo`: `5c0a396`; validated 2026-02-16 via `./hack/lima/cluster-e2e.sh` -> `Cluster e2e complete.` and `systemctl is-active openwonton opengyoza chubo-agent` = `active` on `chubo-1..3`)
- [x] Publish migration notes: Talos naming/CLI/env compatibility and sunset timeline. (`chubo`: `897125b`; `docs/talos/migration-notes.md`)
- [x] Re-run full local validation pass and refresh source-clean audit inventory. (`chubo`: `3d4337d`; `talos`: `f45a33472`; validated 2026-02-16 via `CHUBO_GUARDRAILS_SKIP_BUILD=1 make chubo-guardrails`, `sudo -n ./hack/chubo/e2e-core-qemu.sh --skip-build`, `sudo -n SKIP_BUILD=1 INSTALLER_TAG=helper1771191386 ./hack/chubo/e2e-helper-bundles-qemu.sh`, `sudo -n ./hack/chubo/e2e-opengyoza-quorum-qemu.sh --skip-build`; refreshed inventories in `docs/talos/audits/2026-02-16/`)
- [x] Re-run the full local validation matrix and lock CLI debranding smoke checks in CI. (`talos`: `1070f9e22`, `028343493`; validated 2026-02-22 via `make local-chuboctl-all DEST=/tmp/chuboctl-all PUSH=false`, `make local-talosctl-all DEST=/tmp/talosctl-all PUSH=false`, `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1`, `sudo -n ./hack/chubo/e2e-core-qemu.sh --skip-build`, `sudo -n ./hack/chubo/e2e-helper-bundles-qemu.sh --skip-build`, `sudo -n ./hack/chubo/e2e-opengyoza-quorum-qemu.sh --skip-build`, `sudo -n SKIP_BUILD=1 CONTROLPLANE_COUNT=2 SKIP_NOMAD_JOB_PROBE=0 ./hack/chubo/e2e-cluster-qemu.sh`, `make target-chuboctl-linux-amd64`, `make target-talosctl-linux-amd64`; regenerated CLI docs with `./_out/chuboctl-$(go env GOOS)-$(go env GOARCH) docs website/content/v1.13/reference --cli`)
- [x] Continue source-clean slices and refresh audit inventory after each slice. (`talos`: `767e234c0`, `70b0cf8e9`, `938071262`, `c23ea3c71`, `d5fa3ec27`, `5f756b333`, `4ec2e34e4`, `e05268c92`, `98a14373f`, `89d22333c`, `6e9543e01`, `50e66020e`, `ab16d73e0`, `704bb9b31`, `6a0bbe3fb`, `2f5e8199c`, `dfe1ae188`, `ca2eef261`, `2d26ff1e6`, `0d9622c9c`, `b2c9e1592`, `a5f8b94c5`, `ff58e1483`, `0acd34e8a`, `085ded57f`, `dd4ec0c3a`, `1377c7b58`, `8a11c6f3b`, `50c9556d5`, `78f9a20dd`, `52d89acf1`, `0c85b94ef`, `56e60557f`, `5ab0b8446`, `957bde6f9`, `3153d6a60`, `c7f710896`, `f7c0d8d33`, `1bdbf9476`, `484624cb4`, `30b1452a3`; `chubo`: `42cc2db`, `cd2bc4c`, `4313f22`, `3945347`, `b3ecd52`, `0392c39`; validated 2026-02-16 by rerunning `docs/talos/audits/2026-02-16/collect.sh` after removing dead Kubernetes helpers/local fork wiring, unused etcd check helpers, machine etcd/kubeconfig RPC surface, dead etcd COSI resource/proto surfaces, stale etcd secrets protobuf messages, Kubernetes support-bundle collector plumbing, dead etcd bootstrap shim wiring, unused kube/etcd constants, residual kube/etcd runtime path constants/volume wiring, the legacy Kubernetes PKI import/seed path in `gen secrets`, the stale Kubernetes version compatibility matrix/suite, the stale `kubernetes-version` config-generation knob, remaining legacy `v1alpha1` kubelet/admin-kubeconfig config surface, remaining legacy `v1alpha1` Kubernetes control-plane/CNI config surfaces plus stale cluster-create CNI plumbing, dead legacy machine generateConfiguration proto messages + custom CNI URL CLI flag, dead kubelet/k8s SELinux policy modules/labels, dead Talos API PKI rotation integration-only helpers, dead k8s compatibility hooks + the unused `os:etcd:backup` role constant, residual Kubernetes artifact hooks in `Makefile` plus rootfs Kubernetes scaffolding and stale machine API kube/etcd wording, retiring Kubernetes-coupled integration suites from the default API lane, removing the legacy `k8s.gcr.io` mirror fallback from config generation, normalizing residual runtime/config wording, pruning dead Kubernetes-era `VersionContract` flags while renaming remaining legacy compatibility toggles to neutral names, neutralizing residual kubelet wording in runtime/logind paths, neutralizing residual kube/etcd wording in runtime, config docs, and CLI internals, dropping legacy k8s label parsing from the containerd inspector path, moving chubo-tagged workload CLI/runtime tooling to containerd-only drivers, enforcing namespace checks in chubo container inspector, build-tagging CRI inspector implementation out of chubo builds, shifting registries/image-cache/seccomp resources to workload aliases, dropping stale kube/etcd module requirements while neutralizing legacy integration calls to removed Kubernetes cluster helpers, deleting legacy `internal/integration/k8s` suites/testdata, removing non-chubo `talosctl` jsonpath output roots, bounding core E2E post-install probes to prevent indefinite hangs, retiring remaining `integration_k8s` API/base suites plus legacy kubeconfig integration coverage, trimming stale Kubernetes module metadata from `hack/third_party/go-talos-support`, and renaming legacy secret PKI fields to neutral names while keeping legacy bundle decode compatibility; active refs `2522 -> 2125 -> 1920 -> 1817 -> 1806 -> 1772 -> 1767 -> 1656 -> 1484 -> 1419 -> 1344 -> 1326 -> 1116 -> 1045 -> 1044 -> 954 -> 948 -> 938 -> 909 -> 641 -> 617 -> 604`; with audit split applied, kube/etcd active refs `604 -> 374 -> 280 -> 258 -> 207 -> 183 -> 147 -> 117 -> 103`, tracked separately from CRI naming refs in `docs/talos/audits/2026-02-16/active-cri-refs.txt` (`243 -> 242 -> 238 -> 229 -> 228 -> 158`))
- [x] Add active kube/etcd + CRI refs regression guardrail to `make chubo-guardrails` (normalized baselines in-repo) and trim another CLI/source-clean wording slice. (`talos`: `ba3c22ba5`; validated 2026-02-16 via `./hack/chubo/check-active-refs.sh`, `go test ./cmd/chuboctl/cmd/nodes ./cmd/chuboctl/cmd/mgmt/cluster/create ./pkg/machinery/resources/secrets`, `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1`; active refs now `89`, CRI refs `153`)
- [x] Remove remaining legacy kube/etcd naming refs from active source paths (`recover_etcd`, `registry_kubernetes_enabled`, `KUBE-IPTABLES-HINT`, `kubepods/*`), including API docs alignment and OOM fixture path rename. (`talos`: `62ca5d57d`; validated 2026-02-16 via `go test -count=1 ./internal/app/machined/pkg/controllers/runtime/internal/oom ./pkg/machinery/resources/cluster`, `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go test -c ./internal/app/machined/pkg/controllers/cluster`, `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1`; active refs `89 -> 80`, CRI refs `153` unchanged)
- [x] Harden active-refs enforcement with explicit compatibility paths + forbidden-outside-compat checks (core), trim stale release tooling references (`k8s-bundle`, `registry.k8s.io`) plus stale `deepcopy-gen` tooling deps, and de-noise scans by excluding non-source artifacts (`go.sum`, CA bundle data). (`talos`: `658df6673`, `cdf8849e0`, `06a1b6194`, `9597a9451`, `60b9a4895`, `facdd9790`; validated 2026-02-17 via `./hack/chubo/check-active-refs.sh`, `./hack/chubo/check-active-refs.sh --update-baseline`, `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1`; compat baselines now `0` core refs, `0` CRI refs)

## Exit Criteria

- [x] Product Clean: chubo targets are Kubernetes/etcd free by runtime/API/dependency gates. (2026-02-20 evidence: `hack/chubo/check-go-deps.sh` reports `forbidden deps for internal/app/machined: 0` and `forbidden deps for cmd/chuboctl: 0`; runtime surface check passes in core QEMU lane)
- [x] Replacement Complete: openwonton/opengyoza have Talos-grade OS machinery (bootstrap, trust, lifecycle, upgrade hooks, diagnostics), with no mock-only control path. (2026-02-16 evidence: helper-bundles lane + opengyoza quorum lane both pass)
- [x] Naming: user-facing primary is Chubo (`chuboctl`, `CHUBO*`, `chubo-*`). (2026-02-16 evidence: migration notes published + chubo guardrails CLI/docs scan pass)
- [x] Source Clean: no active Kubernetes/etcd code paths in Chubo mainline. (2026-02-17 evidence: `hack/chubo/active-refs-baseline.txt` and `hack/chubo/active-cri-refs-baseline.txt` both empty after `facdd9790`; `./hack/chubo/check-active-refs.sh` and `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1` pass)
- [x] CI enforces all above without manual auditing. (2026-02-18: `talos/.github/workflows/ci.yaml` runs `unit-tests`, `chubo-guardrails`, and all self-hosted QEMU lanes (`chubo-e2e-core-qemu`, `chubo-e2e-helper-bundles-qemu`, `chubo-e2e-opengyoza-quorum-qemu`, `chubo-e2e-cluster-qemu`) on push, workflow_dispatch, and same-repo PRs; includes non-interactive sudo precheck `sudo -n true` in each QEMU lane.)
