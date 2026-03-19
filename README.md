# Chubo OS

Chubo OS is a Talos-derived, API-managed operating system for the Chubo stack:

- OpenWonton (Nomad)
- OpenGyoza (Consul)
- OpenBao

It is intentionally "no shell":

- no SSH
- no console login
- day-2 operations happen through the OS API over mTLS
- workload-native access happens through helper bundles for the native CLIs: `wonton`, `gyoza`, and `bao`/`vault`

Kubernetes and etcd are not part of the intended product surface.

## Status

This repository is still in an alpha, deep-fork stage.

- the OS and CLI are usable for local development and targeted QEMU validation
- the public docs and examples are still being built out
- many docs under [`docs/talos/`](docs/talos/) are internal migration and design notes, not newcomer guides

If you want a repo you can clone and understand quickly, start with the docs below rather than the internal planning material.

## Start Here

- [docs/README.md](docs/README.md): docs index
- [docs/quickstart.md](docs/quickstart.md): fastest local paths
- [docs/workload-access.md](docs/workload-access.md): helper bundles, native CLIs, and install notes
- [docs/examples/README.md](docs/examples/README.md): concrete alpha-phase walkthroughs
- [docs/reference/cli.md](docs/reference/cli.md): generated `chuboctl` command reference
- [docs/guides/README.md](docs/guides/README.md): hand-written capability and operator guides

## Fast Local Loops

Build the CLI:

```sh
make chuboctl
```

Run the authoritative local QEMU lane:

```sh
sudo -n ./hack/chubo/e2e-core-qemu.sh
```

Run the helper-bundle smoke lane:

```sh
sudo -n ./hack/chubo/e2e-helper-bundles-qemu.sh
```

Run the OpenGyoza quorum lane:

```sh
sudo -n ./hack/chubo/e2e-opengyoza-quorum-qemu.sh
```

For a faster, narrower inner loop, use:

```sh
./hack/qemu/chubo-qemu.sh
```

## Build And Test

Useful local targets from the repo root:

- `make help`
- `make unit-tests`
- `make chuboctl`
- `make chubo-guardrails`

On macOS, the QEMU lanes are the authoritative validation path. The Docker fallback is still non-authoritative there because host kernel features are missing.

## Repository Layout

- [`cmd/chuboctl/`](cmd/chuboctl/): primary CLI
- [`cmd/talosctl/`](cmd/talosctl/): compatibility shim during the rename wave
- [`internal/app/machined/`](internal/app/machined/): boot/runtime controllers and services
- [`internal/app/apid/`](internal/app/apid/): OS API server
- [`internal/app/trustd/`](internal/app/trustd/): trust and PKI services
- [`pkg/provision/`](pkg/provision/): local cluster provisioning
- [`hack/chubo/`](hack/chubo/): local E2E and validation scripts

## Current Gaps

The main missing pieces for external-readiness are:

- a shorter public overview
- a more polished 5-minute quickstart
- example machine configs and local scenarios
- clearer separation between public docs and internal migration notes

Those improvements now start under [`docs/`](docs/).
