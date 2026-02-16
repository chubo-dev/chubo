// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package secrets

import (
	"context"
	"net/netip"

	"github.com/cosi-project/runtime/pkg/controller"
	"github.com/cosi-project/runtime/pkg/controller/generic/transform"
	"github.com/siderolabs/crypto/x509"
	"github.com/siderolabs/gen/optional"
	"go.uber.org/zap"

	"github.com/chubo-dev/chubo/pkg/machinery/resources/config"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/secrets"
)

// RootOSController manages secrets.OSRoot based on configuration.
type RootOSController = transform.Controller[*config.MachineConfig, *secrets.OSRoot]

// NewRootOSController instanciates the controller.
func NewRootOSController() *RootOSController {
	return transform.NewController(
		transform.Settings[*config.MachineConfig, *secrets.OSRoot]{
			Name: "secrets.RootOSController",
			// Talos upstream always has a Cluster config, but `chubo` intentionally
			// synthesizes a minimal config which may have Cluster() == nil.
			// OS root secrets only depend on Machine().Security(), so do not gate on Cluster().
			MapMetadataOptionalFunc: func(cfg *config.MachineConfig) optional.Optional[*secrets.OSRoot] {
				if cfg.Metadata().ID() != config.ActiveID {
					return optional.None[*secrets.OSRoot]()
				}

				if cfg.Config().Machine() == nil {
					return optional.None[*secrets.OSRoot]()
				}

				return optional.Some(secrets.NewOSRoot(secrets.OSRootID))
			},
			TransformFunc: func(ctx context.Context, r controller.Reader, logger *zap.Logger, cfg *config.MachineConfig, res *secrets.OSRoot) error {
				cfgProvider := cfg.Config()
				osSecrets := res.TypedSpec()

				osSecrets.IssuingCA = cfgProvider.Machine().Security().IssuingCA()
				osSecrets.AcceptedCAs = cfgProvider.Machine().Security().AcceptedCAs()

				if osSecrets.IssuingCA != nil {
					osSecrets.AcceptedCAs = append(osSecrets.AcceptedCAs, &x509.PEMEncodedCertificate{
						Crt: osSecrets.IssuingCA.Crt,
					})

					if len(osSecrets.IssuingCA.Key) == 0 {
						// drop incomplete issuing CA, as the machine config for workers contains just the cert
						osSecrets.IssuingCA = nil
					}
				}

				osSecrets.CertSANIPs = nil
				osSecrets.CertSANDNSNames = nil

				for _, san := range cfgProvider.Machine().Security().CertSANs() {
					if ip, err := netip.ParseAddr(san); err == nil {
						osSecrets.CertSANIPs = append(osSecrets.CertSANIPs, ip)
					} else {
						osSecrets.CertSANDNSNames = append(osSecrets.CertSANDNSNames, san)
					}
				}

				osSecrets.Token = cfgProvider.Machine().Security().Token()

				return nil
			},
		},
	)
}
