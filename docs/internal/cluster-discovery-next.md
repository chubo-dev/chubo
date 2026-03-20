# Cluster Discovery Next

This document records the post-alpha follow-up for the inherited external discovery subsystem.

## Alpha Stance

For alpha, this subsystem can be skipped as a product feature.

- it is not part of the recommended first-run path
- cluster discovery is disabled by default in the alpha CLI/config surface
- current QEMU alpha fixtures already commonly run with `--with-cluster-discovery=false`
- workload-native access does not depend on it

What alpha still needs is narrower:

- local QEMU validation
- direct OS API access through `chuboctl`
- helper-bundle access for `wonton`, `gyoza`, and `bao`

## Why The Current Design Is Not Chubo-Native

The current path is still structurally inherited from Talos:

1. external discovery registry
2. raw affiliates
3. merged affiliates
4. members
5. discovery-derived control-plane endpoints

Problems with the current shape:

- default endpoint points at `https://discovery.chubo.dev/`, which is not a real supported public service today
- resource types and naming still use `*.cluster.talos.dev`
- the model still revolves around `controlplane` compatibility concepts
- worker certificate generation currently depends on discovery-derived control-plane endpoints
- none of this is OpenWonton/OpenGyoza-specific; it is node-membership plumbing inherited from the Talos era

## What Chubo Actually Needs

Chubo still needs three concrete capabilities:

1. a way for workers to find OS API and trustd endpoints
2. a way to represent cluster members inside the OS API
3. a way for local DNS and operator UX to resolve node names and addresses

Chubo does not need a broken public-hosted default to provide those capabilities.

## Proposed Replacement Direction

### 1. Split Endpoint Discovery From Membership Discovery

The current subsystem mixes two different jobs:

- discovering who is in the cluster
- discovering where workers should reach trustd/apid

Those should be separated.

Recommended direction:

- treat OS API/trustd endpoint publication as explicit control-plane state
- derive worker join targets from provisioned/configured control-plane endpoints, not from an external public registry

### 2. Make OS API Endpoint Publication Explicit

Workers currently wait on `ControlPlaneEndpoint` resources. Keep that resource class or an equivalent one, but change how it is produced.

Preferred sources:

- provisioner-generated endpoint state for local clusters
- machine config / cluster config declared endpoint state
- local control-plane self-publication into OS API state

Do not make worker bootstrap depend on an external hosted registry by default.

### 3. Keep Membership Inside The OS API

For alpha and near-term Chubo, cluster membership should come from OS API-visible state rather than a public discovery registry.

Candidate shape:

- each node publishes its own node identity and routed addresses into local OS API state
- control-plane nodes aggregate that state into cluster member resources
- operator UX, local DNS, and node completion read that member view

This keeps node membership as an internal OS concern.

### 4. Make Any External Registry Explicit And Optional

If Chubo later needs WAN or unmanaged-node discovery, keep that as an opt-in extension.

That future layer should:

- not be the default
- not block single-node or control-plane-only alpha workflows
- not be required for local provisioning or worker bootstrap unless explicitly enabled

### 5. Finish Naming Cleanup As Part Of The Rewrite

When this subsystem is revisited, clean up:

- `*.cluster.talos.dev` resource types
- stale `controlplane`-era naming where it no longer expresses the Chubo model
- tests and fixtures that still assume Talos/Kubernetes-era semantics

## Suggested Exit Criteria

Do not re-enable discovery by default until all of the following are true:

1. workers can bootstrap without any public discovery host
2. `ControlPlaneEndpoint` resources are produced from Chubo-native sources
3. member resources are available from OS API state without the external registry
4. local DNS and operator completions still work in the supported alpha/beta paths
5. docs describe the feature honestly as either opt-in or supported

## Near-Term Recommendation

Until the replacement exists:

- keep external cluster discovery out of the alpha story
- prefer `--with-cluster-discovery=false` in supported local flows
- document worker-bearing discovery-dependent topologies as follow-up work, not as supported alpha behavior
