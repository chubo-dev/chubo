# Reference

This section contains generated and low-level reference material.

## CLI

- [cli.md](cli.md): generated `chuboctl` command reference

## Regeneration

Rebuild the local CLI docs with a freshly built binary:

```sh
go build -o /tmp/chuboctl-docs ./cmd/chuboctl
/tmp/chuboctl-docs docs docs/reference --cli
```

For alpha, this local reference is preferred over the website tree.
