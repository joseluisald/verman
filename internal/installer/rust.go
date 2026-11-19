package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/joseluisald/verman/internal/tool"
)

type rustProvider struct {
	baseURL, apiURL string
	client          *http.Client
}

func (p *rustProvider) Source() string { return p.apiURL }
func (p *rustProvider) artifact(version, hash string, platform Platform) Artifact {
	return Artifact{Language: "rust", Version: version, Build: version, Platform: platform, URL: p.baseURL + "/rustup/dist/x86_64-pc-windows-msvc/rustup-init.exe", SHA256: hash, ChecksumURL: p.baseURL + "/rustup/dist/x86_64-pc-windows-msvc/rustup-init.exe.sha256", Format: "file", ArchiveRoot: "rustup-init.exe", Variant: "rustup-minimal-msvc", Executables: map[string]string{"rust": "cargo/bin/rustc.exe", "rustc": "cargo/bin/rustc.exe", "rustdoc": "cargo/bin/rustdoc.exe", "cargo": "cargo/bin/cargo.exe", "rustup": "cargo/bin/rustup.exe"}}
}
func (p *rustProvider) ValidateArtifact(a Artifact) error {
	if err := tool.Validate("rust", a.Version); err != nil {
		return err
	}
	if err := windowsTarget("rust", a.Platform); err != nil {
		return err
	}
	if !validSHA256(a.SHA256) || !reflect.DeepEqual(a, p.artifact(a.Version, a.SHA256, a.Platform)) {
		return fmt.Errorf("artefato Rust inválido")
	}
	return nil
}
func (p *rustProvider) initHash(ctx context.Context) (string, error) {
	a := p.artifact("1.0.0", "", Platform{"windows", "amd64"})
	data, err := get(ctx, p.client, a.ChecksumURL, 4096)
	if err != nil {
		return "", err
	}
	parts := strings.Fields(string(data))
	if len(parts) == 0 || !validSHA256(parts[0]) {
		return "", fmt.Errorf("hash do rustup-init inválido")
	}
	return parts[0], nil
}
func (p *rustProvider) List(ctx context.Context, platform Platform) ([]Artifact, error) {
	if err := windowsTarget("Rust", platform); err != nil {
		return nil, err
	}
	data, err := get(ctx, p.client, p.Source(), 16<<20)
	if err != nil {
		return nil, err
	}
	var releases []githubRelease
	if err := json.Unmarshal(data, &releases); err != nil || releases == nil {
		return nil, fmt.Errorf("catálogo Rust inválido: %v", err)
	}
	hash, err := p.initHash(ctx)
	if err != nil {
		return nil, err
	}
	result := []Artifact{}
	seen := map[string]bool{}
	for _, r := range releases {
		version := strings.TrimPrefix(r.Tag, "v")
		if r.Draft || r.Prerelease || seen[version] || tool.Validate("rust", version) != nil {
			continue
		}
		result = append(result, p.artifact(version, hash, platform))
		seen[version] = true
	}
	sort.Slice(result, func(i, j int) bool { return versionGreater(result[i].Version, result[j].Version) })
	return result, nil
}
func (p *rustProvider) ResolveExact(ctx context.Context, version string, platform Platform) (Artifact, error) {
	if err := tool.Validate("rust", version); err != nil {
		return Artifact{}, err
	}
	if err := windowsTarget("Rust", platform); err != nil {
		return Artifact{}, err
	}
	data, err := get(ctx, p.client, p.baseURL+"/dist/channel-rust-"+version+".toml", 12<<20)
	if err != nil {
		return Artifact{}, err
	}
	var manifest struct {
		Pkg map[string]struct {
			Version string
			Target  map[string]struct{ Available bool }
		}
	}
	if _, err := toml.Decode(string(data), &manifest); err != nil {
		return Artifact{}, err
	}
	pkg, ok := manifest.Pkg["rust"]
	fields := strings.Fields(pkg.Version)
	if !ok || len(fields) == 0 || fields[0] != version || !pkg.Target["x86_64-pc-windows-msvc"].Available {
		return Artifact{}, fmt.Errorf("Rust %s não disponível para Windows/amd64", version)
	}
	hash, err := p.initHash(ctx)
	if err != nil {
		return Artifact{}, err
	}
	return p.artifact(version, hash, platform), nil
}
func (p *rustProvider) Resolve(ctx context.Context, a Artifact) (Artifact, error) {
	if err := p.ValidateArtifact(a); err != nil {
		return Artifact{}, err
	}
	return p.ResolveExact(ctx, a.Version, a.Platform)
}
