# Talos Review - OS updates / update_engine

Subsystem name: OS upgrade sequencing

- Chubo goal: Coordinate Flatcar updates (update_engine) with drain, maintenance windows, reboot, and verification.
- Talos equivalent (module/area): runtime sequencer with upgrade phases + installer-based upgrade tasks + bootloader fallback.
- Talos code paths (file/dir):
  - `internal/app/machined/pkg/runtime/sequencer.go`
  - `internal/app/machined/pkg/runtime/v1alpha1/v1alpha1_sequencer.go`
  - `internal/app/machined/pkg/runtime/v1alpha1/v1alpha1_sequencer_tasks.go` (Upgrade task)
  - `internal/app/machined/revert.go`
  - `internal/app/machined/pkg/runtime/v1alpha1/bootloader/` (sdboot/grub)
  - `internal/app/machined/pkg/controllers/runtime/drop_upgrade_fallback.go`
- Key algorithms / state machines: upgrade sequences with ordered phases (drain, stop services, unmount, upgrade, reboot); Upgrade task runs installer container; upgrade fallback tags in META and bootloader revert.
- Config model / API surface: `machine.UpgradeRequest` gRPC API; sequencer chooses sequence based on runtime mode and request flags. Flatcar DBus interface: `com.coreos.update1.Manager` with `GetStatus`, `AttemptUpdate`, `ResetStatus`, and `StatusUpdate` signal (signature `xdssx`).
- Failure handling / recovery: revert on failed upgrade (meta tag + bootloader revert); drop fallback tag after successful boot; errors abort sequence.
- Test strategy (unit/contract/integration): `internal/app/machined/pkg/runtime/sequencer_test.go`, `internal/app/machined/pkg/controllers/runtime/drop_upgrade_fallback_test.go`, `internal/app/machined/pkg/runtime/v1alpha1/bootloader/sdboot/sdboot_test.go`.
- What we will reuse: explicit upgrade sequencing model and “fallback tag” concept; ordered shutdown/unmount steps before upgrade.
- What we will adapt: swap Talos installer/bootloader logic for Flatcar update_engine DBus (com.coreos.update1.Manager GetStatus) + chubo maintenance-window gating.
- What we will avoid (and why): Talos-specific installer container flow and bootloader management, since Flatcar uses update_engine.
- Action items: map chubo update states to update_engine status (UPDATE_STATUS_*), gate reboot on UPDATED_NEED_REBOOT, define verification checks post-reboot, document rollback criteria, and wire updateChannel to Flatcar update_engine config (TODO). Openwonton drain is implemented (set node ineligible, wait allocations).
