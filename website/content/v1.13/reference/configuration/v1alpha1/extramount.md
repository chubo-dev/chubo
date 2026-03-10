---
description: ExtraMount wraps OCI Mount specification.
title: ExtraMount
---

<!-- markdownlint-disable -->










| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`destination` |string |Destination is the absolute path where the mount will be placed in the container.  | |
|`type` |string |Type specifies the mount kind.  | |
|`source` |string |Source specifies the source path of the mount.  | |
|`options` |[]string |Options are fstab style mount options.  | |
|`uidMappings` |<a href="#ExtraMount.uidMappings.">[]LinuxIDMapping</a> |UID/GID mappings used for changing file owners w/o calling chown, fs should support it.<br><br>Every mount point could have its own mapping.  | |
|`gidMappings` |<a href="#ExtraMount.gidMappings.">[]LinuxIDMapping</a> |UID/GID mappings used for changing file owners w/o calling chown, fs should support it.<br><br>Every mount point could have its own mapping.  | |




## uidMappings[] {#ExtraMount.uidMappings.}

LinuxIDMapping represents the Linux ID mapping.




| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`containerID` |uint32 |ContainerID is the starting UID/GID in the container.  | |
|`hostID` |uint32 |HostID is the starting UID/GID on the host to be mapped to 'ContainerID'.  | |
|`size` |uint32 |Size is the number of IDs to be mapped.  | |






## gidMappings[] {#ExtraMount.gidMappings.}

LinuxIDMapping represents the Linux ID mapping.




| Field | Type | Description | Value(s) |
|-------|------|-------------|----------|
|`containerID` |uint32 |ContainerID is the starting UID/GID in the container.  | |
|`hostID` |uint32 |HostID is the starting UID/GID on the host to be mapped to 'ContainerID'.  | |
|`size` |uint32 |Size is the number of IDs to be mapped.  | |








