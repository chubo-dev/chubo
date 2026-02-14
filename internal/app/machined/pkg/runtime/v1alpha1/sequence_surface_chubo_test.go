//go:build chubo

package v1alpha1

import (
	"testing"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
)

func TestShouldStartDashboardDisabledInChubo(t *testing.T) {
	t.Parallel()

	for _, mode := range []runtime.Mode{
		runtime.ModeContainer,
		runtime.ModeMetal,
		runtime.ModeMetalAgent,
	} {
		if shouldStartDashboard(mode) {
			t.Fatalf("shouldStartDashboard(%q) = true, want false", mode)
		}
	}
}
