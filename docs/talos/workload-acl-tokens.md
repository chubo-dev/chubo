# Workload ACL Token Model (OpenWonton/OpenGyoza)

Chubo bootstraps OpenWonton (Nomad) and OpenGyoza (Consul) ACLs without introducing a second secret distribution path.

## Derivation

- The management ACL token for each workload is derived deterministically from `spec.trust.token` and a fixed name:
  - OpenWonton: `nomad`
  - OpenGyoza: `consul`
- Implementation: HMAC-SHA256 over the trust token, formatted as a UUIDv4 for API compatibility.

## Usage

- OS controllers use the derived token to:
  - Bootstrap ACLs via the native API (server role only).
  - Validate readiness via `GET /v1/acl/token/self`.
  - Emit helper bundles (`chuboctl nomadconfig`, `chuboctl consulconfig`) which include `acl.token`.
- OpenGyoza also renders the token into `acl.tokens.master` and `acl.tokens.agent` for agent self-auth.

## Persistence and Rotation

- The token is stable as long as `spec.trust.token` stays the same.
- Rotating `spec.trust.token` after the workload ACL bootstrap is complete is not currently supported:
  - the cluster remains bootstrapped with the old token
  - new nodes will report `ACL already bootstrapped but derived token is not accepted`
- Recovery options:
  1. Revert to the previous `spec.trust.token` and re-apply machine config.
  2. Reset/wipe the cluster and re-bootstrap with the new token.
  3. Manual migration (future): use the old management token to mint a new token matching the newly derived token.

## Observability

- `chuboctl get openwontonbootstrapstatus` and `chuboctl get opengyozabootstrapstatus` report `ACLReady`/`ACLLastError`.
- Bootstrap status resources include `ACLTokenSHA256` so you can correlate changes without printing the token.

