// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package install

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/chubo-dev/chubo/pkg/machinery/client"
	"github.com/chubo-dev/chubo/pkg/machinery/compatibility"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	"github.com/chubo-dev/chubo/pkg/machinery/role"
	"github.com/chubo-dev/chubo/pkg/machinery/version"
)

// PreflightChecks runs the preflight checks.
type PreflightChecks struct {
	disabled bool
	client   *client.Client

	installerChuboVersion *compatibility.ChuboVersion
	hostChuboVersion      *compatibility.ChuboVersion
}

// NewPreflightChecks initializes and returns the installation PreflightChecks.
func NewPreflightChecks(ctx context.Context) (*PreflightChecks, error) {
	if _, err := os.Stat(constants.MachineSocketPath); err != nil {
		log.Printf("pre-flight checks disabled, as host Chubo version is too old")

		return &PreflightChecks{disabled: true}, nil //nolint:nilerr
	}

	c, err := client.New(ctx,
		client.WithUnixSocket(constants.MachineSocketPath),
		client.WithGRPCDialOptions(
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("error connecting to the machine service: %w", err)
	}

	return &PreflightChecks{
		client: c,
	}, nil
}

// Close closes the client.
func (checks *PreflightChecks) Close() error {
	if checks.disabled {
		return nil
	}

	return checks.client.Close()
}

// Run the checks, return the error if the check fails.
func (checks *PreflightChecks) Run(ctx context.Context) error {
	if checks.disabled {
		return nil
	}

	log.Printf("running pre-flight checks")

	// inject "fake" authorization
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs(constants.APIAuthzRoleMetadataKey, string(role.Admin)))

	for _, check := range []func(context.Context) error{
		checks.chuboVersion,
	} {
		if err := check(ctx); err != nil {
			return fmt.Errorf("pre-flight checks failed: %w", err)
		}
	}

	log.Printf("all pre-flight checks successful")

	return nil
}

func (checks *PreflightChecks) chuboVersion(ctx context.Context) error {
	resp, err := checks.client.Version(ctx)
	if err != nil {
		return fmt.Errorf("error getting Chubo version: %w", err)
	}

	hostVersion := unpack(resp.Messages)

	log.Printf("host Chubo version: %s", hostVersion.Version.Tag)

	checks.hostChuboVersion, err = compatibility.ParseChuboVersion(hostVersion.Version)
	if err != nil {
		return fmt.Errorf("error parsing host Chubo version: %w", err)
	}

	checks.installerChuboVersion, err = compatibility.ParseChuboVersion(version.NewVersion())
	if err != nil {
		return fmt.Errorf("error parsing installer Chubo version: %w", err)
	}

	return checks.installerChuboVersion.UpgradeableFrom(checks.hostChuboVersion)
}

func unpack[T any](s []T) T {
	if len(s) != 1 {
		panic("unpack: slice length is not 1")
	}

	return s[0]
}
