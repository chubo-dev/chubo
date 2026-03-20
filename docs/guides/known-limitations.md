# Known Limitations

This repo is still in alpha.

## Documentation

Current docs are improving, but still incomplete:

- examples are still limited in scope
- some areas are described at the command level more clearly than at the operator-workflow level
- the internal fork notes are still larger than the public docs surface

## Platform Reality

- QEMU is the authoritative local validation path
- Docker fallback is not a full substitute, especially on macOS
- some validation paths require root privileges and local host setup

## Product Surface

The intended product surface excludes Kubernetes and etcd, but the codebase is still a deep fork in progress. Some compatibility paths and legacy naming remain in place during the transition.

## Cluster Discovery

The inherited external discovery subsystem is not part of the supported alpha story.

- cluster discovery is disabled by default in the CLI/config surface used for alpha
- the existing discovery path is still structurally Talos-era: external registry -> affiliates -> members -> control-plane endpoints
- parts of that path still carry Talos naming and legacy control-plane assumptions
- a Chubo-native replacement is still work to do next, not a feature to rely on during alpha

For alpha, the supported focus is:

- local QEMU validation
- direct OS API operations through `chuboctl`
- workload-native access through helper bundles and `wonton` / `gyoza` / `bao`

## Public Presentation

The repo is not yet at a polished “clone this and understand everything in five minutes” stage.

What it does have today:

- a usable CLI
- local QEMU validation flows
- helper-bundle workflows for workload-native access
- machine config generation and validation
- OS lifecycle and support-bundle operations

What still needs work:

- richer examples
- more operator walkthroughs
- a cleaner publishable public docs site after alpha
- a Chubo-native replacement for the inherited external discovery subsystem
