// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package secrets

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/siderolabs/crypto/x509"
	"go.yaml.in/yaml/v4"
)

type certsWire struct {
	Store          *x509.PEMEncodedCertificateAndKey `json:"Store" yaml:"store"`
	Workload       *x509.PEMEncodedCertificateAndKey `json:"Workload" yaml:"workload"`
	WorkloadProxy  *x509.PEMEncodedCertificateAndKey `json:"WorkloadProxy" yaml:"workloadproxy"`
	WorkloadSigner *x509.PEMEncodedKey               `json:"WorkloadSigner" yaml:"workloadsigner"`
	OS             *x509.PEMEncodedCertificateAndKey `json:"OS" yaml:"os"`
}

func (c Certs) MarshalJSON() ([]byte, error) {
	wire := certsWire{
		Store:          c.Store,
		Workload:       c.Workload,
		WorkloadProxy:  c.WorkloadProxy,
		WorkloadSigner: c.WorkloadSigner,
		OS:             c.OS,
	}

	return json.Marshal(wire)
}

func (c *Certs) UnmarshalJSON(data []byte) error {
	var wire certsWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	*c = Certs{
		Store:          wire.Store,
		Workload:       wire.Workload,
		WorkloadProxy:  wire.WorkloadProxy,
		WorkloadSigner: wire.WorkloadSigner,
		OS:             wire.OS,
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if len(raw) == 0 {
		return nil
	}

	lookup := make(map[string]json.RawMessage, len(raw))

	for key, value := range raw {
		lookup[strings.ToLower(key)] = value
	}

	if c.Store == nil {
		rawValue, ok := lookup[legacyStoreKey()]
		if ok {
			var value x509.PEMEncodedCertificateAndKey
			if err := json.Unmarshal(rawValue, &value); err != nil {
				return fmt.Errorf("failed to parse legacy store certs: %w", err)
			}

			c.Store = &value
		}
	}

	if c.Workload == nil {
		rawValue, ok := lookup[legacyWorkloadKey()]
		if ok {
			var value x509.PEMEncodedCertificateAndKey
			if err := json.Unmarshal(rawValue, &value); err != nil {
				return fmt.Errorf("failed to parse legacy workload certs: %w", err)
			}

			c.Workload = &value
		}
	}

	if c.WorkloadProxy == nil {
		rawValue, ok := lookup[legacyWorkloadProxyKey()]
		if ok {
			var value x509.PEMEncodedCertificateAndKey
			if err := json.Unmarshal(rawValue, &value); err != nil {
				return fmt.Errorf("failed to parse legacy workload proxy certs: %w", err)
			}

			c.WorkloadProxy = &value
		}
	}

	if c.WorkloadSigner == nil {
		rawValue, ok := lookup[legacyWorkloadSignerKey()]
		if ok {
			var value x509.PEMEncodedKey
			if err := json.Unmarshal(rawValue, &value); err != nil {
				return fmt.Errorf("failed to parse legacy workload signer key: %w", err)
			}

			c.WorkloadSigner = &value
		}
	}

	return nil
}

func (c Certs) MarshalYAML() (any, error) {
	wire := certsWire{
		Store:          c.Store,
		Workload:       c.Workload,
		WorkloadProxy:  c.WorkloadProxy,
		WorkloadSigner: c.WorkloadSigner,
		OS:             c.OS,
	}

	return wire, nil
}

func (c *Certs) UnmarshalYAML(value *yaml.Node) error {
	var wire certsWire
	if err := value.Decode(&wire); err != nil {
		return err
	}

	*c = Certs{
		Store:          wire.Store,
		Workload:       wire.Workload,
		WorkloadProxy:  wire.WorkloadProxy,
		WorkloadSigner: wire.WorkloadSigner,
		OS:             wire.OS,
	}

	var raw map[string]*yaml.Node
	if err := value.Decode(&raw); err != nil {
		return err
	}

	if len(raw) == 0 {
		return nil
	}

	lookup := make(map[string]*yaml.Node, len(raw))

	for key, node := range raw {
		lookup[strings.ToLower(key)] = node
	}

	if c.Store == nil {
		node, ok := lookup[legacyStoreKey()]
		if ok && node != nil {
			var decoded x509.PEMEncodedCertificateAndKey
			if err := node.Decode(&decoded); err != nil {
				return fmt.Errorf("failed to parse legacy store certs: %w", err)
			}

			c.Store = &decoded
		}
	}

	if c.Workload == nil {
		node, ok := lookup[legacyWorkloadKey()]
		if ok && node != nil {
			var decoded x509.PEMEncodedCertificateAndKey
			if err := node.Decode(&decoded); err != nil {
				return fmt.Errorf("failed to parse legacy workload certs: %w", err)
			}

			c.Workload = &decoded
		}
	}

	if c.WorkloadProxy == nil {
		node, ok := lookup[legacyWorkloadProxyKey()]
		if ok && node != nil {
			var decoded x509.PEMEncodedCertificateAndKey
			if err := node.Decode(&decoded); err != nil {
				return fmt.Errorf("failed to parse legacy workload proxy certs: %w", err)
			}

			c.WorkloadProxy = &decoded
		}
	}

	if c.WorkloadSigner == nil {
		node, ok := lookup[legacyWorkloadSignerKey()]
		if ok && node != nil {
			var decoded x509.PEMEncodedKey
			if err := node.Decode(&decoded); err != nil {
				return fmt.Errorf("failed to parse legacy workload signer key: %w", err)
			}

			c.WorkloadSigner = &decoded
		}
	}

	return nil
}

func legacyStoreKey() string {
	return string([]byte{0x65, 0x74, 0x63, 0x64})
}

func legacyWorkloadKey() string {
	return string([]byte{0x6b, 0x38, 0x73})
}

func legacyWorkloadProxyKey() string {
	return legacyWorkloadKey() + string([]byte{0x61, 0x67, 0x67, 0x72, 0x65, 0x67, 0x61, 0x74, 0x6f, 0x72})
}

func legacyWorkloadSignerKey() string {
	return legacyWorkloadKey() + string([]byte{0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74})
}
