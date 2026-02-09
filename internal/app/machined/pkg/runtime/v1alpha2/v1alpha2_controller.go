// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package v1alpha2

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/cosi-project/runtime/pkg/controller"
	osruntime "github.com/cosi-project/runtime/pkg/controller/runtime"
	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/siderolabs/gen/xslices"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/siderolabs/talos/internal/app/machined/pkg/runtime"
	runtimelogging "github.com/siderolabs/talos/internal/app/machined/pkg/runtime/logging"
	"github.com/siderolabs/talos/pkg/logging"
	talosconfig "github.com/siderolabs/talos/pkg/machinery/config/config"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	configresource "github.com/siderolabs/talos/pkg/machinery/resources/config"
	"github.com/siderolabs/talos/pkg/xfs"
	"github.com/siderolabs/talos/pkg/xfs/fsopen"
)

// Controller implements runtime.V1alpha2Controller.
type Controller struct {
	controllerRuntime *osruntime.Runtime

	loggingManager  runtime.LoggingManager
	consoleLogLevel zap.AtomicLevel
	logger          *zap.Logger

	v1alpha1Runtime runtime.Runtime
}

// NewController creates Controller.
func NewController(v1alpha1Runtime runtime.Runtime) (*Controller, error) {
	ctrl := &Controller{
		consoleLogLevel: zap.NewAtomicLevel(),
		loggingManager:  v1alpha1Runtime.Logging(),
		v1alpha1Runtime: v1alpha1Runtime,
	}

	var err error

	ctrl.logger, err = ctrl.MakeLogger("controller-runtime")
	if err != nil {
		return nil, err
	}

	ctrl.controllerRuntime, err = osruntime.NewRuntime(v1alpha1Runtime.State().V1Alpha2().Resources(), ctrl.logger)

	return ctrl, err
}

// Run the controller runtime.
func (ctrl *Controller) Run(ctx context.Context, drainer *runtime.Drainer) error {
	// adjust the log level based on machine configuration
	go ctrl.watchMachineConfig(ctx)

	dnsCacheLogger, err := ctrl.MakeLogger("dns-resolve-cache")
	if err != nil {
		return err
	}

	var (
		etcRoot                xfs.Root
		networkEtcRoot         xfs.Root
		networkBindMountTarget string
	)

	etcRoot = &xfs.UnixRoot{
		FS: fsopen.New(
			"tmpfs",
			fsopen.WithStringParameter("mode", "0755"),
			fsopen.WithStringParameter("size", "8M"),
		),
	}

	networkEtcRoot = &xfs.UnixRoot{
		FS: fsopen.New(
			"tmpfs",
			fsopen.WithStringParameter("mode", "0755"),
			fsopen.WithStringParameter("size", "4M"),
		),
	}

	networkBindMountTarget = constants.SystemResolvedPath

	// While running in container, we don't have control over kernel version
	// shipped with the machine. If the kernel does not support open_tree syscall
	// on anonymous filesystem file descriptors, we need to fallback to the classic,
	// less secure mode. This capability was added in kernel 6.15.0.
	if ctrl.v1alpha1Runtime.State().Platform().Mode().InContainer() {
		opentreeOnAnonymous, err := runtime.KernelCapabilities().OpentreeOnAnonymousFS()
		if err != nil {
			return err
		}

		if !opentreeOnAnonymous {
			etcRoot = &xfs.OSRoot{
				Shadow: constants.SystemEtcPath,
			}

			networkEtcRoot = &xfs.OSRoot{
				Shadow: constants.SystemResolvedPath,
			}

			networkBindMountTarget = ""
		}
	}

	if err := etcRoot.OpenFS(); err != nil {
		return fmt.Errorf("failed to open etc root: %w", err)
	}
	defer etcRoot.Close() //nolint:errcheck

	if err := networkEtcRoot.OpenFS(); err != nil {
		return fmt.Errorf("failed to open network etc root: %w", err)
	}
	defer networkEtcRoot.Close() //nolint:errcheck

	for _, c := range ctrl.controllers(drainer, etcRoot, networkEtcRoot, networkBindMountTarget, dnsCacheLogger) {
		if err := ctrl.controllerRuntime.RegisterController(c); err != nil {
			return err
		}
	}

	return ctrl.controllerRuntime.Run(ctx)
}

// DependencyGraph returns controller-resources dependencies.
func (ctrl *Controller) DependencyGraph() (*controller.DependencyGraph, error) {
	return ctrl.controllerRuntime.GetDependencyGraph()
}

type loggingDestination struct {
	Format    string
	Endpoint  *url.URL
	ExtraTags map[string]string
}

func (a *loggingDestination) Equal(b *loggingDestination) bool {
	if a.Format != b.Format {
		return false
	}

	if a.Endpoint.String() != b.Endpoint.String() {
		return false
	}

	if len(a.ExtraTags) != len(b.ExtraTags) {
		return false
	}

	for k, v := range a.ExtraTags {
		if vv, ok := b.ExtraTags[k]; !ok || vv != v {
			return false
		}
	}

	return true
}

func (ctrl *Controller) watchMachineConfig(ctx context.Context) {
	watchCh := make(chan state.Event)

	if err := ctrl.v1alpha1Runtime.State().V1Alpha2().Resources().Watch(
		ctx,
		resource.NewMetadata(configresource.NamespaceName, configresource.MachineConfigType, configresource.ActiveID, resource.VersionUndefined),
		watchCh,
	); err != nil {
		ctrl.logger.Warn("error watching machine configuration", zap.Error(err))

		return
	}

	var loggingDestinations []loggingDestination

	for {
		var cfg talosconfig.Config

		select {
		case event := <-watchCh:
			if event.Type != state.Created && event.Type != state.Updated {
				continue
			}

			cfg = event.Resource.(*configresource.MachineConfig).Config()

		case <-ctx.Done():
			return
		}

		ctrl.updateConsoleLoggingConfig(cfg.Debug())

		if cfg.Machine() == nil {
			ctrl.updateLoggingConfig(ctx, nil, &loggingDestinations)
		} else {
			ctrl.updateLoggingConfig(ctx, cfg.Machine().Logging().Destinations(), &loggingDestinations)
		}
	}
}

func (ctrl *Controller) updateConsoleLoggingConfig(debug bool) {
	newLogLevel := zapcore.InfoLevel
	if debug {
		newLogLevel = zapcore.DebugLevel
	}

	if newLogLevel != ctrl.consoleLogLevel.Level() {
		ctrl.logger.Info("setting console log level", zap.Stringer("level", newLogLevel))
		ctrl.consoleLogLevel.SetLevel(newLogLevel)
	}
}

func (ctrl *Controller) updateLoggingConfig(ctx context.Context, dests []talosconfig.LoggingDestination, prevLoggingDestinations *[]loggingDestination) {
	loggingDestinations := make([]loggingDestination, len(dests))

	for i, dest := range dests {
		switch f := dest.Format(); f {
		case constants.LoggingFormatJSONLines:
			loggingDestinations[i] = loggingDestination{
				Format:    f,
				Endpoint:  dest.Endpoint(),
				ExtraTags: dest.ExtraTags(),
			}
		default:
			// should not be possible due to validation
			panic(fmt.Sprintf("unhandled log destination format %q", f))
		}
	}

	loggingChanged := len(*prevLoggingDestinations) != len(loggingDestinations)
	if !loggingChanged {
		for i, u := range *prevLoggingDestinations {
			if !u.Equal(&loggingDestinations[i]) {
				loggingChanged = true

				break
			}
		}
	}

	if !loggingChanged {
		return
	}

	*prevLoggingDestinations = loggingDestinations

	var prevSenders []runtime.LogSender

	if len(loggingDestinations) > 0 {
		senders := xslices.Map(dests, runtimelogging.NewJSONLines)

		ctrl.logger.Info("enabling JSON logging")
		prevSenders = ctrl.loggingManager.SetSenders(senders)
	} else {
		ctrl.logger.Info("disabling JSON logging")
		prevSenders = ctrl.loggingManager.SetSenders(nil)
	}

	closeCtx, closeCancel := context.WithTimeout(ctx, 3*time.Second)
	defer closeCancel()

	var wg sync.WaitGroup

	for _, sender := range prevSenders {
		wg.Go(func() {
			err := sender.Close(closeCtx)
			ctrl.logger.Info("log sender closed", zap.Error(err))
		})
	}

	wg.Wait()
}

// MakeLogger creates a logger for a service.
func (ctrl *Controller) MakeLogger(serviceName string) (*zap.Logger, error) {
	logWriter, err := ctrl.loggingManager.ServiceLog(serviceName).Writer()
	if err != nil {
		return nil, err
	}

	return logging.ZapLogger(
		logging.NewLogDestination(logWriter, zapcore.DebugLevel,
			logging.WithColoredLevels(),
		),
		logging.NewLogDestination(logging.StdWriter, ctrl.consoleLogLevel,
			logging.WithoutTimestamp(),
			logging.WithoutLogLevels(),
			logging.WithControllerErrorSuppressor(constants.ConsoleLogErrorSuppressThreshold),
		),
	).With(logging.Component(serviceName)), nil
}
