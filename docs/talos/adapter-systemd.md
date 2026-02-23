# Talos Review - Adapter / systemd

Subsystem name: Adapter / systemd (service & unit management)

- Chubo goal: Manage systemd units via DBus (apply unit + drop-ins atomically, enable/disable, restart only on spec changes) and report observed status/hashes.
- Talos equivalent (module/area): Talos does not use systemd; it ships a custom service supervisor and runners for process/containerd services.
- Talos code paths (file/dir):
  - `internal/app/machined/pkg/system/` (service supervisor + state machine)
  - `internal/app/machined/pkg/system/runner/` (process/containerd runners, restart policies)
  - `internal/app/machined/pkg/system/services/` (service definitions like `containerd`, `kubelet`, `machined`)
- Key algorithms / state machines: `ServiceRunner` state transitions + health checks, dependency ordering in `system.go`, restart policies in `runner/restart`.
- Config model / API surface: service IDs and dependency graph in the supervisor; gRPC MachineService methods like ServiceStart/Stop/Restart are authorized in `internal/app/machined/pkg/system/services/machined.go`.
- Failure handling / recovery: runner emits state events on failure, restart loops can be configured, health checks publish degraded state; supervisor tracks running set and unloads cleanly.
- Test strategy (unit/contract/integration): `internal/app/machined/pkg/system/service_runner_test.go`, `internal/app/machined/pkg/system/system_test.go`, `internal/app/machined/pkg/system/runner/runner_test.go`.
- What we will reuse: explicit service state machine + event history patterns; health check integration; deterministic dependency ordering concepts.
- What we will adapt: map Talos-style service IDs + restart semantics to systemd unit names; emit observed hash/status in chubo state.
- What we will avoid (and why): Talos’s custom supervisor/runner implementation and containerd-based service execution, since chubo relies on systemd DBus for lifecycle.
- Action items: define a minimal “service state model” for systemd units, decide on restart/backoff behavior, and map ServiceSpec fields to systemd unit + drop-ins.
