package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/tool"
)

type directoryKey struct{}

func WithWorkingDirectory(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, directoryKey{}, dir)
}
func workingDirectory(ctx context.Context) (string, error) {
	if dir, _ := ctx.Value(directoryKey{}).(string); dir != "" {
		return dir, nil
	}
	return os.Getwd()
}

// ToolVariables também é usada pela ativação e nunca modifica o processo pai.
func ToolVariables(language, version, root string) map[string]string {
	switch language {
	case "go":
		return map[string]string{"GOROOT": root, "GOTOOLCHAIN": "local"}
	case "rust":
		return map[string]string{"CARGO_HOME": filepath.Join(root, "cargo"), "RUSTUP_HOME": filepath.Join(root, "rustup"), "RUSTUP_TOOLCHAIN": version + "-x86_64-pc-windows-msvc"}
	case "vcpkg":
		return map[string]string{"VCPKG_ROOT": root}
	case "chocolatey":
		return map[string]string{"CHOCOLATEYINSTALL": filepath.Join(root, "tools", "chocolateyInstall")}
	}
	return nil
}

func toolEnvironment(language, version, root string) []string {
	values := ToolVariables(language, version, root)
	env := []string{}
	for _, entry := range os.Environ() {
		key := strings.SplitN(entry, "=", 2)[0]
		skip := false
		for name := range values {
			if strings.EqualFold(key, name) {
				skip = true
			}
		}
		upper := strings.ToUpper(key)
		if upper == "NODE_OPTIONS" || upper == "NODE_PATH" || upper == "RUSTUP_DIST_SERVER" || upper == "RUSTUP_UPDATE_ROOT" || upper == "RUSTUP_INIT_SKIP_PATH_CHECK" || upper == "RUSTUP_TOOLCHAIN" || upper == "PHPRC" || upper == "PHP_INI_SCAN_DIR" || strings.HasPrefix(upper, "NPM_CONFIG_") {
			skip = true
		}
		if !skip {
			env = append(env, entry)
		}
	}
	for key, value := range values {
		env = append(env, key+"="+value)
	}
	return env
}
func nativeCheck(ctx context.Context, root, binary string, args, env []string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = root
	cmd.Env = env
	cmd.WaitDelay = 2 * time.Second
	var stdout, stderr limitedOutput
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("executar %s: %w (%s %s)", filepath.Base(binary), err, stdout.data, stderr.data)
	}
	return strings.TrimSpace(string(stdout.data) + " " + string(stderr.data)), nil
}
func selectedDependency(ctx context.Context, language string) (string, string, func(), error) {
	dir, err := workingDirectory(ctx)
	if err != nil {
		return "", "", nil, err
	}
	cfg, err := config.LoadEffectiveConfig(dir)
	if err != nil {
		return "", "", nil, err
	}
	version := cfg.Tools[language]
	if version == "" {
		return "", "", nil, fmt.Errorf("selecione %s primeiro: verman use %s latest", language, language)
	}
	lease, err := Lease(ctx, language, version)
	if err != nil {
		return "", "", nil, err
	}
	meta, err := ReadMetadata(language, version)
	if err != nil {
		lease()
		return "", "", nil, err
	}
	root, err := config.InstallDir(language, version)
	if err != nil {
		lease()
		return "", "", nil, err
	}
	return filepath.Join(root, filepath.FromSlash(meta.Executables[language])), root, lease, nil
}
func prepareTool(ctx context.Context, root string, a Artifact) error {
	env := toolEnvironment(a.Language, a.Version, root)
	if a.Language == "rust" {
		// rustup verifica os manifests e hashes da toolchain; todos os seus dados ficam no staging.
		env = append(env, "RUSTUP_DIST_SERVER=https://static.rust-lang.org", "RUSTUP_UPDATE_ROOT=https://static.rust-lang.org/rustup", "RUSTUP_INIT_SKIP_MSVC_CHECK=yes", "RUSTUP_INIT_SKIP_PATH_CHECK=yes")
		_, err := nativeCheck(ctx, root, filepath.Join(root, "rustup-init.exe"), []string{"-y", "--no-modify-path", "--profile", "minimal", "--default-toolchain", a.Version + "-x86_64-pc-windows-msvc"}, env, 10*time.Minute)
		if err != nil {
			return err
		}
	}
	if a.Language == "vcpkg" {
		if err := prepareVcpkg(ctx, root, a); err != nil {
			return err
		}
	}
	for _, relative := range a.Executables {
		if _, err := executablePath(root, relative); err != nil {
			return err
		}
	}
	binary := filepath.Join(root, filepath.FromSlash(a.Executables[a.Language]))
	args := []string{"--version"}
	if d, ok := tool.Lookup(a.Language); ok && d.Dependency != "" {
		executable, dependencyRoot, release, err := selectedDependency(ctx, d.Dependency)
		if err != nil {
			return err
		}
		defer release()
		args = append([]string{binary}, args...)
		binary = executable
		if d.Dependency == "php" {
			env = append(env, "PHPRC="+dependencyRoot, "PHP_INI_SCAN_DIR="+filepath.Join(dependencyRoot, "conf.d"))
		}
	}
	if a.Language == "go" {
		args = []string{"version"}
	}
	if a.Language == "vcpkg" {
		args = []string{"version"}
		env = append(env, "VCPKG_DISABLE_METRICS=1")
	}
	result, err := nativeCheck(ctx, root, binary, args, env, 90*time.Second)
	if err != nil {
		return err
	}
	expected := a.Version
	if a.Language == "vcpkg" {
		expected = a.Build
	}
	if a.Language == "go" {
		expected = goBuild(a.Version)
	}
	if !strings.Contains(result, expected) {
		return fmt.Errorf("%s retornou %q; esperado %s", a.Language, result, expected)
	}
	if a.Language == "rust" {
		_, err = nativeCheck(ctx, root, filepath.Join(root, "cargo", "bin", "cargo.exe"), []string{"--version"}, env, 30*time.Second)
	}
	return err
}
func prepareVcpkg(ctx context.Context, root string, a Artifact) error {
	provider, err := providerFor("vcpkg")
	if err != nil {
		return err
	}
	p := provider.(*githubToolProvider)
	data, err := get(ctx, p.client, p.apiURL+"/tags/"+a.Build, 12<<20)
	if err != nil {
		return err
	}
	var release githubRelease
	if err := json.Unmarshal(data, &release); err != nil {
		return err
	}
	if release.Tag != a.Build || release.Draft || release.Prerelease {
		return fmt.Errorf("release vcpkg inconsistente")
	}
	name := "vcpkg-standalone-bundle.tar.gz"
	url := p.downloadURL + "/" + a.Build + "/" + name
	hash := ""
	for _, asset := range release.Assets {
		if asset.Name == name {
			if hash != "" || asset.URL != url {
				return fmt.Errorf("bundle vcpkg inconsistente")
			}
			hash = strings.TrimPrefix(asset.Digest, "sha256:")
		}
	}
	if !validSHA256(hash) {
		return fmt.Errorf("bundle vcpkg sem digest SHA-256 oficial")
	}
	path := filepath.Join(root, ".verman-bundle.tar.gz")
	if err := download(ctx, p.client, Artifact{URL: url, SHA256: hash}, path); err != nil {
		return err
	}
	if err := extractTarGzip(ctx, path, root, ""); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	receipt, _ := json.Marshal(map[string]string{"url": url, "sha256": hash})
	return os.WriteFile(filepath.Join(root, ".verman-bundle.json"), receipt, 0600)
}
