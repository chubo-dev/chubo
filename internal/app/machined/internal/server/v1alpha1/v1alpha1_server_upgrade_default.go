// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !chubo && !chuboos

package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/google/uuid"
	"go.etcd.io/etcd/client/v3/concurrency"

	runtimepkg "github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
	"github.com/chubo-dev/chubo/internal/pkg/etcd"
	"github.com/chubo-dev/chubo/internal/pkg/install"
	"github.com/chubo-dev/chubo/pkg/machinery/api/machine"
	machinetype "github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	"github.com/chubo-dev/chubo/pkg/machinery/meta"
	crires "github.com/chubo-dev/chubo/pkg/machinery/resources/cri"
)

// MinimumEtcdUpgradeLeaseLockSeconds indicates the minimum number of seconds for which we open a lease lock for upgrading Etcd nodes.
// This is not intended to lock for the duration of an upgrade.
// Rather, it is intended to make sure only one node processes the various pre-upgrade checks at a time.
// Thus, this timeout should be reflective of the expected time for the pre-upgrade checks, NOT the time to perform the upgrade itself.
const MinimumEtcdUpgradeLeaseLockSeconds = 60

// Upgrade initiates an upgrade.
//
//nolint:gocyclo
func (s *Server) Upgrade(ctx context.Context, in *machine.UpgradeRequest) (*machine.UpgradeResponse, error) {
	actorID := uuid.New().String()

	ctx = context.WithValue(ctx, runtimepkg.ActorIDCtxKey{}, actorID)

	if err := s.checkSupported(runtimepkg.Upgrade); err != nil {
		return nil, err
	}

	log.Printf("upgrade request received: staged %v, force %v, reboot mode %v", in.GetStage(), in.GetForce(), in.GetRebootMode().String())
	log.Printf("validating %q", in.GetImage())

	if err := install.PullAndValidateInstallerImage(ctx, crires.RegistryBuilder(s.Controller.Runtime().State().V1Alpha2().Resources()), in.GetImage()); err != nil {
		return nil, fmt.Errorf("error validating installer image %q: %w", in.GetImage(), err)
	}

	serviceIDs := runtimeServiceIDs(s.Controller.Runtime())

	if shouldRunEtcdUpgradePrechecks(s.Controller.Runtime().Config().Machine().Type(), in.GetForce(), serviceIDs) {
		etcdClient, err := etcd.NewClientFromControlPlaneIPs(ctx, s.Controller.Runtime().State().V1Alpha2().Resources())
		if err != nil {
			return nil, fmt.Errorf("failed to create etcd client: %w", err)
		}

		// Acquire the upgrade mutex.
		unlocker, err := tryLockUpgradeMutex(ctx, etcdClient)
		if err != nil {
			return nil, fmt.Errorf("failed to acquire upgrade mutex: %w", err)
		}

		// Unlock once API call is done, as it protects only pre-upgrade checks.
		defer unlocker()

		if err = etcdClient.ValidateForUpgrade(ctx, s.Controller.Runtime().Config()); err != nil {
			return nil, fmt.Errorf("error validating etcd for upgrade: %w", err)
		}
	} else if !in.GetForce() && s.Controller.Runtime().Config().Machine().Type() != machinetype.TypeWorker {
		log.Printf("skipping etcd upgrade validation: etcd service is not registered")
	}

	runCtx := context.WithValue(context.Background(), runtimepkg.ActorIDCtxKey{}, actorID)

	if in.GetStage() {
		if ok, err := s.Controller.Runtime().State().Machine().Meta().SetTag(ctx, meta.StagedUpgradeImageRef, in.GetImage()); !ok || err != nil {
			return nil, fmt.Errorf("error adding staged upgrade image ref tag: %w", err)
		}

		opts := install.DefaultInstallOptions()
		if err := opts.Apply(install.OptionsFromUpgradeRequest(s.Controller.Runtime(), in)...); err != nil {
			return nil, fmt.Errorf("error applying install options: %w", err)
		}

		serialized, err := json.Marshal(opts)
		if err != nil {
			return nil, fmt.Errorf("error serializing install options: %s", err)
		}

		var ok bool

		if ok, err = s.Controller.Runtime().State().Machine().Meta().SetTag(ctx, meta.StagedUpgradeInstallOptions, string(serialized)); !ok || err != nil {
			return nil, fmt.Errorf("error adding staged upgrade install options tag: %w", err)
		}

		if err = s.Controller.Runtime().State().Machine().Meta().Flush(); err != nil {
			return nil, fmt.Errorf("error writing meta: %w", err)
		}

		go func() {
			if err := s.Controller.Run(runCtx, runtimepkg.SequenceStageUpgrade, in); err != nil {
				if !runtimepkg.IsRebootError(err) {
					log.Println("reboot for staged upgrade failed:", err)
				}
			}
		}()
	} else {
		go func() {
			if err := s.Controller.Run(runCtx, runtimepkg.SequenceUpgrade, in); err != nil {
				if !runtimepkg.IsRebootError(err) {
					log.Println("upgrade failed:", err)
				}
			}
		}()
	}

	return &machine.UpgradeResponse{
		Messages: []*machine.Upgrade{
			{
				Ack:     "Upgrade request received",
				ActorId: actorID,
			},
		},
	}, nil
}

func shouldRunEtcdUpgradePrechecks(machineType machinetype.Type, force bool, serviceIDs []string) bool {
	if force {
		return false
	}

	if machineType == machinetype.TypeWorker {
		return false
	}

	return slices.Contains(serviceIDs, "etcd")
}

func tryLockUpgradeMutex(ctx context.Context, etcdClient *etcd.Client) (unlock func(), err error) {
	sess, err := concurrency.NewSession(etcdClient.Client,
		concurrency.WithContext(ctx),
		concurrency.WithTTL(MinimumEtcdUpgradeLeaseLockSeconds),
	)
	if err != nil {
		return nil, fmt.Errorf("error establishing etcd concurrency session: %w", err)
	}

	mu := concurrency.NewMutex(sess, constants.EtcdTalosEtcdUpgradeMutex)

	if err = mu.TryLock(ctx); err != nil {
		return nil, fmt.Errorf("error trying to lock etcd upgrade mutex: %w", err)
	}

	log.Printf("etcd upgrade mutex locked with session ID %08x", sess.Lease())

	return func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := mu.Unlock(unlockCtx); err != nil {
			log.Printf("error unlocking etcd upgrade mutex: %v", err)
		}

		if err := sess.Close(); err != nil {
			log.Printf("error closing etcd upgrade mutex session: %v", err)
		}

		log.Printf("etcd upgrade mutex unlocked and session closed")
	}, nil
}
