package network

import (
	"net/netip"
	"testing"

	"github.com/jsimonetti/rtnetlink/v2"
	"github.com/siderolabs/gen/value"

	"github.com/siderolabs/talos/pkg/machinery/nethelpers"
	resourcenetwork "github.com/siderolabs/talos/pkg/machinery/resources/network"
)

func TestRouteMatchesSpec(t *testing.T) {
	t.Parallel()

	spec := &resourcenetwork.RouteSpecSpec{
		Family:      nethelpers.FamilyInet4,
		Destination: netip.Prefix{},
		Source:      netip.MustParseAddr("192.168.0.142"),
		Gateway:     netip.MustParseAddr("192.168.0.1"),
		Table:       nethelpers.TableMain,
		Priority:    1024,
		Scope:       nethelpers.ScopeGlobal,
		Type:        nethelpers.TypeUnicast,
		Protocol:    nethelpers.ProtocolBoot,
		MTU:         1500,
	}

	msg := &rtnetlink.RouteMessage{
		Family:    uint8(nethelpers.FamilyInet4),
		DstLength: 0,
		Protocol:  uint8(nethelpers.ProtocolBoot),
		Scope:     uint8(nethelpers.ScopeGlobal),
		Type:      uint8(nethelpers.TypeUnicast),
		Attributes: rtnetlink.RouteAttributes{
			Dst:      nil,
			Src:      netip.MustParseAddr("192.168.0.142").AsSlice(),
			Gateway:  netip.MustParseAddr("192.168.0.1").AsSlice(),
			OutIface: 9,
			Priority: 1024,
			Table:    uint32(nethelpers.TableMain),
			Metrics: &rtnetlink.RouteMetrics{
				MTU: 1500,
			},
		},
	}

	if !routeMatchesSpec(msg, spec, 9) {
		t.Fatal("expected route to match spec")
	}
}

func TestRouteMatchesSpecSourceOptional(t *testing.T) {
	t.Parallel()

	spec := &resourcenetwork.RouteSpecSpec{
		Family:      nethelpers.FamilyInet4,
		Destination: netip.Prefix{},
		Source:      netip.Addr{},
		Gateway:     netip.MustParseAddr("10.0.2.2"),
		Table:       nethelpers.TableMain,
		Priority:    1024,
		Scope:       nethelpers.ScopeGlobal,
		Type:        nethelpers.TypeUnicast,
		Protocol:    nethelpers.ProtocolBoot,
	}

	msg := &rtnetlink.RouteMessage{
		Family:    uint8(nethelpers.FamilyInet4),
		DstLength: 0,
		Protocol:  uint8(nethelpers.ProtocolBoot),
		Scope:     uint8(nethelpers.ScopeGlobal),
		Type:      uint8(nethelpers.TypeUnicast),
		Attributes: rtnetlink.RouteAttributes{
			Dst:      nil,
			Src:      netip.MustParseAddr("10.0.2.15").AsSlice(),
			Gateway:  netip.MustParseAddr("10.0.2.2").AsSlice(),
			OutIface: 8,
			Priority: 1024,
			Table:    uint32(nethelpers.TableMain),
		},
	}

	if !value.IsZero(spec.Source) {
		t.Fatal("test requires zero-value source")
	}

	if !routeMatchesSpec(msg, spec, 8) {
		t.Fatal("expected route to match spec when source is unspecified")
	}
}

func TestHasDefaultRoutePriorityCollision(t *testing.T) {
	t.Parallel()

	spec := &resourcenetwork.RouteSpecSpec{
		Family:      nethelpers.FamilyInet4,
		Destination: netip.Prefix{},
		Gateway:     netip.MustParseAddr("192.168.0.1"),
		Table:       nethelpers.TableMain,
		Priority:    1024,
	}

	routes := []rtnetlink.RouteMessage{
		{
			Family:    uint8(nethelpers.FamilyInet4),
			DstLength: 0,
			Attributes: rtnetlink.RouteAttributes{
				Dst:      nil,
				Gateway:  netip.MustParseAddr("10.0.2.2").AsSlice(),
				OutIface: 8,
				Priority: 1024,
				Table:    uint32(nethelpers.TableMain),
			},
		},
	}

	if !hasDefaultRoutePriorityCollision(routes, spec) {
		t.Fatal("expected default route priority collision")
	}

	spec.Priority = 2048
	if hasDefaultRoutePriorityCollision(routes, spec) {
		t.Fatal("did not expect collision for different priority")
	}
}
