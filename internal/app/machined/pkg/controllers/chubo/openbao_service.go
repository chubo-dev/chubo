// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package chubo

import (
	"bytes"
	"context"
	"crypto/tls"
	stdlibx509 "crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cosi-project/runtime/pkg/controller"
	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/siderolabs/gen/optional"
	"go.uber.org/zap"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/services"
	machineryclient "github.com/chubo-dev/chubo/pkg/machinery/client"
	gensecrets "github.com/chubo-dev/chubo/pkg/machinery/config/generate/secrets"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	chubores "github.com/chubo-dev/chubo/pkg/machinery/resources/chubo"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/config"
	secretres "github.com/chubo-dev/chubo/pkg/machinery/resources/secrets"
	v1alpha1res "github.com/chubo-dev/chubo/pkg/machinery/resources/v1alpha1"
	"github.com/chubo-dev/chubo/pkg/machinery/role"
)

const (
	openBaoHostModePath    = "/var/lib/chubo/config/openbao.mode"
	openBaoHostConfigPath  = "/var/lib/chubo/config/openbao.hcl"
	openBaoHostSpecPath    = "/var/lib/chubo/config/openbao.host.json"
	openBaoModeHostService = "hostService"
	openBaoHTTPAddress     = "http://127.0.0.1:8200"
	openBaoQueryTimeout    = 5 * time.Second
	openBaoInitTimeout     = 30 * time.Second
	openBaoUnsealTimeout   = 10 * time.Second
)

var openBaoInitPath = "/var/lib/chubo/certs/openbao-init.json"

// OpenBaoServiceManager is the interface to v1alpha1 service manager.
type OpenBaoServiceManager interface {
	IsRunning(id string) (system.Service, bool, error)
	Load(services ...system.Service) []string
	Start(serviceIDs ...string) error
	Stop(ctx context.Context, serviceIDs ...string) error
}

// OpenBaoServiceController starts/stops and initializes the OS-managed OpenBao service.
type OpenBaoServiceController struct {
	V1Alpha1ServiceManager OpenBaoServiceManager
}

// Name implements controller.Controller interface.
func (ctrl *OpenBaoServiceController) Name() string {
	return "chubo.OpenBaoServiceController"
}

// Inputs implements controller.Controller interface.
func (ctrl *OpenBaoServiceController) Inputs() []controller.Input {
	return []controller.Input{
		{
			Namespace: config.NamespaceName,
			Type:      config.MachineConfigType,
			ID:        optional.Some(config.ActiveID),
			Kind:      controller.InputWeak,
		},
		{
			Namespace: v1alpha1res.NamespaceName,
			Type:      v1alpha1res.ServiceType,
			ID:        optional.Some(services.OpenBaoServiceID),
			Kind:      controller.InputWeak,
		},
	}
}

// Outputs implements controller.Controller interface.
func (ctrl *OpenBaoServiceController) Outputs() []controller.Output {
	return []controller.Output{
		{
			Type: chubores.OpenBaoStatusType,
			Kind: controller.OutputExclusive,
		},
		{
			Type: secretres.OpenBaoInitType,
			Kind: controller.OutputExclusive,
		},
	}
}

// Run implements controller.Controller interface.
func (ctrl *OpenBaoServiceController) Run(ctx context.Context, r controller.Runtime, _ *zap.Logger) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.EventCh():
		case <-ticker.C:
		}

		r.StartTrackingOutputs()

		mc, err := safe.ReaderGetByID[*config.MachineConfig](ctx, r, config.ActiveID)
		if err != nil && !state.IsNotFoundError(err) {
			return fmt.Errorf("error getting machine config: %w", err)
		}

		configured, mode, err := openBaoConfigured(mc)
		if err != nil {
			return fmt.Errorf("error reading openbao intent: %w", err)
		}

		svcRes, err := safe.ReaderGetByID[*v1alpha1res.Service](ctx, r, services.OpenBaoServiceID)
		if err != nil && !state.IsNotFoundError(err) {
			return fmt.Errorf("error getting openbao service state: %w", err)
		}

		if configured {
			_, running, runErr := ctrl.V1Alpha1ServiceManager.IsRunning(services.OpenBaoServiceID)
			if runErr != nil {
				ctrl.V1Alpha1ServiceManager.Load(&services.OpenBao{})
				running = false
			}

			if !running {
				if err := ctrl.V1Alpha1ServiceManager.Start(services.OpenBaoServiceID); err != nil {
					return fmt.Errorf("error starting openbao service: %w", err)
				}
			}
		} else {
			_, running, runErr := ctrl.V1Alpha1ServiceManager.IsRunning(services.OpenBaoServiceID)
			if runErr == nil && running {
				if err := ctrl.V1Alpha1ServiceManager.Stop(ctx, services.OpenBaoServiceID); err != nil {
					return fmt.Errorf("error stopping openbao service: %w", err)
				}
			}
		}

		running := false
		healthy := false

		if svcRes != nil {
			running = svcRes.TypedSpec().Running
			healthy = svcRes.TypedSpec().Healthy
		}

		initialized := false
		sealed := false
		lastError := ""

		if configured && healthy {
			initialized, sealed, lastError = ensureOpenBaoInitState(ctx, r, mc)
		}

		if err := safe.WriterModify(ctx, r, chubores.NewOpenBaoStatus(), func(res *chubores.OpenBaoStatus) error {
			res.TypedSpec().Configured = configured
			res.TypedSpec().Mode = mode
			res.TypedSpec().Running = running
			res.TypedSpec().Healthy = healthy
			res.TypedSpec().Initialized = initialized
			res.TypedSpec().Sealed = sealed
			res.TypedSpec().LastError = lastError

			return nil
		}); err != nil {
			return fmt.Errorf("error updating openbao status: %w", err)
		}

		if err := r.CleanupOutputs(
			ctx,
			resource.NewMetadata(chubores.NamespaceName, chubores.OpenBaoStatusType, chubores.OpenBaoStatusID, resource.VersionUndefined),
			resource.NewMetadata(secretres.NamespaceName, secretres.OpenBaoInitType, secretres.OpenBaoInitID, resource.VersionUndefined),
		); err != nil {
			return fmt.Errorf("failed to cleanup outputs: %w", err)
		}
	}
}

func openBaoConfigured(mc *config.MachineConfig) (bool, string, error) {
	if mc == nil || mc.Config() == nil || mc.Config().Machine() == nil {
		return false, "", nil
	}

	files, err := mc.Config().Machine().Files()
	if err != nil {
		return false, "", err
	}

	mode := ""
	configured := false

	for _, f := range files {
		switch f.Path() {
		case openBaoHostModePath:
			mode = strings.TrimSpace(f.Content())
		case openBaoHostConfigPath:
			configured = true
		}
	}

	if mode != openBaoModeHostService {
		return false, mode, nil
	}

	return configured, mode, nil
}

type openBaoInitStatus struct {
	Initialized bool `json:"initialized"`
}

type openBaoSealStatus struct {
	Initialized bool `json:"initialized"`
	Sealed      bool `json:"sealed"`
}

type openBaoInitResponse struct {
	RootToken  string   `json:"root_token"`
	KeysBase64 []string `json:"keys_base64"`
}

type openBaoHostSpec struct {
	NetworkInterface string   `json:"networkInterface"`
	RetryJoin        []string `json:"retryJoin"`
}

type openBaoHostPlan struct {
	LocalIP       netip.Addr
	NetworkIface  string
	RetryJoin     []string
	BootstrapNode bool
}

func ensureOpenBaoInitState(ctx context.Context, r controller.Runtime, mc *config.MachineConfig) (initialized bool, sealed bool, lastError string) {
	plan, err := loadOpenBaoHostPlan(mc)
	if err != nil {
		return false, false, err.Error()
	}

	queryCtx, cancel := context.WithTimeout(ctx, openBaoQueryTimeout)
	status, err := queryOpenBaoSealStatus(queryCtx)
	cancel()
	if err != nil {
		return false, false, err.Error()
	}

	if !status.Initialized {
		if !plan.BootstrapNode {
			fetched, fetchErr := syncOpenBaoInitFromPeers(ctx, r, mc, plan)
			if fetchErr != nil && fetched {
				return false, true, fetchErr.Error()
			}

			return false, true, "waiting for host-native openbao raft join"
		}

		initCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openBaoInitTimeout)
		resp, initErr := initOpenBao(initCtx)
		cancel()
		if initErr != nil {
			postInitCtx, cancel := context.WithTimeout(ctx, openBaoQueryTimeout)
			postInitStatus, postInitErr := queryOpenBaoSealStatus(postInitCtx)
			cancel()
			if postInitErr == nil && postInitStatus.Initialized {
				return true, postInitStatus.Sealed, "openbao initialized but init response was lost; wipe /var/lib/chubo/openbao to retry"
			}

			return false, true, initErr.Error()
		}

		if writeErr := persistOpenBaoInitState(ctx, r, resp); writeErr != nil {
			return false, true, writeErr.Error()
		}

		// Keep the legacy file write as a compatibility path for helper bundles,
		// but treat the sensitive runtime resource as the source of truth.
		if writeErr := persistOpenBaoInit(resp); writeErr != nil {
			lastError = writeErr.Error()
		}

		status.Initialized = true
		status.Sealed = true
	}

	if status.Sealed {
		initData, readErr := readOpenBaoInitState(ctx, r)
		if readErr != nil {
			fetched, fetchErr := syncOpenBaoInitFromPeers(ctx, r, mc, plan)
			if !fetched {
				if fetchErr != nil {
					return status.Initialized, true, fetchErr.Error()
				}

				return status.Initialized, true, "waiting for host-native openbao init material"
			}

			initData, readErr = readOpenBaoInitState(ctx, r)
			if readErr != nil {
				return status.Initialized, true, readErr.Error()
			}
		}

		if len(initData.KeysBase64) == 0 {
			return status.Initialized, true, "openbao init data has no unseal keys"
		}

		unsealCtx, cancel := context.WithTimeout(ctx, openBaoUnsealTimeout)
		err = unsealOpenBao(unsealCtx, initData.KeysBase64[0])
		cancel()
		if err != nil {
			return status.Initialized, true, err.Error()
		}

		queryCtx, cancel = context.WithTimeout(ctx, openBaoQueryTimeout)
		status, err = queryOpenBaoSealStatus(queryCtx)
		cancel()
		if err != nil {
			return true, true, err.Error()
		}
	}

	return status.Initialized, status.Sealed, ""
}

func loadOpenBaoHostPlan(mc *config.MachineConfig) (openBaoHostPlan, error) {
	if mc == nil || mc.Config() == nil || mc.Config().Machine() == nil {
		return openBaoHostPlan{}, fmt.Errorf("host-native openbao requires active machine config")
	}

	files, err := mc.Config().Machine().Files()
	if err != nil {
		return openBaoHostPlan{}, err
	}

	var spec openBaoHostSpec
	found := false

	for _, f := range files {
		if f.Path() != openBaoHostSpecPath {
			continue
		}

		if err := json.Unmarshal([]byte(f.Content()), &spec); err != nil {
			return openBaoHostPlan{}, fmt.Errorf("failed to parse openbao host spec: %w", err)
		}

		found = true

		break
	}

	if !found {
		return openBaoHostPlan{}, fmt.Errorf("host-native openbao spec file is missing")
	}

	if strings.TrimSpace(spec.NetworkInterface) == "" {
		return openBaoHostPlan{}, fmt.Errorf("host-native openbao spec is missing network interface")
	}

	localIP, err := interfaceFirstGlobalUnicast(spec.NetworkInterface)
	if err != nil {
		return openBaoHostPlan{}, err
	}

	return openBaoHostPlan{
		LocalIP:       localIP,
		NetworkIface:  spec.NetworkInterface,
		RetryJoin:     append([]string(nil), spec.RetryJoin...),
		BootstrapNode: isOpenBaoBootstrapNode(localIP, spec.RetryJoin),
	}, nil
}

func interfaceFirstGlobalUnicast(name string) (netip.Addr, error) {
	iface, err := net.InterfaceByName(strings.TrimSpace(name))
	if err != nil {
		return netip.Addr{}, fmt.Errorf("failed to find interface %q for host-native openbao: %w", name, err)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return netip.Addr{}, fmt.Errorf("failed to list addresses for %q: %w", name, err)
	}

	var ipv6Candidate netip.Addr

	for _, addr := range addrs {
		prefix, err := netip.ParsePrefix(addr.String())
		if err != nil {
			continue
		}

		ip := prefix.Addr().Unmap()
		if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsLoopback() {
			continue
		}

		if ip.Is4() {
			return ip, nil
		}

		if !ipv6Candidate.IsValid() {
			ipv6Candidate = ip
		}
	}

	if ipv6Candidate.IsValid() {
		return ipv6Candidate, nil
	}

	return netip.Addr{}, fmt.Errorf("interface %q has no global unicast address for host-native openbao", name)
}

func isOpenBaoBootstrapNode(localIP netip.Addr, peers []string) bool {
	candidates := []netip.Addr{localIP}

	for _, peer := range peers {
		ip, err := netip.ParseAddr(strings.TrimSpace(peer))
		if err != nil {
			continue
		}

		candidates = append(candidates, ip.Unmap())
	}

	slices.SortFunc(candidates, func(a, b netip.Addr) int {
		return a.Compare(b)
	})

	return len(candidates) > 0 && candidates[0] == localIP
}

func syncOpenBaoInitFromPeers(ctx context.Context, r controller.Runtime, mc *config.MachineConfig, plan openBaoHostPlan) (bool, error) {
	var lastErr error

	for _, peer := range plan.RetryJoin {
		if strings.TrimSpace(peer) == "" {
			continue
		}

		resp, err := readOpenBaoInitFromPeer(ctx, mc, peer)
		if err != nil {
			lastErr = err
			continue
		}

		if err := persistOpenBaoInitState(ctx, r, resp); err != nil {
			return false, err
		}

		if err := persistOpenBaoInit(resp); err != nil {
			return false, err
		}

		return true, nil
	}

	if lastErr != nil {
		return false, lastErr
	}

	return false, nil
}

func persistOpenBaoInitState(ctx context.Context, r controller.Runtime, resp openBaoInitResponse) error {
	return safe.WriterModify(ctx, r, secretres.NewOpenBaoInit(), func(res *secretres.OpenBaoInit) error {
		res.TypedSpec().RootToken = strings.TrimSpace(resp.RootToken)
		res.TypedSpec().KeysBase64 = append([]string(nil), resp.KeysBase64...)

		return nil
	})
}

func readOpenBaoInitState(ctx context.Context, r controller.Runtime) (openBaoInitResponse, error) {
	initRes, err := safe.ReaderGetByID[*secretres.OpenBaoInit](ctx, r, secretres.OpenBaoInitID)
	if err == nil {
		out := openBaoInitResponse{
			RootToken:  strings.TrimSpace(initRes.TypedSpec().RootToken),
			KeysBase64: append([]string(nil), initRes.TypedSpec().KeysBase64...),
		}

		if out.RootToken != "" || len(out.KeysBase64) > 0 {
			return out, nil
		}
	}

	return readOpenBaoInit()
}

func queryOpenBaoSealStatus(ctx context.Context) (openBaoSealStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openBaoHTTPAddress+"/v1/sys/seal-status", nil)
	if err != nil {
		return openBaoSealStatus{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return openBaoSealStatus{}, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return openBaoSealStatus{}, fmt.Errorf("openbao seal-status returned %s", resp.Status)
	}

	var out openBaoSealStatus
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return openBaoSealStatus{}, err
	}

	return out, nil
}

func initOpenBao(ctx context.Context) (openBaoInitResponse, error) {
	body, err := json.Marshal(map[string]int{
		"secret_shares":    1,
		"secret_threshold": 1,
	})
	if err != nil {
		return openBaoInitResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, openBaoHTTPAddress+"/v1/sys/init", bytes.NewReader(body))
	if err != nil {
		return openBaoInitResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return openBaoInitResponse{}, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return openBaoInitResponse{}, fmt.Errorf("openbao init returned %s", resp.Status)
	}

	var out openBaoInitResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return openBaoInitResponse{}, err
	}

	return out, nil
}

func unsealOpenBao(ctx context.Context, key string) error {
	body, err := json.Marshal(map[string]string{"key": key})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, openBaoHTTPAddress+"/v1/sys/unseal", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("openbao unseal returned %s", resp.Status)
	}

	return nil
}

func persistOpenBaoInit(resp openBaoInitResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(openBaoInitPath), 0o700); err != nil {
		return err
	}

	return os.WriteFile(openBaoInitPath, data, 0o600)
}

func readOpenBaoInit() (openBaoInitResponse, error) {
	data, err := os.ReadFile(openBaoInitPath)
	if err != nil {
		return openBaoInitResponse{}, err
	}

	var out openBaoInitResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return openBaoInitResponse{}, err
	}

	return out, nil
}

func readOpenBaoInitFromPeer(ctx context.Context, mc *config.MachineConfig, peer string) (openBaoInitResponse, error) {
	c, err := buildOpenBaoPeerClient(ctx, mc, peer)
	if err != nil {
		return openBaoInitResponse{}, err
	}
	defer c.Close() //nolint:errcheck

	readCtx, cancel := context.WithTimeout(ctx, openBaoQueryTimeout)
	defer cancel()

	reader, err := c.Read(readCtx, openBaoInitPath)
	if err != nil {
		return openBaoInitResponse{}, err
	}
	defer reader.Close() //nolint:errcheck

	data, err := io.ReadAll(reader)
	if err != nil {
		return openBaoInitResponse{}, err
	}

	var out openBaoInitResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return openBaoInitResponse{}, err
	}

	if strings.TrimSpace(out.RootToken) == "" || len(out.KeysBase64) == 0 {
		return openBaoInitResponse{}, fmt.Errorf("peer %s returned incomplete openbao init material", peer)
	}

	return out, nil
}

func buildOpenBaoPeerClient(ctx context.Context, mc *config.MachineConfig, peer string) (*machineryclient.Client, error) {
	if mc == nil || mc.Config() == nil {
		return nil, fmt.Errorf("machine config is required to build peer client")
	}

	now := time.Now()
	secretsBundle := gensecrets.NewBundleFromConfig(gensecrets.NewFixedClock(now), mc.Config())

	caBlock, _ := pem.Decode(secretsBundle.Certs.OS.Crt)
	if caBlock == nil || caBlock.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("failed to decode OS CA certificate")
	}

	caCert, err := stdlibx509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OS CA certificate: %w", err)
	}

	clientNotBefore := caCert.NotBefore
	clientNotAfter := now.Add(24 * time.Hour)
	if clientNotAfter.After(caCert.NotAfter) {
		clientNotAfter = caCert.NotAfter
	}
	if !clientNotAfter.After(clientNotBefore) {
		clientNotBefore = now.Add(-time.Hour)
		clientNotAfter = now.Add(24 * time.Hour)
	}

	clientCert, err := gensecrets.NewAdminCertificateAndKey(clientNotBefore, secretsBundle.Certs.OS, role.MakeSet(role.Admin), clientNotAfter.Sub(clientNotBefore))
	if err != nil {
		return nil, err
	}

	tlsCert, err := tls.X509KeyPair(clientCert.Crt, clientCert.Key)
	if err != nil {
		return nil, err
	}

	rootCAs := stdlibx509.NewCertPool()
	if !rootCAs.AppendCertsFromPEM(secretsBundle.Certs.OS.Crt) {
		return nil, fmt.Errorf("failed to append OS CA to root pool")
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		RootCAs:      rootCAs,
		Certificates: []tls.Certificate{tlsCert},
	}

	endpoint := net.JoinHostPort(strings.TrimSpace(peer), strconv.Itoa(constants.ApidPort))

	return machineryclient.New(ctx,
		machineryclient.WithTLSConfig(tlsConfig),
		machineryclient.WithEndpoints(endpoint),
	)
}
