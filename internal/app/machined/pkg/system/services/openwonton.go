// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package services

import (
	"context"
	"debug/elf"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/chubo-dev/chubo/internal/app/machined/pkg/runtime"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/events"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/health"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/runner"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/runner/process"
	"github.com/chubo-dev/chubo/internal/app/machined/pkg/system/runner/restart"
	"github.com/chubo-dev/chubo/internal/pkg/environment"
	"github.com/chubo-dev/chubo/pkg/conditions"
	"github.com/chubo-dev/chubo/pkg/machinery/constants"
	"github.com/chubo-dev/chubo/pkg/machinery/resources/network"
	timeresource "github.com/chubo-dev/chubo/pkg/machinery/resources/time"
)

const (
	// OpenWontonServiceID is the service ID exposed by machined.
	OpenWontonServiceID = "openwonton"

	openWontonBinaryPath      = "/var/lib/chubo/bin/openwonton"
	openWontonWontonPath      = "/var/lib/chubo/bin/wonton"
	openWontonDFShimPath      = "/var/lib/chubo/bin/df"
	openWontonConfigPath      = "/var/lib/chubo/config/openwonton.hcl"
	openWontonArtifactURLPath = "/var/lib/chubo/config/openwonton.artifact_url"
	openWontonDataDir         = "/var/lib/chubo/openwonton"
	openWontonAllocDir        = "/var/lib/chubo/openwonton/alloc"
	openWontonGlibcDir        = "/var/lib/chubo/lib/openwonton"
	// Keep the fallback interpreter path short enough to fit PT_INTERP.
	openWontonInterpreterFallback = "/var/lib/chubo/ld-linux.so"
	chuboAgentBinaryPath          = "/usr/local/lib/containers/chubo-agent/usr/bin/chubo-agent"
	openWontonFallback            = "/usr/bin/init"
)

var _ system.HealthcheckedService = (*OpenWonton)(nil)

// OpenWonton implements an OS-managed Nomad-compatible service.
type OpenWonton struct{}

// ID implements the Service interface.
func (s *OpenWonton) ID(runtime.Runtime) string {
	return OpenWontonServiceID
}

// PreFunc implements the Service interface.
func (s *OpenWonton) PreFunc(ctx context.Context, r runtime.Runtime) error {
	if err := os.MkdirAll(filepath.Dir(openWontonBinaryPath), 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(openWontonConfigPath), 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(openWontonDataDir, 0o700); err != nil {
		return err
	}

	// openwonton client fingerprinting expects the alloc directory to exist.
	if err := os.MkdirAll(openWontonAllocDir, 0o700); err != nil {
		return err
	}

	if err := os.MkdirAll(openWontonGlibcDir, 0o755); err != nil {
		return err
	}

	if err := ensureOpenWontonDFShim(); err != nil {
		return err
	}

	if err := EnsureChuboServiceTLSMaterial(ctx, r, OpenWontonServiceID); err != nil {
		return err
	}

	release := openWontonOCIRelease

	if raw, err := os.ReadFile(openWontonArtifactURLPath); err == nil {
		override := strings.TrimSpace(string(raw))
		if override != "" {
			release = openWontonOCIRelease
			release.AssetURLs = make(map[string]string, len(openWontonOCIRelease.AssetURLs))

			for k, v := range openWontonOCIRelease.AssetURLs {
				release.AssetURLs[k] = v
			}

			release.AssetURLs[goruntime.GOARCH] = override
		}
	}

	// Prefer real openwonton release artifacts (wonton + glibc bundle from the OCI image tar).
	// Fall back to the local mock binary path if artifact install is unavailable (for example
	// network-restricted boots).
	if err := ensureOpenWontonRuntime(ctx, openWontonBinaryPath, openWontonWontonPath, openWontonGlibcDir, release); err == nil {
		// Ensure direct binary execution is possible so re-exec paths (task executor/plugin launch)
		// resolve os.Executable() to openwonton itself, not the dynamic loader path.
		if err = ensureOpenWontonInterpreter(openWontonBinaryPath, openWontonGlibcDir); err != nil {
			return err
		}

		return nil
	}

	return ensureServiceBinary(openWontonBinaryPath, openWontonFallback)
}

// PostFunc implements the Service interface.
func (s *OpenWonton) PostFunc(runtime.Runtime, events.ServiceState) error {
	return nil
}

// Condition implements the Service interface.
func (s *OpenWonton) Condition(r runtime.Runtime) conditions.Condition {
	return conditions.WaitForAll(
		timeresource.NewSyncCondition(r.State().V1Alpha2().Resources()),
		network.NewReadyCondition(r.State().V1Alpha2().Resources(), network.AddressReady, network.HostnameReady),
		conditions.WaitForFileToExist(openWontonConfigPath),
	)
}

// DependsOn implements the Service interface.
func (s *OpenWonton) DependsOn(runtime.Runtime) []string {
	return []string{machinedServiceID, DockerServiceID}
}

// Volumes implements the Service interface.
func (s *OpenWonton) Volumes(runtime.Runtime) []string {
	return nil
}

// Runner implements the Service interface.
func (s *OpenWonton) Runner(r runtime.Runtime) (runner.Runner, error) {
	debug := false

	if r.Config() != nil {
		debug = r.Config().Debug()
	}

	isDynamic, err := openWontonBinaryIsDynamic(openWontonBinaryPath)
	if err != nil {
		return nil, err
	}

	processArgs := []string{
		openWontonBinaryPath,
		"agent",
		"-config",
		openWontonConfigPath,
	}

	envVars := setEnvVar(environment.Get(r.Config()), "PATH", filepath.Dir(openWontonDFShimPath)+":"+constants.PATH)
	envVars = append(envVars, constants.EnvXDGRuntimeDir)

	if isDynamic {
		// Re-check in runner context to guarantee interpreter materialization/patching
		// right before process launch (mount namespaces may differ from PreFunc path).
		if err = ensureOpenWontonInterpreter(openWontonBinaryPath, openWontonGlibcDir); err != nil {
			return nil, err
		}

		_, libraryPath, loaderErr := openWontonLoaderAndLibraryPath(openWontonGlibcDir)
		if loaderErr != nil {
			return nil, loaderErr
		}

		// Prefer direct exec with LD_LIBRARY_PATH so os.Executable()/re-exec paths resolve
		// to openwonton itself (not the dynamic loader binary path).
		envVars = setEnvVar(envVars, "LD_LIBRARY_PATH", libraryPath)
	}

	args := &runner.Args{
		ID:          s.ID(r),
		ProcessArgs: processArgs,
	}

	return restart.New(process.NewRunner(
		debug,
		args,
		runner.WithLoggingManager(r.Logging()),
		runner.WithEnv(envVars),
		runner.WithOOMScoreAdj(-700),
		runner.WithCgroupPath(constants.CgroupSystem+"/openwonton"),
		runner.WithDroppedCapabilities(constants.DefaultDroppedCapabilities),
	),
		restart.WithType(restart.Forever),
	), nil
}

func openWontonBinaryIsDynamic(path string) (bool, error) {
	ef, err := elf.Open(path)
	if err != nil {
		return false, err
	}
	defer ef.Close() //nolint:errcheck

	for _, prog := range ef.Progs {
		if prog.Type == elf.PT_INTERP {
			return true, nil
		}
	}

	return false, nil
}

func openWontonInterpreterPath(path string) (string, error) {
	ef, err := elf.Open(path)
	if err != nil {
		return "", err
	}

	defer ef.Close() //nolint:errcheck

	for _, prog := range ef.Progs {
		if prog.Type != elf.PT_INTERP {
			continue
		}

		if prog.Filesz == 0 {
			break
		}

		buf := make([]byte, prog.Filesz)

		n, readErr := prog.ReadAt(buf, 0)
		if readErr != nil {
			return "", readErr
		}

		return strings.TrimRight(string(buf[:n]), "\x00"), nil
	}

	return "", errors.New("missing PT_INTERP")
}

func ensureOpenWontonInterpreter(binaryPath, glibcDir string) error {
	loaderPath, _, err := openWontonLoaderAndLibraryPath(glibcDir)
	if err != nil {
		return err
	}

	interpreterPath, err := openWontonInterpreterPath(binaryPath)
	if err != nil {
		return err
	}

	if pathExists(interpreterPath) {
		return nil
	}

	// First try to materialize the interpreter path requested by the ELF as-is.
	if err = ensureSymlinkTarget(loaderPath, interpreterPath); err == nil {
		return nil
	}

	// If the OS root is immutable and the ELF interpreter points to an unwriteable location
	// (for example /lib64/ld-linux-x86-64.so.2), patch PT_INTERP to a writable stable path
	// under /var/lib/chubo and link that path to the bundled loader.
	if err = ensureSymlinkTarget(loaderPath, openWontonInterpreterFallback); err != nil {
		return err
	}

	return patchELFInterpreter(binaryPath, openWontonInterpreterFallback)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

func ensureSymlinkTarget(target, linkPath string) error {
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		return err
	}

	info, err := os.Lstat(linkPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			currentTarget, readErr := os.Readlink(linkPath)
			if readErr == nil && currentTarget == target {
				return nil
			}
		}

		if removeErr := os.Remove(linkPath); removeErr != nil {
			return removeErr
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.Symlink(target, linkPath)
}

func patchELFInterpreter(binaryPath, interpreterPath string) error {
	ef, err := elf.Open(binaryPath)
	if err != nil {
		return err
	}

	defer ef.Close() //nolint:errcheck

	var interpProg *elf.Prog

	for _, prog := range ef.Progs {
		if prog.Type == elf.PT_INTERP {
			interpProg = prog

			break
		}
	}

	if interpProg == nil {
		return errors.New("missing PT_INTERP")
	}

	if int(interpProg.Filesz) < len(interpreterPath)+1 {
		return fmt.Errorf("fallback PT_INTERP path %q too long for binary (%d bytes)", interpreterPath, interpProg.Filesz)
	}

	f, err := os.OpenFile(binaryPath, os.O_WRONLY, 0)
	if err != nil {
		return err
	}

	defer f.Close() //nolint:errcheck

	buf := make([]byte, interpProg.Filesz)
	copy(buf, []byte(interpreterPath))

	if _, err = f.WriteAt(buf, int64(interpProg.Off)); err != nil {
		return err
	}

	return nil
}

// HealthFunc implements the HealthcheckedService interface.
func (s *OpenWonton) HealthFunc(runtime.Runtime) health.Check {
	return func(ctx context.Context) error {
		client, err := NewChuboServiceHTTPClient(OpenWontonServiceID, 2*time.Second)
		if err != nil {
			return err
		}

		return simpleHealthCheckWithClient(ctx, "https://127.0.0.1:4646/v1/status/leader", client)
	}
}

// HealthSettings implements the HealthcheckedService interface.
func (s *OpenWonton) HealthSettings(runtime.Runtime) *health.Settings {
	settings := health.DefaultSettings
	// OpenWonton can take a bit longer to bind TLS listeners after first boot/install.
	// Avoid a false-negative first probe which leaves the service marked unhealthy.
	settings.InitialDelay = 30 * time.Second

	return &settings
}

// APIRestartAllowed implements APIRestartableService.
func (s *OpenWonton) APIRestartAllowed(runtime.Runtime) bool {
	return true
}

// APIStartAllowed implements APIStartableService.
func (s *OpenWonton) APIStartAllowed(runtime.Runtime) bool {
	return true
}

// APIStopAllowed implements APIStoppableService.
func (s *OpenWonton) APIStopAllowed(runtime.Runtime) bool {
	return true
}

func ensureOpenWontonDFShim() error {
	info, err := os.Lstat(openWontonDFShimPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			target, readErr := os.Readlink(openWontonDFShimPath)
			if readErr == nil && target == chuboAgentBinaryPath {
				return nil
			}
		}

		if removeErr := os.Remove(openWontonDFShimPath); removeErr != nil {
			return removeErr
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	return os.Symlink(chuboAgentBinaryPath, openWontonDFShimPath)
}

func setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	result := make([]string, 0, len(env)+1)
	result = append(result, prefix+value)

	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			continue
		}

		result = append(result, entry)
	}

	return result
}
