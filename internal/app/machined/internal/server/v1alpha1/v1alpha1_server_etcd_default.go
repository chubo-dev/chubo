// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo && !chuboos

package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/siderolabs/gen/xslices"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/siderolabs/talos/internal/pkg/etcd"
	"github.com/siderolabs/talos/pkg/chunker/stream"
	"github.com/siderolabs/talos/pkg/machinery/api/common"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	"github.com/siderolabs/talos/pkg/machinery/nethelpers"
	etcdresource "github.com/siderolabs/talos/pkg/machinery/resources/etcd"
)

// EtcdMemberList implements the machine.MachineServer interface.
func (s *Server) EtcdMemberList(ctx context.Context, in *machine.EtcdMemberListRequest) (*machine.EtcdMemberListResponse, error) {
	if err := s.checkControlplaneService("member list", "etcd"); err != nil {
		return nil, err
	}

	var (
		client *etcd.Client
		err    error
	)

	if in.QueryLocal {
		client, err = etcd.NewLocalClient(ctx)
	} else {
		client, err = etcd.NewClientFromControlPlaneIPs(ctx, s.Controller.Runtime().State().V1Alpha2().Resources())
	}

	if err != nil {
		return nil, err
	}

	//nolint:errcheck
	defer client.Close()

	ctx = clientv3.WithRequireLeader(ctx)

	resp, err := client.MemberList(ctx)
	if err != nil {
		return nil, err
	}

	return &machine.EtcdMemberListResponse{
		Messages: []*machine.EtcdMembers{
			{
				LegacyMembers: xslices.Map(resp.Members, (*etcdserverpb.Member).GetName),
				Members: xslices.Map(resp.Members, func(member *etcdserverpb.Member) *machine.EtcdMember {
					return &machine.EtcdMember{
						Id:         member.GetID(),
						Hostname:   member.GetName(),
						PeerUrls:   member.GetPeerURLs(),
						ClientUrls: member.GetClientURLs(),
						IsLearner:  member.GetIsLearner(),
					}
				}),
			},
		},
	}, nil
}

// EtcdRemoveMemberByID implements the machine.MachineServer interface.
func (s *Server) EtcdRemoveMemberByID(ctx context.Context, in *machine.EtcdRemoveMemberByIDRequest) (*machine.EtcdRemoveMemberByIDResponse, error) {
	if err := s.checkControlplaneService("etcd remove member", "etcd"); err != nil {
		return nil, err
	}

	client, err := etcd.NewClientFromControlPlaneIPs(ctx, s.Controller.Runtime().State().V1Alpha2().Resources())
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	defer client.Close() //nolint:errcheck

	ctx = clientv3.WithRequireLeader(ctx)

	if err = client.RemoveMemberByMemberID(ctx, in.MemberId); err != nil {
		if errors.Is(err, rpctypes.ErrMemberNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		return nil, fmt.Errorf("failed to remove member: %w", err)
	}

	return &machine.EtcdRemoveMemberByIDResponse{
		Messages: []*machine.EtcdRemoveMemberByID{
			{},
		},
	}, nil
}

// EtcdLeaveCluster implements the machine.MachineServer interface.
func (s *Server) EtcdLeaveCluster(ctx context.Context, in *machine.EtcdLeaveClusterRequest) (*machine.EtcdLeaveClusterResponse, error) {
	if err := s.checkControlplaneService("etcd leave", "etcd"); err != nil {
		return nil, err
	}

	client, err := etcd.NewClientFromControlPlaneIPs(ctx, s.Controller.Runtime().State().V1Alpha2().Resources())
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	defer client.Close() //nolint:errcheck

	ctx = clientv3.WithRequireLeader(ctx)

	if err = client.LeaveCluster(ctx, s.Controller.Runtime().State().V1Alpha2().Resources()); err != nil {
		return nil, fmt.Errorf("failed to leave cluster: %w", err)
	}

	return &machine.EtcdLeaveClusterResponse{
		Messages: []*machine.EtcdLeaveCluster{
			{},
		},
	}, nil
}

// EtcdForfeitLeadership implements the machine.MachineServer interface.
func (s *Server) EtcdForfeitLeadership(ctx context.Context, in *machine.EtcdForfeitLeadershipRequest) (*machine.EtcdForfeitLeadershipResponse, error) {
	if err := s.checkControlplaneService("etcd forfeit leadership", "etcd"); err != nil {
		return nil, err
	}

	client, err := etcd.NewClientFromControlPlaneIPs(ctx, s.Controller.Runtime().State().V1Alpha2().Resources())
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	defer client.Close() //nolint:errcheck

	ctx = clientv3.WithRequireLeader(ctx)

	memberID, err := etcd.GetLocalMemberID(ctx, s.Controller.Runtime().State().V1Alpha2().Resources())
	if err != nil {
		return nil, err
	}

	leader, err := client.ForfeitLeadership(ctx, etcdresource.FormatMemberID(memberID))
	if err != nil {
		return nil, fmt.Errorf("failed to forfeit leadership: %w", err)
	}

	return &machine.EtcdForfeitLeadershipResponse{
		Messages: []*machine.EtcdForfeitLeadership{
			{
				Member: leader,
			},
		},
	}, nil
}

// EtcdSnapshot implements the machine.MachineServer interface.
func (s *Server) EtcdSnapshot(in *machine.EtcdSnapshotRequest, srv machine.MachineService_EtcdSnapshotServer) error {
	if err := s.checkControlplaneService("etcd snapshot", "etcd"); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	client, err := etcd.NewLocalClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}

	//nolint:errcheck
	defer client.Close()

	rd, err := client.Snapshot(srv.Context())
	if err != nil {
		return fmt.Errorf("failed reading etcd snapshot: %w", err)
	}

	chunker := stream.NewChunker(ctx, rd)
	chunkCh := chunker.Read()

	for data := range chunkCh {
		err := srv.SendMsg(&common.Data{Bytes: data})
		if err != nil {
			cancel()

			return err
		}
	}

	return nil
}

// EtcdRecover implements the machine.MachineServer interface.
//
//nolint:gocyclo
func (s *Server) EtcdRecover(srv machine.MachineService_EtcdRecoverServer) error {
	if _, err := os.Stat(filepath.Dir(constants.EtcdRecoverySnapshotPath)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return status.Error(codes.FailedPrecondition, "etcd service is not ready for recovery yet")
		}

		return err
	}

	if err := s.checkControlplaneService("etcd recover", "etcd"); err != nil {
		return err
	}

	snapshot, err := os.OpenFile(constants.EtcdRecoverySnapshotPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
	if err != nil {
		return fmt.Errorf("error creating etcd recovery snapshot: %w", err)
	}

	defer snapshot.Close() //nolint:errcheck

	successfulUpload := false

	defer func() {
		if !successfulUpload {
			os.Remove(snapshot.Name()) //nolint:errcheck
		}
	}()

	for {
		var msg *common.Data

		msg, err = srv.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}

			return err
		}

		_, err = snapshot.Write(msg.Bytes)
		if err != nil {
			return fmt.Errorf("error writing snapshot: %w", err)
		}
	}

	if err = snapshot.Sync(); err != nil {
		return fmt.Errorf("error fsyncing snapshot: %w", err)
	}

	if err = snapshot.Close(); err != nil {
		return fmt.Errorf("error closing snapshot: %w", err)
	}

	successfulUpload = true

	return srv.SendAndClose(&machine.EtcdRecoverResponse{
		Messages: []*machine.EtcdRecover{
			{},
		},
	})
}

func mapAlarms(alarms []*etcdserverpb.AlarmMember) []*machine.EtcdMemberAlarm {
	mapAlarmType := func(alarmType etcdserverpb.AlarmType) machine.EtcdMemberAlarm_AlarmType {
		switch alarmType {
		case etcdserverpb.AlarmType_NOSPACE:
			return machine.EtcdMemberAlarm_NOSPACE
		case etcdserverpb.AlarmType_CORRUPT:
			return machine.EtcdMemberAlarm_CORRUPT
		case etcdserverpb.AlarmType_NONE:
			return machine.EtcdMemberAlarm_NONE
		default:
			return machine.EtcdMemberAlarm_NONE
		}
	}

	return xslices.Map(alarms, func(alarm *etcdserverpb.AlarmMember) *machine.EtcdMemberAlarm {
		return &machine.EtcdMemberAlarm{
			MemberId: alarm.MemberID,
			Alarm:    mapAlarmType(alarm.Alarm),
		}
	})
}

// EtcdAlarmList lists etcd alarms for the current node.
//
// This method is available only on control plane nodes (which run etcd).
func (s *Server) EtcdAlarmList(ctx context.Context, in *emptypb.Empty) (*machine.EtcdAlarmListResponse, error) {
	if err := s.checkControlplaneService("etcd alarm list", "etcd"); err != nil {
		return nil, err
	}

	client, err := etcd.NewLocalClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	//nolint:errcheck
	defer client.Close()

	resp, err := client.AlarmList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list etcd alarms: %w", err)
	}

	return &machine.EtcdAlarmListResponse{
		Messages: []*machine.EtcdAlarm{
			{
				MemberAlarms: mapAlarms(resp.Alarms),
			},
		},
	}, nil
}

// EtcdAlarmDisarm disarms etcd alarms for the current node.
//
// This method is available only on control plane nodes (which run etcd).
func (s *Server) EtcdAlarmDisarm(ctx context.Context, in *emptypb.Empty) (*machine.EtcdAlarmDisarmResponse, error) {
	if err := s.checkControlplaneService("etcd alarm list", "etcd"); err != nil {
		return nil, err
	}

	client, err := etcd.NewLocalClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	//nolint:errcheck
	defer client.Close()

	resp, err := client.AlarmDisarm(ctx, &clientv3.AlarmMember{})
	if err != nil {
		return nil, fmt.Errorf("failed to disarm etcd alarm: %w", err)
	}

	return &machine.EtcdAlarmDisarmResponse{
		Messages: []*machine.EtcdAlarmDisarm{
			{
				MemberAlarms: mapAlarms(resp.Alarms),
			},
		},
	}, nil
}

// EtcdDefragment defragments etcd data directory for the current node.
//
// Defragmentation is a resource-heavy operation, so it should only run on a specific
// node.
//
// This method is available only on control plane nodes (which run etcd).
func (s *Server) EtcdDefragment(ctx context.Context, in *emptypb.Empty) (*machine.EtcdDefragmentResponse, error) {
	if err := s.checkControlplaneService("etcd defragment", "etcd"); err != nil {
		return nil, err
	}

	client, err := etcd.NewLocalClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	//nolint:errcheck
	defer client.Close()

	_, err = client.Defragment(ctx, nethelpers.JoinHostPort("localhost", constants.EtcdClientPort))
	if err != nil {
		return nil, fmt.Errorf("failed to defragment etcd: %w", err)
	}

	return &machine.EtcdDefragmentResponse{
		Messages: []*machine.EtcdDefragment{
			{},
		},
	}, nil
}

// EtcdStatus returns etcd status for the member of the cluster.
//
// This method is available only on control plane nodes (which run etcd).
func (s *Server) EtcdStatus(ctx context.Context, in *emptypb.Empty) (*machine.EtcdStatusResponse, error) {
	if err := s.checkControlplaneService("etcd status", "etcd"); err != nil {
		return nil, err
	}

	client, err := etcd.NewLocalClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	//nolint:errcheck
	defer client.Close()

	resp, err := client.Status(ctx, nethelpers.JoinHostPort("localhost", constants.EtcdClientPort))
	if err != nil {
		return nil, fmt.Errorf("failed to query etcd status: %w", err)
	}

	storageVersion := resp.StorageVersion
	// NOTE: this field is only filled on >3.6.0, thus we need a workaround for previous ETCD versions
	if storageVersion == "" {
		if v, err := semver.Parse(resp.Version); err == nil {
			storageVersion = fmt.Sprintf("%d.%d.0", v.Major, v.Minor)
		} else {
			// we swallow the error here, as we don't want to fail the request
			// over something that is not critical
			storageVersion = "unknown"
		}
	}

	return &machine.EtcdStatusResponse{
		Messages: []*machine.EtcdStatus{
			{
				MemberStatus: &machine.EtcdMemberStatus{
					MemberId:         resp.Header.MemberId,
					ProtocolVersion:  resp.Version,
					DbSize:           resp.DbSize,
					DbSizeInUse:      resp.DbSizeInUse,
					Leader:           resp.Leader,
					RaftIndex:        resp.RaftIndex,
					RaftTerm:         resp.RaftTerm,
					RaftAppliedIndex: resp.RaftAppliedIndex,
					StorageVersion:   storageVersion,
					Errors:           resp.Errors,
					IsLearner:        resp.IsLearner,
				},
			},
		},
	}, nil
}

// EtcdDowngradeCancel cancels etcd cluster downgrade that is in progress.
//
// This method is available only on control plane nodes (which run etcd).
func (s *Server) EtcdDowngradeCancel(ctx context.Context, _ *emptypb.Empty) (*machine.EtcdDowngradeCancelResponse, error) {
	if err := s.checkControlplaneService("etcd downgrade cancel", "etcd"); err != nil {
		return nil, err
	}

	client, err := etcd.NewLocalClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	//nolint:errcheck
	defer client.Close()

	resp, err := client.Downgrade(ctx, clientv3.DowngradeCancel, "")
	if err != nil {
		return nil, fmt.Errorf("failed to query etcd status: %w", err)
	}

	return &machine.EtcdDowngradeCancelResponse{
		Messages: []*machine.EtcdDowngradeCancel{
			{
				ClusterDowngrade: &machine.EtcdClusterDowngrade{
					ClusterVersion: resp.Version,
				},
			},
		},
	}, nil
}

// EtcdDowngradeEnable enables etcd cluster downgrade to a specific version.
//
// This method is available only on control plane nodes (which run etcd).
//
//nolint:dupl
func (s *Server) EtcdDowngradeEnable(ctx context.Context, in *machine.EtcdDowngradeEnableRequest) (*machine.EtcdDowngradeEnableResponse, error) {
	if err := s.checkControlplaneService("etcd downgrade cancel", "etcd"); err != nil {
		return nil, err
	}

	if err := validateDowngrade(in.Version); err != nil {
		return nil, err
	}

	client, err := etcd.NewLocalClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	//nolint:errcheck
	defer client.Close()

	resp, err := client.Downgrade(ctx, clientv3.DowngradeEnable, in.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to query etcd status: %w", err)
	}

	return &machine.EtcdDowngradeEnableResponse{
		Messages: []*machine.EtcdDowngradeEnable{
			{
				ClusterDowngrade: &machine.EtcdClusterDowngrade{
					ClusterVersion: resp.Version,
				},
			},
		},
	}, nil
}

// EtcdDowngradeValidate validates etcd cluster for downgrade to a specific version.
//
// This method is available only on control plane nodes (which run etcd).
//
//nolint:dupl
func (s *Server) EtcdDowngradeValidate(ctx context.Context, in *machine.EtcdDowngradeValidateRequest) (*machine.EtcdDowngradeValidateResponse, error) {
	if err := s.checkControlplaneService("etcd downgrade cancel", "etcd"); err != nil {
		return nil, err
	}

	if err := validateDowngrade(in.Version); err != nil {
		return nil, err
	}

	client, err := etcd.NewLocalClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	//nolint:errcheck
	defer client.Close()

	resp, err := client.Downgrade(ctx, clientv3.DowngradeValidate, in.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to query etcd status: %w", err)
	}

	return &machine.EtcdDowngradeValidateResponse{
		Messages: []*machine.EtcdDowngradeValidate{
			{
				ClusterDowngrade: &machine.EtcdClusterDowngrade{
					ClusterVersion: resp.Version,
				},
			},
		},
	}, nil
}

var minEtcdDowngradeVersion = semver.Version{Major: 3, Minor: 5}

func validateDowngrade(version string) error {
	if version == "" {
		return status.Error(codes.InvalidArgument, "version is required for etcd downgrade")
	}

	parts := strings.Split(version, ".")
	if len(parts) != 2 {
		return status.Error(codes.InvalidArgument, "version should be in MAJOR.MINOR format")
	}

	major, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return status.Error(codes.InvalidArgument, "major version should be a number")
	}

	minor, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return status.Error(codes.InvalidArgument, "minor version should be a number")
	}

	semverVersion := semver.Version{Major: uint64(major), Minor: uint64(minor)}
	if semverVersion.LT(minEtcdDowngradeVersion) {
		return status.Error(codes.InvalidArgument, "etcd downgrade is only supported to 3.5 and later versions")
	}

	return nil
}
