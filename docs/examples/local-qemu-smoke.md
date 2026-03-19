# Local QEMU Smoke

This is the shortest realistic local validation path for the current alpha codebase.

## 1. Build The CLI

```sh
make chuboctl
```

Quick sanity check:

```sh
./_out/chuboctl-host --help
./_out/chuboctl-host cluster create --help
```

## 2. Pick The Right Local Lane

### macOS

Use the authoritative QEMU fixture:

```sh
sudo -n ./hack/chubo/e2e-core-qemu.sh
```

If it fails immediately on privilege or host-network requirements, stop there and fix the local host setup first.

### Linux

Use the same authoritative QEMU fixture first:

```sh
sudo -n ./hack/chubo/e2e-core-qemu.sh
```

You can experiment with Docker-based paths later, but QEMU is still the better validation path for install/lifecycle behavior.

## 3. Follow The Run

While the fixture is running:

```sh
sudo tail -f /tmp/chubo-*-e2e-state/*/*.log
```

## 4. What This Proves

This lane is the repo's main local proof that:

- boot artifacts are usable
- install completes
- runtime API comes up
- helper bundles can be downloaded
- lifecycle actions remain viable in the local authoritative path

## 5. What To Read Next

- [../guides/local-development.md](../guides/local-development.md)
- [../guides/cluster-lifecycle.md](../guides/cluster-lifecycle.md)
- [helper-bundles.md](helper-bundles.md)
