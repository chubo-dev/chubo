// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"

	runtimepkg "github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
	"github.com/chubo-dev/chubo/internal/pkg/install"
	"github.com/chubo-dev/chubo/pkg/machinery/api/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/meta"
	crires "github.com/chubo-dev/chubo/pkg/machinery/resources/cri"
)

// Upgrade initiates an upgrade.
//
// Chubo mode skips legacy control-plane upgrade prechecks.
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
