# Chubo Migration Notes (Talos Compatibility Sunset)

This note defines the compatibility window while moving from Talos naming to Chubo naming.

## Primary Interface (use now)

- CLI: `chuboctl`
- Env vars: `CHUBOCONFIG`, `CHUBO_HOME`, `CHUBO_EDITOR`
- Prefixes: `chubo-*` targets, scripts, and artifacts

## Compatibility Interface (temporary)

The following remain supported only during the transition window:

- `talosctl` compatibility shim
- `TALOSCONFIG`, `TALOS_HOME`, `TALOS_EDITOR` aliases
- legacy `chuboos-*` wrappers/aliases

No new features should be added to compatibility aliases.

## Required Operator Changes

1. Replace `talosctl` with `chuboctl` in scripts/CI/jobs.
2. Replace `TALOS*` env vars with `CHUBO*`.
3. Rename local workflow scripts/targets to `chubo-*`.
4. Regenerate docs/help output from `chuboctl` and republish.

## Sunset Policy

- Sunset target version: `v1.14.0`
- Sunset target date: `2026-08-31`
- Rule: remove legacy wrappers at the first release that meets either target.

After sunset:

- `talosctl` shim removed
- `TALOS*` aliases removed
- `chuboos-*` wrappers removed

## Validation Checklist

- `rg -n -i 'talosctl|TALOSCONFIG|chuboos-' docs README.md`
- `chuboctl --help` and generated CLI docs contain only Chubo naming
- CI jobs use `chuboctl` and `CHUBO*` variables
