# OS API Operations

`chuboctl` is the primary operator tool for the Chubo OS API.

The OS API is the control plane for node management. It is not a shell replacement layered over SSH.

## Configuration

Main config-related operations:

- `apply-config`
- `patch`
- `edit`
- `validate`
- `gen`
- `machineconfig gen`
- `machineconfig patch`
- `config` subcommands for client config management

This supports the main alpha workflows of:

- generating machine configs
- validating them
- applying them to nodes
- managing client contexts and endpoints

Suggested entry commands:

```sh
./_out/chuboctl-host gen --help
./_out/chuboctl-host machineconfig --help
./_out/chuboctl-host validate --help
./_out/chuboctl-host apply-config --help
```

## Node State And Resources

Main read/observe operations:

- `get`
- `logs`
- `events`
- `service`
- `version`
- `time`
- `memory`
- `processes`
- `mounts`
- `netstat`
- `dmesg`
- `cgroups`

Use these commands when you want to inspect the OS and its managed services through the API.

Suggested entry commands:

```sh
./_out/chuboctl-host get --help
./_out/chuboctl-host logs --help
./_out/chuboctl-host events --help
./_out/chuboctl-host service --help
```

## Data Access And Debugging

Additional operator/debugging surfaces:

- `read`
- `copy`
- `debug`
- `inspect`
- `support`
- `pcap`

These commands cover many node-level investigations without turning SSH into the primary operator path.

Suggested entry commands:

```sh
./_out/chuboctl-host debug --help
./_out/chuboctl-host inspect --help
./_out/chuboctl-host support --help
```

## Lifecycle

Main lifecycle actions:

- `reboot`
- `shutdown`
- `upgrade`
- `rollback`
- `reset`
- `wipe`

These should be treated as high-impact operations and validated in the smallest relevant loop before broad use.

## Workload Access Boundary

The OS API does not replace the workload APIs.

For OpenWonton, OpenGyoza, and OpenBao:

- use `chuboctl` to fetch the helper bundle
- use `wonton`, `gyoza`, and `bao`/`vault` to talk to the workload API itself

See [../workload-access.md](../workload-access.md).
