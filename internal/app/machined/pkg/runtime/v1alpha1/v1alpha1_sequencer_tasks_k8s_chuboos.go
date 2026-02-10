// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build chuboos

package v1alpha1

import (
	"context"
	"log"

	"github.com/siderolabs/talos/internal/app/machined/pkg/runtime"
)

// ChuboOS doesn't ship Kubernetes, etcd, or CRI management.
// Keep the Talos sequencer API shape, but make these tasks no-ops.

func CordonAndDrainNode(runtime.Sequence, any) (runtime.TaskExecutionFunc, string) {
	return func(ctx context.Context, logger *log.Logger, r runtime.Runtime) error {
		logger.Printf("skipping cordon/drain (chuboos build)")

		return nil
	}, "cordonAndDrainNode"
}

func LeaveEtcd(runtime.Sequence, any) (runtime.TaskExecutionFunc, string) {
	return func(ctx context.Context, logger *log.Logger, r runtime.Runtime) error {
		logger.Printf("skipping etcd leave (chuboos build)")

		return nil
	}, "leaveEtcd"
}

func RemoveAllPods(runtime.Sequence, any) (runtime.TaskExecutionFunc, string) {
	return func(ctx context.Context, logger *log.Logger, r runtime.Runtime) error {
		logger.Printf("skipping pod removal (chuboos build)")

		return nil
	}, "removeAllPods"
}

func StopAllPods(runtime.Sequence, any) (runtime.TaskExecutionFunc, string) {
	return func(ctx context.Context, logger *log.Logger, r runtime.Runtime) error {
		logger.Printf("skipping pod stop (chuboos build)")

		return nil
	}, "stopAllPods"
}
