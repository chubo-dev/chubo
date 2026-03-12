# Deep Talos Fork Plan (No K8s/Etcd/CRI) for a Chubo Distribution

This document is a concrete plan for a deep Talos fork to produce a Talos-like distribution which:

- targets bare metal and cloud
- supports amd64 and arm64
- enforces "no local shell ever" (no SSH, no console login, no break-glass shell)
- removes Kubernetes, etcd, and CRI as first-class subsystems
- provides a simplified, non-Kubernetes configuration model
- runs the Chubo control loop as a first-class OS module, with management surfaced via the OS API (Talos-like)

The goal is to reuse Talos for the hard OS problems (installer, signed boot assets, atomic upgrades, rollback, diagnostics, "no shell" posture), while deleting the Kubernetes product surface area and replacing it with a minimal "appliance OS" that exists to run Chubo.

## Decisions (Current)

- Nomad and Consul are OS-owned (baked into the OS image, upgraded via OS upgrades), and run as OS-managed host process services.
- OpenBao runs as a Nomad job (not an OS host service).
- Upstream strategy: hard fork (branch out). No ongoing rebases.
- Phase 1 compromise: keep CRI *config-only* controllers required by the installer flow (image cache + registries), but do not run CRI as a first-class runtime/service.
- Secrets: OS-only trust (OS issuing CA + trustd token). No Kubernetes/etcd PKI in the `chubo` build (legacy `chuboos` paths/targets remain as wrappers during transition).
- Bootstrap: signed payload (JWS compact, `alg=EdDSA`/Ed25519) verified by a pinned signer certificate in config; verified JSON is rendered to `/var/lib/chubo/bootstrap/bootstrap.json`.
- API surface: one external management plane (OS API via `apid`/`trustd`) plus kubeconfig-like helpers to bootstrap access to the native workload APIs. Chubo does not expose a separate remote API.
- Naming convergence source of truth: `docs/talos/rename-map.md` (Wave B).

## Upstream Strategy: Hard Fork (Branch Out)

This project will branch out from Talos and treat Talos as an initial code drop, not a dependency to continuously rebase onto.

This does not require keeping Kubernetes code. Kubernetes/etcd/CRI can be deleted.

What changes when you hard fork:

- Security: you own the CVE response process (kernel, userland, Go deps, container images, build tooling).
- Platforms: you own long-term amd64/arm64 enablement and quirks (UEFI, NIC/storage variance, cloud metadata drift).
- Updates: you own upgrade/rollback correctness and test coverage across all supported platforms.
- Supply chain: you own signing keys, SBOM/provenance generation, and release artifact hosting.

Recommended operating stance even for a hard fork:

- Keep an internal mirror of Talos upstream for reference.
- Cherry-pick upstream fixes selectively (especially security and platform fixes) when it is cheaper than re-implementing.
- Maintain a strict module boundary so most future work lands in modules, not the base OS.

## Scope and Principles

### Invariants

- No local shell ever: no getty, no SSH, no emergency shell as a recovery path.
- Remote-first management: everything day-2 must be possible via API(s).
- Image-based OS: updates are atomic and rollbackable; signed artifacts.
- Deterministic behavior: same inputs produce same behavior; no interactive mutation.
- Minimal attack surface: only the processes required to run the OS and Chubo.

### Explicit Non-Goals

- Being a general-purpose Linux distribution.
- Supporting Kubernetes as a workload orchestrator.
- Maintaining compatibility with Talos machine/cluster config formats for users.

### Compatibility Target

The fork should keep the following Talos properties:

- Installer + boot assets for bare metal (ISO/PXE) and cloud images.
- Atomic upgrades + rollback.
- "No shell" posture and diagnostics over API.
- A container runtime for OS-managed workloads (system containerd + extensions).

And explicitly remove:

- Kubernetes control plane and worker semantics.
- kubelet lifecycle and kubelet APIs.
- etcd membership orchestration.
- CRI-oriented container runtime pipeline.

## Strategy Overview

This is a deep fork, not a "thin wrapper". The plan uses three tactics to reduce risk:

1. Compile-time removal first (build tags / build variants) to get a booting system with deleted subsystems.
2. Config surface replacement next (new config schema + toolchain), once we know the minimal set of remaining controllers/services.
3. Image composition cleanup last (remove binaries, images, registries, and unused features), once behavior is validated.

This order prevents "rewrite the config" from becoming a multi-month detour before we can even boot and test.

## Boundary: Base OS vs Modules

This fork should be structured as a k8s-less "base OS" plus optional modules.
The base OS must remain useful without any opinionated "cluster primitives" baked in.

### Base OS Responsibilities

- Enforce "no local shell ever" by construction.
- Provide boot assets + installer for bare metal and cloud images.
- Provide atomic OS upgrades with rollback and health gating.
- Provide a remote OS API which is sufficient to recover a node when higher-level software is unhealthy.
- Provide the minimum runtime needed for system workloads.
- Runtime: system `containerd` (not CRI) for extension containers.
- Runtime: process supervision for host daemons.
- Provide core node primitives.
- Primitives: disks/mounts.
- Primitives: networking and DNS resolver.
- Primitives: time sync.
- Primitives: logging and support bundle.
- Primitives: certificate and trust management for the OS API.

### Module Responsibilities

- Add product-specific daemons and controllers on top of the base OS without changing base invariants.
- Be build-time selectable (build tags or separate targets) and runtime-selectable (enabled/disabled via config).
- Own their own config schema section and status surface.
- Integrate with OS upgrades by providing explicit drain/stop/start hooks.

### Base OS: Controllers and Services (Keep)

Controllers to keep (Talos upstream, k8s-less):

- `internal/app/machined/pkg/controllers/block`
- `internal/app/machined/pkg/controllers/files`
- `internal/app/machined/pkg/controllers/hardware`
- `internal/app/machined/pkg/controllers/network`
- `internal/app/machined/pkg/controllers/perf`
- `internal/app/machined/pkg/controllers/runtime` (excluding CRI runtime controllers)
- `internal/app/machined/pkg/controllers/cri` (config-only: image cache + registries; required for installer flow)
- `internal/app/machined/pkg/controllers/time`
- `internal/app/machined/pkg/controllers/config` (to be replaced/rewired for the new config schema)
- `internal/app/machined/pkg/controllers/secrets` (trim to OS API trust only)

Services to keep (Talos upstream, k8s-less):

- `internal/app/machined/pkg/system/services/machined.go`
- `internal/app/machined/pkg/system/services/apid.go` (OS API)
- `internal/app/machined/pkg/system/services/containerd.go` (system containerd)
- `internal/app/machined/pkg/system/services/extension.go` (extension services)
- `internal/app/machined/pkg/system/services/trustd.go` (evaluate: keep if still required for OS trust flows)
- `internal/app/machined/pkg/system/services/udevd.go`

Services to keep (optional, based on desired OS surface):

- `internal/app/machined/pkg/system/services/auditd.go`
- `internal/app/machined/pkg/system/services/syslogd.go`
- `internal/app/machined/pkg/system/services/registryd.go`
- `internal/app/machined/pkg/system/services/dashboard.go`

### Module: Chubo Bootstrap Agent (First-Class Workload)

Purpose:

- Run `chubo-agent` as a first-class system workload.
- Provide a stable bootstrap path for Chubo config and trust under `/var/lib/chubo`.

Expected additions:

- Extension service spec and packaging for `chubo-agent` (or a host process service if required).
- Controllers to render/manage Chubo OS-integration files and health reporting.
- A minimal status surface which proves the Chubo module is up and observable via the OS API (no separate remote listener).

### Module: Chubo (Nomad + Consul, OS-Owned)

Purpose:

- Provide Nomad and Consul as OS-managed host process services.
- Provide deterministic bootstrap and upgrade orchestration without a shell.

Expected additions:

- New services (host processes, supervised by the OS service manager).
- Service: `nomad` (new).
- Service: `consul` (new).
- New controllers.
- Controller: config rendering for Nomad and Consul.
- Controller: TLS material generation/rotation for Nomad and Consul.
- Controller: ACL/bootstrap token handling (must not depend on OpenBao).
- Controller: health and readiness gating resources.
- OS API helpers: generate Nomad/Consul client configuration bundles (addresses, CA/certs, ACL tokens) to access their native APIs (Talos analogy: `talosctl kubeconfig`).
- Upgrade hooks.
- Hook: drain workloads (Nomad) and stop Nomad before OS upgrade/reboot.
- Hook: enforce Consul quorum safety rules for servers.

### Module: OpenBao (Nomad Job)

Purpose:

- Run OpenBao as a scheduled workload once Nomad is healthy.
- Keep OpenBao upgrades decoupled from OS upgrades.

Expected additions:

- A controller which ensures the OpenBao job exists in Nomad and is updated idempotently when its spec changes.
- Init and unseal automation which does not require a shell.
- Auto-unseal is effectively required for "no local shell ever".
- A status surface which reports:
- job presence and version
- sealed/unsealed
- reachable/unreachable
- OS API helpers: generate OpenBao client configuration (and any required bootstrap material) to access the native OpenBao API without a shell.

## What We Reuse From Talos

Keep these Talos subsystems largely intact:

- Installer and boot asset pipeline: `cmd/installer`, build tooling, signed assets.
- Runtime/controller framework and state model (COSI runtime).
- Core controllers: storage/block, files, network, time, hardware, runtime essentials, diagnostics, logging, SBOM reporting.
- system containerd and extension service execution model.
- Upgrade/rollback sequencing and "health-gated" reboot semantics.
- API plumbing that supports remote logs/status/support bundles (exact APIs to be decided).

Key Talos code areas which are valuable and mostly non-Kubernetes:

- Controller runtime assembly: `internal/app/machined/pkg/runtime/v1alpha2/v1alpha2_controller.go`
- Extension services runtime: `internal/app/machined/pkg/system/services/extension.go`
- system containerd: `internal/app/machined/pkg/system/services/containerd.go`
- Network/time/block controllers under `internal/app/machined/pkg/controllers/{network,time,block,files,hardware,runtime}`

## What We Remove (and Why It's Not Just Deleting Folders)

Kubernetes and etcd are not isolated. They are coupled into:

- controller registration (`v1alpha2_controller.go`)
- boot sequences (`v1alpha1_sequencer_tasks.go` starts `CRI`, loads `Kubelet`, conditionally starts `Etcd`)
- secrets/PKI controllers (`controllers/secrets` has Kubernetes/etcd related CAs and certs)
- runtime health and image GC (`controllers/runtime` and `controllers/cri`)
- config types and validation (`pkg/machinery/config/types/v1alpha1/*` is Kubernetes-first)

The plan deletes the subsystems and then repairs the seams by:

- redefining boot sequences to start only OS essentials + Chubo
- shrinking secrets/PKI to only what the OS + Chubo require
- replacing config schema and validation (no `cluster:` / kubelet image logic)

### Primary Removal Targets

- Controller: `internal/app/machined/pkg/controllers/k8s`
- Controller: `internal/app/machined/pkg/controllers/etcd`
- Controller: `internal/app/machined/pkg/controllers/cri` (remove CRI runtime controllers; keep config-only controllers required by installer)
- Controller: `internal/app/machined/pkg/controllers/kubeaccess` (likely; Kubernetes API access)
- Controller: `internal/app/machined/pkg/controllers/kubespan` (evaluate if any non-k8s value remains)
- Service: `internal/app/machined/pkg/system/services/cri.go`
- Service: `internal/app/machined/pkg/system/services/kubelet.go`
- Service: `internal/app/machined/pkg/system/services/etcd.go`
- Sequencer: `internal/app/machined/pkg/runtime/v1alpha1/v1alpha1_sequencer_tasks.go` (remove kube/etcd/cri start/stop/drain tasks; replace with OS-only service lifecycle)
- Secrets: Kubernetes/etcd specific controllers in `internal/app/machined/pkg/controllers/secrets/*`
- Config: Kubernetes-first config types in `pkg/machinery/config/types/v1alpha1/*` need replacement or a new "chubo OS" config package.

## Proposed New Config Model (Simplified, Non-Kubernetes)

The fork needs a config schema that can:

- install the OS (disk target, wipe policy, encryption if required)
- bring up networking (DHCP first, static optional)
- establish a trust root for remote API access and for Chubo bootstrap
- configure time/logging basics
- define and configure the Chubo agent workload

### Config Principle

The OS config should be "bootstrap minimal": only what is needed to:

- boot securely
- obtain network
- establish the management trust chain
- start `chubo-agent`

Everything else should be managed by Chubo via its own API and stored under `/var/lib/chubo`.

### Sketch: `ChuboOSConfig` (strawman)

This is not a final schema; it is a target for Phase 2.

```yaml
apiVersion: chubo.dev/v1alpha1
kind: MachineConfig
metadata:
  id: <optional stable node id>
spec:
  install:
    disk: /dev/nvme0n1
    wipe: false
    # optional: encryption, partitions, raid

  network:
    dhcp: true
    # optional: static config for bare metal

  time:
    servers: [ "time.cloudflare.com" ]

  logging:
    consoleLevel: info
    # optional: remote sink configuration

  trust:
    # trust inputs for OS API + bootstrap (refined in Phase 3)
    token: <string> # trustd token (Talos-like)
    ca:
      crt: <pem>
      key: <pem>    # Phase 2: keep issuing CA key local; rotation/pinning later
    acceptedCAs: [] # optional extra roots

  registry:
    mirrors:
      "10.0.2.2:5001":
        endpoints: [ "http://10.0.2.2:5001" ]

  modules:
    chubo:
      enabled: true
      # how the node obtains initial Chubo MachineConfig + trust
      bootstrap:
        mode: signedPayload
        signerCert: <pem>   # PEM-encoded cert; must be Ed25519 public key
        payload: <jws>      # compact JWS string, protected header must include {"alg":"EdDSA"}
      agent:
        image: <oci image>  # or "binary" install mode
        version: <tag>
        # no remote listener; management flows go through the OS API
        # minimal flags only; Chubo owns the rest via its MachineConfig
      consul:
        enabled: true
        role: server # server|client|server-client
      nomad:
        enabled: true
        role: server # server|client|server-client
      openbao:
        enabled: false
        mode: external # external|hostService|nomadJob
```

## Architecture of the Forked OS

### Runtime Model

Use Talos "system containerd" + extension services for containerized system workloads, and add OS-managed
process services for host-only workloads (Nomad/Consul/OpenBao).

`Nomad` is commonly treated as a host process (especially for clients), so the fork should not assume it can
run as an extension container.

Baseline approach:

- Keep `containerd` service (`internal/app/machined/pkg/system/services/containerd.go`).
- Use `Extension` service to run a Chubo extension/container (`internal/app/machined/pkg/system/services/extension.go`).
- Add new process-managed services (Talos-style).
- Service: `nomad`.
- Service: `consul`.
- OpenBao runs as an OS-managed host service in the steady-state runtime path; the legacy Nomad-job mode remains only for explicit bootstrap/dev flows.
- Do not ship/run CRI containerd (`cri.go`) or Kubernetes services.

This yields:

- a minimal runtime to execute Chubo without systemd
- lifecycle control and health checks via the Talos service manager
- the ability to manage host-only daemons without adding systemd

### Control Plane Surfaces (Talos-like)

Decision: single external management plane (OS API).

- OS API (`apid` + COSI state, and `trustd` for cert flows) is the backstop for:
  - logs, network state, upgrade/rollback, reset, support bundle
  - Chubo module status and operations (apply desired state, read status, stream logs)
- OS API also provides minimal "workload access helpers" to bootstrap access to the native APIs (Talos analogy: `MachineService.Kubeconfig()`):
  - Nomad: client config + mTLS/ACL material to talk to openwonton
  - Consul: client config + mTLS/ACL material to talk to opengyoza
  - OpenBao: client config + bootstrap material to talk to openbao once the local host service has initialized
- `chuboctl` targets the OS API (analogous to `talosctl`).
- Chubo runs locally (extension or host service) but does not expose a separate remote API listener.
- openwonton/opengyoza/openbao keep their native APIs as workload control planes (analogous to the Kubernetes API).

Alternative (not chosen): dual API surfaces (OS API + a separate Chubo API). This increases surface area and creates "two control planes".

## Replacing K8s/Etcd/CRI with Nomad/Consul/OpenBao

Removing Kubernetes-related subsystems is necessary but not sufficient if the distribution is intended to
provide "cluster primitives" out of the box.

Two viable models:

Model 1: OS provides only "OS primitives"; Chubo provides "cluster primitives".

- OS primitives: install, upgrade/rollback, networking/time/logging/diagnostics, and a way to run `chubo-agent`.
- Cluster primitives (Nomad/Consul/OpenBao): downloaded/configured/orchestrated by Chubo.

Pros:

- OS stays simpler and closer to Talos upstream.
- Clear product boundary: Chubo is the control plane.

Cons:

- Chubo must manage host processes in a systemd-less OS (needs an integration mechanism with the OS service manager).
- More failure modes during early boot (before Chubo is fully healthy) and during upgrades.

Model 2: OS provides both "OS primitives" and "cluster primitives".

- OS runs/supervises Nomad/Consul (and optionally OpenBao) as first-class services.
- Chubo configures them (or they are configured directly via the OS config), but their lifecycle is OS-owned.

Pros:

- Fewer chicken-and-egg loops; easier bootstrap and recovery in a no-shell system.
- OS upgrades can explicitly drain/stop/restart these services safely.

Cons:

- The OS fork becomes "Nomad OS", not just "Talos without Kubernetes".
- Higher rebase and maintenance burden (you will own these lifecycle controllers).

This plan assumes Model 2 for Nomad and Consul (OS-owned lifecycle).
OpenBao runs as a Nomad job, with job lifecycle owned by the OS Chubo module (not by Chubo control-plane API).

## Phased Work Plan

### Phase 0: Inventory and Minimal Build Variant

Deliverables:

- A list of all Kubernetes/etcd/CRI touchpoints by package and build artifact.
- A dependency map which shows what needs to be removed vs stubbed.
- A compile-time build variant (build tag or separate main) which excludes selected subsystems.
- Exclude: `controllers/k8s`, `controllers/etcd`, and CRI runtime controllers (keep CRI config-only controllers required by installer).
- Exclude: `services/{kubelet,etcd,cri}`.
- Exclude: sequencer tasks which start/stop those services.

Notes:

- The first success gate is: "forked machined compiles and boots on qemu amd64 and arm64 without those subsystems".

### Phase 1: Boot and Service Graph Without K8s/Etcd/CRI

Deliverables:

- Modified controller registration in `internal/app/machined/pkg/runtime/v1alpha2/v1alpha2_controller.go` to exclude removed controllers.
- Modified boot sequences in `internal/app/machined/pkg/runtime/v1alpha1/v1alpha1_sequencer_tasks.go`.
- Boot: `StartAllServices` should start only OS services required for management + extensions.
- Upgrade: `StopServicesEphemeral` should stop only those services.
- Remove or stub any sequence tasks that assume Kubernetes drain/etcd leave.

Success gates:

- System reaches "ready" state and is remotely observable via API.
- `containerd` service starts and can run at least one extension service.

### Phase 2: Config Simplification (New Schema + Tooling)

Deliverables:

- New config package (either replacement of `pkg/machinery/config/types/v1alpha1` or a new package selected at build time).
- Validation and defaults logic updated to remove Kubernetes image/version assumptions.
- CLI toolchain to generate and apply the new config (transition `talosctl` to `chuboctl` per `docs/talos/rename-map.md`, or implement minimal equivalent).

Success gates:

- Install + first boot + remote API access works using only the new config.
- No user-visible Kubernetes concepts in docs or CLI output.

### Phase 3: Secrets/Trust Reshape (No K8s/Etcd PKI)

Deliverables:

- Remove Kubernetes/etcd specific secrets controllers.
- Secrets: eliminate kubelet/k8s/etcd CA and cert generation paths.
- Secrets: keep only OS API certs and any trust required for extensions/registry access.
- Define the trust relationship between OS API CA(s), Chubo mTLS CA pinning and rotation, and the Chubo bootstrap signer (if used).

Success gates:

- Node can rotate OS API certs without Kubernetes fields.
- Chubo bootstrap and mTLS rotation work without coupling to Talos cluster token semantics.

### Phase 4: Image Composition Cleanup (Actually No K8s Bits Shipped)

Deliverables:

- Remove kubelet/etcd/CRI binaries, images, configs, and registries artifacts from the built rootfs.
- Remove unused controllers/resources from build outputs to reduce attack surface and size.
- SBOM and artifact signing updated to reflect the new content set.

Success gates:

- Built images contain no kubelet/etcd/CRI binaries and no Kubernetes container images.
- Security review pass on process list and network listeners (only OS API + required internals, plus enabled workload APIs like Nomad/Consul/OpenBao).

### Phase 5: Core Services (Chubo + Nomad + Consul + OpenBao)

Deliverables:

- `chubo-agent` as a first-class OS workload.
- Chubo packaging: extension service (OCI) or host process service.
- Chubo delivery: versioned, signed, delivered as part of OS release (preferred early) or via a controlled update channel.
- Nomad and Consul as OS-managed host process services.
- Nomad/Consul: service definitions (process runner, restart policy, resource limits).
- Nomad/Consul: config rendering pipeline (from OS config and/or Chubo config) into OS-managed `/etc`-like paths.
- Nomad/Consul: health checks (HTTP/RPC checks, readiness gates).
- Nomad/Consul: clean shutdown semantics for reboot/upgrade.
- OpenBao runs as a Nomad job (system job recommended for servers; evaluate clients separately).
- OpenBao: init/unseal automation and a remote operational API surface (no shell).
- Boot order: OS API up.
- Boot order: chubo-agent up.
- Boot order: Consul up (if used as a dependency).
- Boot order: Nomad up.
- Boot order: OpenBao job submitted and healthy.
- Upgrade integration: before OS upgrade, drain workloads (Nomad), then stop Nomad cleanly.
- Upgrade integration: enforce Consul server quorum safety rules (servers vs clients).
- Upgrade integration: health-gated reboot and rollback still works when Nomad is present.

Success gates:

- Fresh install boots, starts the OS API and chubo-agent, and is fully manageable remotely with no shell access.
- Nomad and Consul come up deterministically and report health over API.
- OS upgrades drain and reboot safely with clear failure reporting and rollback behavior.

### Phase 6: Platforms, Cloud Images, and Release Engineering

Deliverables:

- Bare metal: ISO and PXE artifacts for amd64/arm64.
- Cloud: published images for the chosen providers (at minimum AWS + one more).
- CI: build + unit tests.
- CI: qemu-based boot tests (amd64/arm64).
- CI: upgrade/rollback tests.
- CI: smoke tests (Chubo module healthy/observable via OS API, Nomad/Consul health checks pass).
- Release policy: channels (stable/beta/nightly).
- Release policy: signing keys and rotation.
- Release policy: compatibility policy between OS version and chubo-agent version.

Success gates:

- End-to-end install + bootstrap + remote manage + upgrade + rollback in qemu amd64.
- End-to-end install + bootstrap + remote manage + upgrade + rollback in qemu arm64.
- End-to-end install + bootstrap + remote manage + upgrade + rollback in one bare metal lab run.
- End-to-end install + bootstrap + remote manage + upgrade + rollback in at least one cloud provider.

## Testing Strategy (No-Shell Reality)

"No shell ever" means tests must validate remote observability and recovery.

Minimum test suites:

- Boot smoke: OS API reachable, network configured, logs accessible.
- Service smoke: start/stop/restart `chubo-agent`, `nomad`, `consul`, verify health.
- Workload smoke: OpenBao Nomad job exists, is running, and is unsealed (per the chosen unseal method).
- Upgrade: apply OS upgrade, reboot, validate rollback on injected failure.
- Reset: factory reset/reinstall flow is remotely triggerable and audited.

## Developer Loop (Lima)

We use Lima for fast local iteration on macOS when validating end-to-end behavior (mTLS, apply/status, logs, DNS checks, updates).
These workflows are allowed to use `limactl shell` for development even though the product invariant is "no local shell ever".

Primary references in this repo:

- `docs/dev/lima-smoke-test.md` (single VM: mTLS + apply/status + logs)
- `docs/dev/lima-cluster-e2e.md` (multi-VM: Nomad jobs)
- `docs/dev/flatcar-update-test.md` (Flatcar update_engine + reboot flow; useful reference while implementing the OS upgrade gate/rollback story)
- `docs/dev/lima-test.yaml` (reference MachineConfig used by the smoke test)

Code and templates:

- `hack/lima/cluster-e2e.sh` (automates the cluster workflow)
- `hack/lima/chubo.yaml` (Fedora-based VM template)
- `hack/lima/flatcar.yaml` (Flatcar-based VM template)

Hard-fork follow-up:

- Add a dedicated Lima template for the forked OS images (for example `hack/lima/chubo-os.yaml`) once we can produce qcow2/ISO outputs.
- Extend `hack/lima/cluster-e2e.sh` (or add a sibling script) to run install/upgrade/rollback smoke tests against the forked OS.

## Risks and Mitigations

- Hard fork maintenance burden (security fixes and platform enablement become internal work).
- Mitigation: maintain an internal Talos upstream mirror for reference and selective cherry-picks; keep the base OS small and modular; automate boot/upgrade tests across amd64/arm64.
- Hidden k8s coupling outside obvious controllers.
- Mitigation: Phase 0 inventory, build artifact scanning, and "no-k8s-on-disk" tests.
- Losing critical OS features while deleting subsystems.
- Mitigation: preserve controller-runtime core and sequences; change only what is required, then shrink.
- Dual-management-plane confusion (OS API vs Chubo API).
- Mitigation: keep a single external management plane (OS API); keep any Chubo API internal-only or remove it.

## Deliverable Definition of Done

The fork is "real" (not a prototype) when:

- Images are published (bare metal + cloud) for amd64 and arm64.
- The node is operable without any local shell for install.
- The node is operable without any local shell for bootstrap.
- The node is operable without any local shell for day-2 config changes.
- The node is operable without any local shell for OS upgrades + rollback.
- The node is operable without any local shell for reset/reinstall.
- The node is operable without any local shell for diagnostics/support bundle.
- There is a single external management endpoint (OS API via `apid`/`trustd`); Chubo does not expose a separate remote API.
- Client access material for openwonton/opengyoza/openbao can be obtained via OS API helpers (kubeconfig-like), without a shell.
- No kubelet/etcd/CRI binaries or images ship in the OS.
- CI proves boot and upgrade on qemu for both architectures.
