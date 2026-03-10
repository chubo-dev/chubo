---
description: ClusterInlineManifest struct describes inline bootstrap manifests for the user.
title: ClusterInlineManifest
---

<!-- markdownlint-disable -->










| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`name` |string |Name of the manifest.<br>Name should be unique. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
name: csi
{{< /highlight >}}</details> | |
|`contents` |string |Manifest contents as a string. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
contents: /etc/workload/auth
{{< /highlight >}}</details> | |






