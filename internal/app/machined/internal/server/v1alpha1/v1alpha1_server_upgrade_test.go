package runtime

import (
	"testing"

	machinetype "github.com/chubo-dev/chubo/pkg/machinery/config/machine"
)

func TestShouldRunEtcdUpgradePrechecks(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		machine    machinetype.Type
		force      bool
		serviceIDs []string
		want       bool
	}{
		{
			name:       "controlplane with etcd runs prechecks",
			machine:    machinetype.TypeControlPlane,
			serviceIDs: []string{"machined", "etcd"},
			want:       true,
		},
		{
			name:       "controlplane without etcd skips prechecks",
			machine:    machinetype.TypeControlPlane,
			serviceIDs: []string{"machined", "apid"},
			want:       false,
		},
		{
			name:       "worker always skips prechecks",
			machine:    machinetype.TypeWorker,
			serviceIDs: []string{"machined", "etcd"},
			want:       false,
		},
		{
			name:       "force flag skips prechecks",
			machine:    machinetype.TypeControlPlane,
			force:      true,
			serviceIDs: []string{"machined", "etcd"},
			want:       false,
		},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shouldRunEtcdUpgradePrechecks(tc.machine, tc.force, tc.serviceIDs)
			if got != tc.want {
				t.Fatalf("shouldRunEtcdUpgradePrechecks() = %v, want %v", got, tc.want)
			}
		})
	}
}
