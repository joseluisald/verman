package installer

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAdditionalGitHubCatalogsRevalidateOriginsAndDigests(t *testing.T) {
	tags := map[string]string{"bun": "bun-v1.4.4", "deno": "v2.9.8", "git": "v2.56.1.windows.1", "pnpm": "v12.21.0", "vcpkg": "2026-10-29", "chocolatey": "2.7.5"}
	for language, spec := range githubToolSpecs {
		t.Run(language, func(t *testing.T) {
			var release githubRelease
			calls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if strings.Contains(r.URL.Path, "/tags/") {
					json.NewEncoder(w).Encode(release)
				} else {
					json.NewEncoder(w).Encode([]githubRelease{release})
				}
			}))
			defer server.Close()
			p := &githubToolProvider{spec: spec, apiURL: server.URL + "/releases", downloadURL: server.URL + "/downloads", client: server.Client()}
			tag := tags[language]
			version := p.version(tag)
			hash := strings.Repeat("a", 64)
			release = githubRelease{Tag: tag, Assets: []githubAsset{{Name: p.filename(version, tag), URL: p.artifact(version, tag, hash, Platform{"windows", "amd64"}).URL, Digest: "sha256:" + hash}}}
			items, err := p.List(context.Background(), Platform{"windows", "amd64"})
			if err != nil || len(items) != 1 {
				t.Fatalf("list: %v %v", items, err)
			}
			a := items[0]
			if _, err := p.Resolve(context.Background(), a); err != nil || calls != 2 {
				t.Fatalf("resolve não reconsultou: %d %v", calls, err)
			}
			a.URL = "https://untrusted.example/runtime.exe"
			if err := p.ValidateArtifact(a); err == nil {
				t.Fatal("URL adulterada aceita")
			}
			release.Assets[0].Digest = ""
			items, err = p.List(context.Background(), Platform{"windows", "amd64"})
			if err == nil || len(items) != 0 {
				t.Fatal("asset sem digest aceito")
			}
			release.Assets[0].Digest = "sha256:" + hash
			release.Assets[0].URL = "https://untrusted.example/runtime.exe"
			if _, err := p.List(context.Background(), Platform{"windows", "amd64"}); err == nil {
				t.Fatal("URL remota adulterada aceita")
			}
		})
	}
}
func TestSHA512DownloadAuthenticatesBeforePublishing(t *testing.T) {
	content := []byte("pacote verificado pelo SRI SHA-512 do npm")
	sum := sha512.Sum512(content)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(content) }))
	defer server.Close()
	a := Artifact{Language: "yarn", URL: server.URL, SHA512: hex.EncodeToString(sum[:])}
	dir := t.TempDir()
	path := filepath.Join(dir, "verified")
	if err := download(context.Background(), server.Client(), a, path); err != nil {
		t.Fatal(err)
	}
	expected := sha256.Sum256(content)
	actual, err := hashFile(path)
	if err != nil || actual != hex.EncodeToString(expected[:]) {
		t.Fatal("hash local inconsistente")
	}
	a.SHA512 = strings.Repeat("0", 128)
	if err := download(context.Background(), server.Client(), a, filepath.Join(dir, "tampered")); err == nil {
		t.Fatal("SHA-512 inválido aceito")
	}
	if !Describe(Artifact{Language: "yarn", SHA512: hex.EncodeToString(sum[:])}).Installable {
		t.Fatal("Yarn com SRI válido bloqueado")
	}
}
func TestSingleFileExtractionRejectsPaths(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "source")
	if err := os.WriteFile(archive, []byte("fixture somente leitura"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../escape.exe", "C:/escape.exe", "sub/file.exe", ".", `sub\file.exe`} {
		if err := extractArchive(context.Background(), archive, dir, Artifact{Format: "file", ArchiveRoot: name}); err == nil {
			t.Fatalf("caminho aceito: %s", name)
		}
	}
	if err := extractArchive(context.Background(), archive, dir, Artifact{Format: "file", ArchiveRoot: "tool.exe"}); err != nil {
		t.Fatal(err)
	}
}

func TestGoCatalogNormalizesOldAndNewPatchZero(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]any{
			map[string]any{"version": "go1.27.0", "stable": true, "files": []any{map[string]string{"filename": "go1.27.0.windows-amd64.zip", "os": "windows", "arch": "amd64", "kind": "archive", "sha256": strings.Repeat("a", 64)}}},
			map[string]any{"version": "go1.20", "stable": true, "files": []any{map[string]string{"filename": "go1.20.windows-amd64.zip", "os": "windows", "arch": "amd64", "kind": "archive", "sha256": strings.Repeat("b", 64)}}},
			map[string]any{"version": "go1.4.3", "stable": true, "files": []any{map[string]string{"filename": "go1.4.3.windows-amd64.zip", "os": "windows", "arch": "amd64", "kind": "archive"}}},
		})
	}))
	defer server.Close()
	p := &goProvider{indexURL: server.URL, baseURL: server.URL, client: server.Client()}
	items, err := p.List(context.Background(), Platform{"windows", "amd64"})
	if err != nil || len(items) != 2 || items[0].Version != "1.27.0" || items[1].Version != "1.20.0" {
		t.Fatalf("%+v %v", items, err)
	}
	if normalizedDate("2026-13-31") != "" || normalizedDate("2026-02-31") != "" {
		t.Fatal("data inválida aceita")
	}
	if !newerToolBuild("git", "v2.56.1.windows.10", "v2.56.1.windows.2") {
		t.Fatal("revisão Git comparada como texto")
	}
}

func TestRustRepairPreservesUserBinariesAndFreshProxies(t *testing.T) {
	source, destination := t.TempDir(), t.TempDir()
	for _, root := range []string{source, destination} {
		if err := os.MkdirAll(filepath.Join(root, "cargo", "bin"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	for path, value := range map[string]string{filepath.Join(source, "cargo", "bin", "rustc.exe"): "old proxy", filepath.Join(source, "cargo", "bin", "user-tool.exe"): "user binary", filepath.Join(destination, "cargo", "bin", "rustc.exe"): "fresh proxy"} {
		if err := os.WriteFile(path, []byte(value), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := preserveMutable(context.Background(), source, destination, "rust"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "cargo", "bin", "rustc.exe"))
	if err != nil || string(data) != "fresh proxy" {
		t.Fatal("proxy substituído por versão antiga")
	}
	data, err = os.ReadFile(filepath.Join(destination, "cargo", "bin", "user-tool.exe"))
	if err != nil || string(data) != "user binary" {
		t.Fatal("binário do usuário não preservado")
	}
	if mutableFile("rust", "cargo/bin/rustc.exe") || !mutableFile("rust", "cargo/bin/user-tool.exe") {
		t.Fatal("limites de integridade incorretos")
	}
}

func TestChocolateyUninstallPreservesInstalledPackages(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "installs", "chocolatey", "2.7.5")
	lib := filepath.Join(root, "tools", "chocolateyInstall", "lib", "example-package")
	if err := os.MkdirAll(lib, 0700); err != nil {
		t.Fatal(err)
	}
	relative := "tools/chocolateyInstall/choco.exe"
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(relative)), []byte("fixture: nunca executar"), 0700); err != nil {
		t.Fatal(err)
	}
	meta := Metadata{SchemaVersion: 1, Language: "chocolatey", Version: "2.7.5", OS: "windows", Arch: "amd64", Source: "https://github.com/chocolatey/choco/releases/download/2.7.5/chocolatey.2.7.5.nupkg", Executables: map[string]string{"chocolatey": relative}}
	meta.OS = runtime.GOOS
	meta.Arch = runtime.GOARCH
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, MetadataFileName), data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := uninstallAt(context.Background(), filepath.Join(base, "installs"), "chocolatey", "2.7.5"); err == nil || !strings.Contains(err.Error(), "contém pacotes") {
		t.Fatalf("remoção deveria ser bloqueada: %v", err)
	}
	if _, err := os.Stat(lib); err != nil {
		t.Fatal("pacote apagado")
	}
}
