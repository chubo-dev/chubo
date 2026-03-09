// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package v1alpha1

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime/v1alpha1/platform"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/services"
	"github.com/chubo-dev/chubo/pkg/conditions"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
)

func StartAllServices(runtime.Sequence, any) (runtime.TaskExecutionFunc, string) {
	return func(ctx context.Context, logger *log.Logger, r runtime.Runtime) (err error) {
		// Reuse the same "activate" event hook, but keep semantics remote-first.
		platform.FireEvent(
			ctx,
			r.State().Platform(),
			platform.Event{
				Type:    platform.EventTypeActivate,
				Message: "Chubo-OS is ready for remote interaction.",
			},
		)

		required := []string{
			"machined",
			"containerd",
			"apid",
		}

		if shouldStartDockerForOpenWonton("/var/lib/chubo/config/openwonton.hcl", "/var/lib/chubo/config/openwonton.role") {
			logger.Printf("OpenWonton config detected, starting docker service")
			system.Services(r).LoadAndStart(&services.Docker{})
			required = append(required, services.DockerServiceID)
		}

		all := make([]conditions.Condition, 0, len(required))

		for _, id := range required {
			all = append(all, system.WaitForService(system.StateEventUp, id))
		}

		ctx, cancel := context.WithTimeout(ctx, constants.BootTimeout)
		defer cancel()

		aggregateCondition := conditions.WaitForAll(all...)

		errChan := make(chan error)

		go func() {
			errChan <- aggregateCondition.Wait(ctx)
		}()

		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			logger.Printf("%s", aggregateCondition.String())

			select {
			case err := <-errChan:
				return err
			case <-ticker.C:
			}
		}
	}, "startAllServices"
}

func shouldStartDockerForOpenWonton(configPath, rolePath string) bool {
	if _, err := os.Stat(configPath); err == nil {
		return true
	}

	if _, err := os.Stat(rolePath); err == nil {
		return true
	}

	return false
}
