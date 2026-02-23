# Talos Review - Config, auth, and trust chain

Subsystem name: Config ingestion + auth + trust chain

- Chubo goal: Load/validate MachineConfig from disk/bootstrap/API, persist canonically, and enforce mTLS + RBAC for remote control.
- Talos equivalent (module/area): Config acquisition controller + config schema/validation + secrets controllers + gRPC authz middleware.
- Talos code paths (file/dir):
  - `internal/app/machined/pkg/controllers/config/acquire.go` (multi-source config load + state machine)
  - `pkg/machinery/config/` (types, loader, validation, schemas)
  - `pkg/machinery/resources/config/` (MachineConfig resources)
  - `internal/app/machined/pkg/controllers/secrets/` and `pkg/machinery/resources/secrets/`
  - `pkg/grpc/middleware/authz/` and `pkg/machinery/role/`
- Key algorithms / state machines: AcquireController state machine loads from disk/cmdline/platform; config validation via `config/validation`; secrets controllers generate/rotate API/etcd/k8s certs; Trustd regenerates certs on time changes and machine type changes.
- Config model / API surface: gRPC MachineService `ApplyConfiguration/Bootstrap` in machined server; roles extracted from client cert organization strings with authz rules; secrets exposed as COSI resources.
- Failure handling / recovery: AcquireController publishes ConfigLoadErrorEvent + platform failure events; controllers gate on required inputs and clean up outputs; authz returns PermissionDenied for missing roles.
- Test strategy (unit/contract/integration): `pkg/machinery/config/contract_test.go`, `pkg/machinery/config/config_schema_test.go`, `internal/app/machined/pkg/controllers/secrets/api_test.go`, `pkg/grpc/middleware/authz/authorizer_test.go`.
- What we will reuse: multi-source config loading + validation pipeline; strict RBAC from client certs; separation of secrets into dedicated resources.
- What we will adapt: chubo MachineConfig schema + bootstrap token/signature flow; lighter-weight secret store vs Talos’s full COSI resource graph.
- What we will avoid (and why): Talos-specific maintenance mode flows and Kubernetes secret rotation details that are out of chubo Phase 1 scope.
- Action items: define bootstrap trust flow (token vs signed config), map chubo roles to cert orgs, and specify when cert rotation is triggered.
- Chubo status:
  - Server TLS reloads from disk on each new connection; CA is pinned in `/var/lib/chubo/state/ca.pin`.
  - Added `GenerateServerCSR` RPC to stage a pending server key + CSR.
  - Added `RotateServerCert` RPC to swap the server cert/key (must be signed by the pinned CA).
  - Added client cert revocation list (`/var/lib/chubo/state/revoked.json`) + `RevokeClientCert`.
  - Added bootstrap payload ingestion (`/var/lib/chubo/bootstrap/bootstrap.json`) signed by `/var/lib/chubo/bootstrap/signer.crt`, with replay protection recorded in `/var/lib/chubo/state/bootstrap.json`.
