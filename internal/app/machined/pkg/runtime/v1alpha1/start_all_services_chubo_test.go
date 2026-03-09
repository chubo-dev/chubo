package v1alpha1

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldStartDockerForOpenWonton(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	configPath := filepath.Join(base, "openwonton.hcl")
	rolePath := filepath.Join(base, "openwonton.role")

	if shouldStartDockerForOpenWonton(configPath, rolePath) {
		t.Fatal("expected false when no openwonton config files exist")
	}

	if err := os.WriteFile(configPath, []byte("data"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if !shouldStartDockerForOpenWonton(configPath, rolePath) {
		t.Fatal("expected true when openwonton config file exists")
	}

	if err := os.Remove(configPath); err != nil {
		t.Fatalf("remove config: %v", err)
	}

	if err := os.WriteFile(rolePath, []byte("server-client"), 0o600); err != nil {
		t.Fatalf("write role: %v", err)
	}

	if !shouldStartDockerForOpenWonton(configPath, rolePath) {
		t.Fatal("expected true when openwonton role file exists")
	}
}
