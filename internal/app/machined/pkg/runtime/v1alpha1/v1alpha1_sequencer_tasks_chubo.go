// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package v1alpha1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime/v1alpha1/internal/opengyozaleave"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime/v1alpha1/internal/opengyozaquorum"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime/v1alpha1/internal/openwontondrain"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime/v1alpha1/internal/openwontonleave"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/services"
	chuboacl "github.com/chubo-dev/chubo/pkg/chubo/acl"
	"github.com/chubo-dev/chubo/pkg/machinery/meta"
)

// Chubo OS doesn't ship legacy control-plane services.
//
// Keep the Chubo sequencer API shape for now (upgrade/reset entry points call these tasks),
// but implement them in terms of OpenWonton/OpenGyoza.

const (
	openGyozaHTTPAddress         = "https://127.0.0.1:8500"
	openGyozaRolePath            = "/var/lib/chubo/config/opengyoza.role"
	openWontonHTTPAddress        = "https://127.0.0.1:4646"
	openWontonRolePath           = "/var/lib/chubo/config/openwonton.role"
	openWontonDrainDeadline      = 10 * time.Minute
	openWontonDefaultHTTPTimeout = 5 * time.Second
)

func CordonAndDrainNode(_ runtime.Sequence, in any) (runtime.TaskExecutionFunc, string) {
	type forceGetter interface {
		GetForce() bool
	}

	return func(ctx context.Context, logger *log.Logger, r runtime.Runtime) error {
		// Treat opengyoza quorum checks like legacy health checks: blocking by default,
		// skippable only when the caller explicitly forces the operation.
		force := false
		if fg, ok := in.(forceGetter); ok {
			force = fg.GetForce()
		}

		openGyozaRole, openGyozaConfigured, err := opengyozaquorum.ReadRole(openGyozaRolePath)
		if err != nil {
			logger.Printf("skipping opengyoza quorum check: failed to read role: %v", err)
		}

		if openGyozaConfigured && opengyozaquorum.IsServerRole(openGyozaRole) {
			trustToken := ""
			if r.Config() != nil && r.Config().Machine() != nil && r.Config().Machine().Security() != nil {
				trustToken = strings.TrimSpace(r.Config().Machine().Security().Token())
			}
			consulToken := chuboacl.WorkloadToken(trustToken, "consul")

			if force {
				logger.Printf("skipping opengyoza quorum check: forced operation")
				return nil
			}

			peersOverrideJSON, ok := r.State().Machine().Meta().ReadTag(meta.ChuboOpenGyozaPeersOverride)
			peersOverrideJSON = strings.TrimSpace(peersOverrideJSON)

			if ok && peersOverrideJSON != "" {
				var peers []string

				if uerr := json.Unmarshal([]byte(peersOverrideJSON), &peers); uerr != nil {
					err = fmt.Errorf("failed to decode opengyoza peers override meta: %w", uerr)
				} else {
					err = opengyozaquorum.CheckSafeServerStopFromPeers(peers)
				}
			} else {
				client, err := services.NewChuboServiceHTTPClient(services.OpenGyozaServiceID, openWontonDefaultHTTPTimeout)
				if err != nil {
					return fmt.Errorf("failed to create opengyoza HTTP client: %w", err)
				}

				// Retry briefly so transient readiness issues (e.g. local agent not yet listening) don't
				// silently skip the check on the first connection attempt.
				var lastErr error
				for attempt := 0; attempt < 20; attempt++ {
					lastErr = opengyozaquorum.CheckSafeServerStopWithToken(ctx, client, openGyozaHTTPAddress, consulToken)
					if lastErr == nil || errors.Is(lastErr, opengyozaquorum.ErrUnsafeServerStop) {
						break
					}

					if ctx.Err() != nil {
						break
					}

					time.Sleep(100 * time.Millisecond)
				}

				err = lastErr
			}

			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}

				logger.Printf("opengyoza quorum check failed: %v", err)
				return fmt.Errorf("opengyoza quorum check failed: %w", err)
			}

			logger.Printf("opengyoza quorum check passed")
		}

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

		client, err := services.NewChuboServiceHTTPClient(services.OpenWontonServiceID, openWontonDefaultHTTPTimeout)
		if err != nil {
			logger.Printf("skipping openwonton drain: failed to create HTTP client: %v", err)
			return nil
		}

		trustToken := ""
		if r.Config() != nil && r.Config().Machine() != nil && r.Config().Machine().Security() != nil {
			trustToken = strings.TrimSpace(r.Config().Machine().Security().Token())
		}
		nomadToken := chuboacl.WorkloadToken(trustToken, "nomad")

		nodeName, err := r.NodeName()
		if err != nil || strings.TrimSpace(nodeName) == "" {
			nodeName, _ = os.Hostname() //nolint:errcheck
		}

		if err := openwontondrain.DrainNodeWithToken(ctx, client, openWontonHTTPAddress, nodeName, openWontonDrainDeadline, nomadToken); err != nil {
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

func LeaveClusterMembership(runtime.Sequence, any) (runtime.TaskExecutionFunc, string) {
	return func(ctx context.Context, logger *log.Logger, r runtime.Runtime) error {
		// On reset, best-effort leave the Chubo control plane so peers don't retain stale
		// membership records (analogous to legacy leave behavior).
		trustToken := ""
		if r.Config() != nil && r.Config().Machine() != nil && r.Config().Machine().Security() != nil {
			trustToken = strings.TrimSpace(r.Config().Machine().Security().Token())
		}

		nomadToken := chuboacl.WorkloadToken(trustToken, "nomad")
		consulToken := chuboacl.WorkloadToken(trustToken, "consul")

		nodeName, err := r.NodeName()
		if err != nil || strings.TrimSpace(nodeName) == "" {
			nodeName, _ = os.Hostname() //nolint:errcheck
		}

		role, configured, err := openwontondrain.ReadRole(openWontonRolePath)
		if err != nil {
			logger.Printf("skipping openwonton leave: failed to read role: %v", err)
		} else if configured {
			client, err := services.NewChuboServiceHTTPClient(services.OpenWontonServiceID, openWontonDefaultHTTPTimeout)
			if err != nil {
				logger.Printf("skipping openwonton leave: failed to create HTTP client: %v", err)
			} else {
				leaveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()

				clientRole := openwontonleave.IsClientRole(role)
				serverRole := openwontonleave.IsServerRole(role)

				if !clientRole && !serverRole {
					logger.Printf("skipping openwonton leave: unknown role=%q", strings.TrimSpace(role))
				}

				if clientRole {
					if err := openwontonleave.PurgeNodeWithToken(leaveCtx, client, openWontonHTTPAddress, nodeName, nomadToken); err != nil {
						logger.Printf("openwonton node purge skipped: %v", err)
					} else {
						logger.Printf("requested openwonton node purge for %q", nodeName)
					}
				}

				if serverRole {
					if err := openwontonleave.RemoveServerPeerWithToken(leaveCtx, client, openWontonHTTPAddress, nodeName, nomadToken); err != nil {
						logger.Printf("openwonton raft peer removal skipped: %v", err)
					} else {
						logger.Printf("requested openwonton raft peer removal for %q", nodeName)
					}
				}
			}
		}

		role, configured, err = opengyozaquorum.ReadRole(openGyozaRolePath)
		if err != nil {
			logger.Printf("skipping opengyoza leave: failed to read role: %v", err)
		} else if configured {
			client, err := services.NewChuboServiceHTTPClient(services.OpenGyozaServiceID, openWontonDefaultHTTPTimeout)
			if err != nil {
				logger.Printf("skipping opengyoza leave: failed to create HTTP client: %v", err)
			} else {
				leaveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()

				if err := opengyozaleave.LeaveWithToken(leaveCtx, client, openGyozaHTTPAddress, consulToken); err != nil {
					logger.Printf("opengyoza agent leave skipped: %v", err)
				} else {
					logger.Printf("requested opengyoza agent leave")
				}
			}
		}

		return nil
	}, "leaveClusterMembership"
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
