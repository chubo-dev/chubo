# Documentation

This folder is the public docs entry point for the repository.

If you are new to the repo, stay in the pages linked below first. The `docs/talos/` tree is internal engineering material for the deep fork and should not be treated as the default reading path.

## Start Here

- [quickstart.md](quickstart.md): fastest local paths for trying Chubo OS
- [workload-access.md](workload-access.md): how helper bundles map to `wonton`, `gyoza`, and `bao`/`vault`
- [examples/README.md](examples/README.md): concrete alpha walkthroughs
- [guides/README.md](guides/README.md): what is currently possible in the alpha codebase

## Reference

- [reference/cli.md](reference/cli.md): generated `chuboctl` command reference
- local CLI help remains authoritative for ad-hoc inspection:
  - `./_out/chuboctl-host --help`
  - `./_out/chuboctl-host <command> --help`
  - `./_out/chuboctl-host gen --help`

## Internal Docs

- [internal/README.md](internal/README.md): engineering notes, migration plans, and execution checklists

The internal docs are still important, but they are not newcomer docs. Use them when you need implementation background, rename history, or execution notes for the deep fork.
