# Helper Bundles

This example shows the intended alpha operator path for workload access.

## Prerequisites

- a running node or local fixture
- a working `chuboconfig`
- `wonton` installed
- `gyoza` installed
- `bao` or `vault` installed

See [../workload-access.md](../workload-access.md) for install notes and background.

## 1. Download The Bundles

```sh
mkdir -p ./helpers
chuboctl nomadconfig ./helpers
chuboctl consulconfig ./helpers
chuboctl openbaoconfig ./helpers
```

After extraction you should have:

- `./helpers/nomadconfig/`
- `./helpers/consulconfig/`
- `./helpers/openbaoconfig/`

## 2. Use OpenWonton With `wonton`

```sh
cd ./helpers/nomadconfig
set -a
. ./nomad.env
set +a
wonton status
```

## 3. Use OpenGyoza With `gyoza`

```sh
cd ../consulconfig
set -a
. ./consul.env
set +a
gyoza members
```

## 4. Use OpenBao With `bao`

```sh
cd ../openbaoconfig
set -a
. ./openbao.env
set +a
bao status
```

## 5. What The Bundle Gives You

Each bundle provides the connection material the native CLI needs:

- API address
- CA certificate
- client certificate
- client key
- token

That is why the default operator flow is:

- `chuboctl` for OS API access
- `wonton`, `gyoza`, and `bao` for workload API access
