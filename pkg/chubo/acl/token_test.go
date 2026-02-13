// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package acl

import (
	"regexp"
	"testing"
)

func TestWorkloadTokenDeterministic(t *testing.T) {
	t.Parallel()

	got1 := WorkloadToken("token", "nomad")
	got2 := WorkloadToken("token", "nomad")

	if got1 == "" {
		t.Fatalf("expected non-empty token")
	}

	if got1 != got2 {
		t.Fatalf("expected deterministic token, got %q != %q", got1, got2)
	}

	uuidRe := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidRe.MatchString(got1) {
		t.Fatalf("expected UUID token, got %q", got1)
	}

	// UUIDv4 version nibble (start of 3rd group) should be '4'.
	if got1[14] != '4' {
		t.Fatalf("expected v4 UUID, got %q", got1)
	}

	// Variant nibble (start of 4th group) should be 8/9/a/b.
	switch got1[19] {
	case '8', '9', 'a', 'b':
	default:
		t.Fatalf("expected RFC4122 variant, got %q", got1)
	}
}
