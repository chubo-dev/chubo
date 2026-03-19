# Debugging And Support

The current codebase already exposes a useful debugging surface even without SSH.

## First Places To Look

For local fixture runs:

```sh
sudo tail -f /tmp/chubo-*-e2e-state/*/*.log
```

This is often the quickest path to understanding a failed local boot or install transition.

## API-Level Observation

Useful commands:

- `logs`
- `events`
- `service`
- `dmesg`
- `inspect`
- `support`

These let you observe controller behavior, service failures, and node state directly through the OS API.

## Debug Containers

`chuboctl debug` can run a debug container from:

- a local image archive
- a remote image reference

This is useful when you need focused tooling on a running node without turning shell access into the primary operator model.

## Support Bundles

`chuboctl support` generates a support archive containing:

- kernel logs
- internal service logs
- COSI resources without secrets
- runtime state graph
- processes
- IO pressure snapshot
- mounts
- PCI device info
- OS version

This is the main artifact to collect when a failure needs to be analyzed outside an interactive session.

## Narrow Before You Escalate

Follow the smallest authoritative loop:

- config issue: validate generated config first
- node/controller issue: targeted `go test` plus local QEMU
- cluster/bootstrap issue: smallest relevant local fixture

Avoid rerunning broad fixtures until a narrower hypothesis has been falsified.
