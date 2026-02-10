// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chuboos

//docgen:jsonschema

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/siderolabs/crypto/x509"
	"github.com/siderolabs/gen/optional"

	"github.com/siderolabs/talos/pkg/machinery/config/config"
	"github.com/siderolabs/talos/pkg/machinery/config/internal/registry"
	"github.com/siderolabs/talos/pkg/machinery/config/machine"
	"github.com/siderolabs/talos/pkg/machinery/config/types/meta"
	"github.com/siderolabs/talos/pkg/machinery/config/types/v1alpha1"
	"github.com/siderolabs/talos/pkg/machinery/config/validation"
)

// MachineConfigKind is the `chuboos` minimal machine config document kind.
const MachineConfigKind = "MachineConfig"

// MachineConfigAPIVersion is the API version string for the minimal machine config.
//
// Note: Talos upstream uses plain "v1alpha1" for many config documents, but we
// intentionally scope this to a group-like string to avoid collisions with Talos kinds.
const MachineConfigAPIVersion = "chubo.dev/v1alpha1"

func init() {
	registry.Register(MachineConfigKind, func(version string) config.Document {
		switch version {
		case MachineConfigAPIVersion:
			return &MachineConfigV1Alpha1{}
		default:
			return nil
		}
	})
}

// Check interfaces.
var (
	_ config.Document  = &MachineConfigV1Alpha1{}
	_ config.Validator = &MachineConfigV1Alpha1{}
)

// MachineConfigV1Alpha1 is the minimal, non-Kubernetes config document for the `chuboos` fork.
//
// schemaRoot: true
// schemaMeta: chubo.dev/v1alpha1/MachineConfig
type MachineConfigV1Alpha1 struct {
	meta.Meta `yaml:",inline"`

	Metadata MachineConfigMetadata `yaml:"metadata,omitempty"`
	Spec     MachineConfigSpec     `yaml:"spec"`
}

type MachineConfigMetadata struct {
	// Optional stable node ID.
	ID string `yaml:"id,omitempty"`
}

type MachineConfigSpec struct {
	Install   *InstallSpec   `yaml:"install,omitempty"`
	Network   *NetworkSpec   `yaml:"network,omitempty"`
	Time      *TimeSpec      `yaml:"time,omitempty"`
	Logging   *LoggingSpec   `yaml:"logging,omitempty"`
	Trust     *TrustSpec     `yaml:"trust,omitempty"`
	Registry  *RegistrySpec  `yaml:"registry,omitempty"`
	Modules   *ModulesSpec   `yaml:"modules,omitempty"`
	Bootstrap *BootstrapSpec `yaml:"bootstrap,omitempty"`
}

type InstallSpec struct {
	Disk  string `yaml:"disk,omitempty"`
	Wipe  *bool  `yaml:"wipe,omitempty"`
	Image string `yaml:"image,omitempty"`
}

type NetworkSpec struct {
	// DHCP defaults to true when unspecified.
	DHCP *bool `yaml:"dhcp,omitempty"`
}

type TimeSpec struct {
	Servers []string `yaml:"servers,omitempty"`
}

type LoggingSpec struct {
	// Placeholder for Phase 2 schema; not yet wired to runtime knobs.
	ConsoleLevel string `yaml:"consoleLevel,omitempty"`
}

type TrustSpec struct {
	// Token used by trustd flows (Talos-like).
	Token string `yaml:"token,omitempty"`

	// Issuing CA for the OS API (PEM-encoded). In Talos this key typically lives on
	// control plane nodes; for `chuboos` we start with the simple model and evolve
	// it in Phase 3.
	CA *CASpec `yaml:"ca,omitempty"`

	// Optional extra accepted CA certificates (PEM-encoded).
	AcceptedCAs []string `yaml:"acceptedCAs,omitempty"`
}

type CASpec struct {
	Crt string `yaml:"crt,omitempty"`
	Key string `yaml:"key,omitempty"`
}

type RegistrySpec struct {
	// Mirrors configures pull-through/caching mirrors.
	// Keys are registry host[:port], values are mirror endpoints.
	Mirrors map[string]RegistryMirrorSpec `yaml:"mirrors,omitempty"`
}

type RegistryMirrorSpec struct {
	Endpoints    []string `yaml:"endpoints,omitempty"`
	OverridePath *bool    `yaml:"overridePath,omitempty"`
	SkipFallback *bool    `yaml:"skipFallback,omitempty"`
}

type ModulesSpec struct {
	Chubo     *ChuboModuleSpec     `yaml:"chubo,omitempty"`
	Hashistack *HashistackModuleSpec `yaml:"hashistack,omitempty"`
}

type ChuboModuleSpec struct {
	Enabled   *bool              `yaml:"enabled,omitempty"`
	Bootstrap *ChuboBootstrapSpec `yaml:"bootstrap,omitempty"`
}

type ChuboBootstrapSpec struct {
	Mode       string `yaml:"mode,omitempty"`
	SignerCert string `yaml:"signerCert,omitempty"`
	Payload    string `yaml:"payload,omitempty"`
}

type HashistackModuleSpec struct {
	Enabled *bool                 `yaml:"enabled,omitempty"`
	Nomad   *HashistackRoleSpec   `yaml:"nomad,omitempty"`
	Consul  *HashistackRoleSpec   `yaml:"consul,omitempty"`
	OpenBao *HashistackOpenBaoSpec `yaml:"openbao,omitempty"`
}

type HashistackRoleSpec struct {
	Enabled *bool  `yaml:"enabled,omitempty"`
	Role    string `yaml:"role,omitempty"` // server|client
}

type HashistackOpenBaoSpec struct {
	Enabled *bool  `yaml:"enabled,omitempty"`
	Mode    string `yaml:"mode,omitempty"` // nomadJob
}

type BootstrapSpec struct {
	// Reserved for future OS-owned bootstrap mechanisms.
}

// NewMachineConfigV1Alpha1 creates a new minimal machine config.
func NewMachineConfigV1Alpha1() *MachineConfigV1Alpha1 {
	return &MachineConfigV1Alpha1{
		Meta: meta.Meta{
			MetaKind:       MachineConfigKind,
			MetaAPIVersion: MachineConfigAPIVersion,
		},
	}
}

// Clone implements config.Document interface.
func (s *MachineConfigV1Alpha1) Clone() config.Document {
	cp := *s

	// Deep copy maps/slices we own (keep it minimal and explicit).
	if s.Spec.Time != nil && len(s.Spec.Time.Servers) > 0 {
		servers := make([]string, len(s.Spec.Time.Servers))
		copy(servers, s.Spec.Time.Servers)
		cp.Spec.Time = &TimeSpec{Servers: servers}
	}

	if s.Spec.Trust != nil && len(s.Spec.Trust.AcceptedCAs) > 0 {
		accepted := make([]string, len(s.Spec.Trust.AcceptedCAs))
		copy(accepted, s.Spec.Trust.AcceptedCAs)
		cp.Spec.Trust = &TrustSpec{
			Token:       s.Spec.Trust.Token,
			CA:          s.Spec.Trust.CA,
			AcceptedCAs: accepted,
		}
	}

	if s.Spec.Registry != nil && len(s.Spec.Registry.Mirrors) > 0 {
		m := make(map[string]RegistryMirrorSpec, len(s.Spec.Registry.Mirrors))
		for k, v := range s.Spec.Registry.Mirrors {
			// Copy slice.
			endpoints := make([]string, len(v.Endpoints))
			copy(endpoints, v.Endpoints)

			v.Endpoints = endpoints
			m[k] = v
		}

		cp.Spec.Registry = &RegistrySpec{Mirrors: m}
	}

	return &cp
}

// Validate implements config.Validator interface.
func (s *MachineConfigV1Alpha1) Validate(mode validation.RuntimeMode, _ ...validation.Option) ([]string, error) {
	if s.MetaKind != MachineConfigKind {
		return nil, fmt.Errorf("kind must be %q", MachineConfigKind)
	}

	if s.MetaAPIVersion != MachineConfigAPIVersion {
		return nil, fmt.Errorf("apiVersion must be %q", MachineConfigAPIVersion)
	}

	if mode.RequiresInstall() {
		if s.Spec.Install == nil || strings.TrimSpace(s.Spec.Install.Disk) == "" {
			return nil, errors.New("spec.install.disk is required in install mode")
		}
	}

	if s.Spec.Trust == nil {
		return nil, errors.New("spec.trust is required")
	}

	if strings.TrimSpace(s.Spec.Trust.Token) == "" {
		return nil, errors.New("spec.trust.token is required")
	}

	if s.Spec.Trust.CA == nil {
		return nil, errors.New("spec.trust.ca is required")
	}

	if strings.TrimSpace(s.Spec.Trust.CA.Crt) == "" {
		return nil, errors.New("spec.trust.ca.crt is required")
	}

	if strings.TrimSpace(s.Spec.Trust.CA.Key) == "" {
		return nil, errors.New("spec.trust.ca.key is required")
	}

	// Basic registry mirror URL validation (best-effort, avoids surprising runtime errors).
	if s.Spec.Registry != nil {
		for host, mirror := range s.Spec.Registry.Mirrors {
			host = strings.TrimSpace(host)
			if host == "" {
				return nil, errors.New("spec.registry.mirrors has an empty key")
			}

			for _, ep := range mirror.Endpoints {
				u, err := url.Parse(ep)
				if err != nil || u.Scheme == "" || u.Host == "" {
					return nil, fmt.Errorf("spec.registry.mirrors[%q] has invalid endpoint %q", host, ep)
				}
			}
		}
	}

	return nil, nil
}

// ToV1Alpha1 synthesizes a minimal internal v1alpha1.Config suitable for the current `chuboos` boot pipeline.
//
// This preserves Talos' internal config.Provider interface contract (which still depends on v1alpha1.Config),
// while allowing the external config surface to remain small and non-Kubernetes.
func (s *MachineConfigV1Alpha1) ToV1Alpha1() (*v1alpha1.Config, error) {
	cfg := &v1alpha1.Config{
		ConfigVersion: v1alpha1.Version,
		MachineConfig: &v1alpha1.MachineConfig{
			// In `chuboos` we treat Talos' "controlplane" machine type as "managed node"
			// to keep OS API certificate flows (trustd) enabled everywhere.
			MachineType: machine.TypeControlPlane.String(),
		},
		ClusterConfig: nil,
	}

	// Install.
	if s.Spec.Install != nil {
		cfg.MachineConfig.MachineInstall = &v1alpha1.InstallConfig{
			InstallDisk:  strings.TrimSpace(s.Spec.Install.Disk),
			InstallWipe:  s.Spec.Install.Wipe,
			InstallImage: strings.TrimSpace(s.Spec.Install.Image),
		}
	}

	// Time.
	if s.Spec.Time != nil && len(s.Spec.Time.Servers) > 0 {
		cfg.MachineConfig.MachineTime = &v1alpha1.TimeConfig{
			TimeServers: slicesClone(s.Spec.Time.Servers),
		}
	}

	// Trust.
	if s.Spec.Trust != nil {
		cfg.MachineConfig.MachineToken = strings.TrimSpace(s.Spec.Trust.Token)

		if s.Spec.Trust.CA != nil {
			cfg.MachineConfig.MachineCA = &x509.PEMEncodedCertificateAndKey{
				Crt: []byte(s.Spec.Trust.CA.Crt),
				Key: []byte(s.Spec.Trust.CA.Key),
			}
		}

		if len(s.Spec.Trust.AcceptedCAs) > 0 {
			cfg.MachineConfig.MachineAcceptedCAs = make([]*x509.PEMEncodedCertificate, 0, len(s.Spec.Trust.AcceptedCAs))
			for _, pem := range s.Spec.Trust.AcceptedCAs {
				if strings.TrimSpace(pem) == "" {
					continue
				}

				cfg.MachineConfig.MachineAcceptedCAs = append(cfg.MachineConfig.MachineAcceptedCAs, &x509.PEMEncodedCertificate{
					Crt: []byte(pem),
				})
			}
		}
	}

	// Registry mirrors (CRI config-only controllers still consume these in `chuboos` installer flows).
	if s.Spec.Registry != nil && len(s.Spec.Registry.Mirrors) > 0 {
		if cfg.MachineConfig.MachineRegistries.RegistryMirrors == nil {
			cfg.MachineConfig.MachineRegistries.RegistryMirrors = make(map[string]*v1alpha1.RegistryMirrorConfig, len(s.Spec.Registry.Mirrors))
		}

		for host, mirror := range s.Spec.Registry.Mirrors {
			host = strings.TrimSpace(host)
			if host == "" {
				continue
			}

			m := mirror // copy
			cfg.MachineConfig.MachineRegistries.RegistryMirrors[host] = &v1alpha1.RegistryMirrorConfig{
				MirrorEndpoints:    slicesClone(m.Endpoints),
				MirrorOverridePath: m.OverridePath,
				MirrorSkipFallback: m.SkipFallback,
			}
		}
	}

	return cfg, nil
}

func slicesClone[T any](in []T) []T {
	if len(in) == 0 {
		return nil
	}

	out := make([]T, len(in))
	copy(out, in)

	return out
}

// NodeID returns the optional stable node ID, if set.
func (s *MachineConfigV1Alpha1) NodeID() optional.Optional[string] {
	if strings.TrimSpace(s.Metadata.ID) == "" {
		return optional.None[string]()
	}

	return optional.Some(s.Metadata.ID)
}

