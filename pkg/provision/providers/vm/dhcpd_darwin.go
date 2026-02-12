// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package vm

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/siderolabs/go-retry/retry"

	"github.com/chubo-dev/chubo/pkg/provision"
)

// CreateDHCPd creates a DHCP server on darwin.
// It waits for the interface to appear, shut's down the apple bootp DHCPd server created by qemu by default,
// starts the talos DHCP server and then starts the apple bootp server again, which is configured such
// that it detects existing dhcp servers on interfaces and doesn't interfare with them.
func (p *Provisioner) CreateDHCPd(ctx context.Context, state *provision.State, clusterReq provision.ClusterRequest) error {
	// QEMU's vmnet-shared backend chooses the bridge name internally. We try to predict it
	// (state.BridgeName), but on some setups that prediction can be wrong. In that case, fall back
	// to discovering the interface by the expected gateway IP (vmnet assigns the gateway to the host bridge).
	gateway, hasGateway := pickGatewayIPv4(clusterReq.Network.GatewayAddrs)

	bridgeName, err := waitForInterface(ctx, state.BridgeName, gateway, hasGateway)
	if err != nil {
		return err
	}

	state.BridgeName = bridgeName

	cmd := exec.CommandContext(ctx, "/bin/launchctl", "unload", "-w", "/System/Library/LaunchDaemons/bootps.plist")

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to stop native dhcp server: %w", err)
	}

	err = p.startDHCPd(state, clusterReq)
	if err != nil {
		return err
	}

	err = waitForDHCPServerUp(ctx, state)
	if err != nil {
		return err
	}

	time.Sleep(time.Second)

	cmd = exec.CommandContext(ctx, "/bin/launchctl", "load", "-w", "/System/Library/LaunchDaemons/bootps.plist")

	err = cmd.Run()
	if err != nil {
		fmt.Printf("warning: failed to start native dhcp server after creating a talos dhcp server: %s", err)
	}

	return nil
}

func pickGatewayIPv4(gateways []netip.Addr) (netip.Addr, bool) {
	for _, gw := range gateways {
		if gw.Is4() && gw.IsValid() {
			return gw, true
		}
	}

	return netip.Addr{}, false
}

func parseBridgeIndex(name string) (int, bool) {
	if !strings.HasPrefix(name, "bridge") {
		return 0, false
	}

	suffix := strings.TrimPrefix(name, "bridge")
	if suffix == "" {
		return 0, false
	}

	n, err := strconv.Atoi(suffix)
	if err != nil {
		return 0, false
	}

	return n, true
}

func ifaceHasIP(iface net.Interface, ip netip.Addr) bool {
	addrs, err := iface.Addrs()
	if err != nil {
		return false
	}

	for _, addr := range addrs {
		prefix, err := netip.ParsePrefix(addr.String())
		if err != nil {
			continue
		}

		if prefix.Addr() == ip {
			return true
		}
	}

	return false
}

// waitForInterface returns the name of a vmnet bridge interface once it is available.
//
// On Darwin, QEMU creates the vmnet bridge interface asynchronously. We primarily wait for the predicted
// interfaceName, but if a gateway is provided we can also discover the actual interface by looking for a
// vmnet-style bridge that owns the gateway IP.
func waitForInterface(ctx context.Context, interfaceName string, gateway netip.Addr, hasGateway bool) (string, error) {
	var found string

	err := retry.Constant(1*time.Minute, retry.WithUnits(50*time.Millisecond)).RetryWithContext(ctx, func(_ context.Context) error {
		ifaces, err := net.Interfaces()
		if err != nil {
			return err
		}

		for _, iface := range ifaces {
			if iface.Name == interfaceName {
				found = interfaceName
				return nil
			}
		}

		if hasGateway {
			for _, iface := range ifaces {
				bridgeIndex, ok := parseBridgeIndex(iface.Name)
				if !ok || bridgeIndex < 100 {
					continue
				}

				if ifaceHasIP(iface, gateway) {
					found = iface.Name
					return nil
				}
			}
		}

		return retry.ExpectedError(fmt.Errorf("interface %s not found", interfaceName))
	})

	if err != nil {
		return "", err
	}

	if found == "" {
		return "", fmt.Errorf("interface %s not found (retry finished without match)", interfaceName)
	}

	return found, nil
}

func waitForDHCPServerUp(ctx context.Context, state *provision.State) error {
	return retry.Constant(1*time.Minute, retry.WithUnits(100*time.Millisecond)).RetryWithContext(ctx, func(_ context.Context) error {
		logFileData, err := os.ReadFile(state.GetRelativePath(dhcpLog))
		if err != nil {
			return retry.ExpectedError(err)
		}

		if strings.Contains(string(logFileData), "Ready to handle requests") {
			return nil
		}

		return retry.ExpectedError(fmt.Errorf("failure: DHCPd server has not started"))
	})
}
