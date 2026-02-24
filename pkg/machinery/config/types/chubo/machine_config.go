// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

//docgen:jsonschema

import (
	"crypto/ed25519"
	"crypto/sha256"
	stdlibx509 "crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/siderolabs/crypto/x509"
	"github.com/siderolabs/gen/optional"

	chuboacl "github.com/chubo-dev/chubo/pkg/chubo/acl"
	"github.com/chubo-dev/chubo/pkg/machinery/config/config"
	"github.com/chubo-dev/chubo/pkg/machinery/config/internal/registry"
	"github.com/chubo-dev/chubo/pkg/machinery/config/machine"
	"github.com/chubo-dev/chubo/pkg/machinery/config/types/meta"
	"github.com/chubo-dev/chubo/pkg/machinery/config/types/v1alpha1"
	"github.com/chubo-dev/chubo/pkg/machinery/config/validation"
)

// MachineConfigKind is the `chubo` minimal machine config document kind.
const MachineConfigKind = "MachineConfig"

const (
	// ChuboBootstrapModeSignedPayload enables signed bootstrap payload ingestion.
	//
	// The payload is expected to be a JWS compact string using Ed25519 (alg=EdDSA).
	ChuboBootstrapModeSignedPayload = "signedPayload"

	// chuboBootstrapPayloadPath is where the verified bootstrap payload is written.
	chuboBootstrapPayloadPath      = "/var/lib/chubo/bootstrap/bootstrap.json"
	chuboOpenWontonConfigPath      = "/var/lib/chubo/config/openwonton.hcl"
	chuboOpenWontonRolePath        = "/var/lib/chubo/config/openwonton.role"
	chuboOpenWontonArtifactURLPath = "/var/lib/chubo/config/openwonton.artifact_url"
	chuboOpenGyozaConfigPath       = "/var/lib/chubo/config/opengyoza.hcl"
	chuboOpenGyozaRolePath         = "/var/lib/chubo/config/opengyoza.role"
	chuboOpenGyozaArtifactURLPath  = "/var/lib/chubo/config/opengyoza.artifact_url"
	chuboOpenWontonTLSDir          = "/var/lib/chubo/certs/openwonton"
	chuboOpenGyozaTLSDir           = "/var/lib/chubo/certs/opengyoza"
	chuboOpenBaoJobPath            = "/var/lib/chubo/config/openbao.nomad.json"
	chuboOpenBaoModePath           = "/var/lib/chubo/config/openbao.mode"

	chuboRoleServer = "server"
	chuboRoleClient = "client"
	// chuboRoleServerClient enables both server and client modes for OpenWonton.
	//
	// For OpenGyoza, this role is treated as server.
	chuboRoleServerClient = "server-client"

	chuboOpenBaoModeNomadJob = "nomadJob"
	chuboOpenBaoDefaultJobID = "openbao"
	chuboOpenBaoDefaultImage = "ghcr.io/openbao/openbao:latest"
)

// MachineConfigAPIVersion is the API version string for the minimal machine config.
//
// Note: Chubo upstream uses plain "v1alpha1" for many config documents, but we
// intentionally scope this to a group-like string to avoid collisions with Chubo kinds.
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

// MachineConfigV1Alpha1 is the minimal config document for the `chubo` fork.
//
// schemaRoot: true
// schemaMeta: chubo.dev/v1alpha1/MachineConfig
type MachineConfigV1Alpha1 struct {
	meta.Meta `yaml:",inline"`

	// Metadata contains stable identity information for the node.
	Metadata MachineConfigMetadata `yaml:"metadata,omitempty"`

	// Spec holds the desired machine configuration.
	Spec MachineConfigSpec `yaml:"spec"`
}

type MachineConfigMetadata struct {
	// Optional stable node ID.
	ID string `yaml:"id,omitempty"`
}

type MachineConfigSpec struct {
	// Install configures on-disk installation behavior.
	Install *InstallSpec `yaml:"install,omitempty"`

	// Network configures basic host networking.
	Network *NetworkSpec `yaml:"network,omitempty"`

	// Time configures time synchronization settings.
	Time *TimeSpec `yaml:"time,omitempty"`

	// Logging configures OS logging behavior.
	Logging *LoggingSpec `yaml:"logging,omitempty"`

	// Trust configures OS trust primitives and API access requirements.
	Trust *TrustSpec `yaml:"trust,omitempty"`

	// Registry configures container image registry settings.
	Registry *RegistrySpec `yaml:"registry,omitempty"`

	// Modules configures optional OS-managed modules.
	Modules *ModulesSpec `yaml:"modules,omitempty"`

	// Bootstrap configures bootstrap behavior (payload ingestion, signer pinning).
	Bootstrap *BootstrapSpec `yaml:"bootstrap,omitempty"`
}

type InstallSpec struct {
	// Disk is the target disk (for example: /dev/sda).
	Disk string `yaml:"disk,omitempty"`

	// Wipe controls whether to wipe the target disk before installing.
	Wipe *bool `yaml:"wipe,omitempty"`

	// Image is the installer image reference (OCI ref) used during install.
	Image string `yaml:"image,omitempty"`
}

type NetworkSpec struct {
	// DHCP defaults to true when unspecified.
	DHCP *bool `yaml:"dhcp,omitempty"`
}

type TimeSpec struct {
	// Servers is the list of NTP servers to use.
	Servers []string `yaml:"servers,omitempty"`
}

type LoggingSpec struct {
	// ConsoleLevel configures console log level (placeholder; not yet wired to runtime knobs).
	ConsoleLevel string `yaml:"consoleLevel,omitempty"`
}

type TrustSpec struct {
	// Token used by trustd flows (Chubo-like).
	Token string `yaml:"token,omitempty"`

	// Issuing CA for the OS API (PEM-encoded). In Chubo this key typically lives on
	// control plane nodes; for `chubo` we start with the simple model and evolve
	// it in Phase 3.
	CA *CASpec `yaml:"ca,omitempty"`

	// Optional extra accepted CA certificates (PEM-encoded).
	AcceptedCAs []string `yaml:"acceptedCAs,omitempty"`
}

type CASpec struct {
	// Crt is a PEM-encoded CA certificate.
	Crt string `yaml:"crt,omitempty"`

	// Key is a PEM-encoded CA private key.
	Key string `yaml:"key,omitempty"`
}

type RegistrySpec struct {
	// Mirrors configures pull-through/caching mirrors.
	// Keys are registry host[:port], values are mirror endpoints.
	Mirrors map[string]RegistryMirrorSpec `yaml:"mirrors,omitempty"`
}

type RegistryMirrorSpec struct {
	// Endpoints is the ordered list of mirror endpoints (URLs).
	Endpoints []string `yaml:"endpoints,omitempty"`

	// OverridePath controls whether to override the default image path when using mirrors.
	OverridePath *bool `yaml:"overridePath,omitempty"`

	// SkipFallback controls whether to skip fallback to the original registry.
	SkipFallback *bool `yaml:"skipFallback,omitempty"`
}

type ModulesSpec struct {
	// Chubo configures the Chubo module (openwonton/opengyoza/openbao bootstrap).
	Chubo *ChuboModuleSpec `yaml:"chubo,omitempty"`
}

type ChuboModuleSpec struct {
	// Enabled turns the module on/off.
	Enabled *bool `yaml:"enabled,omitempty"`

	// Bootstrap configures trust/bootstrap payload ingestion.
	Bootstrap *ChuboBootstrapSpec `yaml:"bootstrap,omitempty"`

	// Nomad configures the openwonton role and settings.
	Nomad *ChuboRoleSpec `yaml:"nomad,omitempty"`

	// Consul configures the opengyoza role and settings.
	Consul *ChuboRoleSpec `yaml:"consul,omitempty"`

	// OpenBao configures OpenBao integration (currently via Nomad job).
	OpenBao *ChuboOpenBaoSpec `yaml:"openbao,omitempty"`
}

type ChuboBootstrapSpec struct {
	// Mode selects the bootstrap mechanism (for example: signedPayload).
	Mode string `yaml:"mode,omitempty"`

	// SignerCert is the PEM-encoded signer certificate used to verify bootstrap payloads.
	SignerCert string `yaml:"signerCert,omitempty"`

	// Payload is the bootstrap payload (format depends on Mode).
	Payload string `yaml:"payload,omitempty"`
}

type ChuboRoleSpec struct {
	// Enabled turns the role on/off.
	Enabled *bool `yaml:"enabled,omitempty"`

	// Role selects the role (server|client|server-client).
	Role string `yaml:"role,omitempty"`

	// ArtifactURL overrides the default release artifact URL used to install the component binary.
	//
	// This is primarily intended for airgapped/internal mirrors and for development when upstream
	// release assets are not publicly accessible.
	ArtifactURL string `yaml:"artifactURL,omitempty"`

	// BootstrapExpect controls the expected number of peers for quorum/bootstrap.
	// When unset, server roles default to 1 and client roles default to 0.
	BootstrapExpect *int `yaml:"bootstrapExpect,omitempty"`

	// Join is the ordered list of peer addresses (IP/host) to join/retry-join.
	Join []string `yaml:"join,omitempty"`

	// NetworkInterface sets openwonton client.network_interface when client mode is enabled.
	//
	// This avoids default-route interface auto-detection paths that may rely on
	// distro-specific helper binaries.
	NetworkInterface string `yaml:"networkInterface,omitempty"`
}

type ChuboOpenBaoSpec struct {
	// Enabled turns OpenBao integration on/off.
	Enabled *bool `yaml:"enabled,omitempty"`

	// Mode selects the integration mode (nomadJob).
	Mode string `yaml:"mode,omitempty"`
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

	// Chubo bootstrap (Phase 3): signed payload verification.
	if s.Spec.Modules != nil && s.Spec.Modules.Chubo != nil {
		enabled := s.Spec.Modules.Chubo.Enabled == nil || *s.Spec.Modules.Chubo.Enabled

		if !enabled && s.Spec.Modules.Chubo.Bootstrap != nil {
			return nil, errors.New("spec.modules.chubo.bootstrap is set but spec.modules.chubo.enabled is false")
		}

		if enabled && s.Spec.Modules.Chubo.Bootstrap != nil {
			bootstrap := s.Spec.Modules.Chubo.Bootstrap
			switch strings.TrimSpace(bootstrap.Mode) {
			case "":
				// no-op
			case ChuboBootstrapModeSignedPayload:
				if strings.TrimSpace(bootstrap.SignerCert) == "" {
					return nil, errors.New("spec.modules.chubo.bootstrap.signerCert is required for signedPayload mode")
				}

				if strings.TrimSpace(bootstrap.Payload) == "" {
					return nil, errors.New("spec.modules.chubo.bootstrap.payload is required for signedPayload mode")
				}

				decoded, fp, err := verifyChuboBootstrapJWS(strings.TrimSpace(bootstrap.Payload), bootstrap.SignerCert)
				if err != nil {
					return nil, fmt.Errorf("invalid chubo bootstrap payload: %w", err)
				}

				if !json.Valid(decoded) {
					return nil, errors.New("chubo bootstrap payload is not valid JSON")
				}

				// fp is currently informational; keep it computed here to ensure the cert parses.
				_ = fp
			default:
				return nil, fmt.Errorf("unknown spec.modules.chubo.bootstrap.mode %q", bootstrap.Mode)
			}
		}

		if !enabled && (s.Spec.Modules.Chubo.Nomad != nil || s.Spec.Modules.Chubo.Consul != nil || s.Spec.Modules.Chubo.OpenBao != nil) {
			return nil, errors.New("spec.modules.chubo.nomad/consul/openbao are set but spec.modules.chubo.enabled is false")
		}

		if enabled {
			if s.Spec.Modules.Chubo.Nomad != nil && (s.Spec.Modules.Chubo.Nomad.Enabled == nil || *s.Spec.Modules.Chubo.Nomad.Enabled) {
				if err := validateChuboRole("spec.modules.chubo.nomad.role", s.Spec.Modules.Chubo.Nomad.Role); err != nil {
					return nil, err
				}

				if err := validateChuboArtifactURL("spec.modules.chubo.nomad.artifactURL", s.Spec.Modules.Chubo.Nomad.ArtifactURL); err != nil {
					return nil, err
				}

				if err := validateChuboBootstrapExpect("spec.modules.chubo.nomad.bootstrapExpect", s.Spec.Modules.Chubo.Nomad.BootstrapExpect); err != nil {
					return nil, err
				}

				if err := validateChuboJoin("spec.modules.chubo.nomad.join", s.Spec.Modules.Chubo.Nomad.Join); err != nil {
					return nil, err
				}

				if err := validateChuboNetworkInterface("spec.modules.chubo.nomad.networkInterface", s.Spec.Modules.Chubo.Nomad.NetworkInterface); err != nil {
					return nil, err
				}
			}

			if s.Spec.Modules.Chubo.Consul != nil && (s.Spec.Modules.Chubo.Consul.Enabled == nil || *s.Spec.Modules.Chubo.Consul.Enabled) {
				if err := validateChuboRole("spec.modules.chubo.consul.role", s.Spec.Modules.Chubo.Consul.Role); err != nil {
					return nil, err
				}

				if err := validateChuboArtifactURL("spec.modules.chubo.consul.artifactURL", s.Spec.Modules.Chubo.Consul.ArtifactURL); err != nil {
					return nil, err
				}

				if err := validateChuboBootstrapExpect("spec.modules.chubo.consul.bootstrapExpect", s.Spec.Modules.Chubo.Consul.BootstrapExpect); err != nil {
					return nil, err
				}

				if err := validateChuboJoin("spec.modules.chubo.consul.join", s.Spec.Modules.Chubo.Consul.Join); err != nil {
					return nil, err
				}
			}

			if s.Spec.Modules.Chubo.OpenBao != nil && (s.Spec.Modules.Chubo.OpenBao.Enabled == nil || *s.Spec.Modules.Chubo.OpenBao.Enabled) {
				mode := strings.TrimSpace(s.Spec.Modules.Chubo.OpenBao.Mode)
				switch mode {
				case "", chuboOpenBaoModeNomadJob:
					// valid
				default:
					return nil, fmt.Errorf("unknown spec.modules.chubo.openbao.mode %q", s.Spec.Modules.Chubo.OpenBao.Mode)
				}
			}
		}
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

// ToV1Alpha1 synthesizes a minimal internal v1alpha1.Config suitable for the current `chubo` boot pipeline.
//
// This preserves Chubo' internal config.Provider interface contract (which still depends on v1alpha1.Config),
// while allowing the external config surface to remain small and workload-focused.
func (s *MachineConfigV1Alpha1) ToV1Alpha1() (*v1alpha1.Config, error) {
	cfg := &v1alpha1.Config{
		ConfigVersion: v1alpha1.Version,
		MachineConfig: &v1alpha1.MachineConfig{
			// In `chubo` we treat Chubo' "controlplane" machine type as "managed node"
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

	// Chubo bootstrap: write the verified payload into /var so chubo-agent (or a future OS module)
	// can consume it without adding a separate remote API surface.
	if s.Spec.Modules != nil && s.Spec.Modules.Chubo != nil {
		enabled := s.Spec.Modules.Chubo.Enabled == nil || *s.Spec.Modules.Chubo.Enabled

		if enabled && s.Spec.Modules.Chubo.Bootstrap != nil && strings.TrimSpace(s.Spec.Modules.Chubo.Bootstrap.Mode) == ChuboBootstrapModeSignedPayload {
			decoded, fp, err := verifyChuboBootstrapJWS(strings.TrimSpace(s.Spec.Modules.Chubo.Bootstrap.Payload), s.Spec.Modules.Chubo.Bootstrap.SignerCert)
			if err != nil {
				return nil, fmt.Errorf("invalid chubo bootstrap payload: %w", err)
			}

			if !json.Valid(decoded) {
				return nil, errors.New("chubo bootstrap payload is not valid JSON")
			}

			cfg.MachineConfig.MachineFiles = append(cfg.MachineConfig.MachineFiles, &v1alpha1.MachineFile{
				FileContent:     string(decoded),
				FilePermissions: v1alpha1.FileMode(0o600),
				FilePath:        chuboBootstrapPayloadPath,
				FileOp:          "create",
			})

			// Store signer fingerprint for debugging/auditing (not a trust anchor).
			cfg.MachineConfig.MachineFiles = append(cfg.MachineConfig.MachineFiles, &v1alpha1.MachineFile{
				FileContent:     fp + "\n",
				FilePermissions: v1alpha1.FileMode(0o644),
				FilePath:        "/var/lib/chubo/bootstrap/signer.sha256",
				FileOp:          "create",
			})
		}

		chuboConsulEnabled := false
		chuboConsulACLToken := ""

		if s.Spec.Modules.Chubo.Consul != nil {
			chuboConsulEnabled = s.Spec.Modules.Chubo.Consul.Enabled == nil || *s.Spec.Modules.Chubo.Consul.Enabled
			if chuboConsulEnabled {
				chuboConsulACLToken = chuboacl.WorkloadToken(s.Spec.Trust.Token, "consul")
			}
		}

		// Chubo Nomad/OpenWonton: render minimal host-process config for the OS-managed service.
		if enabled && s.Spec.Modules.Chubo.Nomad != nil {
			nomadEnabled := s.Spec.Modules.Chubo.Nomad.Enabled == nil || *s.Spec.Modules.Chubo.Nomad.Enabled
			if nomadEnabled {
				role := normalizeChuboRole(s.Spec.Modules.Chubo.Nomad.Role)
				bootstrapExpect := defaultChuboBootstrapExpect(role, s.Spec.Modules.Chubo.Nomad.BootstrapExpect)
				join := normalizeChuboJoin(s.Spec.Modules.Chubo.Nomad.Join)
				networkInterface := normalizeChuboNetworkInterface(s.Spec.Modules.Chubo.Nomad.NetworkInterface)

				if url := strings.TrimSpace(s.Spec.Modules.Chubo.Nomad.ArtifactURL); url != "" {
					cfg.MachineConfig.MachineFiles = append(cfg.MachineConfig.MachineFiles, &v1alpha1.MachineFile{
						FileContent:     url + "\n",
						FilePermissions: v1alpha1.FileMode(0o644),
						FilePath:        chuboOpenWontonArtifactURLPath,
						FileOp:          "create",
					})
				}

				cfg.MachineConfig.MachineFiles = append(cfg.MachineConfig.MachineFiles, &v1alpha1.MachineFile{
					FileContent:     renderOpenWontonConfig(role, bootstrapExpect, join, networkInterface, chuboConsulEnabled, chuboConsulACLToken),
					FilePermissions: v1alpha1.FileMode(0o600),
					FilePath:        chuboOpenWontonConfigPath,
					FileOp:          "create",
				})

				cfg.MachineConfig.MachineFiles = append(cfg.MachineConfig.MachineFiles, &v1alpha1.MachineFile{
					FileContent:     role + "\n",
					FilePermissions: v1alpha1.FileMode(0o644),
					FilePath:        chuboOpenWontonRolePath,
					FileOp:          "create",
				})
			}
		}

		// Chubo Consul/OpenGyoza: render minimal host-process config for the OS-managed service.
		if enabled && s.Spec.Modules.Chubo.Consul != nil {
			consulEnabled := chuboConsulEnabled
			if consulEnabled {
				role := normalizeChuboRole(s.Spec.Modules.Chubo.Consul.Role)
				bootstrapExpect := defaultChuboBootstrapExpect(role, s.Spec.Modules.Chubo.Consul.BootstrapExpect)
				join := normalizeChuboJoin(s.Spec.Modules.Chubo.Consul.Join)
				aclToken := chuboConsulACLToken

				if url := strings.TrimSpace(s.Spec.Modules.Chubo.Consul.ArtifactURL); url != "" {
					cfg.MachineConfig.MachineFiles = append(cfg.MachineConfig.MachineFiles, &v1alpha1.MachineFile{
						FileContent:     url + "\n",
						FilePermissions: v1alpha1.FileMode(0o644),
						FilePath:        chuboOpenGyozaArtifactURLPath,
						FileOp:          "create",
					})
				}

				cfg.MachineConfig.MachineFiles = append(cfg.MachineConfig.MachineFiles, &v1alpha1.MachineFile{
					FileContent:     renderOpenGyozaConfig(role, bootstrapExpect, join, aclToken),
					FilePermissions: v1alpha1.FileMode(0o600),
					FilePath:        chuboOpenGyozaConfigPath,
					FileOp:          "create",
				})

				cfg.MachineConfig.MachineFiles = append(cfg.MachineConfig.MachineFiles, &v1alpha1.MachineFile{
					FileContent:     role + "\n",
					FilePermissions: v1alpha1.FileMode(0o644),
					FilePath:        chuboOpenGyozaRolePath,
					FileOp:          "create",
				})
			}
		}

		// Chubo OpenBao: render default Nomad job payload for the job-presence controller.
		if enabled && s.Spec.Modules.Chubo.OpenBao != nil {
			openBaoEnabled := s.Spec.Modules.Chubo.OpenBao.Enabled == nil || *s.Spec.Modules.Chubo.OpenBao.Enabled
			mode := strings.TrimSpace(s.Spec.Modules.Chubo.OpenBao.Mode)
			if mode == "" {
				mode = chuboOpenBaoModeNomadJob
			}

			if openBaoEnabled && mode == chuboOpenBaoModeNomadJob {
				cfg.MachineConfig.MachineFiles = append(cfg.MachineConfig.MachineFiles, &v1alpha1.MachineFile{
					FileContent:     renderOpenBaoNomadJobPayload(),
					FilePermissions: v1alpha1.FileMode(0o600),
					FilePath:        chuboOpenBaoJobPath,
					FileOp:          "create",
				})

				cfg.MachineConfig.MachineFiles = append(cfg.MachineConfig.MachineFiles, &v1alpha1.MachineFile{
					FileContent:     mode + "\n",
					FilePermissions: v1alpha1.FileMode(0o644),
					FilePath:        chuboOpenBaoModePath,
					FileOp:          "create",
				})
			}
		}
	}

	// Registry mirrors (CRI config-only controllers still consume these in `chubo` installer flows).
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

func validateChuboRole(path string, role string) error {
	switch strings.TrimSpace(role) {
	case "", chuboRoleServer, chuboRoleClient, chuboRoleServerClient:
		return nil
	default:
		return fmt.Errorf("unknown %s %q", path, role)
	}
}

func normalizeChuboRole(role string) string {
	switch strings.TrimSpace(role) {
	case chuboRoleClient:
		return chuboRoleClient
	case chuboRoleServerClient:
		return chuboRoleServerClient
	default:
		return chuboRoleServer
	}
}

func isChuboServerRole(role string) bool {
	switch normalizeChuboRole(role) {
	case chuboRoleServer, chuboRoleServerClient:
		return true
	default:
		return false
	}
}

func isChuboClientRole(role string) bool {
	switch normalizeChuboRole(role) {
	case chuboRoleClient, chuboRoleServerClient:
		return true
	default:
		return false
	}
}

func validateChuboBootstrapExpect(path string, v *int) error {
	if v == nil {
		return nil
	}

	if *v < 0 {
		return fmt.Errorf("%s must be >= 0", path)
	}

	return nil
}

func validateChuboJoin(path string, addrs []string) error {
	for _, raw := range addrs {
		if strings.TrimSpace(raw) == "" {
			return fmt.Errorf("%s must not contain empty entries", path)
		}
	}

	return nil
}

func validateChuboNetworkInterface(path string, iface string) error {
	if iface == "" {
		return nil
	}

	if strings.TrimSpace(iface) == "" {
		return fmt.Errorf("%s must not be empty", path)
	}

	return nil
}

func validateChuboArtifactURL(path string, raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s must be a valid URL: %w", path, err)
	}

	switch u.Scheme {
	case "http", "https":
		// ok
	default:
		return fmt.Errorf("%s must be an http(s) URL", path)
	}

	if strings.TrimSpace(u.Host) == "" {
		return fmt.Errorf("%s must include a host", path)
	}

	return nil
}

func normalizeChuboJoin(addrs []string) []string {
	if len(addrs) == 0 {
		return nil
	}

	out := make([]string, 0, len(addrs))

	for _, a := range addrs {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}

		out = append(out, a)
	}

	return out
}

func normalizeChuboNetworkInterface(iface string) string {
	return strings.TrimSpace(iface)
}

func defaultChuboBootstrapExpect(role string, v *int) int {
	if v != nil {
		return *v
	}

	if isChuboServerRole(role) {
		return 1
	}

	return 0
}

func renderHCLStringArray(values []string) string {
	// HCL accepts JSON list syntax, so json.Marshal gives us correct quoting.
	b, err := json.Marshal(values)
	if err != nil {
		// json.Marshal shouldn't fail for []string; fall back to empty list.
		return "[]"
	}

	return string(b)
}

func renderOpenWontonClientServers(join []string) string {
	servers := make([]string, 0, len(join))

	for _, raw := range join {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		if _, _, err := net.SplitHostPort(raw); err == nil {
			servers = append(servers, raw)

			continue
		}

		servers = append(servers, net.JoinHostPort(raw, "4647"))
	}

	return renderHCLStringArray(servers)
}

func renderOpenWontonConfig(role string, bootstrapExpect int, join []string, networkInterface string, consulEnabled bool, consulACLToken string) string {
	serverEnabled := isChuboServerRole(role)
	clientEnabled := isChuboClientRole(role)

	joinBlock := ""
	if serverEnabled && len(join) > 0 {
		joinBlock = fmt.Sprintf(`  server_join {
    retry_join = %s
    retry_max = 0
    retry_interval = "15s"
  }
`, renderHCLStringArray(join))
	}

	clientServersLine := ""
	if clientEnabled && len(join) > 0 {
		clientServersLine = fmt.Sprintf("  servers = %s\n", renderOpenWontonClientServers(join))
	}

	clientNetworkInterfaceLine := ""
	if clientEnabled && networkInterface != "" {
		clientNetworkInterfaceLine = fmt.Sprintf("  network_interface = %q\n", networkInterface)
	}

	advertiseBlock := ""
	if networkInterface != "" {
		quotedNetworkInterface := fmt.Sprintf("\\\"%s\\\"", networkInterface)

		// Force Nomad/OpenWonton to advertise the configured host interface IP.
		// This avoids accidental selection of docker0 (172.17.0.1), which breaks raft in
		// multi-node clusters because all nodes otherwise advertise the same address.
		advertiseBlock = fmt.Sprintf(`advertise {
  http = "{{ GetInterfaceIP %s }}"
  rpc = "{{ GetInterfaceIP %s }}"
  serf = "{{ GetInterfaceIP %s }}"
}

`, quotedNetworkInterface, quotedNetworkInterface, quotedNetworkInterface)
	}

	clientOptionsBlock := ""
	if clientEnabled {
		// Keep raw_exec enabled so minimal smoke jobs can run without the exec plugin runtime.
		clientOptionsBlock = `  options = {
    "driver.raw_exec.enable" = "1"
  }
`
	}

	consulTokenLine := ""
	if consulACLToken != "" {
		consulTokenLine = fmt.Sprintf("  token = %q\n", consulACLToken)
	}

	consulBlock := ""
	if consulEnabled {
		consulBlock = fmt.Sprintf(`consul {
  address = "127.0.0.1:8500"
  ssl = false
%s
}

`, consulTokenLine)
	}

	return fmt.Sprintf(`data_dir = "/var/lib/chubo/openwonton"
bind_addr = "0.0.0.0"
log_level = "INFO"

acl {
  enabled = true
}

tls {
  http = true
  rpc = true
  ca_file = "%s/ca.pem"
  cert_file = "%s/server.pem"
  key_file = "%s/server-key.pem"
  verify_https_client = true
}

%s
%s

server {
  enabled = %t
  bootstrap_expect = %d
%s
}

client {
  enabled = %t
%s
%s
%s
}
`, chuboOpenWontonTLSDir, chuboOpenWontonTLSDir, chuboOpenWontonTLSDir, consulBlock, advertiseBlock, serverEnabled, bootstrapExpect, joinBlock, clientEnabled, clientOptionsBlock, clientServersLine, clientNetworkInterfaceLine)
}

func renderOpenGyozaConfig(role string, bootstrapExpect int, join []string, aclToken string) string {
	serverEnabled := isChuboServerRole(role)

	joinLine := ""
	if len(join) > 0 {
		joinLine = fmt.Sprintf("retry_join = %s\n", renderHCLStringArray(join))
	}

	return fmt.Sprintf(`data_dir = "/var/lib/chubo/opengyoza"
bind_addr = "0.0.0.0"
client_addr = "0.0.0.0"
log_level = "INFO"
ports {
  https = 8500
  http = -1
}
acl {
  enabled = true
  default_policy = "deny"
  enable_token_persistence = true
  tokens {
    master = %q
    agent = %q
  }
}
ca_file = "%s/ca.pem"
cert_file = "%s/server.pem"
key_file = "%s/server-key.pem"
verify_incoming = true
verify_outgoing = true
%sserver = %t
bootstrap_expect = %d
`, aclToken, aclToken, chuboOpenGyozaTLSDir, chuboOpenGyozaTLSDir, chuboOpenGyozaTLSDir, joinLine, serverEnabled, bootstrapExpect)
}

func renderOpenBaoNomadJobPayload() string {
	return fmt.Sprintf(`{
  "Job": {
    "ID": %q,
    "Name": %q,
    "Type": "service",
    "Datacenters": ["dc1"],
    "TaskGroups": [
      {
        "Name": "openbao",
        "Count": 1,
        "Networks": [
          {
            "Mode": "bridge",
            "DynamicPorts": [
              {
                "Label": "http",
                "To": 8200
              }
            ]
          }
        ],
        "Tasks": [
          {
            "Name": "openbao",
            "Driver": "docker",
            "Config": {
              "image": %q,
              "ports": ["http"],
              "args": ["server", "-dev", "-dev-listen-address=0.0.0.0:8200"]
            },
            "Resources": {
              "CPU": 500,
              "MemoryMB": 512
            },
            "Services": [
              {
                "Name": "openbao",
                "PortLabel": "http"
              }
            ]
          }
        ]
      }
    ]
  }
}
`, chuboOpenBaoDefaultJobID, chuboOpenBaoDefaultJobID, chuboOpenBaoDefaultImage)
}

// NodeID returns the optional stable node ID, if set.
func (s *MachineConfigV1Alpha1) NodeID() optional.Optional[string] {
	if strings.TrimSpace(s.Metadata.ID) == "" {
		return optional.None[string]()
	}

	return optional.Some(s.Metadata.ID)
}

// chuboBootstrapJWSHeader is the protected header for the JWS compact bootstrap payload.
type chuboBootstrapJWSHeader struct {
	// Alg is the JWS algorithm identifier (expected "EdDSA").
	Alg string `json:"alg" yaml:"alg"`
}

func verifyChuboBootstrapJWS(jwsCompact string, signerCertPEM string) ([]byte, string, error) {
	cert, pub, fp, err := parseSignerEd25519Cert(signerCertPEM)
	if err != nil {
		return nil, "", err
	}

	_ = cert // reserved for future chain/pinning logic

	parts := strings.Split(jwsCompact, ".")
	if len(parts) != 3 {
		return nil, "", errors.New("payload must be a JWS compact string (<protected>.<payload>.<signature>)")
	}

	protectedRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, "", fmt.Errorf("invalid JWS protected header encoding: %w", err)
	}

	var header chuboBootstrapJWSHeader
	if err := json.Unmarshal(protectedRaw, &header); err != nil {
		return nil, "", fmt.Errorf("invalid JWS protected header JSON: %w", err)
	}

	if header.Alg != "EdDSA" {
		return nil, "", fmt.Errorf("unsupported JWS alg %q (expected %q)", header.Alg, "EdDSA")
	}

	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, "", fmt.Errorf("invalid JWS payload encoding: %w", err)
	}

	sigRaw, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, "", fmt.Errorf("invalid JWS signature encoding: %w", err)
	}

	signingInput := []byte(parts[0] + "." + parts[1])
	if !ed25519.Verify(pub, signingInput, sigRaw) {
		return nil, "", errors.New("invalid JWS signature")
	}

	return payloadRaw, fp, nil
}

func parseSignerEd25519Cert(pemStr string) (*stdlibx509.Certificate, ed25519.PublicKey, string, error) {
	pemStr = strings.TrimSpace(pemStr)
	if pemStr == "" {
		return nil, nil, "", errors.New("signer cert is empty")
	}

	block, _ := pem.Decode([]byte(pemStr))
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, nil, "", errors.New("signerCert must be a PEM-encoded CERTIFICATE")
	}

	cert, err := stdlibx509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to parse signer certificate: %w", err)
	}

	pub, ok := cert.PublicKey.(ed25519.PublicKey)
	if !ok {
		return nil, nil, "", errors.New("signerCert public key must be Ed25519")
	}

	sum := sha256.Sum256(cert.Raw)
	fp := hex.EncodeToString(sum[:])

	return cert, pub, fp, nil
}
