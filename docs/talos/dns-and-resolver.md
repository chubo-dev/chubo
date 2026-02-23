# Talos Review - DNS and resolver plumbing

Subsystem name: DNS cache + resolver wiring

- Chubo goal: Ensure host DNS and allocation DNS work deterministically (systemd-resolved + optional stub) and expose resolver status.
- Talos equivalent (module/area): Host DNS controller + DNS resolve cache service (CoreDNS-based) running on loopback.
- Talos code paths (file/dir):
  - `internal/pkg/dns/` (DNS server, cache, runner manager)
  - `internal/app/machined/pkg/controllers/network/hostdns_config.go`
  - `internal/app/machined/pkg/controllers/network/dns_resolve_cache.go`
  - `internal/app/machined/pkg/controllers/network/probe.go`
  - `internal/app/machined/pkg/controllers/network/internal/probe/probe.go`
  - `pkg/machinery/resources/network/` (HostDNSConfig, DNSUpstream, DNSResolveCache)
- Key algorithms / state machines: DNS manager runs runners per address pair (UDP/TCP), clears cache on upstream change, and uses suture supervisor; controller reconciles HostDNSConfig and upstreams to running DNS status.
- Config model / API surface: MachineConfig HostDNS feature flags; network resources define upstreams and listen addresses.
- Failure handling / recovery: DNSResolveCacheController tolerates IPv6 runner errors, clears/tears down runners on disable, and records per-runner status resources.
- Test strategy (unit/contract/integration): `internal/pkg/dns/dns_test.go`, `internal/app/machined/pkg/controllers/network/dns_resolve_cache_test.go`.
- What we will reuse: upstream management + cache invalidation patterns; explicit status resources for running DNS endpoints.
- What we will adapt: replace Talos loopback DNS server with systemd-resolved + optional CoreDNS stub for allocation namespaces.
- What we will avoid (and why): Talos-specific HostDNS forwarding to Kubernetes service CIDR, unless chubo needs it.
- Chubo status:
  - Implemented systemd-resolved drop-in: `Domains=~<opengyozaDomain>`, `DNS=127.0.0.1`, and optional `DNSStubListenerExtra=<allocationDNSServer>`.
  - `InspectState` reads the drop-in to avoid reconcile drift.
  - Added `allocationDNSMode`:
    - `resolved` (default) keeps `DNSStubListenerExtra`.
    - `coredns` runs `chubo-dns-stub.service` (CoreDNS) bound to the allocation DNS IP and forwards to `127.0.0.53`.
  - Added allocation DNS check in reconcile (Talos probe-inspired): resolves an internal `.consul` name and `example.com` via the allocation DNS server and surfaces results in `GetStatus`.
  - If `/var/run/netns/*` entries exist (Nomad allocations), the check also runs inside a few netns to validate allocator DNS from allocation contexts (throttled to ~30s).
- Action items: validate `DNSStubListenerExtra` on Flatcar; add dns-check job for allocations; ensure CoreDNS stub behavior is exercised in integration tests.
