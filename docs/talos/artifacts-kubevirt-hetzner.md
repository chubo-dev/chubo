# Artifact publishing and consumption (KubeVirt + Hetzner)

This repo publishes release artifacts via `.github/workflows/release-artifacts.yaml`.

Primary artifacts for homelab/KubeVirt and Hetzner testing:

- `chuboctl-linux-amd64`
- `chuboctl-linux-arm64`
- `chuboctl-darwin-amd64`
- `chuboctl-darwin-arm64`
- `metal-*.iso`
- `*nocloud*.raw*`
- `*hcloud*.raw*`
- `SHA256SUMS`

## Produce artifacts

Automatic:

- Push a tag `v*` (for example `v1.13.0-alpha.1`).
- Workflow builds artifacts under `_out/`, uploads them as workflow artifacts, and attaches them to the GitHub Release for the tag.

Manual:

- Run `release-artifacts` via `workflow_dispatch`.
- Optional input `tag` sets the image/artifact tag (defaults to `sha-<short>` when not provided).

## Verify downloaded artifacts

```bash
sha256sum -c SHA256SUMS
```

## KubeVirt (local homelab) flow

Use the `nocloud` disk image.

1. Download release assets:

```bash
TAG=v1.13.0-alpha.1
BASE_URL="https://github.com/chubo-dev/chubo/releases/download/${TAG}"
curl -L -o nocloud-amd64.raw.zst "${BASE_URL}/nocloud-amd64.raw.zst"
curl -L -o SHA256SUMS "${BASE_URL}/SHA256SUMS"
sha256sum -c SHA256SUMS --ignore-missing
```

2. Decompress:

```bash
unzstd -f nocloud-amd64.raw.zst
```

3. Upload into CDI as a `DataVolume`:

```bash
virtctl image-upload dv chubo-nocloud-${TAG} \
  --namespace default \
  --storage-class <your-storage-class> \
  --size 20Gi \
  --image-path ./nocloud-amd64.raw \
  --access-mode ReadWriteOnce \
  --volume-mode Block
```

4. Reference that `DataVolume` from your VM spec and boot.

## Hetzner flow

Use the `hcloud` disk image.

1. Download and verify:

```bash
TAG=v1.13.0-alpha.1
BASE_URL="https://github.com/chubo-dev/chubo/releases/download/${TAG}"
curl -L -o hcloud-amd64.raw.zst "${BASE_URL}/hcloud-amd64.raw.zst"
curl -L -o SHA256SUMS "${BASE_URL}/SHA256SUMS"
sha256sum -c SHA256SUMS --ignore-missing
```

2. Decompress:

```bash
unzstd -f hcloud-amd64.raw.zst
```

3. Import as a Hetzner snapshot (for example with `hcloud-upload-image`), then create servers from that snapshot.

## Notes

- Release artifacts are `chuboctl`-first. Do not rely on `talosctl` naming in automation.
- `hcloud` and `nocloud` artifacts are built with `PLATFORM=linux/amd64` in CI.

## Known-Good Hetzner Pin (2026-02-24)

For the current Hetzner Chubo lane in this workspace, the tested pin is:

- `hcloud_snapshot_id=361277929`
- `chubo_release_tag=hetzner-dev-20260224194658`
- `chubo_image_sha256=b79e2a5773eebb0f0a6d350c6c6b30dabc4a2a95d15b4d21084965bb349f17d6`

This snapshot ID is Hetzner project-scoped. If your project does not contain this
snapshot, import the matching `hcloud-amd64.raw.zst` and pin the new image ID with
the same tag/SHA pair.
