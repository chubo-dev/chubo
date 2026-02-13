// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package acl provides deterministic ACL token derivation for Chubo-managed services.
//
// Tokens are derived from the OS trust token so the cluster can bootstrap without an
// extra secret distribution mechanism.
package acl

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// WorkloadToken returns a deterministic UUIDv4-formatted token for the given name.
//
// The same trust token + name always yields the same token. This is intended for
// bootstrapping OpenWonton/OpenGyoza ACLs and for generating helper bundles.
func WorkloadToken(trustToken, name string) string {
	trustToken = strings.TrimSpace(trustToken)
	name = strings.TrimSpace(name)

	if trustToken == "" || name == "" {
		return ""
	}

	sum := hmac.New(sha256.New, []byte(trustToken))
	_, _ = sum.Write([]byte("chubo-workload-acl:" + name))

	raw := sum.Sum(nil)[:16]

	// Force RFC 4122 variant + v4 format for broad compatibility with Nomad/Consul ACL APIs.
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80

	return formatUUID(raw)
}

func formatUUID(raw []byte) string {
	// 16 bytes -> 36 chars (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
	var out [36]byte

	hex.Encode(out[0:8], raw[0:4])
	out[8] = '-'
	hex.Encode(out[9:13], raw[4:6])
	out[13] = '-'
	hex.Encode(out[14:18], raw[6:8])
	out[18] = '-'
	hex.Encode(out[19:23], raw[8:10])
	out[23] = '-'
	hex.Encode(out[24:36], raw[10:16])

	return string(out[:])
}
