# Local Development

This repo is optimized around local iteration with `chuboctl`, targeted `go test`, and QEMU-based fixture runs.

## Fastest Loops

Choose the smallest loop that exercises the code you changed:

- config/rendering changes: targeted `go test` plus generated config inspection
- CLI/help/docs changes: `make chuboctl` plus `./_out/chuboctl-host --help`
- service/runtime controller changes: local QEMU or the smallest relevant E2E lane
- cluster/bootstrap behavior: local multi-node QEMU before broader fixtures

## Main Local Commands

Build the CLI:

```sh
make chuboctl
```

Run unit tests:

```sh
make unit-tests
```

Run guardrails:

```sh
make chubo-guardrails
```

Run a fast single-node QEMU dev loop:

```sh
./hack/qemu/chubo-qemu.sh
```

Run the main authoritative local QEMU fixture:

```sh
sudo -n ./hack/chubo/e2e-core-qemu.sh
```

## Host Expectations

### macOS

- QEMU lanes are the authoritative validation path
- local Docker/Colima can help with build artifacts
- the Docker cluster fallback is still non-authoritative on macOS
- some root-run fixtures require vmnet/root privileges

Suggested alpha sequence:

```sh
make chuboctl
./_out/chuboctl-host --help
sudo -n ./hack/chubo/e2e-core-qemu.sh
```

### Linux

- Docker fallback is more viable
- QEMU is still the most representative validation path for local cluster flows

Suggested alpha sequence:

```sh
make chuboctl
./_out/chuboctl-host --help
sudo -n ./hack/chubo/e2e-core-qemu.sh
```

## What To Use For Reference

For alpha, use:

- [../reference/cli.md](../reference/cli.md)
- local `--help` output from `./_out/chuboctl-host`
- [../examples/local-qemu-smoke.md](../examples/local-qemu-smoke.md)

Do not treat the website tree as the primary documentation path yet.
