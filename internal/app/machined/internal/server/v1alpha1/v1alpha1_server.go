// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package runtime

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	stdlibx509 "crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	cosiv1alpha1 "github.com/cosi-project/runtime/api/v1alpha1"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/cosi-project/runtime/pkg/state/protobuf/server"
	"github.com/google/uuid"
	"github.com/gopacket/gopacket/afpacket"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/nberlee/go-netstat/netstat"
	"github.com/pkg/xattr"
	"github.com/prometheus/procfs"
	"github.com/rs/xid"
	"github.com/siderolabs/gen/xslices"
	"github.com/siderolabs/go-kmsg"
	"github.com/siderolabs/go-pointer"
	"golang.org/x/net/bpf"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/chubo-dev/chubo/internal/app/debug"
	"github.com/chubo-dev/chubo/internal/app/images"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime/v1alpha1/bootloader"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime/v1alpha1/bootloader/options"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system"
	"github.com/chubo-dev/chubo/internal/app/resources"
	storaged "github.com/chubo-dev/chubo/internal/app/storaged"
	"github.com/chubo-dev/chubo/internal/pkg/miniprocfs"
	"github.com/chubo-dev/chubo/internal/pkg/partition"
	"github.com/chubo-dev/chubo/internal/pkg/pcap"
	"github.com/chubo-dev/chubo/pkg/archiver"
	chuboacl "github.com/chubo-dev/chubo/pkg/chubo/acl"
	"github.com/chubo-dev/chubo/pkg/chunker"
	"github.com/chubo-dev/chubo/pkg/chunker/stream"
	"github.com/chubo-dev/chubo/pkg/machinery/api/cluster"
	"github.com/chubo-dev/chubo/pkg/machinery/api/common"
	"github.com/chubo-dev/chubo/pkg/machinery/api/inspect"
	"github.com/chubo-dev/chubo/pkg/machinery/api/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/api/storage"
	timeapi "github.com/chubo-dev/chubo/pkg/machinery/api/time"
	clientconfig "github.com/chubo-dev/chubo/pkg/machinery/client/config"
	"github.com/chubo-dev/chubo/pkg/machinery/config"
	"github.com/chubo-dev/chubo/pkg/machinery/config/configdiff"
	"github.com/chubo-dev/chubo/pkg/machinery/config/configloader"
	"github.com/chubo-dev/chubo/pkg/machinery/config/generate/secrets"
	machinetype "github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	"github.com/chubo-dev/chubo/pkg/machinery/nethelpers"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/block"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/network"
	"github.com/chubo-dev/chubo/pkg/machinery/role"
	"github.com/chubo-dev/chubo/pkg/machinery/version"
)

// OSPathSeparator is the string version of the os.PathSeparator.
const OSPathSeparator = string(os.PathSeparator)

// Server implements ClusterService and MachineService APIs
// and is also responsible for registering ResourceServer and InspectServer.
type Server struct {
	cluster.UnimplementedClusterServiceServer
	machine.UnimplementedMachineServiceServer

	Controller runtime.Controller

	// ShutdownCtx signals that the server is shutting down.
	ShutdownCtx context.Context //nolint:containedctx

	server *grpc.Server
}

func (s *Server) checkSupported(feature runtime.ModeCapability) error {
	mode := s.Controller.Runtime().State().Platform().Mode()

	if !mode.Supports(feature) {
		return status.Errorf(codes.FailedPrecondition, "method is not supported in %s mode", mode.String())
	}

	return nil
}

func (s *Server) checkControlplane(apiName string) error {
	switch s.Controller.Runtime().Config().Machine().Type() { //nolint:exhaustive
	case machinetype.TypeControlPlane:
		fallthrough
	case machinetype.TypeInit:
		return nil
	}

	return status.Errorf(codes.Unimplemented, "%s is only available on control plane nodes", apiName)
}

func (s *Server) checkControlplaneService(apiName string, serviceID string) error {
	if err := s.checkControlplane(apiName); err != nil {
		return err
	}

	if !slices.Contains(runtimeServiceIDs(s.Controller.Runtime()), serviceID) {
		return status.Errorf(codes.Unimplemented, "%s is not available: %q service is not registered on this node", apiName, serviceID)
	}

	return nil
}

// Register implements the factory.Registrator interface.
func (s *Server) Register(obj *grpc.Server) {
	s.server = obj

	// wrap resources with access filter
	resourceState := s.Controller.Runtime().State().V1Alpha2().Resources()
	resourceState = state.WrapCore(state.Filter(resourceState, resources.AccessPolicy(resourceState)))

	machine.RegisterMachineServiceServer(obj, s)
	machine.RegisterImageServiceServer(obj, images.NewService(s.Controller))
	machine.RegisterDebugServiceServer(obj, &debug.Service{})
	cluster.RegisterClusterServiceServer(obj, s)
	cosiv1alpha1.RegisterStateServer(obj, server.NewState(resourceState))
	inspect.RegisterInspectServiceServer(obj, &InspectServer{server: s})
	storage.RegisterStorageServiceServer(obj, &storaged.Server{Controller: s.Controller})
	timeapi.RegisterTimeServiceServer(obj, &TimeServer{ConfigProvider: s.Controller.Runtime()})
}

// modeWrapper overrides RequiresInstall() based on actual installed status.
type modeWrapper struct {
	runtime.Mode

	installed bool
}

func (m modeWrapper) RequiresInstall() bool {
	return m.Mode.RequiresInstall() && !m.installed
}

// ApplyConfiguration implements machine.MachineService.
//
//nolint:gocyclo,cyclop
func (s *Server) ApplyConfiguration(ctx context.Context, in *machine.ApplyConfigurationRequest) (*machine.ApplyConfigurationResponse, error) {
	mode := in.Mode.String()
	modeDetails := "Applied configuration with a reboot"
	modeErr := ""

	if in.Mode != machine.ApplyConfigurationRequest_TRY {
		s.Controller.Runtime().CancelConfigRollbackTimeout()
	}

	cfgProvider, err := configloader.NewFromBytes(in.GetData())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// as we are not in maintenance mode, the v1alpha1 config should be always present
	// in the future we should allow to remove v1alpha1, but for now for better UX we deny
	// such requests to avoid confusion
	if cfgProvider.RawV1Alpha1() == nil {
		return nil, status.Error(codes.InvalidArgument, "the applied machine configuration doesn't contain v1alpha1 config, did you mean to patch the machine config instead?")
	}

	validationMode := modeWrapper{
		Mode:      s.Controller.Runtime().State().Platform().Mode(),
		installed: s.Controller.Runtime().State().Machine().Installed(),
	}

	warnings, err := cfgProvider.Validate(validationMode)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	warningsRuntime, err := cfgProvider.RuntimeValidate(ctx, s.Controller.Runtime().State().V1Alpha2().Resources(), validationMode)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	warnings = slices.Concat(warnings, warningsRuntime)

	//nolint:exhaustive
	switch in.Mode {
	// --mode=try
	case machine.ApplyConfigurationRequest_TRY:
		fallthrough
	// --mode=no-reboot
	case machine.ApplyConfigurationRequest_NO_REBOOT:
		if err = s.Controller.Runtime().CanApplyImmediate(cfgProvider); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		modeDetails = "Applied configuration without a reboot"
	// --mode=staged
	case machine.ApplyConfigurationRequest_STAGED:
		modeDetails = "Staged configuration to be applied after the next reboot"
	// --mode=auto detect actual update mode
	case machine.ApplyConfigurationRequest_AUTO:
		if err = s.Controller.Runtime().CanApplyImmediate(cfgProvider); err != nil {
			in.Mode = machine.ApplyConfigurationRequest_REBOOT
			modeDetails = "Applied configuration with a reboot"
			modeErr = ": " + err.Error()
		} else {
			in.Mode = machine.ApplyConfigurationRequest_NO_REBOOT
			modeDetails = "Applied configuration without a reboot"
		}

		mode = fmt.Sprintf("%s(%s)", mode, in.Mode)
	}

	if in.DryRun {
		details, err := generateDiff(s.Controller.Runtime(), cfgProvider)
		if err != nil {
			return nil, fmt.Errorf("failed to generate diff: %w", err)
		}

		return &machine.ApplyConfigurationResponse{
			Messages: []*machine.ApplyConfiguration{
				{
					Mode: in.Mode,
					ModeDetails: fmt.Sprintf(`Dry run summary:
%s (skipped in dry-run).
%s`, modeDetails, details),
				},
			},
		}, nil
	}

	log.Printf("apply config request: mode %s", strings.ToLower(mode))

	//nolint:exhaustive
	switch in.Mode {
	// --mode=try
	case machine.ApplyConfigurationRequest_TRY:
		timeout := constants.ConfigTryTimeout
		if in.TryModeTimeout != nil {
			timeout = in.TryModeTimeout.AsDuration()
		}

		modeDetails += fmt.Sprintf("\nThe config is applied in 'try' mode and will be automatically reverted back in %s", timeout.String())

		if err := s.Controller.Runtime().RollbackToConfigAfter(timeout); err != nil {
			return nil, err
		}

		if err := s.Controller.Runtime().SetConfig(cfgProvider); err != nil {
			return nil, err
		}
	// --mode=no-reboot
	case machine.ApplyConfigurationRequest_NO_REBOOT:
		if err := s.Controller.Runtime().SetPersistedConfig(cfgProvider); err != nil {
			return nil, err
		}

		if err := s.Controller.Runtime().SetConfig(cfgProvider); err != nil {
			return nil, err
		}
	// --mode=staged
	case machine.ApplyConfigurationRequest_STAGED:
		if err := s.Controller.Runtime().SetPersistedConfig(cfgProvider); err != nil {
			return nil, err
		}
	// --mode=reboot
	case machine.ApplyConfigurationRequest_REBOOT:
		if err := s.Controller.Runtime().SetPersistedConfig(cfgProvider); err != nil {
			return nil, err
		}

		go func() {
			if err := s.Controller.Run(context.Background(), runtime.SequenceReboot, nil, runtime.WithTakeover()); err != nil {
				if !runtime.IsRebootError(err) {
					log.Println("apply configuration failed:", err)
				}
			}
		}()
	default:
		return nil, fmt.Errorf("incorrect mode '%s' specified for the apply config call", in.Mode.String())
	}

	return &machine.ApplyConfigurationResponse{
		Messages: []*machine.ApplyConfiguration{
			{
				Mode:        in.Mode,
				Warnings:    warnings,
				ModeDetails: modeDetails + modeErr,
			},
		},
	}, nil
}

func generateDiff(r runtime.Runtime, provider config.Provider) (string, error) {
	documentsDiff, err := configdiff.DiffToString(r.ConfigContainer(), provider)
	if err != nil {
		return "", err
	}

	if documentsDiff == "" {
		documentsDiff = "No changes."
	}

	return "Config diff:\n\n" + documentsDiff, nil
}

// Reboot implements the machine.MachineServer interface.
func (s *Server) Reboot(ctx context.Context, in *machine.RebootRequest) (reply *machine.RebootResponse, err error) {
	actorID := uuid.New().String()

	log.Printf("reboot via API received. actor id: %s", actorID)

	if err := s.checkSupported(runtime.Reboot); err != nil {
		return nil, err
	}

	rebootCtx := context.WithValue(context.Background(), runtime.ActorIDCtxKey{}, actorID)

	go func() {
		if err := s.Controller.Run(rebootCtx, runtime.SequenceReboot, in); err != nil {
			if !runtime.IsRebootError(err) {
				log.Println("reboot failed:", err)
			}
		}
	}()

	reply = &machine.RebootResponse{
		Messages: []*machine.Reboot{
			{
				ActorId: actorID,
			},
		},
	}

	return reply, nil
}

// Rollback implements the machine.MachineServer interface.
func (s *Server) Rollback(ctx context.Context, in *machine.RollbackRequest) (*machine.RollbackResponse, error) {
	log.Printf("rollback via API received")

	if err := s.checkSupported(runtime.Rollback); err != nil {
		return nil, err
	}

	systemDisk, err := block.GetSystemDisk(ctx, s.Controller.Runtime().State().V1Alpha2().Resources())
	if err != nil {
		return nil, fmt.Errorf("system disk lookup failed: %w", err)
	}

	if systemDisk == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "system disk not found")
	}

	if err := func() error {
		config, err := bootloader.Probe(systemDisk.DevPath, options.ProbeOptions{})
		if err != nil {
			return err
		}

		return config.Revert(systemDisk.DevPath)
	}(); err != nil {
		return nil, err
	}

	go func() {
		if err := s.Controller.Run(context.Background(), runtime.SequenceReboot, in, runtime.WithTakeover()); err != nil {
			if !runtime.IsRebootError(err) {
				log.Println("reboot failed:", err)
			}
		}
	}()

	return &machine.RollbackResponse{
		Messages: []*machine.Rollback{
			{},
		},
	}, nil
}

// Bootstrap implements machine.MachineService.
//
// Chubo bootstrap is configuration-driven and continuously reconciled, so there is no
// imperative bootstrap RPC like legacy Talos used.
func (s *Server) Bootstrap(context.Context, *machine.BootstrapRequest) (*machine.BootstrapResponse, error) {
	return nil, status.Error(codes.Unimplemented, "bootstrap RPC is not available in chubo mode; apply MachineConfig and monitor chubobootstrapstatus/openwontonbootstrapstatus/opengyozabootstrapstatus resources")
}

// Shutdown implements the machine.MachineServer interface.
func (s *Server) Shutdown(ctx context.Context, in *machine.ShutdownRequest) (reply *machine.ShutdownResponse, err error) {
	actorID := uuid.New().String()

	log.Printf("shutdown via API received. actor id: %s", actorID)

	if err = s.checkSupported(runtime.Shutdown); err != nil {
		return nil, err
	}

	shutdownCtx := context.WithValue(context.Background(), runtime.ActorIDCtxKey{}, actorID)

	go func() {
		if err := s.Controller.Run(shutdownCtx, runtime.SequenceShutdown, in, runtime.WithTakeover()); err != nil {
			if !runtime.IsRebootError(err) {
				log.Println("shutdown failed:", err)
			}
		}
	}()

	reply = &machine.ShutdownResponse{
		Messages: []*machine.Shutdown{
			{
				ActorId: actorID,
			},
		},
	}

	return reply, nil
}

func runtimeServiceIDs(rt runtime.Runtime) []string {
	services := system.Services(rt).List()

	ids := make([]string, 0, len(services))

	for _, svc := range services {
		ids = append(ids, svc.AsProto().GetId())
	}

	return ids
}

// ResetOptions implements runtime.ResetOptions interface.
type ResetOptions struct {
	*machine.ResetRequest

	systemDiskTargets []*partition.VolumeWipeTarget
	systemDiskPaths   []string
}

// GetSystemDiskTargets implements runtime.ResetOptions interface.
func (opt *ResetOptions) GetSystemDiskTargets() []runtime.PartitionTarget {
	if opt.systemDiskTargets == nil {
		return nil
	}

	return xslices.Map(opt.systemDiskTargets, func(t *partition.VolumeWipeTarget) runtime.PartitionTarget { return t })
}

// GetSystemDiskPaths implements runtime.ResetOptions interface.
func (opt *ResetOptions) GetSystemDiskPaths() []string {
	return opt.systemDiskPaths
}

// String implements runtime.ResetOptions interface.
func (opt *ResetOptions) String() string {
	return strings.Join(xslices.Map(opt.systemDiskTargets, func(t *partition.VolumeWipeTarget) string { return t.String() }), ", ")
}

// Reset resets the node.
//
//nolint:gocyclo,cyclop
func (s *Server) Reset(ctx context.Context, in *machine.ResetRequest) (reply *machine.ResetResponse, err error) {
	actorID := uuid.New().String()

	log.Printf("reset request received. actorID: %s", actorID)

	opts := ResetOptions{
		ResetRequest: in,
	}

	if len(in.GetUserDisksToWipe()) > 0 {
		if in.Mode == machine.ResetRequest_SYSTEM_DISK {
			return nil, errors.New("reset failed: invalid input, wipe mode SYSTEM_DISK doesn't support UserDisksToWipe parameter")
		}

		diskList, err := safe.StateListAll[*block.Disk](ctx, s.Controller.Runtime().State().V1Alpha2().Resources())
		if err != nil {
			return nil, fmt.Errorf("listing disks failed: %w", err)
		}

		disks := xslices.ToMap(
			safe.ToSlice(diskList, func(d *block.Disk) *block.Disk { return d }),
			func(disk *block.Disk) (string, *block.Disk) {
				return disk.TypedSpec().DevPath, disk
			},
		)

		systemDisk, err := block.GetSystemDisk(ctx, s.Controller.Runtime().State().V1Alpha2().Resources())
		if err != nil {
			return nil, fmt.Errorf("system disk lookup failed: %w", err)
		}

		// validate input
		for _, deviceName := range in.GetUserDisksToWipe() {
			disk, ok := disks[deviceName]
			if !ok {
				return nil, fmt.Errorf("reset user disk failed: device %s wasn't found", deviceName)
			}

			if disk.TypedSpec().Readonly {
				return nil, fmt.Errorf("reset user disk failed: device %s is readonly", deviceName)
			}

			if systemDisk != nil && deviceName == systemDisk.DevPath {
				return nil, fmt.Errorf("reset user disk failed: device %s is the system disk", deviceName)
			}
		}
	}

	if len(in.GetSystemPartitionsToWipe()) > 0 {
		if in.Mode == machine.ResetRequest_USER_DISKS {
			return nil, errors.New("reset failed: invalid input, wipe mode USER_DISKS doesn't support SystemPartitionsToWipe parameter")
		}

		for _, spec := range in.GetSystemPartitionsToWipe() {
			volumeStatus, err := safe.ReaderGetByID[*block.VolumeStatus](ctx, s.Controller.Runtime().State().V1Alpha2().Resources(), spec.Label)
			if err != nil {
				return nil, fmt.Errorf("failed to get volume status with label %q: %w", spec.Label, err)
			}

			if volumeStatus.TypedSpec().Location == "" {
				return nil, fmt.Errorf("failed to reset: volume %q is not located", spec.Label)
			}

			target := partition.VolumeWipeTargetFromVolumeStatus(volumeStatus)

			if spec.Wipe {
				opts.systemDiskTargets = append(opts.systemDiskTargets, target)
			}
		}
	}

	if in.Mode != machine.ResetRequest_USER_DISKS && len(in.GetSystemPartitionsToWipe()) == 0 {
		opts.systemDiskPaths, err = block.GetSystemDiskPaths(ctx, s.Controller.Runtime().State().V1Alpha2().Resources())
		if err != nil {
			return nil, fmt.Errorf("system disk paths lookup failed: %w", err)
		}
	}

	resetCtx := context.WithValue(context.Background(), runtime.ActorIDCtxKey{}, actorID)

	go func() {
		if err := s.Controller.Run(resetCtx, runtime.SequenceReset, &opts); err != nil {
			if !runtime.IsRebootError(err) {
				log.Println("reset failed:", err)
			}
		}
	}()

	reply = &machine.ResetResponse{
		Messages: []*machine.Reset{
			{
				ActorId: actorID,
			},
		},
	}

	return reply, nil
}

// ServiceList returns list of the registered services and their status.
func (s *Server) ServiceList(ctx context.Context, in *emptypb.Empty) (result *machine.ServiceListResponse, err error) {
	services := system.Services(s.Controller.Runtime()).List()

	result = &machine.ServiceListResponse{
		Messages: []*machine.ServiceList{
			{
				Services: xslices.Map(services, (*system.ServiceRunner).AsProto),
			},
		},
	}

	return result, nil
}

// ServiceStart implements the machine.MachineServer interface and starts a
// service running on Talos.
func (s *Server) ServiceStart(ctx context.Context, in *machine.ServiceStartRequest) (reply *machine.ServiceStartResponse, err error) {
	if err = system.Services(s.Controller.Runtime()).APIStart(ctx, in.Id); err != nil {
		return &machine.ServiceStartResponse{}, err
	}

	reply = &machine.ServiceStartResponse{
		Messages: []*machine.ServiceStart{
			{
				Resp: fmt.Sprintf("Service %q started", in.Id),
			},
		},
	}

	return reply, err
}

// ServiceStop implements the machine.MachineServer interface and stops a
// service running on Talos.
func (s *Server) ServiceStop(ctx context.Context, in *machine.ServiceStopRequest) (reply *machine.ServiceStopResponse, err error) {
	if err = system.Services(s.Controller.Runtime()).APIStop(ctx, in.Id); err != nil {
		return &machine.ServiceStopResponse{}, err
	}

	reply = &machine.ServiceStopResponse{
		Messages: []*machine.ServiceStop{
			{
				Resp: fmt.Sprintf("Service %q stopped", in.Id),
			},
		},
	}

	return reply, err
}

// ServiceRestart implements the machine.MachineServer interface and stops a
// service running on Talos.
func (s *Server) ServiceRestart(ctx context.Context, in *machine.ServiceRestartRequest) (reply *machine.ServiceRestartResponse, err error) {
	if err = system.Services(s.Controller.Runtime()).APIRestart(ctx, in.Id); err != nil {
		return &machine.ServiceRestartResponse{}, err
	}

	reply = &machine.ServiceRestartResponse{
		Messages: []*machine.ServiceRestart{
			{
				Resp: fmt.Sprintf("Service %q restarted", in.Id),
			},
		},
	}

	return reply, err
}

// Copy implements the machine.MachineServer interface and copies data out of Talos node.
func (s *Server) Copy(req *machine.CopyRequest, obj machine.MachineService_CopyServer) error {
	path := req.RootPath
	path = filepath.Clean(path)

	if !filepath.IsAbs(path) {
		return fmt.Errorf("path is not absolute %v", path)
	}

	pr, pw := io.Pipe()

	errCh := make(chan error, 1)

	ctx, ctxCancel := context.WithCancel(obj.Context())
	defer ctxCancel()

	go func() {
		//nolint:errcheck
		defer pw.Close()

		errCh <- archiver.TarGz(ctx, path, pw)
	}()

	chunker := stream.NewChunker(ctx, pr)
	chunkCh := chunker.Read()

	for data := range chunkCh {
		err := obj.SendMsg(&common.Data{Bytes: data})
		if err != nil {
			ctxCancel()
		}
	}

	archiveErr := <-errCh
	if archiveErr != nil {
		return obj.SendMsg(&common.Data{
			Metadata: &common.Metadata{
				Error: archiveErr.Error(),
			},
		})
	}

	return nil
}

// List implements the machine.MachineServer interface.
//
//nolint:gocyclo,cyclop
func (s *Server) List(req *machine.ListRequest, obj machine.MachineService_ListServer) error {
	if req == nil {
		req = new(machine.ListRequest)
	}

	if !strings.HasPrefix(req.Root, OSPathSeparator) {
		// Make sure we use complete paths
		req.Root = OSPathSeparator + req.Root
	}

	req.Root = strings.TrimSuffix(req.Root, OSPathSeparator)
	if req.Root == "" {
		req.Root = "/"
	}

	var recursionDepth int

	if req.Recurse {
		if req.RecursionDepth == 0 {
			recursionDepth = -1
		} else {
			recursionDepth = int(req.RecursionDepth)
		}
	}

	opts := []archiver.WalkerOption{
		archiver.WithMaxRecurseDepth(recursionDepth),
	}

	if len(req.Types) > 0 {
		types := make([]archiver.FileType, 0, len(req.Types))

		for _, t := range req.Types {
			switch t {
			case machine.ListRequest_REGULAR:
				types = append(types, archiver.RegularFileType)
			case machine.ListRequest_DIRECTORY:
				types = append(types, archiver.DirectoryFileType)
			case machine.ListRequest_SYMLINK:
				types = append(types, archiver.SymlinkFileType)
			}
		}

		opts = append(opts, archiver.WithFileTypes(types...))
	}

	files, err := archiver.Walker(obj.Context(), req.Root, opts...)
	if err != nil {
		return err
	}

	for fi := range files {
		xattrs := []*machine.Xattr{}

		if req.ReportXattrs {
			// On filesystems such as devtmpfs and sysfs, xattrs are not supported.
			// However, we can still get the label from the security.selinux xattr for automatic labels.
			foundSelinux := false

			if list, err := xattr.List(fi.FullPath); err == nil {
				for _, attr := range list {
					if data, err := xattr.Get(fi.FullPath, attr); err == nil {
						if attr == "security.selinux" {
							foundSelinux = true
						}

						xattrs = append(xattrs, &machine.Xattr{Name: attr, Data: data})
					}
				}
			}

			if !foundSelinux {
				if data, err := xattr.Get(fi.FullPath, "security.selinux"); err == nil {
					xattrs = append(xattrs, &machine.Xattr{Name: "security.selinux", Data: data})
				}
			}
		}

		if fi.Error != nil {
			err = obj.Send(&machine.FileInfo{
				Name:         fi.FullPath,
				RelativeName: fi.RelPath,
				Error:        fi.Error.Error(),
				Xattrs:       xattrs,
			})
		} else {
			err = obj.Send(&machine.FileInfo{
				Name:         fi.FullPath,
				RelativeName: fi.RelPath,
				Size:         fi.FileInfo.Size(),
				Mode:         uint32(fi.FileInfo.Mode()),
				Modified:     fi.FileInfo.ModTime().Unix(),
				IsDir:        fi.FileInfo.IsDir(),
				Link:         fi.Link,
				Uid:          fi.FileInfo.Sys().(*syscall.Stat_t).Uid,
				Gid:          fi.FileInfo.Sys().(*syscall.Stat_t).Gid,
				Xattrs:       xattrs,
			})
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// DiskUsage implements the machine.MachineServer interface.
//
//nolint:cyclop
func (s *Server) DiskUsage(req *machine.DiskUsageRequest, obj machine.MachineService_DiskUsageServer) error { //nolint:gocyclo
	if req == nil {
		req = new(machine.DiskUsageRequest)
	}

	for _, path := range req.Paths {
		if !strings.HasPrefix(path, OSPathSeparator) {
			// Make sure we use complete paths
			path = OSPathSeparator + path
		}

		path = strings.TrimSuffix(path, OSPathSeparator)
		if path == "" {
			path = "/"
		}

		_, err := os.Stat(path)
		if err == os.ErrNotExist {
			err = obj.Send(
				&machine.DiskUsageInfo{
					Name:         path,
					RelativeName: path,
					Error:        err.Error(),
				},
			)
			if err != nil {
				return err
			}

			continue
		}

		files, err := archiver.Walker(obj.Context(), path, archiver.WithMaxRecurseDepth(-1))
		if err != nil {
			err = obj.Send(
				&machine.DiskUsageInfo{
					Name:         path,
					RelativeName: path,
					Error:        err.Error(),
				},
			)
			if err != nil {
				return err
			}

			continue
		}

		folders := map[string]*machine.DiskUsageInfo{}

		// send a record back to client if the message shouldn't be skipped
		// at the same time use record information for folder size estimation
		sendSize := func(info *machine.DiskUsageInfo, depth int32, isDir bool) error {
			prefix := strings.TrimRight(filepath.Dir(info.Name), "/")
			if folder, ok := folders[prefix]; ok {
				folder.Size += info.Size
			}

			// recursion depth check
			skip := depth >= req.RecursionDepth && req.RecursionDepth > 0
			// skip files check
			skip = skip || !isDir && !req.All
			// threshold check
			skip = skip || req.Threshold > 0 && info.Size < req.Threshold
			skip = skip || req.Threshold < 0 && info.Size > -req.Threshold

			if skip {
				return nil
			}

			return obj.Send(info)
		}

		var (
			depth     int32
			prefix    = path
			rootDepth = int32(strings.Count(path, archiver.OSPathSeparator))
		)

		// flush all folder sizes until we get to the common prefix
		flushFolders := func(prefix, nextPrefix string) error {
			for !strings.HasPrefix(nextPrefix, prefix) {
				currentDepth := int32(strings.Count(prefix, archiver.OSPathSeparator)) - rootDepth

				if folder, ok := folders[prefix]; ok {
					err = sendSize(folder, currentDepth, true)
					if err != nil {
						return err
					}

					delete(folders, prefix)
				}

				prefix = strings.TrimRight(filepath.Dir(prefix), "/")
			}

			return nil
		}

		for fi := range files {
			if fi.Error != nil {
				err = obj.Send(
					&machine.DiskUsageInfo{
						Name:         fi.FullPath,
						RelativeName: fi.RelPath,
						Error:        fi.Error.Error(),
					},
				)
			} else {
				currentDepth := int32(strings.Count(fi.FullPath, archiver.OSPathSeparator)) - rootDepth

				size := max(fi.FileInfo.Size(), 0)

				// kcore file size gives wrong value, this code should be smarter when it reads it
				// TODO: figure out better way to skip such file
				if fi.FullPath == "/proc/kcore" {
					size = 0
				}

				if fi.FileInfo.IsDir() {
					folders[strings.TrimRight(fi.FullPath, "/")] = &machine.DiskUsageInfo{
						Name:         fi.FullPath,
						RelativeName: fi.RelPath,
						Size:         size,
					}
				} else {
					err = sendSize(&machine.DiskUsageInfo{
						Name:         fi.FullPath,
						RelativeName: fi.RelPath,
						Size:         size,
					}, currentDepth, false)
					if err != nil {
						return err
					}
				}

				// depth goes down when walker gets to the next sibling folder
				if currentDepth < depth {
					nextPrefix := fi.FullPath

					if err = flushFolders(prefix, nextPrefix); err != nil {
						return err
					}

					prefix = nextPrefix
				}

				if fi.FileInfo.IsDir() {
					prefix = fi.FullPath
				}

				depth = currentDepth
			}
		}

		if path != "" {
			p := strings.TrimRight(path, "/")
			if folder, ok := folders[p]; ok {
				err = flushFolders(prefix, p)
				if err != nil {
					return err
				}

				err = sendSize(folder, 0, true)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}

	return nil
}

// Mounts implements the machine.MachineServer interface.
func (s *Server) Mounts(ctx context.Context, in *emptypb.Empty) (reply *machine.MountsResponse, err error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	//nolint:errcheck
	defer file.Close()

	var (
		stat     unix.Statfs_t
		multiErr *multierror.Error
	)

	var stats []*machine.MountStat

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())

		if len(fields) < 2 {
			continue
		}

		filesystem := fields[0]
		mountpoint := fields[1]

		var (
			totalSize  uint64
			totalAvail uint64
		)

		if statInfo, err := os.Stat(mountpoint); err == nil && statInfo.Mode().IsDir() {
			if err := unix.Statfs(mountpoint, &stat); err != nil {
				multiErr = multierror.Append(multiErr, err)
			} else {
				totalSize = uint64(stat.Bsize) * stat.Blocks
				totalAvail = uint64(stat.Bsize) * stat.Bavail
			}
		}

		stat := &machine.MountStat{
			Filesystem: filesystem,
			Size:       totalSize,
			Available:  totalAvail,
			MountedOn:  mountpoint,
		}

		stats = append(stats, stat)
	}

	if err := scanner.Err(); err != nil {
		multiErr = multierror.Append(multiErr, err)
	}

	reply = &machine.MountsResponse{
		Messages: []*machine.Mounts{
			{
				Stats: stats,
			},
		},
	}

	return reply, multiErr.ErrorOrNil()
}

// Version implements the machine.MachineServer interface.
func (s *Server) Version(ctx context.Context, in *emptypb.Empty) (reply *machine.VersionResponse, err error) {
	var platform *machine.PlatformInfo

	if s.Controller.Runtime().State().Platform() != nil {
		platform = &machine.PlatformInfo{
			Name: s.Controller.Runtime().State().Platform().Name(),
			Mode: s.Controller.Runtime().State().Platform().Mode().String(),
		}
	}

	var features *machine.FeaturesInfo

	config := s.Controller.Runtime().Config()
	if config != nil && config.Machine() != nil {
		features = &machine.FeaturesInfo{
			Rbac: true,
		}
	}

	return &machine.VersionResponse{
		Messages: []*machine.Version{
			{
				Version:  version.NewVersion(),
				Platform: platform,
				Features: features,
			},
		},
	}, nil
}

func (s *Server) routedNodeIP(ctx context.Context) (string, error) {
	nodeAddrs, err := safe.ReaderGetByID[*network.NodeAddress](ctx, s.Controller.Runtime().State().V1Alpha2().Resources(), network.NodeAddressRoutedID)
	if err != nil {
		if state.IsNotFoundError(err) {
			return "", status.Errorf(codes.FailedPrecondition, "node addresses not ready")
		}

		return "", err
	}

	ips := nodeAddrs.TypedSpec().IPs()
	if len(ips) == 0 {
		return "", status.Errorf(codes.FailedPrecondition, "no routed node addresses found")
	}

	// `proxyfrom` metadata is populated by apid from the original :authority
	// header. If that authority matches one of the routed node addresses,
	// prefer it so helper bundles advertise the client-reachable API path
	// (for example vmnet/bridged IP over QEMU slirp NAT).
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if preferred, ok := preferredRoutedNodeIP(ips, md.Get("proxyfrom")); ok {
			return preferred.String(), nil
		}
	}

	return ips[0].String(), nil
}

func preferredRoutedNodeIP(ips []netip.Addr, authorities []string) (netip.Addr, bool) {
	for _, authority := range authorities {
		host := strings.TrimSpace(authority)
		if host == "" || host == "unknown" {
			continue
		}

		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}

		host = strings.Trim(host, "[]")

		addr, err := netip.ParseAddr(host)
		if err != nil {
			continue
		}

		for _, ip := range ips {
			if ip == addr {
				return ip, true
			}
		}
	}

	return netip.Addr{}, false
}

type tarGzFile struct {
	Name string
	Mode int64
	Data []byte
}

func tarGzFiles(files ...tarGzFile) ([]byte, error) {
	var buf bytes.Buffer

	zw := gzip.NewWriter(&buf)
	tarW := tar.NewWriter(zw)

	for _, file := range files {
		mode := file.Mode
		if mode == 0 {
			mode = 0o600
		}

		err := tarW.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     file.Name,
			Size:     int64(len(file.Data)),
			ModTime:  time.Now(),
			Mode:     mode,
		})
		if err != nil {
			return nil, err
		}

		if _, err = tarW.Write(file.Data); err != nil {
			return nil, err
		}
	}

	if err := tarW.Close(); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *Server) workloadHelperClientMaterial() ([]byte, []byte, []byte, error) {
	// openwonton currently validates client certs against a frozen verifier clock.
	// Anchor helper cert validity to the OS CA notBefore so they remain valid in that window.
	const helperCertFutureWindow = 365 * 24 * time.Hour

	now := time.Now()
	secretsBundle := secrets.NewBundleFromConfig(secrets.NewFixedClock(now), s.Controller.Runtime().Config())

	caBlock, _ := pem.Decode(secretsBundle.Certs.OS.Crt)
	if caBlock == nil || caBlock.Type != "CERTIFICATE" {
		return nil, nil, nil, fmt.Errorf("failed to decode OS CA certificate for workload helper")
	}

	caCert, err := stdlibx509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse OS CA certificate for workload helper: %w", err)
	}

	helperNotBefore := caCert.NotBefore
	helperNotAfter := now.Add(helperCertFutureWindow)
	if helperNotAfter.After(caCert.NotAfter) {
		helperNotAfter = caCert.NotAfter
	}

	// Defensive fallback for malformed CA windows.
	if !helperNotAfter.After(helperNotBefore) {
		helperNotBefore = now.Add(-24 * time.Hour)
		helperNotAfter = now.Add(helperCertFutureWindow)
	}

	helperTTL := helperNotAfter.Sub(helperNotBefore)

	cert, err := secrets.NewAdminCertificateAndKey(helperNotBefore, secretsBundle.Certs.OS, role.MakeSet(role.Admin), helperTTL)
	if err != nil {
		return nil, nil, nil, err
	}

	return slices.Clone(secretsBundle.Certs.OS.Crt), slices.Clone(cert.Crt), slices.Clone(cert.Key), nil
}

func (s *Server) workloadHelperACLToken(name string) string {
	trustToken := strings.TrimSpace(s.Controller.Runtime().Config().Machine().Security().Token())
	if trustToken == "" {
		return ""
	}

	return chuboacl.WorkloadToken(trustToken, name)
}

// NomadConfig implements machine.MachineService.
func (s *Server) NomadConfig(empty *emptypb.Empty, obj machine.MachineService_NomadConfigServer) error {
	if err := s.checkControlplane("NomadConfig"); err != nil {
		return err
	}

	ip, err := s.routedNodeIP(obj.Context())
	if err != nil {
		return err
	}

	ca, crt, key, err := s.workloadHelperClientMaterial()
	if err != nil {
		return err
	}

	token := s.workloadHelperACLToken("nomad")
	addr := "https://" + net.JoinHostPort(ip, "4646")

	env := []byte(fmt.Sprintf("NOMAD_ADDR=%s\nNOMAD_CACERT=./ca.pem\nNOMAD_CLIENT_CERT=./client.pem\nNOMAD_CLIENT_KEY=./client-key.pem\nNOMAD_TOKEN=%s\n", addr, token))
	hcl := []byte(fmt.Sprintf("address = %q\nca_file = \"./ca.pem\"\nclient_cert = \"./client.pem\"\nclient_key = \"./client-key.pem\"\nsecret_id = %q\n", addr, token))
	readme := []byte("Nomad helper bundle generated by the OS API.\nopenwonton is configured for HTTPS + client certificate auth.\nThe token is required when ACLs are enabled.\n")

	tgz, err := tarGzFiles(
		tarGzFile{Name: "nomad.env", Data: env},
		tarGzFile{Name: "nomad.hcl", Data: hcl},
		tarGzFile{Name: "ca.pem", Data: ca},
		tarGzFile{Name: "client.pem", Data: crt},
		tarGzFile{Name: "client-key.pem", Data: key},
		tarGzFile{Name: "acl.token", Data: []byte(token + "\n")},
		tarGzFile{Name: "README", Mode: 0o644, Data: readme},
	)
	if err != nil {
		return err
	}

	return obj.Send(&common.Data{Bytes: tgz})
}

// ConsulConfig implements machine.MachineService.
func (s *Server) ConsulConfig(empty *emptypb.Empty, obj machine.MachineService_ConsulConfigServer) error {
	if err := s.checkControlplane("ConsulConfig"); err != nil {
		return err
	}

	ip, err := s.routedNodeIP(obj.Context())
	if err != nil {
		return err
	}

	ca, crt, key, err := s.workloadHelperClientMaterial()
	if err != nil {
		return err
	}

	token := s.workloadHelperACLToken("consul")
	addr := "https://" + net.JoinHostPort(ip, "8500")

	env := []byte(fmt.Sprintf("CONSUL_HTTP_ADDR=%s\nCONSUL_CACERT=./ca.pem\nCONSUL_CLIENT_CERT=./client.pem\nCONSUL_CLIENT_KEY=./client-key.pem\nCONSUL_HTTP_TOKEN=%s\n", addr, token))
	hcl := []byte(fmt.Sprintf("address = %q\nca_file = \"./ca.pem\"\ncert_file = \"./client.pem\"\nkey_file = \"./client-key.pem\"\ntoken = %q\n", addr, token))
	readme := []byte("Consul helper bundle generated by the OS API.\nopengyoza is configured for HTTPS + client certificate auth.\nThe token is required when ACLs are enabled.\n")

	tgz, err := tarGzFiles(
		tarGzFile{Name: "consul.env", Data: env},
		tarGzFile{Name: "consul.hcl", Data: hcl},
		tarGzFile{Name: "ca.pem", Data: ca},
		tarGzFile{Name: "client.pem", Data: crt},
		tarGzFile{Name: "client-key.pem", Data: key},
		tarGzFile{Name: "acl.token", Data: []byte(token + "\n")},
		tarGzFile{Name: "README", Mode: 0o644, Data: readme},
	)
	if err != nil {
		return err
	}

	return obj.Send(&common.Data{Bytes: tgz})
}

// OpenBaoConfig implements machine.MachineService.
func (s *Server) OpenBaoConfig(empty *emptypb.Empty, obj machine.MachineService_OpenBaoConfigServer) error {
	if err := s.checkControlplane("OpenBaoConfig"); err != nil {
		return err
	}

	ip, err := s.routedNodeIP(obj.Context())
	if err != nil {
		return err
	}

	ca, crt, key, err := s.workloadHelperClientMaterial()
	if err != nil {
		return err
	}

	addr := "http://" + net.JoinHostPort(ip, "8200")
	token := "root" // OpenBao dev-mode default token from the rendered Nomad job payload.

	// Set both prefixes for CLI compatibility across Vault/OpenBao tooling.
	env := []byte(fmt.Sprintf("VAULT_ADDR=%s\nBAO_ADDR=%s\nVAULT_CACERT=./ca.pem\nBAO_CACERT=./ca.pem\nVAULT_CLIENT_CERT=./client.pem\nBAO_CLIENT_CERT=./client.pem\nVAULT_CLIENT_KEY=./client-key.pem\nBAO_CLIENT_KEY=./client-key.pem\nVAULT_TOKEN=%s\nBAO_TOKEN=%s\n", addr, addr, token, token))
	hcl := []byte(fmt.Sprintf("address = %q\nca_cert = \"./ca.pem\"\nclient_cert = \"./client.pem\"\nclient_key = \"./client-key.pem\"\ntoken = %q\n", addr, token))
	readme := []byte("OpenBao helper bundle generated by the OS API.\nToken defaults to OpenBao dev-mode root token until OS-managed OpenBao bootstrap is implemented.\n")

	tgz, err := tarGzFiles(
		tarGzFile{Name: "openbao.env", Data: env},
		tarGzFile{Name: "openbao.hcl", Data: hcl},
		tarGzFile{Name: "ca.pem", Data: ca},
		tarGzFile{Name: "client.pem", Data: crt},
		tarGzFile{Name: "client-key.pem", Data: key},
		tarGzFile{Name: "acl.token", Data: []byte(token + "\n")},
		tarGzFile{Name: "README", Mode: 0o644, Data: readme},
	)
	if err != nil {
		return err
	}

	return obj.Send(&common.Data{Bytes: tgz})
}

// Logs provides a service or container logs can be requested and the contents of the
// log file are streamed in chunks.
func (s *Server) Logs(req *machine.LogsRequest, l machine.MachineService_LogsServer) (err error) {
	var chunk chunker.Chunker

	switch {
	case req.Namespace == constants.SystemContainerdNamespace:
		var options []runtime.LogOption

		if req.Follow {
			options = append(options, runtime.WithFollow())
		}

		if req.TailLines >= 0 {
			options = append(options, runtime.WithTailLines(int(req.TailLines)))
		}

		var logR io.ReadCloser

		logR, err = s.Controller.Runtime().Logging().ServiceLog(req.Id).Reader(options...)
		if err != nil {
			return err
		}

		//nolint:errcheck
		defer logR.Close()

		chunk = stream.NewChunker(l.Context(), logR)
	default:
		var file io.Closer

		if chunk, file, err = workloadLogs(l.Context(), req); err != nil {
			return err
		}
		//nolint:errcheck
		defer file.Close()
	}

	for data := range chunk.Read() {
		if err = l.Send(&common.Data{Bytes: data}); err != nil {
			return err
		}
	}

	return nil
}

// LogsContainers provide a list of registered log containers.
func (s *Server) LogsContainers(context.Context, *emptypb.Empty) (*machine.LogsContainersResponse, error) {
	return &machine.LogsContainersResponse{
		Messages: []*machine.LogsContainer{
			{
				Ids: s.Controller.Runtime().Logging().RegisteredLogs(),
			},
		},
	}, nil
}

func workloadLogs(ctx context.Context, req *machine.LogsRequest) (chunker.Chunker, io.Closer, error) {
	inspector, err := getContainerInspector(ctx, req.Namespace, req.Driver)
	if err != nil {
		return nil, nil, err
	}
	//nolint:errcheck
	defer inspector.Close()

	container, err := inspector.Container(req.Id)
	if err != nil {
		return nil, nil, err
	}

	if container == nil {
		return nil, nil, fmt.Errorf("container %q not found", req.Id)
	}

	return container.GetLogChunker(ctx, req.Follow, int(req.TailLines))
}

// Read implements the read API.
func (s *Server) Read(in *machine.ReadRequest, srv machine.MachineService_ReadServer) (err error) {
	stat, err := os.Stat(in.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return status.Error(codes.NotFound, err.Error())
		}

		return err
	}

	switch mode := stat.Mode(); {
	case mode.IsRegular():
		f, err := os.OpenFile(in.Path, os.O_RDONLY, 0)
		if err != nil {
			return err
		}

		defer f.Close() //nolint:errcheck

		ctx, cancel := context.WithCancel(srv.Context())
		defer cancel()

		chunker := stream.NewChunker(ctx, f)
		chunkCh := chunker.Read()

		for data := range chunkCh {
			err := srv.SendMsg(&common.Data{Bytes: data})
			if err != nil {
				cancel()
			}
		}

		return nil
	default:
		return errors.New("path must be a regular file")
	}
}

// Events streams runtime events.
//
//nolint:gocyclo
func (s *Server) Events(req *machine.EventsRequest, l machine.MachineService_EventsServer) error {
	// send an empty (hello) event to indicate to client that streaming has started
	err := sendEmptyEvent(req, l)
	if err != nil {
		return err
	}

	errCh := make(chan error)

	var opts []runtime.WatchOptionFunc

	if req.TailEvents != 0 {
		opts = append(opts, runtime.WithTailEvents(int(req.TailEvents)))
	}

	if req.TailId != "" {
		tailID, err := xid.FromString(req.TailId)
		if err != nil {
			return fmt.Errorf("error parsing tail_id: %w", err)
		}

		opts = append(opts, runtime.WithTailID(tailID))
	}

	if req.TailSeconds != 0 {
		opts = append(opts, runtime.WithTailDuration(time.Duration(req.TailSeconds)*time.Second))
	}

	if req.WithActorId != "" {
		opts = append(opts, runtime.WithActorID(req.WithActorId))
	}

	if err := s.Controller.Runtime().Events().Watch(func(events <-chan runtime.EventInfo) {
		errCh <- func() error {
			for {
				select {
				case <-s.ShutdownCtx.Done():
					return nil
				case <-l.Context().Done():
					return l.Context().Err()
				case event, ok := <-events:
					if !ok {
						return nil
					}

					msg, err := event.ToMachineEvent()
					if err != nil {
						return err
					}

					if err = l.Send(msg); err != nil {
						return err
					}
				}
			}
		}()
	}, opts...); err != nil {
		return err
	}

	return <-errCh
}

func sendEmptyEvent(req *machine.EventsRequest, l machine.MachineService_EventsServer) error {
	emptyEvent, err := pointer.To(runtime.NewEvent(nil, req.WithActorId)).ToMachineEvent()
	if err != nil {
		return err
	}

	return l.Send(emptyEvent)
}

// Containers implements the machine.MachineServer interface.
func (s *Server) Containers(ctx context.Context, in *machine.ContainersRequest) (reply *machine.ContainersResponse, err error) {
	inspector, err := getContainerInspector(ctx, in.Namespace, in.Driver)
	if err != nil {
		return nil, err
	}
	//nolint:errcheck
	defer inspector.Close()

	pods, err := inspector.Pods()
	if err != nil {
		// fatal error
		if pods == nil {
			return nil, err
		}
		// TODO: only some failed, need to handle it better via client
		log.Println(err.Error())
	}

	var containers []*machine.ContainerInfo

	for _, pod := range pods {
		for _, container := range pod.Containers {
			container := &machine.ContainerInfo{
				Namespace:        in.Namespace,
				Id:               container.Display,
				InternalId:       container.ID,
				Uid:              container.UID,
				PodId:            pod.Name,
				Name:             container.Name,
				Image:            container.Image,
				Pid:              container.Pid,
				Status:           container.Status,
				NetworkNamespace: container.NetworkNamespace,
			}
			containers = append(containers, container)
		}
	}

	reply = &machine.ContainersResponse{
		Messages: []*machine.Container{
			{
				Containers: containers,
			},
		},
	}

	return reply, nil
}

// Stats implements the machine.MachineServer interface.
func (s *Server) Stats(ctx context.Context, in *machine.StatsRequest) (reply *machine.StatsResponse, err error) {
	inspector, err := getContainerInspector(ctx, in.Namespace, in.Driver)
	if err != nil {
		return nil, err
	}
	//nolint:errcheck
	defer inspector.Close()

	pods, err := inspector.Pods()
	if err != nil {
		// fatal error
		if pods == nil {
			return nil, err
		}
		// TODO: only some failed, need to handle it better via client
		log.Println(err.Error())
	}

	var stats []*machine.Stat

	for _, pod := range pods {
		for _, container := range pod.Containers {
			if container.Metrics == nil {
				continue
			}

			stat := &machine.Stat{
				Namespace:   in.Namespace,
				Id:          container.Display,
				PodId:       pod.Name,
				Name:        container.Name,
				MemoryUsage: container.Metrics.MemoryUsage,
				CpuUsage:    container.Metrics.CPUUsage,
			}

			stats = append(stats, stat)
		}
	}

	reply = &machine.StatsResponse{
		Messages: []*machine.Stats{
			{
				Stats: stats,
			},
		},
	}

	return reply, nil
}

// Restart implements the machine.MachineServer interface.
func (s *Server) Restart(ctx context.Context, in *machine.RestartRequest) (*machine.RestartResponse, error) {
	inspector, err := getContainerInspector(ctx, in.Namespace, in.Driver)
	if err != nil {
		return nil, err
	}
	//nolint:errcheck
	defer inspector.Close()

	container, err := inspector.Container(in.Id)
	if err != nil {
		return nil, err
	}

	if container == nil {
		return nil, fmt.Errorf("container %q not found", in.Id)
	}

	err = container.Kill(syscall.SIGTERM)
	if err != nil {
		return nil, err
	}

	return &machine.RestartResponse{
		Messages: []*machine.Restart{
			{},
		},
	}, nil
}

// Dmesg implements the machine.MachineServer interface.
//
//nolint:gocyclo
func (s *Server) Dmesg(req *machine.DmesgRequest, srv machine.MachineService_DmesgServer) error {
	ctx := srv.Context()

	var options []kmsg.Option

	if req.Follow {
		options = append(options, kmsg.Follow())
	}

	if req.Tail {
		options = append(options, kmsg.FromTail())
	}

	reader, err := kmsg.NewReader(options...)
	if err != nil {
		return fmt.Errorf("error opening /dev/kmsg reader: %w", err)
	}
	defer reader.Close() //nolint:errcheck

	ch := reader.Scan(ctx)

	for {
		select {
		case <-s.ShutdownCtx.Done():
			if err = reader.Close(); err != nil {
				return err
			}
		case <-ctx.Done():
			if err = reader.Close(); err != nil {
				return err
			}
		case packet, ok := <-ch:
			if !ok {
				return nil
			}

			if packet.Err != nil {
				err = srv.Send(&common.Data{
					Metadata: &common.Metadata{
						Error: packet.Err.Error(),
					},
				})
			} else {
				msg := packet.Message
				err = srv.Send(&common.Data{
					Bytes: fmt.Appendf(nil, "%s: %7s: [%s]: %s", msg.Facility, msg.Priority, msg.Timestamp.Format(time.RFC3339Nano), msg.Message),
				})
			}

			if err != nil {
				return err
			}
		}
	}
}

// Processes implements the machine.MachineServer interface.
func (s *Server) Processes(ctx context.Context, in *emptypb.Empty) (reply *machine.ProcessesResponse, err error) {
	var processes []*machine.ProcessInfo

	procs, err := miniprocfs.NewProcesses()
	if err != nil {
		return nil, err
	}

	for {
		info, err := procs.Next()
		if err != nil {
			return nil, err
		}

		if info == nil {
			break
		}

		processes = append(processes, info)
	}

	reply = &machine.ProcessesResponse{
		Messages: []*machine.Process{
			{
				Processes: processes,
			},
		},
	}

	return reply, nil
}

// Memory implements the machine.MachineServer interface.
func (s *Server) Memory(ctx context.Context, in *emptypb.Empty) (reply *machine.MemoryResponse, err error) {
	proc, err := procfs.NewDefaultFS()
	if err != nil {
		return nil, err
	}

	info, err := proc.Meminfo()
	if err != nil {
		return nil, err
	}

	meminfo := &machine.MemInfo{
		Memtotal:          pointer.SafeDeref(info.MemTotal),
		Memfree:           pointer.SafeDeref(info.MemFree),
		Memavailable:      pointer.SafeDeref(info.MemAvailable),
		Buffers:           pointer.SafeDeref(info.Buffers),
		Cached:            pointer.SafeDeref(info.Cached),
		Swapcached:        pointer.SafeDeref(info.SwapCached),
		Active:            pointer.SafeDeref(info.Active),
		Inactive:          pointer.SafeDeref(info.Inactive),
		Activeanon:        pointer.SafeDeref(info.ActiveAnon),
		Inactiveanon:      pointer.SafeDeref(info.InactiveAnon),
		Activefile:        pointer.SafeDeref(info.ActiveFile),
		Inactivefile:      pointer.SafeDeref(info.InactiveFile),
		Unevictable:       pointer.SafeDeref(info.Unevictable),
		Mlocked:           pointer.SafeDeref(info.Mlocked),
		Swaptotal:         pointer.SafeDeref(info.SwapTotal),
		Swapfree:          pointer.SafeDeref(info.SwapFree),
		Dirty:             pointer.SafeDeref(info.Dirty),
		Writeback:         pointer.SafeDeref(info.Writeback),
		Anonpages:         pointer.SafeDeref(info.AnonPages),
		Mapped:            pointer.SafeDeref(info.Mapped),
		Shmem:             pointer.SafeDeref(info.Shmem),
		Slab:              pointer.SafeDeref(info.Slab),
		Sreclaimable:      pointer.SafeDeref(info.SReclaimable),
		Sunreclaim:        pointer.SafeDeref(info.SUnreclaim),
		Kernelstack:       pointer.SafeDeref(info.KernelStack),
		Pagetables:        pointer.SafeDeref(info.PageTables),
		Nfsunstable:       pointer.SafeDeref(info.NFSUnstable),
		Bounce:            pointer.SafeDeref(info.Bounce),
		Writebacktmp:      pointer.SafeDeref(info.WritebackTmp),
		Commitlimit:       pointer.SafeDeref(info.CommitLimit),
		Committedas:       pointer.SafeDeref(info.CommittedAS),
		Vmalloctotal:      pointer.SafeDeref(info.VmallocTotal),
		Vmallocused:       pointer.SafeDeref(info.VmallocUsed),
		Vmallocchunk:      pointer.SafeDeref(info.VmallocChunk),
		Hardwarecorrupted: pointer.SafeDeref(info.HardwareCorrupted),
		Anonhugepages:     pointer.SafeDeref(info.AnonHugePages),
		Shmemhugepages:    pointer.SafeDeref(info.ShmemHugePages),
		Shmempmdmapped:    pointer.SafeDeref(info.ShmemPmdMapped),
		Cmatotal:          pointer.SafeDeref(info.CmaTotal),
		Cmafree:           pointer.SafeDeref(info.CmaFree),
		Hugepagestotal:    pointer.SafeDeref(info.HugePagesTotal),
		Hugepagesfree:     pointer.SafeDeref(info.HugePagesFree),
		Hugepagesrsvd:     pointer.SafeDeref(info.HugePagesRsvd),
		Hugepagessurp:     pointer.SafeDeref(info.HugePagesSurp),
		Hugepagesize:      pointer.SafeDeref(info.Hugepagesize),
		Directmap4K:       pointer.SafeDeref(info.DirectMap4k),
		Directmap2M:       pointer.SafeDeref(info.DirectMap2M),
		Directmap1G:       pointer.SafeDeref(info.DirectMap1G),
	}

	reply = &machine.MemoryResponse{
		Messages: []*machine.Memory{
			{
				Meminfo: meminfo,
			},
		},
	}

	return reply, err
}

// GenerateClientConfiguration implements the machine.MachineServer interface.
func (s *Server) GenerateClientConfiguration(ctx context.Context, in *machine.GenerateClientConfigurationRequest) (*machine.GenerateClientConfigurationResponse, error) {
	if s.Controller.Runtime().Config().Machine().Type() == machinetype.TypeWorker {
		return nil, status.Error(codes.FailedPrecondition, "client configuration (chuboconfig) can't be generated on worker nodes")
	}

	crtTTL := in.CrtTtl.AsDuration()
	if crtTTL <= 0 {
		return nil, status.Error(codes.InvalidArgument, "crt_ttl should be positive")
	}

	roles, _ := role.Parse(in.Roles)

	secretsBundle := secrets.NewBundleFromConfig(secrets.NewFixedClock(time.Now()), s.Controller.Runtime().Config())

	cert, err := secretsBundle.GenerateChuboAPIClientCertificateWithTTL(roles, crtTTL)
	if err != nil {
		return nil, err
	}

	// make a nice context name
	contextName := s.Controller.Runtime().Config().Cluster().Name()
	if r := roles.Strings(); len(r) == 1 {
		contextName = strings.TrimPrefix(r[0], role.Prefix) + "@" + contextName
	}

	clientConfig := clientconfig.NewConfig(contextName, nil, secretsBundle.Certs.OS.Crt, cert)

	b, err := clientConfig.Bytes()
	if err != nil {
		return nil, err
	}

	msg := &machine.GenerateClientConfiguration{
		Ca:  secretsBundle.Certs.OS.Crt,
		Crt: cert.Crt,
		Key: cert.Key,
	}
	msg.SetChuboconfig(b)

	reply := &machine.GenerateClientConfigurationResponse{
		Messages: []*machine.GenerateClientConfiguration{msg},
	}

	return reply, nil
}

type packetStreamWriter struct {
	stream machine.MachineService_PacketCaptureServer
}

func (w *packetStreamWriter) Write(data []byte) (int, error) {
	// copy the data as the stream may not send it immediately
	data = slices.Clone(data)

	err := w.stream.Send(&common.Data{Bytes: data})
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

// PacketCapture performs packet capture and streams the pcap file.
//
//nolint:gocyclo
func (s *Server) PacketCapture(in *machine.PacketCaptureRequest, srv machine.MachineService_PacketCaptureServer) error {
	linkInfo, err := safe.StateGetResource(srv.Context(), s.Controller.Runtime().State().V1Alpha2().Resources(), network.NewLinkStatus(network.NamespaceName, in.Interface))
	if err != nil {
		if state.IsNotFoundError(err) {
			return status.Errorf(codes.NotFound, "interface %q not found", in.Interface)
		}

		return err
	}

	var linkType pcap.LinkType

	switch linkInfo.TypedSpec().Type { //nolint:exhaustive
	case nethelpers.LinkEther, nethelpers.LinkLoopbck:
		linkType = pcap.LinkTypeEthernet
	case nethelpers.LinkNone:
		linkType = pcap.LinkTypeRaw
	default:
		return status.Errorf(codes.InvalidArgument, "unsupported link type %s", linkInfo.TypedSpec().Type)
	}

	if in.SnapLen == 0 {
		in.SnapLen = afpacket.DefaultFrameSize
	}

	filter := make([]bpf.RawInstruction, 0, len(in.BpfFilter))

	for _, f := range in.BpfFilter {
		filter = append(filter, bpf.RawInstruction{
			Op: uint16(f.Op),
			Jt: uint8(f.Jt),
			Jf: uint8(f.Jf),
			K:  f.K,
		})
	}

	handle, err := afpacket.NewTPacket(
		afpacket.OptInterface(in.Interface),
		afpacket.OptPollTimeout(100*time.Millisecond),
		afpacket.OptSocketType(unix.SOCK_RAW|unix.SOCK_CLOEXEC),
	)
	if err != nil {
		return fmt.Errorf("error creating afpacket handle: %w", err)
	}

	if len(filter) > 0 {
		if err = handle.SetBPF(filter); err != nil {
			handle.Close()

			return fmt.Errorf("error setting BPF filter: %w", err)
		}
	}

	if err = handle.SetPromiscuous(in.Promiscuous); err != nil {
		handle.Close()

		return fmt.Errorf("error setting promiscuous mode %v: %w", in.Promiscuous, err)
	}

	return capturePackets(srv.Context(), &packetStreamWriter{srv}, handle, in.SnapLen, linkType)
}

//nolint:gocyclo,cyclop
func capturePackets(ctx context.Context, w io.Writer, handle *afpacket.TPacket, snapLen uint32, linkType pcap.LinkType) error {
	defer handle.Close()

	pcapw := pcap.NewWriter(w)

	if err := pcapw.WriteFileHeader(snapLen, linkType); err != nil {
		return err
	}

	defer func() {
		infoMessage := "pcap: "

		stats, errStats := handle.Stats()
		if errStats == nil {
			infoMessage += fmt.Sprintf("packets captured %d, polls %d", stats.Packets, stats.Polls)
		}

		_, socketStatsV3, socketStatsErr := handle.SocketStats()
		if socketStatsErr == nil {
			infoMessage += fmt.Sprintf(", socket stats: drops %d, packets %d, queue freezes %d", socketStatsV3.Drops(), socketStatsV3.Packets(), socketStatsV3.QueueFreezes())
		}

		log.Print(infoMessage)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		data, captureData, err := handle.ZeroCopyReadPacketData()
		if err == nil {
			if err = pcapw.WritePacket(captureData, data); err != nil {
				return err
			}

			continue
		}

		// Immediately retry for temporary network errors
		if nerr, ok := err.(net.Error); ok && nerr.Temporary() { //nolint:staticcheck
			continue
		}

		// Immediately retry for EAGAIN and poll timeout
		if errors.Is(err, syscall.EAGAIN) || errors.Is(err, afpacket.ErrTimeout) {
			continue
		}

		// Immediately break for known unrecoverable errors
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
			errors.Is(err, io.ErrNoProgress) || errors.Is(err, io.ErrClosedPipe) || errors.Is(err, io.ErrShortBuffer) ||
			errors.Is(err, syscall.EBADF) || errors.Is(err, afpacket.ErrPoll) ||
			strings.Contains(err.Error(), "use of closed file") {
			return err
		}

		time.Sleep(5 * time.Millisecond) // short sleep before retrying some errors
	}
}

// Netstat implements the machine.MachineServer interface.
func (s *Server) Netstat(ctx context.Context, req *machine.NetstatRequest) (*machine.NetstatResponse, error) {
	if req == nil {
		req = new(machine.NetstatRequest)
	}

	features := netstat.EnableFeatures{
		TCP:           req.L4Proto.Tcp,
		TCP6:          req.L4Proto.Tcp6,
		UDP:           req.L4Proto.Udp,
		UDP6:          req.L4Proto.Udp6,
		UDPLite:       req.L4Proto.Udplite,
		UDPLite6:      req.L4Proto.Udplite6,
		Raw:           req.L4Proto.Raw,
		Raw6:          req.L4Proto.Raw6,
		PID:           req.Feature.Pid,
		NoHostNetwork: !req.Netns.Hostnetwork,
		AllNetNs:      req.Netns.Allnetns,
		NetNsName:     req.Netns.Netns,
	}

	var fn netstat.AcceptFn

	switch req.Filter {
	case machine.NetstatRequest_ALL:
		fn = func(*netstat.SockTabEntry) bool { return true }
	case machine.NetstatRequest_LISTENING:
		fn = func(s *netstat.SockTabEntry) bool {
			return s.RemoteEndpoint.IP.IsUnspecified() && s.RemoteEndpoint.Port == 0
		}
	case machine.NetstatRequest_CONNECTED:
		fn = func(s *netstat.SockTabEntry) bool {
			return !s.RemoteEndpoint.IP.IsUnspecified() && s.RemoteEndpoint.Port != 0
		}
	}

	netstatResp, err := netstat.Netstat(ctx, features, fn)
	if err != nil {
		return nil, err
	}

	records := make([]*machine.ConnectRecord, len(netstatResp))

	for i, entry := range netstatResp {
		records[i] = &machine.ConnectRecord{
			L4Proto:    entry.Transport,
			Localip:    entry.LocalEndpoint.IP.String(),
			Localport:  uint32(entry.LocalEndpoint.Port),
			Remoteip:   entry.RemoteEndpoint.IP.String(),
			Remoteport: uint32(entry.RemoteEndpoint.Port),
			State:      machine.ConnectRecord_State(entry.State),
			Txqueue:    entry.TxQueue,
			Rxqueue:    entry.RxQueue,
			Tr:         machine.ConnectRecord_TimerActive(entry.Tr),
			Timerwhen:  entry.TimerWhen,
			Retrnsmt:   entry.Retrnsmt,
			Uid:        entry.UID,
			Timeout:    entry.Timeout,
			Inode:      entry.Inode,
			Ref:        entry.Ref,
			Pointer:    entry.Pointer,
			Process:    &machine.ConnectRecord_Process{},
			Netns:      entry.NetNS,
		}
		if entry.Process != nil {
			records[i].Process = &machine.ConnectRecord_Process{
				Pid:  uint32(entry.Process.Pid),
				Name: entry.Process.Name,
			}
		}
	}

	reply := &machine.NetstatResponse{
		Messages: []*machine.Netstat{
			{
				Connectrecord: records,
			},
		},
	}

	return reply, err
}
