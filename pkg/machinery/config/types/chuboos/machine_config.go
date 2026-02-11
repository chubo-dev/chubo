// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chuboos

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

const (
	// ChuboBootstrapModeSignedPayload enables signed bootstrap payload ingestion.
	//
	// The payload is expected to be a JWS compact string using Ed25519 (alg=EdDSA).
	ChuboBootstrapModeSignedPayload = "signedPayload"

	// chuboBootstrapPayloadPath is where the verified bootstrap payload is written.
	chuboBootstrapPayloadPath = "/var/lib/chubo/bootstrap/bootstrap.json"
	chuboOpenWontonConfigPath = "/var/lib/chubo/config/openwonton.hcl"
	chuboOpenWontonRolePath   = "/var/lib/chubo/config/openwonton.role"

	chuboRoleServer = "server"
	chuboRoleClient = "client"
)

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
	Chubo *ChuboModuleSpec `yaml:"chubo,omitempty"`
}

type ChuboModuleSpec struct {
	Enabled   *bool               `yaml:"enabled,omitempty"`
	Bootstrap *ChuboBootstrapSpec `yaml:"bootstrap,omitempty"`
	Nomad     *ChuboRoleSpec      `yaml:"nomad,omitempty"`
	Consul    *ChuboRoleSpec      `yaml:"consul,omitempty"`
	OpenBao   *ChuboOpenBaoSpec   `yaml:"openbao,omitempty"`
}

type ChuboBootstrapSpec struct {
	Mode       string `yaml:"mode,omitempty"`
	SignerCert string `yaml:"signerCert,omitempty"`
	Payload    string `yaml:"payload,omitempty"`
}

type ChuboRoleSpec struct {
	Enabled *bool  `yaml:"enabled,omitempty"`
	Role    string `yaml:"role,omitempty"` // server|client
}

type ChuboOpenBaoSpec struct {
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
			}

			if s.Spec.Modules.Chubo.Consul != nil && (s.Spec.Modules.Chubo.Consul.Enabled == nil || *s.Spec.Modules.Chubo.Consul.Enabled) {
				if err := validateChuboRole("spec.modules.chubo.consul.role", s.Spec.Modules.Chubo.Consul.Role); err != nil {
					return nil, err
				}
			}

			if s.Spec.Modules.Chubo.OpenBao != nil && (s.Spec.Modules.Chubo.OpenBao.Enabled == nil || *s.Spec.Modules.Chubo.OpenBao.Enabled) {
				mode := strings.TrimSpace(s.Spec.Modules.Chubo.OpenBao.Mode)
				switch mode {
				case "", "nomadJob":
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

		// Chubo Nomad/OpenWonton: render minimal host-process config for the OS-managed service.
		if enabled && s.Spec.Modules.Chubo.Nomad != nil {
			nomadEnabled := s.Spec.Modules.Chubo.Nomad.Enabled == nil || *s.Spec.Modules.Chubo.Nomad.Enabled
			if nomadEnabled {
				role := normalizeChuboRole(s.Spec.Modules.Chubo.Nomad.Role)

				cfg.MachineConfig.MachineFiles = append(cfg.MachineConfig.MachineFiles, &v1alpha1.MachineFile{
					FileContent:     renderOpenWontonConfig(role),
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

func validateChuboRole(path string, role string) error {
	switch strings.TrimSpace(role) {
	case "", chuboRoleServer, chuboRoleClient:
		return nil
	default:
		return fmt.Errorf("unknown %s %q", path, role)
	}
}

func normalizeChuboRole(role string) string {
	switch strings.TrimSpace(role) {
	case chuboRoleClient:
		return chuboRoleClient
	default:
		return chuboRoleServer
	}
}

func renderOpenWontonConfig(role string) string {
	serverEnabled := role == chuboRoleServer
	clientEnabled := role == chuboRoleClient

	return fmt.Sprintf(`data_dir = "/var/lib/chubo/openwonton"
bind_addr = "0.0.0.0"
log_level = "INFO"

server {
  enabled = %t
  bootstrap_expect = 1
}

client {
  enabled = %t
}
`, serverEnabled, clientEnabled)
}

// NodeID returns the optional stable node ID, if set.
func (s *MachineConfigV1Alpha1) NodeID() optional.Optional[string] {
	if strings.TrimSpace(s.Metadata.ID) == "" {
		return optional.None[string]()
	}

	return optional.Some(s.Metadata.ID)
}

type chuboBootstrapJWSHeader struct {
	Alg string `json:"alg"`
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
