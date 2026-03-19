# Workload Access

Chubo does not proxy Nomad, Consul, or OpenBao commands for you.

Instead, `chuboctl` downloads helper bundles from the OS API, and you use those bundles with the workload-native CLIs:

- `wonton` for OpenWonton
- `gyoza` for OpenGyoza
- `bao` or `vault` for OpenBao

## What A Helper Bundle Contains

Each helper bundle is a `.tar.gz` archive extracted by `chuboctl` into a local directory.

The exact files differ slightly by service, but the bundle typically contains:

- an env file such as `nomad.env`, `consul.env`, or `openbao.env`
- an HCL config file such as `nomad.hcl`, `consul.hcl`, or `openbao.hcl`
- `ca.pem`
- `client.pem`
- `client-key.pem`
- `acl.token`
- a short `README`

Those files provide the API address, CA, client certificate, client key, and token needed to talk to the workload API directly.

## Why This Exists

The OS API is Chubo's remote control plane.

Workload APIs are separate surfaces with their own TLS and ACL requirements. The helper bundle gives you the right connection material without making you manually discover:

- which address to use
- which CA to trust
- which client certificate and key to present
- which token to send

## Default CLI Usage

These are the default pairings Chubo should document and expect operators to use:

- `chuboctl nomadconfig` -> `wonton`
- `chuboctl consulconfig` -> `gyoza`
- `chuboctl openbaoconfig` -> `bao` or `vault`

The command names are still `nomadconfig` and `consulconfig` because the emitted config/env material remains Nomad- and Consul-compatible for migration and compatibility reasons.

## Install The Native CLIs

### OpenWonton (`wonton`)

OpenWonton's upstream README says the fastest local trial is:

1. download a release
2. extract the archive
3. move `wonton` onto your `PATH`

Upstream also states that `wonton` is the primary CLI and `nomad` is only a compatibility shim.

Use:

- <https://github.com/openwonton/openwonton>
- <https://github.com/openwonton/openwonton/releases>

### OpenGyoza (`gyoza`)

OpenGyoza's upstream README says:

- the primary CLI is `gyoza`
- `consul` is a compatibility shim
- quick-start docs are still being finalized

The releases page currently publishes `gyoza_<version>_<os>_<arch>.zip` assets, so the practical install flow today is:

1. download the matching release archive
2. extract it
3. move `gyoza` onto your `PATH`

Use:

- <https://github.com/opengyoza/opengyoza>
- <https://github.com/opengyoza/opengyoza/releases>

### OpenBao (`bao` / `vault`)

Use whichever OpenBao-compatible CLI you standardize on locally. The bundle writes both `BAO_*` and `VAULT_*` environment variables for compatibility.

## Example Workflow

### OpenWonton

```sh
chuboctl nomadconfig ./helpers
cd ./helpers/nomadconfig
set -a
. ./nomad.env
set +a
wonton status
```

### OpenGyoza

```sh
chuboctl consulconfig ./helpers
cd ./helpers/consulconfig
set -a
. ./consul.env
set +a
gyoza members
```

### OpenBao

```sh
chuboctl openbaoconfig ./helpers
cd ./helpers/openbaoconfig
set -a
. ./openbao.env
set +a
bao status
```

## Current Naming Caveat

The current helper-command names are compatibility-first:

- `nomadconfig`
- `consulconfig`
- `openbaoconfig`

That does not mean the preferred operator CLIs are `nomad` and `consul`.

For Chubo documentation, the preferred operator path should be:

- `wonton` first
- `gyoza` first
- `bao` first where available
