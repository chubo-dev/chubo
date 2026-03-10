---
description: ClusterNetworkConfig represents cluster networking configuration options.
title: ClusterNetworkConfig
---

<!-- markdownlint-disable -->










| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`dnsDomain` |string |The DNS domain used for cluster service discovery.<br>The default is `cluster.local` <details><summary>Show example(s)</summary>{{< highlight yaml >}}
dnsDomain: cluster.local
{{< /highlight >}}</details> | |
|`serviceSubnets` |[]string |The service subnet CIDR. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
serviceSubnets:
    - 10.96.0.0/12
{{< /highlight >}}</details> | |






