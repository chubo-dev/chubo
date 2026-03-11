// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package services

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFirstPrivateIPv4(t *testing.T) {
	t.Parallel()

	ifaces := []net.Interface{
		{Name: "eth1", Flags: net.FlagUp},
		{Name: "eth0", Flags: net.FlagUp},
		{Name: "lo", Flags: net.FlagUp | net.FlagLoopback},
	}

	addrsByName := map[string][]net.Addr{
		"eth0": {mustCIDR(t, "203.0.113.10/24")},
		"eth1": {mustCIDR(t, "10.0.0.12/24")},
		"lo":   {mustCIDR(t, "127.0.0.1/8")},
	}

	ip, err := firstPrivateIPv4FromInterfaces(ifaces, func(iface net.Interface) ([]net.Addr, error) {
		return addrsByName[iface.Name], nil
	})
	require.NoError(t, err)
	require.Equal(t, "10.0.0.12", ip)
}

func TestFirstPrivateIPv4SkipsContainerBridgeInterfaces(t *testing.T) {
	t.Parallel()

	ifaces := []net.Interface{
		{Name: "docker0", Flags: net.FlagUp},
		{Name: "eth1", Flags: net.FlagUp},
		{Name: "cni0", Flags: net.FlagUp},
	}

	addrsByName := map[string][]net.Addr{
		"docker0": {mustCIDR(t, "172.17.0.1/16")},
		"eth1":    {mustCIDR(t, "10.0.0.12/24")},
		"cni0":    {mustCIDR(t, "10.88.0.1/16")},
	}

	ip, err := firstPrivateIPv4FromInterfaces(ifaces, func(iface net.Interface) ([]net.Addr, error) {
		return addrsByName[iface.Name], nil
	})
	require.NoError(t, err)
	require.Equal(t, "10.0.0.12", ip)
}

func TestFirstPrivateIPv4ErrorWhenNoPrivateAddress(t *testing.T) {
	t.Parallel()

	ifaces := []net.Interface{
		{Name: "eth0", Flags: net.FlagUp},
	}

	addrsByName := map[string][]net.Addr{
		"eth0": {mustCIDR(t, "198.51.100.20/24")},
	}

	_, err := firstPrivateIPv4FromInterfaces(ifaces, func(iface net.Interface) ([]net.Addr, error) {
		return addrsByName[iface.Name], nil
	})
	require.EqualError(t, err, "no private IPv4 address found")
}

func mustCIDR(t *testing.T, cidr string) *net.IPNet {
	t.Helper()

	ip, ipNet, err := net.ParseCIDR(cidr)
	require.NoError(t, err)
	ipNet.IP = ip

	return ipNet
}
