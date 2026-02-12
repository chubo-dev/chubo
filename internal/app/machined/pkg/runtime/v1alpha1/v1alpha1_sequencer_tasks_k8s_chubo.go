// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chubo || chuboos

package v1alpha1

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/siderolabs/talos/internal/app/machined/pkg/runtime"
	"github.com/siderolabs/talos/internal/app/machined/pkg/runtime/v1alpha1/internal/openwontondrain"
)

// Chubo doesn't ship Kubernetes, etcd, or CRI management.
// Keep the Talos sequencer API shape, but make these tasks no-ops.

const (
	openWontonHTTPAddress        = "http://127.0.0.1:4646"
	openWontonRolePath           = "/var/lib/chubo/config/openwonton.role"
	openWontonDrainDeadline      = 10 * time.Minute
	openWontonDefaultHTTPTimeout = 5 * time.Second
)

func CordonAndDrainNode(runtime.Sequence, any) (runtime.TaskExecutionFunc, string) {
	return func(ctx context.Context, logger *log.Logger, r runtime.Runtime) error {
		role, configured, err := openwontondrain.ReadRole(openWontonRolePath)
		if err != nil {
			logger.Printf("skipping openwonton drain: failed to read role: %v", err)

			return nil
		}

		if !configured {
			logger.Printf("skipping openwonton drain: role file not found")

			return nil
		}

		if !openwontondrain.IsClientRole(role) {
			logger.Printf("skipping openwonton drain: role=%q", role)

			return nil
		}

		nodeName, err := r.NodeName()
		if err != nil || strings.TrimSpace(nodeName) == "" {
			nodeName, _ = os.Hostname() //nolint:errcheck
		}

		client := &http.Client{Timeout: openWontonDefaultHTTPTimeout}

		if err := openwontondrain.DrainNode(ctx, client, openWontonHTTPAddress, nodeName, openWontonDrainDeadline); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			logger.Printf("skipping openwonton drain: %v", err)

			return nil
		}

		logger.Printf("requested openwonton drain for node %q", nodeName)

		return nil
	}, "cordonAndDrainNode"
}

func LeaveEtcd(runtime.Sequence, any) (runtime.TaskExecutionFunc, string) {
	return func(ctx context.Context, logger *log.Logger, r runtime.Runtime) error {
		logger.Printf("skipping etcd leave (chubo build)")

		return nil
	}, "leaveEtcd"
}

func RemoveAllPods(runtime.Sequence, any) (runtime.TaskExecutionFunc, string) {
	return func(ctx context.Context, logger *log.Logger, r runtime.Runtime) error {
		logger.Printf("skipping pod removal (chubo build)")

		return nil
	}, "removeAllPods"
}

func StopAllPods(runtime.Sequence, any) (runtime.TaskExecutionFunc, string) {
	return func(ctx context.Context, logger *log.Logger, r runtime.Runtime) error {
		logger.Printf("skipping pod stop (chubo build)")

		return nil
	}, "stopAllPods"
}
