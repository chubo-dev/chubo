---
description: VolumeMountConfig struct describes extra volume mount for the static pods.
title: VolumeMountConfig
---

<!-- markdownlint-disable -->










| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`hostPath` |string |Path on the host. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
hostPath: /var/lib/auth
{{< /highlight >}}</details> | |
|`mountPath` |string |Path in the container. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
mountPath: /etc/workload/auth
{{< /highlight >}}</details> | |
|`readonly` |bool |Mount the volume read only. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
readonly: true
{{< /highlight >}}</details> | |






