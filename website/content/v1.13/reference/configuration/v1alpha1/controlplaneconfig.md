---
description: ControlPlaneConfig represents the control plane configuration options.
title: ControlPlaneConfig
---

<!-- markdownlint-disable -->










| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`endpoint` |<a href="#ControlPlaneConfig.endpoint">Endpoint</a> |Endpoint is the canonical controlplane endpoint, which can be an IP address or a DNS hostname.<br>It is single-valued, and may optionally include a port number. <details><summary>Show example(s)</summary>{{< highlight yaml >}}
endpoint: https://1.2.3.4:6443
{{< /highlight >}}{{< highlight yaml >}}
endpoint: https://cluster1.internal:6443
{{< /highlight >}}</details> | |
|`localAPIServerPort` |int |The port that the API server listens on internally.<br>This may be different than the port portion listed in the endpoint field above.<br>The default is `6443`.  | |




## endpoint {#ControlPlaneConfig.endpoint}

Endpoint represents the endpoint URL parsed out of the machine config.



{{< highlight yaml >}}
endpoint: https://1.2.3.4:6443
{{< /highlight >}}

{{< highlight yaml >}}
endpoint: https://cluster1.internal:6443
{{< /highlight >}}

{{< highlight yaml >}}
endpoint: udp://127.0.0.1:12345
{{< /highlight >}}

{{< highlight yaml >}}
endpoint: tcp://1.2.3.4:12345
{{< /highlight >}}


| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|








