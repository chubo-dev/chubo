# Talos Review - Storage and disks

Subsystem name: Storage, disks, and mounts

- Chubo goal: Detect system disk, ensure `/var/lib/chubo` is present and safe, and optionally support future partitioning/encryption.
- Talos equivalent (module/area): block controllers + volume manager + volume lifecycle + encryption helpers.
- Talos code paths (file/dir):
  - `internal/app/machined/pkg/controllers/block/` (discovery, system disk, mounts, volume manager)
  - `internal/app/machined/pkg/controllers/block/internal/volumes/` (partition/format/encrypt/grow)
  - `pkg/machinery/resources/block/` (DiscoveredVolume, Disk, SystemDisk, VolumeConfig/Status)
  - `internal/pkg/encryption/`
- Key algorithms / state machines: system disk detection via META partition label; discovery refresh gating; VolumeManager state machine with retries and lifecycle phases; idempotent partition/format/expand operations.
- Config model / API surface: VolumeConfig/Status resources, VolumeLifecycle, and MountRequest/Status resources.
- Failure handling / recovery: controller retries with backoff; finalizers guard lifecycle teardown; mount status resources allow idempotent reconcile.
- Test strategy (unit/contract/integration): `internal/app/machined/pkg/controllers/block/system_disk_test.go`, `internal/app/machined/pkg/controllers/block/volume_config_test.go`, `internal/app/machined/pkg/controllers/block/mount_test.go`, `internal/app/machined/pkg/controllers/block/internal/volumes/partition_test.go`.
- What we will reuse: system disk discovery patterns, explicit lifecycle resources, idempotent volume operations.
- What we will adapt: minimal checks (mount present, free space, UUID/label match) instead of full Talos provisioning in Phase 1.
- What we will avoid (and why): Talos full disk partitioning/encryption workflows until chubo Phase 7 (per `plan.md`).
- Action items: define minimal mount + free-space checks for `/var/lib/chubo`, and document safe failure behavior when storage is degraded.
- Chubo status:
  - Added a storage preflight in reconcile that verifies baseDir exists, enforces minimum free space, and (when configured) checks the mount UUID/label for `/var/lib/chubo`.
  - Added an optional wipe workflow (`storage.wipeState` + `storage.wipeID`) that clears state/artifacts/binaries/logs once per ID.
