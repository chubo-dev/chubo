// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/state"
	clientv3 "go.etcd.io/etcd/client/v3"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/events"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/services"
	"github.com/chubo-dev/chubo/internal/pkg/cri"
	"github.com/chubo-dev/chubo/internal/pkg/etcd"
	"github.com/chubo-dev/chubo/internal/pkg/logind"
	"github.com/chubo-dev/chubo/pkg/kubernetes"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/k8s"
)

// CordonAndDrainNode represents the task for stopping all containerd tasks in the
// k8s.io namespace.
func CordonAndDrainNode(runtime.Sequence, any) (runtime.TaskExecutionFunc, string) {
	return func(ctx context.Context, logger *log.Logger, r runtime.Runtime) (err error) {
		// Skip not-exist error as it means that the node hasn't fully joined yet.
		if _, err = os.Stat("/var/lib/kubelet/pki/kubelet-client-current.pem"); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}

			return err
		}

		var nodename string

		if nodename, err = r.NodeName(); err != nil {
			return err
		}

		// Controllers will automatically cordon the node when it enters the appropriate phase,
		// so here we just wait for the node to be cordoned.
		if err = waitForNodeCordoned(ctx, logger, r, nodename); err != nil {
			return err
		}

		var kubeHelper *kubernetes.Client

		if kubeHelper, err = kubernetes.NewClientFromKubeletKubeconfig(); err != nil {
			return err
		}

		defer kubeHelper.Close() //nolint:errcheck

		return kubeHelper.Drain(ctx, nodename)
	}, "cordonAndDrainNode"
}

func waitForNodeCordoned(ctx context.Context, logger *log.Logger, r runtime.Runtime, nodename string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	logger.Print("waiting for node to be cordoned")

	_, err := r.State().V1Alpha2().Resources().WatchFor(
		ctx,
		k8s.NewNodeStatus(k8s.NamespaceName, nodename).Metadata(),
		state.WithCondition(func(r resource.Resource) (bool, error) {
			if resource.IsTombstone(r) {
				return false, nil
			}

			nodeStatus, ok := r.(*k8s.NodeStatus)
			if !ok {
				return false, nil
			}

			return nodeStatus.TypedSpec().Unschedulable, nil
		}),
	)

	return err
}

// LeaveEtcd represents the task for removing a control plane node from etcd.
//
//nolint:gocyclo
func LeaveEtcd(runtime.Sequence, any) (runtime.TaskExecutionFunc, string) {
	return func(ctx context.Context, logger *log.Logger, r runtime.Runtime) (err error) {
		_, err = os.Stat(filepath.Join(constants.EtcdDataPath, "/member"))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}

			return err
		}

		etcdID := (&services.Etcd{}).ID(r)

		services := system.Services(r).List()

		shouldLeaveEtcd := false

		for _, service := range services {
			if service.AsProto().Id != etcdID {
				continue
			}

			switch service.GetState() { //nolint:exhaustive
			case events.StateRunning:
				fallthrough
			case events.StateStopping:
				fallthrough
			case events.StateFailed:
				shouldLeaveEtcd = true
			}

			break
		}

		if !shouldLeaveEtcd {
			return nil
		}

		client, err := etcd.NewClientFromControlPlaneIPs(ctx, r.State().V1Alpha2().Resources())
		if err != nil {
			return fmt.Errorf("failed to create etcd client: %w", err)
		}

		//nolint:errcheck
		defer client.Close()

		ctx = clientv3.WithRequireLeader(ctx)

		if err = client.LeaveCluster(ctx, r.State().V1Alpha2().Resources()); err != nil {
			return fmt.Errorf("failed to leave cluster: %w", err)
		}

		return nil
	}, "leaveEtcd"
}

// RemoveAllPods represents the task for stopping and removing all pods.
func RemoveAllPods(runtime.Sequence, any) (runtime.TaskExecutionFunc, string) {
	return stopAndRemoveAllPods(cri.StopAndRemove), "removeAllPods"
}

// StopAllPods represents the task for stopping all pods.
func StopAllPods(runtime.Sequence, any) (runtime.TaskExecutionFunc, string) {
	return stopAndRemoveAllPods(cri.StopOnly), "stopAllPods"
}

func waitForKubeletLifecycleFinalizers(ctx context.Context, logger *log.Logger, r runtime.Runtime) error {
	logger.Printf("waiting for kubelet lifecycle finalizers")

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	lifecycle := resource.NewMetadata(k8s.NamespaceName, k8s.KubeletLifecycleType, k8s.KubeletLifecycleID, resource.VersionUndefined)

	for {
		ok, err := r.State().V1Alpha2().Resources().Teardown(ctx, lifecycle)
		if err != nil {
			return err
		}

		if ok {
			break
		}

		_, err = r.State().V1Alpha2().Resources().WatchFor(ctx, lifecycle, state.WithFinalizerEmpty())
		if err != nil {
			return err
		}
	}

	return r.State().V1Alpha2().Resources().Destroy(ctx, lifecycle)
}

func stopAndRemoveAllPods(stopAction cri.StopAction) runtime.TaskExecutionFunc {
	return func(ctx context.Context, logger *log.Logger, r runtime.Runtime) (err error) {
		if err = waitForKubeletLifecycleFinalizers(ctx, logger, r); err != nil {
			logger.Printf("failed waiting for kubelet lifecycle finalizers: %s", err)
		}

		logger.Printf("shutting down kubelet gracefully")

		shutdownCtx, shutdownCtxCancel := context.WithTimeout(ctx, logind.InhibitMaxDelay)
		defer shutdownCtxCancel()

		if err = r.State().Machine().DBus().WaitShutdown(shutdownCtx); err != nil {
			logger.Printf("failed waiting for inhibit shutdown lock: %s", err)
		}

		if err = system.Services(nil).Stop(ctx, "kubelet"); err != nil {
			return err
		}

		// Check that the CRI is running and the socket is available, if not, skip the rest.
		if _, err = os.Stat(constants.CRIContainerdAddress); errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		client, err := cri.NewClient("unix://"+constants.CRIContainerdAddress, 10*time.Second)
		if err != nil {
			return err
		}

		//nolint:errcheck
		defer client.Close()

		ctx, cancel := context.WithTimeout(ctx, time.Minute*3)
		defer cancel()

		// We remove pods with POD network mode first so that the CNI can perform
		// any cleanup tasks. If we don't do this, we run the risk of killing the
		// CNI, preventing the CRI from cleaning up the pod's networking.
		if err = client.StopAndRemovePodSandboxes(ctx, stopAction, runtimeapi.NamespaceMode_POD, runtimeapi.NamespaceMode_CONTAINER); err != nil {
			logger.Printf("failed to stop and remove pods with POD network mode: %s", err)
		}

		// With the POD network mode pods out of the way, we kill the remaining pods.
		if err = client.StopAndRemovePodSandboxes(ctx, stopAction); err != nil {
			logger.Printf("failed to stop and remove pods: %s", err)
		}

		return nil
	}
}
