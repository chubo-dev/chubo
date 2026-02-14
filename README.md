# Chubo OS (Talos Fork)

This repository is a Talos-derived OS distribution which targets running the Chubo stack:

- OpenWonton (Nomad)
- OpenGyoza (Consul)
- OpenBao (as a Nomad job)

Chubo OS is API-managed and intentionally "no shell":
- no SSH
- no console login
- day-2 operations happen via the OS API (mTLS), plus helper bundles to access workload-native APIs

Kubernetes/etcd are not part of the product surface (they are being removed from the repository in staged passes).

## Workspace Layout

In the `chubo-os` workspace, docs and control-plane tooling live in the sibling repo:
- `../chubo/docs/talos/deep-fork-plan.md`
- `../chubo/docs/talos/chubo-product-source-clean-plan.md`
- `../chubo/docs/dev/chubo-os-qemu-devloop.md`

## Build, Test, and Fast Iteration

From `talos/`:

- Unit tests: `make unit-tests`
- Guardrails (k8s-less deps + rootfs + CLI surface): `make chubo-guardrails`
- QEMU core E2E (root): `sudo -n ./hack/chubo/e2e-core-qemu.sh`
- Helper bundles smoke (root): `sudo -n ./hack/chubo/e2e-helper-bundles-qemu.sh`
- Opengyoza quorum fixture (root): `sudo -n ./hack/chubo/e2e-opengyoza-quorum-qemu.sh`

To monitor a QEMU fixture run, tail the per-node serial log in the printed state dir, e.g.:

```sh
sudo tail -f /tmp/chubo-*-e2e-state/*/*.log
```

## CI

GitHub Actions workflows are intentionally minimal and Chubo-focused:
- `.github/workflows/ci.yaml`
