//go:build !chuboos

package v1alpha1

import (
	"testing"

	"github.com/siderolabs/talos/internal/app/machined/pkg/runtime"
)

func TestShouldStartDashboardDisabledForMetalAgent(t *testing.T) {
	t.Parallel()

	if shouldStartDashboard(runtime.ModeMetalAgent) {
		t.Fatal("shouldStartDashboard(ModeMetalAgent) = true, want false")
	}
}
