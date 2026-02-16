// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo

package talos

import "testing"

func TestIsWorkloadImageNamespaceDefault(t *testing.T) {
	t.Parallel()

	if !isWorkloadImageNamespace("cri") {
		t.Fatalf("expected cri alias to be accepted")
	}

	if !isWorkloadImageNamespace("workload") {
		t.Fatalf("expected workload alias to be accepted")
	}

	if isWorkloadImageNamespace("system") {
		t.Fatalf("expected non-workload namespace to be rejected")
	}
}
