// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo || chuboos

package services

import (
	"context"
	"log"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
	machineapi "github.com/chubo-dev/chubo/pkg/machinery/api/machine"
)

// BootstrapEtcd is a no-op in chubo mode, where etcd is not part of the runtime.
func BootstrapEtcd(context.Context, runtime.Runtime, *machineapi.BootstrapRequest) error {
	log.Printf("bootstrap: etcd bootstrap skipped in chubo mode")

	return nil
}
