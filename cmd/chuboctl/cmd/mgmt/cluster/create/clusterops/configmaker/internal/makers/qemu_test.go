// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package makers_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/chubo-dev/chubo/cmd/chuboctl/cmd/mgmt/cluster/create/clusterops"
	"github.com/chubo-dev/chubo/cmd/chuboctl/cmd/mgmt/cluster/create/clusterops/configmaker/internal/makers"
	"github.com/chubo-dev/chubo/pkg/machinery/config/generate"
)

func TestQemuMaker_MachineConfig(t *testing.T) {
	cOps := clusterops.GetCommon()
	qOps := clusterops.GetQemu()

	m, err := makers.NewQemu(makers.MakerOptions[clusterops.Qemu]{
		ExtraOps:    qOps,
		CommonOps:   cOps,
		Provisioner: testProvisioner{}, // use test provisioner to simplify the test case.
	})
	require.NoError(t, err)

	desiredExtraGenOps := []generate.Option{}
	if !qOps.BootloaderEnabled || qOps.TargetArch == "arm64" {
		// QEMU maker intentionally disables kexec when bootloader is disabled or on arm64.
		desiredExtraGenOps = append(desiredExtraGenOps, generate.WithSysctls(map[string]string{
			"kernel.kexec_load_disabled": "1",
		}))
	}

	assertConfigDefaultness(t, cOps, *m.Maker, desiredExtraGenOps...)
}
