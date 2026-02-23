# Talos/Talosctl Reference Elimination Plan

Goal: remove `talos`/`talosctl` naming from source code and build/runtime surfaces, while keeping only compatibility references that are technically required.

Scope: `talos/` codebase first, `chubo/` docs and tooling follow-up second.

## Execution Strategy (Explicit)

Decision date: February 20, 2026.

- **Option 1 (active now):** compatibility-first debranding. Keep `talos`/`talosctl` symbols as read-only aliases while shifting all new output/surfaces to `chubo`.
- **Option 2 (next major slice):** API/proto namespace migration (`talos.*` -> `chubo.*`) with compatibility behavior during transition.
- **Option 3 (final cutover):** remove compatibility aliases/shims after the sunset gate is met.

Default order is **Option 1 -> Option 2 -> Option 3**. Do not start Option 3 early unless an explicit product decision overrides this plan.

## Baseline Snapshot (2026-02-20)

Inventory commands (run from `talos/`):

```bash
rg -n -i --hidden --glob '!.git/**' --glob '!_out/**' \
  --glob '!CHANGELOG.md' --glob '!website/**' --glob '!**/*.md' \
  --glob '!**/testdata/**' --glob '!**/*.pb.go' --glob '!**/*.vtproto.go' \
  '\btalosctl\b'

rg -n -i --hidden --glob '!.git/**' --glob '!_out/**' \
  --glob '!CHANGELOG.md' --glob '!website/**' --glob '!**/*.md' \
  --glob '!**/testdata/**' --glob '!**/*.pb.go' --glob '!**/*.vtproto.go' \
  '\btalos\b'
```

Current results:

- `talosctl`: 40 files / 408 matches (largest area: `hack/chubo`, `Dockerfile`, `Makefile`).
- `talos`: 447 files / 2657 matches (largest area: `hack/chubo`, `pkg/machinery`, `internal/app`, `api/resource/definitions`).
- Latest follow-up (2026-02-21 after comment-only debrand sweeps + chuboctl-image/local-target primary wiring + chubo-first CNI/release/docs/source-bundle slices + chubo-primary OS image default switch + resource-definition proto namespace migration): `talosctl` 14 files / 281 matches, `talos` 283 files / 1413 matches.
- Latest follow-up (2026-02-22 after endpoint/key-path debranding + release/build metadata cleanup + chuboctl-primary Docker CLI stages): `talosctl` 14 files / 178 matches, `talos` 278 files / 1245 matches.
- Pathnames still containing `talosctl`: 2 files (`cmd/talosctl/main.go`, `cmd/talosctl/acompat/acompat.go`).

## Temporary Allowlist (Only if Needed)

- [ ] CLI compatibility entrypoint (`cmd/talosctl/main.go`) for one transition window.
- [ ] Legacy env var aliases (`TALOSCONFIG`, `TALOS_HOME`, `TALOS_EDITOR`) as read-only compatibility.
- [ ] Legacy kernel params (`talos.*`) as read aliases only.
- [ ] Legacy EFI entry discovery (`Talos-*.efi`) for upgrade/read compatibility.
- [ ] Legacy protobuf type URL/package compatibility readers during Phase 3 transition.

Any remaining `talos*` outside this allowlist is a bug.

## Phase 0: Guardrails

- [x] Add `hack/chubo/check-talos-refs.sh` with:
  - allowed-path allowlist file
  - forbidden-outside-allowlist checks for `talosctl|talos`
  - `--update-baseline` mode for intentional bulk changes
  (`talos`: `7d9868311`, `5b996d349`)
- [x] Wire it into `make chubo-guardrails` and CI.
  (`talos`: `7d9868311`)
- [x] Publish baseline artifacts under `docs/talos/audits/<date>/`.
  (`chubo`: `bab68fe`; `docs/talos/audits/2026-02-20/`)

## Phase 1: `talosctl` Source Namespace Removal

- [x] Move implementation from `cmd/talosctl/**` to `cmd/chuboctl/**`.
  (`talos`: `1506569d5`, `c98ced6a0`, `f52230876`, `cdda1c896`, `49fb6021c`, `20bfd8bb7`, `7f3564485`; `f52230876` migrates `cmd/chuboctl` from wrapper stubs to full local command/helper trees (`cmd/chuboctl/cmd/**`, `cmd/chuboctl/pkg/**`), renames the node command namespace to `nodes`, and removes `cmd/talosctl/pkg/*` imports from chuboctl code paths. `7f3564485` removes the legacy `cmd/talosctl/cmd/**` and `cmd/talosctl/pkg/**` trees and refreshes the refs baseline.)
  (guardrail follow-up: `talos`: `5e9b558d9` switches the chubo guardrails package compile probe from `./cmd/talosctl/cmd/talos` to `./cmd/chuboctl/cmd/nodes`.)
- [x] Rewrite imports from `.../cmd/talosctl/...` to `.../cmd/chuboctl/...`.
- [x] Keep `cmd/talosctl/main.go` as a thin compatibility shim only.
- [x] Rename build outputs/targets/scripts to chubo-first names (`_out/chuboctl-*`, Make targets, helper scripts).
  (`talos`: `94ebabc66`, `fa73194ec`; `fa73194ec` makes `chuboctl` Make targets primary, keeps `talosctl` targets as compatibility aliases, switches strict dependency guardrails to `cmd/chuboctl`, updates helper script build-flag naming to `GO_BUILDFLAGS_CHUBOCTL`, and refreshes `hack/chubo/talos-refs-baseline.txt`.)
  (follow-up: `talos`: `b304a0a52` makes Dockerfile CLI build stages chuboctl-primary (`cmd/chuboctl`, `GO_BUILDFLAGS_CHUBOCTL`, `/chuboctl-*` artifacts) and keeps `talosctl-*` stages as compatibility aliases that copy from chuboctl outputs; baseline refreshed.)
- [x] Re-run: `make unit-tests`, `make chubo-guardrails`, `sudo -n ./hack/chubo/e2e-core-qemu.sh --skip-build`.
  (latest validation for this slice at `talos`: `fa73194ec`: `make chuboctl-$(go env GOOS)-$(go env GOARCH)`, `make talosctl-$(go env GOOS)-$(go env GOARCH)`, `make unit-tests`, `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1`, `sudo -n ./hack/chubo/e2e-core-qemu.sh --skip-build` (PASS).)
  (follow-up debranding slice: `talos`: `4ca451d35` makes `image source-bundle` primary with legacy `talos-bundle` alias, `6f273775b` switches integration CLI config suite naming/flags to chubo-first forms, `ebd569123` renames integration base suite/config wiring to chubo-first identifiers (`ChuboSuite`, `ChuboconfigPath`, `ChuboctlPath`, `ChuboImage`) and updates API capability naming (`RunsOSKernel`), `dd0b0d50c` makes integration runner flags `chubo.*` primary while keeping legacy `talos.*` aliases, `e3afbad42` renames cluster-create internal option fields to chubo-first names (`ChuboVersion`/`ChuboImage`) across create flows, makers, and presets, `455b80ec0` renames cluster-create legacy alias constant identifiers to neutral names while keeping the same compatibility flag behavior, `eb1558263` does the same neutral-identifier migration for `mgmt gen` legacy aliases/output-type alias constants, and `5e5fbd88d` renames `cmd/chuboctl/cmd/nodes/support.go` helper identifiers from talos-specific to OS-neutral naming.)

## Phase 2: Runtime + Config Debranding (Dual-Read Compatibility)

- [ ] Introduce chubo-primary constants for env vars, paths, kernel params, EFI labels, and user-facing strings.
  (in progress: `talos`: `f31c7cfcb` [client config state-dir helper is now `GetChuboDirectory` with legacy alias retained; `cmd/chuboctl` global args now use `Chuboconfig`; local E2E fixture scripts use `CHUBOCONFIG_FILE` and write `chuboconfig` workdir files])
  (in progress: `talos`: `fa087c9ac` [cluster-create now uses `--chuboconfig-destination` as primary with hidden legacy `--talosconfig-destination`/`--talosconfig` aliases; `gen config` default output type/file is now `chuboconfig` while keeping legacy `talosconfig` output type as accepted alias; local E2E scripts now request `-t chuboconfig`])
  (in progress: `talos`: `4788c6895` [makes `gen secrets` use `--chubo-version` as the primary flag with hidden legacy `--talos-version` alias; renames remaining local `talosconfig` variable/type names in `cmd/chuboctl`/machined helpers and refreshes talos-refs baseline])
  (in progress: `talos`: `5be6077b5` [adds chubo-primary EFI/boot-entry constants (`Chubo-` UKI prefix and `Chubo Linux UKI` description) while keeping legacy Talos constants; uses constants for `CHUBO_EDITOR`/`TALOS_EDITOR` lookup in `cmd/chuboctl` editor flow])
  (in progress: `talos`: `9cc94bb62` [makes `--chubo-version` primary for `gen config` and local `cluster create` flows while keeping hidden legacy `--talos-version`; hides legacy `--talosconfig` on `chuboctl` root; switches maintenance/runtime remediation hints from `talosctl` to `chuboctl`])
  (in progress: `talos`: `755fe9553` [adds chubo-primary config helper aliases (`Bundle.ChuboConfig`, `provision.WithChuboConfig`, `provision.WithChuboClient`, `provision.GetChuboAPIEndpoints`) while keeping legacy Talos aliases; updates cluster-create/gen/integration callers to chubo-primary helpers; makes existing-config bundle loading prefer `chuboconfig` before legacy `talosconfig`; refreshes talos-refs baseline])
  (in progress: `talos`: `c257410b7` [adds `generate.Input.Chuboconfig()` as primary with legacy `Talosconfig()` alias, switches bundle generation/tests/examples to the chubo-primary helper, and refreshes talos-refs baseline])
  (in progress: `talos`: `56f5d200b` [adds chubo-primary provision option fields (`Options.ChuboConfig`/`Options.ChuboClient`) with legacy `Talos*` fallbacks and switches provision access adapter wiring to the chubo-primary effective selectors])
  (in progress: `talos`: `9a7fd717d` [switches local airgapped certificate identity strings from `talos.dev` to `chubo.dev` in `cmd/chuboctl` helper generation and refreshes talos-refs baseline])
  (in progress: `talos`: `20826d9a6` [debrands residual Talos wording in `pkg/machinery/config` and `pkg/provision` package/function comments plus generation example text, and refreshes talos-refs baseline])
  (in progress: `talos`: `f8c98956b` [switches docker provisioner runtime labels to chubo-primary (`chubo.owned`, `chubo.cluster.name`, `chubo.type`) with legacy Talos-label fallback for list/reflect paths, renames mapped API port field to OS-neutral naming, and refreshes talos-refs baseline])
  (in progress: `talos`: `037f73ce6` [debrands remaining request/state and qemu wording in provision paths (state error hint now `chuboctl`, neutral qemu comments/temp-dir prefix), and refreshes talos-refs baseline])
  (in progress: `talos`: `216078f7f` [adds chubo-primary version contract constants (`ChuboVersion*`), keeps `TalosVersion*` aliases for compatibility, migrates machinery/config callers/tests to chubo-primary names, and refreshes talos-refs baseline])
  (in progress: `talos`: `6221c3e47` [adds chubo-primary provisioner endpoint hook (`GetChuboAPIEndpoints`) with legacy `GetTalosAPIEndpoints` wrappers, updates cluster config-provider wiring to chubo-primary fields with Talos fallback, and refreshes talos-refs baseline])
  (in progress: `talos`: `67263cf7d` [adds chubo-primary OS API certificate TTL constant (`ChuboAPIDefaultCertificateValidityDuration`), keeps legacy Talos alias, updates `chuboctl config new --crt-ttl` default and secrets client-certificate generation callsite, and refreshes talos-refs baseline])
  (in progress: `talos`: `9b5afcb7b` [adds chubo-primary support-bundle client option surface (`bundle.Options.ChuboClient`, `bundle.WithChuboClient`, `bundle.Options.EffectiveClient()`), keeps legacy Talos aliases, and migrates bundle collectors plus `cmd/chuboctl/cmd/nodes/support.go` callsites to chubo-primary accessors])
  (in progress: `talos`: `a5e1d9679` [makes provision/support option setters chubo-primary (`WithChuboConfig`, `WithChuboClient`) without mutating legacy Talos fields; keeps legacy setters as compatibility aliases that still hydrate chubo-primary state])
  (in progress: `talos`: `3f818112d` [makes `GetChuboAPIEndpoints` mandatory on `provision.Provisioner`, removes legacy `GetTalosAPIEndpoints` provisioner methods/wrappers, and keeps cluster-create maker coverage green])
  (in progress: `talos`: `567db3d19` [adds chubo-primary API helpers for generated client config payloads (`GetChuboconfig`/`SetChuboconfig`) and client certificate issuance (`GenerateChuboAPIClientCertificate*`), then migrates nodes-config generation and machined server callsites off talos-named helpers while keeping legacy aliases])
  (in progress: `talos`: `336c9f985` [makes `pkg/machinery/config/bundle.Bundle` use `ChuboCfg` as the primary config field, keeps `TalosCfg` as a legacy alias field/fallback, and routes bundle reads/writes through chubo-primary storage])
  (in progress: `talos`: `a35a61fe4` [switches config/secrets generation tests to the chubo-primary API client certificate helper (`GenerateChuboAPIClientCertificate`) instead of the legacy Talos alias])
  (in progress: `talos`: `1607c848d` [refactors `pkg/provision/options.Options` to isolate legacy alias hydration/fallback in dedicated helper paths (`setLegacyTalosConfig`/`setLegacyTalosClient`) and makes `pkg/cluster.ConfigClientProvider` fallback semantics explicitly compatibility-only while keeping external alias fields intact])
  (in progress: `talos`: `8bfa76f83` [switches client-side runtime identity strings to chubo-primary values (`runtime` metadata and SideroV1 `ClientName`), updates empty-config error wording to `chubo`, and refreshes the talos refs baseline])
  (in progress: `talos`: `4bf4adf68` [debrands remaining Talos wording in core `pkg/machinery/client`, `pkg/httpdefaults`, and `pkg/machinery/kernel` comments/docs strings and refreshes the talos refs baseline])
  (in progress: `talos`: `f31aa6387` [debrands machine lifecycle/user-facing messaging in config acquire + sequencer flows, updates matching acquire controller tests, removes residual Talos wording from imager quirks/constants/meta comments/errors, and refreshes the talos refs baseline])
  (in progress: `talos`: `cb45c6b10` [renames internal META ADV state/import identifiers to chubo-neutral names (`advState`/`resourceState`, `metaadv`) while keeping on-disk compatibility behavior unchanged, and refreshes the talos refs baseline])
  (in progress: `talos`: `dd4fed54b` [renames META ADV implementation package path from `internal/pkg/meta/internal/adv/talos` to `.../adv/chubo`, updates package names/imports/tests accordingly, and refreshes the talos refs baseline])
  (in progress: `talos`: `e9cdb1313` [makes compatibility version APIs chubo-primary (`ChuboVersion`/`ParseChuboVersion`) while keeping Talos aliases, migrates installer preflight/errata callsites and wording to chubo-first naming, and refreshes the talos refs baseline])
  (in progress: `talos`: `67dae1141` [makes extension validation APIs chubo-primary (`WithChuboVersion`/`ValidationOptions.ChuboVersion`) while keeping Talos aliases, migrates imager/tests to chubo naming, and refreshes the talos refs baseline])
  (in progress: `talos`: `e8a21a96b` [adds chubo-first extension compatibility constraint support (`compatibility.chubo`) while retaining legacy `compatibility.talos` parsing via a legacy alias field, and refreshes the talos refs baseline])
  (in progress: `talos`: `43d2616f2` [debrands remaining Talos wording in installer/extensions comments to chubo-first messaging and refreshes the talos refs baseline])
  (in progress: `talos`: `4899e25a1` [adds `hack/chubo/debrand-mechanical.sh` for low-risk comment-only codemod passes and applies it to server/platforms/role/client comment slices, then refreshes talos refs baseline])
  (in progress: `talos`: `f643f6418` [makes Chubo image constants primary in `pkg/images` (`DefaultChuboImage*`, `DefaultChuboctlAllImageRepository`) while retaining Talos aliases/legacy repo paths, migrates cluster-create/integration callsites, and refreshes the talos refs baseline])
  (in progress: `talos`: `cb406188c` [debrands generated protobuf comment surfaces in machinery API stubs])
  (in progress: `talos`: `0f72f1938` [debrands runtime/tooling comment surfaces across `internal/*`, `pkg/cluster`, `pkg/imager`, and qemu provisioner comments])
  (in progress: `talos`: `8112e74cb` [debrands machinery comment surfaces across compatibility/config/resources packages])
  (in progress: `talos`: `f6e212211` [refreshes `hack/chubo/talos-refs-baseline.txt` after the comment sweep])
  (in progress: `talos`: `c4c383c10` [makes chuboctl image outputs/repositories primary (`registry-chuboctl*`, chuboctl docs-build stage, `pkg/images` default repo) while keeping Talos compatibility aliases and refreshing talos refs baseline])
  (in progress: `talos`: `6c12535d5` [makes local `chuboctl*` artifact targets first-class in Makefile/Dockerfile while keeping `talosctl*` compatibility targets and refreshing talos refs baseline])
  (in progress: `talos`: `8e8b93d49` [makes `chuboctl-cni-bundle` the primary local artifact output and default integration URL, keeps `talosctl-cni-bundle` as a generated compatibility alias tarball/stage, switches release-note image list generation to `chuboctl image source-bundle`, and refreshes talos refs baseline])
  (in progress: `talos`: `76e462cbe` [debrands release-note copy in `hack/release.toml` from talosctl/talos-bundle to chuboctl/source-bundle wording and refreshes talos refs baseline])
  (in progress: `talos`: `19ab500e9` [debrands remaining `talosctl` wording in machine proto comments and Ethernet feature config docs/schemas to `chuboctl`, then refreshes talos refs baseline])
  (in progress: `talos`: `8f8965093` [switches integration CLI source-bundle fixture expectation from `talosctl-all` to `chuboctl-all`])
  (in progress: `talos`: `9da8c7d88` [makes `pkg/images.ListSourcesFor` and `cmd/chuboctl image source-bundle` output use `Chubo`/`ChuboctlAll` primary names and drops residual `sources.Talos*` callsites, then refreshes talos refs baseline])
  (in progress: `talos`: `8f76f9235` [switches local E2E scripts to chubo-first root/path variable naming (`CHUBO_ROOT`) and chubo-first docker fallback wording while retaining legacy `TALOS*` environment aliases; refreshes talos refs baseline])
  (in progress: `talos`: `532cc31d5` [switches Chubo OS image defaults from `/talos` to `/chubo` across `pkg/images`, Makefile image flows (`all`/`push`/`image-list`/`e2e-*`/reproducibility), Dockerfile stage aliases (`chubo` primary, `talos` compatibility), and integration source-bundle expectations while keeping legacy `talos` target/env compatibility aliases; refreshes talos refs baseline])
  (in progress: `talos`: `c5deb547c` [makes SideroV1 key lookup chubo-first with legacy XDG talos-key fallback, adds env-indirection for discovery and image-factory endpoints (`CHUBO_*` primary with legacy `TALOS_*` aliases), debrands `/etc/os-release` HOME_URL to `chubo.dev`, and refreshes `hack/chubo/talos-refs-baseline.txt`])
- [ ] Keep talos variants as parse aliases only (do not emit talos names in new output).
  (in progress: `talos`: `fd9226f23` [switches default build/display identity from Talos to Chubo by setting `Makefile NAME ?= Chubo`, changing embedded default `pkg/machinery/gendata/data/name` to `Chubo`, regenerating `pkg/machinery/version/os-release` to `NAME=Chubo`/`ID=chubo`, and widening integration CLI read assertion to accept both `ID=chubo` and legacy `ID=talos`])
- [ ] Rename runtime metadata labels/domains where safe (`*.talos.dev` -> `*.chubo.dev`) with compatibility readers where required.
  (in progress: `talos`: `5be6077b5` [switches runtime `Diagnostic` and `Version` resources to `*.chubo.dev` types, keeps `*.talos.dev` as compatibility aliases, and dual-registers protobuf dynamic types for old/new names])
  (in progress: `talos`: `3dd794954` [switches additional core runtime resources (`BootedEntry`, `OOMAction`, `PlatformMetadata`, `SBOMItem`, `SecurityState`) to `*.chubo.dev` types with legacy `*.talos.dev` aliases and dual dynamic registration])
  (in progress: `talos`: `a1a1e501c` [switches additional runtime spec/status resources (`KernelParamSpec`/`KernelParamDefaultSpec`/`KernelParamStatus`, `MountStatus`, `MaintenanceServiceConfig`, `DevicesStatus`, `WatchdogTimerConfig`, `KernelCmdline`, `KernelModuleSpec`) to `*.chubo.dev` with legacy aliases and dual dynamic registration])
  (in progress: `talos`: `613688c09` [completes migration of remaining runtime resources (`Environment`, `EventSinkConfig`, `Extension*`, `KmsgLogConfig`, `LoadedKernelModule`, `Machine*`, `MaintenanceServiceRequest`, `Meta*`, `UniqueMachineToken`, `WatchdogTimerStatus`) to `*.chubo.dev` with legacy aliases and dual dynamic registration])
- [x] Keep upgrade-safe bootloader handling: read old `Talos-*` entries, write new `Chubo-*` entries.
  (`talos`: `5be6077b5`)

## Phase 3: API/Proto Namespace Migration (Major Slice)

- [x] Define versioned API migration (`v1` compatibility + `v2` chubo namespace) before code changes in `docs/talos/api-proto-v2-migration.md`.
- [x] Capture migration blast-radius audit (proto package inventory + structprotogen namespace dependencies + generated stub scope) under `docs/talos/audits/2026-02-21/`.
- [x] Land structprotogen namespace override scaffold (`--resource-namespace`) while keeping the legacy default (`talos.resource.definitions`) for compatibility-first rollout.
  (`talos`: `23a6bb5c9`; validated with `go test ./tools/structprotogen/...` and `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1`)
- [x] Migrate `api/resource/definitions/*.proto` package names from `talos.*` to `chubo.*`.
  (`talos`: `aefddf03a`; validated with `go test ./pkg/machinery/api/resource/definitions/...`)
- [x] Regenerate machinery/API stubs and update server/client bindings.
  (`talos`: `aefddf03a`; validated with `go test ./pkg/machinery/resources/...`, `go test ./pkg/machinery/proto ./pkg/machinery/client`, and `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1`)
- [ ] Provide compatibility behavior for older clients/nodes during transition.
  (in progress: `talos`: `f6bd91314` adds dual-prefix runtime event decoding (`talos/runtime/*` + `chubo/runtime/*`) in `pkg/machinery/client`; `talos`: `b52c82e04` switches runtime event emission to `chubo/runtime/*` and adds runtime event TypeURL unit coverage)
- [ ] Validate compatibility and upgrade/rollback in QEMU fixtures.
  (in progress: validated local lane health on 2026-02-21 with `sudo -n ./hack/chubo/e2e-core-qemu.sh --skip-build` -> PASS (`/tmp/chubo-core-work-16543`); 2026-02-23 mixed-version core PASS with old fork image `localhost:5001/chubo/installer:old-1070f9e22` (`v1.13.0-alpha.1-398-g1070f9e22 -> v1.13.0-alpha.1-399-g028343493-dirty -> v1.13.0-alpha.1-398-g1070f9e22`, artifacts under `/tmp/chubo-core-work-25601`); 2026-02-23 mixed cluster PASS for control-plane coverage with `WORKER_COUNT=0` (artifacts under `/tmp/chubo-cluster-work-31614`); remaining gap: mixed worker path with old installer fails install decode on `spec.modules.chubo.nomad.networkInterface`, and mixed helper-bundles remains blocked until an old arm64 installer artifact exists)

Phase 3 gating decision (February 23, 2026):
- Keep the Phase 3 compatibility gate focused on mixed core + mixed cluster control-plane coverage.
- Treat mixed worker coverage from `v1.13.0-alpha.1-398-g1070f9e22` and mixed helper-bundles arm64 as deferred/non-blocking follow-ups.

## Phase 4: External Surface Cleanup

- Active burn-down order for this phase:
  - endpoint/default source cleanup (`talos.dev` -> chubo-owned or indirection),
  - release/build metadata wording cleanup,
  - docs/runbook regeneration from chubo-first binaries.
- [ ] Replace remaining `talos.dev` endpoints/URLs in code paths (factory/discovery/docs links) with chubo-owned endpoints or config indirection.
  (in progress: `talos`: `c5deb547c`; validated with `go test ./cmd/chuboctl/cmd/constants ./hack/cloud-image-uploader ./pkg/machinery/constants ./pkg/machinery/version ./pkg/machinery/client ./pkg/machinery/config/types/v1alpha1` and `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1`)
  (in progress: `talos`: `1229b0405` [adds chubo-primary factory-proxy env indirection in `hack/start-registry-proxies.sh` (`CHUBO_IMAGE_FACTORY_URL` with legacy `TALOS_IMAGE_FACTORY_URL` fallback) and refreshes `hack/chubo/talos-refs-baseline.txt`])
  (in progress: `talos`: `6e0198a5f` [debrands remaining active endpoint/schema defaults and generated docs/examples to chubo-first URLs (`discovery.chubo.dev`, `factory.chubo.dev`, `chubo.dev/v1.13/schemas/...`) and refreshes `hack/chubo/talos-refs-baseline.txt`], with validation `./hack/chubo/check-talos-refs.sh --update-baseline` and `make chubo-guardrails CHUBO_GUARDRAILS_SKIP_BUILD=1`)
- [ ] Remove residual Talos wording from release tooling, build metadata, and installer UX.
  (in progress: `talos`: `1229b0405` [debrands release note metadata and content wording in `hack/release.toml` (`project_name`, `github_repo`, doc links, and user-facing Talos wording -> Chubo)])
  (in progress: `talos`: `fd9226f23` [debrands Docker build metadata output from `_out/talos-metadata` to `_out/chubo-metadata` and updates Dockerfile commentary to Chubo wording])
  (in progress: `talos`: `da8fb9e21` [switches Docker build cache namespace from `id=talos/.cache` to `id=chubo/.cache` across build/generate/test/lint/sbom stages and refreshes `hack/chubo/talos-refs-baseline.txt`])
  (in progress: `talos`: `b304a0a52` [switches Dockerfile CLI builder stages from `cmd/talosctl` to `cmd/chuboctl`, introduces chuboctl-first stage naming/artifacts, and keeps `talosctl-*` stage aliases for compatibility while reducing talosctl token surface])
- [ ] Rebuild docs from chubo-first binaries and update runbooks.

## Phase 5: Compatibility Sunset

- Target date: Friday, March 6, 2026 (conditional on all prior phases and E2E gates passing).
- [ ] Remove `talosctl` shim.
- [ ] Remove `TALOS*` env aliases and legacy talos output names.
- [ ] Remove legacy talos kernel-param aliases only after documented cutoff and upgrade coverage.
- [ ] Lock with guardrail expectation: zero `talos|talosctl` outside explicit historical migration docs.

## Exit Criteria

- [ ] No `talos|talosctl` references in active source paths outside the temporary allowlist.
- [ ] CI blocks regressions automatically.
- [ ] All local lanes pass: unit tests, guardrails, core QEMU E2E, helper-bundles E2E, opengyoza quorum E2E, cluster E2E.
