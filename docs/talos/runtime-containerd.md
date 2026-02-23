# Talos Review - Container runtime (containerd/CRI)

Subsystem name: Containerd / CRI configuration + lifecycle

- Chubo goal: Manage container runtime config, registry auth, and runtime health for node workloads.
- Talos equivalent (module/area): containerd service + CRI registries config pipeline + CRI base runtime spec.
- Talos code paths (file/dir):
  - `internal/app/machined/pkg/system/services/containerd.go`
  - `internal/app/machined/pkg/controllers/cri/registries_config.go`
  - `internal/app/machined/pkg/controllers/files/cri_registry_config.go`
  - `internal/pkg/containers/cri/containerd/` (hosts + config generators)
  - `internal/app/machined/pkg/controllers/files/cri_base_runtime_spec.go`
  - `internal/app/machined/pkg/controllers/runtime/cri_image_gc.go`
- Key algorithms / state machines: config flow MachineConfig -> RegistriesConfig resource -> etc file specs; hosts.toml generation for mirrors/TLS; base runtime spec mutation; image GC via containerd client.
- Config model / API surface: registry mirror/auth/TLS in machine config; CRI resources in `pkg/machinery/resources/cri`; containerd health check via gRPC health.
- Failure handling / recovery: file controllers sync idempotently (write only on change), service runner restarts containerd on failure, health checks surface non-serving status.
- Test strategy (unit/contract/integration): `internal/pkg/containers/cri/containerd/hosts_test.go`, `internal/app/machined/pkg/controllers/files/cri_base_runtime_spec_test.go`, `internal/app/machined/pkg/controllers/runtime/cri_image_gc_test.go`.
- What we will reuse: registry config rendering patterns, hosts.toml generation, health check approach.
- What we will adapt: narrow scope to chubo components (openwonton/opengyoza) and only the registry features needed for Phase 1.
- What we will avoid (and why): full Talos CRI image caching + Kubernetes-specific assumptions where chubo does not require them.
- Action items: define chubo runtime responsibilities, pick config file locations, and document required registry/TLS inputs.
