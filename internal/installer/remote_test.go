package installer

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var windowsAMD64 = Platform{"windows", "amd64"}

func zipFixture(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var data bytes.Buffer
	w := zip.NewWriter(&data)
	for name, content := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(f, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestNodeCatalogCacheAndResolve(t *testing.T) {
	var calls atomic.Int32
	var broken atomic.Bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if broken.Load() {
			http.Error(w, "offline", 503)
			return
		}
		switch r.URL.Path {
		case "/index.json":
			io.WriteString(w, `[{"version":"v9.0.0","files":["win-x64-zip"]},{"version":"v22.11.0","files":["win-x64-zip"]},{"version":"v23.0.0-rc.1","files":["win-x64-zip"]},{"version":"v25.0.0","files":["linux-x64"]}]`)
		case "/v22.11.0/SHASUMS256.txt":
			fmt.Fprintf(w, "%s  node-v22.11.0-win-x64.zip\n", strings.Repeat("a", 64))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	p := &nodeProvider{baseURL: server.URL, client: server.Client()}
	cache := filepath.Join(t.TempDir(), "catalog.json")
	ctx := context.Background()
	result, err := loadCatalog(ctx, p, windowsAMD64, cache, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Catalog.Artifacts) != 2 || result.Catalog.Artifacts[0].Version != "22.11.0" {
		t.Fatalf("filtragem/ordem: %#v", result)
	}
	if _, err := loadCatalog(ctx, p, windowsAMD64, cache, false); err != nil || calls.Load() != 1 {
		t.Fatalf("cache não usado: %d, %v", calls.Load(), err)
	}
	if _, err := loadCatalog(ctx, p, windowsAMD64, cache, true); err != nil || calls.Load() != 2 {
		t.Fatalf("refresh não consultou fonte: %d, %v", calls.Load(), err)
	}
	artifact, err := p.Resolve(ctx, result.Catalog.Artifacts[0])
	if err != nil || artifact.SHA256 != strings.Repeat("a", 64) {
		t.Fatalf("checksum: %#v %v", artifact, err)
	}
	broken.Store(true)
	stale := result.Catalog
	stale.FetchedAt = time.Now().Add(-2 * time.Hour)
	data, err := json.Marshal(stale)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cache, data, 0600); err != nil {
		t.Fatal(err)
	}
	result, err = loadCatalog(ctx, p, windowsAMD64, cache, false)
	if err != nil || !strings.Contains(result.Warning, "usando cache") {
		t.Fatalf("fallback: %#v %v", result, err)
	}
	if _, err := loadCatalog(ctx, p, windowsAMD64, filepath.Join(t.TempDir(), "missing"), true); err == nil {
		t.Fatal("erro remoto ocultado sem cache")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := loadCatalog(cancelled, p, windowsAMD64, cache, true); err == nil {
		t.Fatal("cancelamento ignorado")
	}
	// Uma URL adulterada no cache nunca deve chegar ao resolvedor ou ao download.
	stale.Artifacts[0].URL = "https://evil.example/runtime.zip"
	data, err = json.Marshal(stale)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cache, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadCatalog(ctx, p, windowsAMD64, cache, false); err == nil {
		t.Fatal("cache adulterado aceito")
	}
	if _, err := p.List(ctx, Platform{"linux", "amd64"}); err == nil {
		t.Fatal("plataforma não suportada aceita")
	}
}

func TestNodeMalformedResponses(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `[]`, `[{"version":"v22.0.0"}]`, `not json`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) }))
			defer server.Close()
			p := &nodeProvider{baseURL: server.URL, client: server.Client()}
			if _, err := p.List(context.Background(), windowsAMD64); err == nil {
				t.Fatal("catálogo inválido aceito")
			}
		})
	}
	for _, checksum := range []string{"", "invalid node-v22.11.0-win-x64.zip", strings.Repeat("a", 64) + "  other.zip"} {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, checksum) }))
		p := &nodeProvider{baseURL: server.URL, client: server.Client()}
		_, err := p.Resolve(context.Background(), p.artifact("22.11.0", windowsAMD64))
		server.Close()
		if err == nil {
			t.Fatal("checksum inválido aceito")
		}
	}
}

func TestZIPRejectsUnsafeEntries(t *testing.T) {
	for _, name := range []string{"outside", "root/../../outside", "root/C:/escape", `root/..\outside`, "root/a/../escape", "root/name.", "root//escape"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			archive := filepath.Join(dir, "file.zip")
			if err := os.WriteFile(archive, zipFixture(t, map[string]string{name: "bad"}), 0600); err != nil {
				t.Fatal(err)
			}
			if err := extractZIP(context.Background(), archive, filepath.Join(dir, "out"), "root"); err == nil {
				t.Fatal("entrada insegura aceita")
			}
		})
	}
	var data bytes.Buffer
	w := zip.NewWriter(&data)
	header := &zip.FileHeader{Name: "root/link"}
	header.SetMode(os.ModeSymlink | 0777)
	f, err := w.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(f, "../../outside"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "link.zip")
	if err := os.WriteFile(archive, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if err := extractZIP(context.Background(), archive, t.TempDir(), "root"); err == nil {
		t.Fatal("link aceito")
	}
}

func TestInstallLifecycle(t *testing.T) {
	for _, failure := range []string{"", "checksum", "truncated", "validation", "traversal"} {
		t.Run(failure, func(t *testing.T) {
			var downloads atomic.Int32
			prefix := "node-v22.11.0-win-x64"
			entries := map[string]string{prefix + "/node.exe": "fixture", prefix + "/npm.cmd": "fixture", prefix + "/npx.cmd": "fixture", prefix + "/node_modules/npm/package.json": "preservar"}
			if failure == "traversal" {
				entries[prefix+"/../escape"] = "bad"
			}
			archive := zipFixture(t, entries)
			sum := sha256.Sum256(archive)
			checksum := hex.EncodeToString(sum[:])
			if failure == "checksum" {
				checksum = strings.Repeat("0", 64)
			}
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasSuffix(r.URL.Path, "index.json"):
					io.WriteString(w, `[{"version":"v22.11.0","files":["win-x64-zip"]}]`)
				case strings.HasSuffix(r.URL.Path, "SHASUMS256.txt"):
					fmt.Fprintf(w, "%s  %s.zip", checksum, prefix)
				case strings.HasSuffix(r.URL.Path, ".zip"):
					downloads.Add(1)
					if failure == "truncated" {
						w.Header().Set("Content-Length", fmt.Sprint(len(archive)+100))
					}
					w.Write(archive)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			p := &nodeProvider{baseURL: server.URL, client: server.Client()}
			base := t.TempDir()
			destination := filepath.Join(base, "installs", "node", "22.11.0")
			if err := os.MkdirAll(destination, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(destination, "previous"), []byte("preservar"), 0600); err != nil {
				t.Fatal(err)
			}
			validator := func(ctx context.Context, root string, artifact Artifact) error {
				if failure == "validation" {
					return fmt.Errorf("versão incorreta")
				}
				_, err := os.Stat(filepath.Join(root, "node_modules", "npm", "package.json"))
				return err
			}
			err := installWith(context.Background(), p, windowsAMD64, base, "node", "22.11.0", io.Discard, server.Client(), validator)
			if failure == "" {
				if err != nil {
					t.Fatal(err)
				}
				if _, err := readMetadataAt(destination, "node", "22.11.0", windowsAMD64); err != nil {
					t.Fatal(err)
				}
				if err := installWith(context.Background(), p, windowsAMD64, base, "node", "22.11.0", io.Discard, server.Client(), validator); err != nil {
					t.Fatal(err)
				}
				if downloads.Load() != 1 {
					t.Fatal("instalação repetida baixou novamente")
				}
			} else {
				if err == nil {
					t.Fatal("falha ignorada")
				}
				if _, err := os.Stat(filepath.Join(destination, MetadataFileName)); !os.IsNotExist(err) {
					t.Fatal("falha publicou metadados")
				}
				if _, err := os.Stat(filepath.Join(destination, "previous")); err != nil {
					t.Fatal("destino anterior perdido")
				}
			}
			items, err := os.ReadDir(filepath.Dir(destination))
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 2 || items[0].Name() != ".lock-22.11.0.lck" || items[1].Name() != "22.11.0" {
				t.Fatalf("resíduos: %v", items)
			}
		})
	}
}

func TestConcurrentInstallation(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		<-release
		http.Error(w, "stop", 503)
	}))
	defer server.Close()
	p := &nodeProvider{baseURL: server.URL, client: server.Client()}
	base := t.TempDir()
	finished := make(chan error, 1)
	go func() {
		finished <- installWith(context.Background(), p, windowsAMD64, base, "node", "22.11.0", io.Discard, server.Client(), validateNode)
	}()
	<-started
	err := installWith(context.Background(), p, windowsAMD64, base, "node", "22.11.0", io.Discard, server.Client(), validateNode)
	close(release)
	<-finished
	if err == nil || !strings.Contains(err.Error(), "bloquear") {
		t.Fatalf("concorrência não bloqueada: %v", err)
	}
}

func TestRequestDeadline(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	p := &nodeProvider{baseURL: server.URL, client: server.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := p.List(ctx, windowsAMD64); err == nil {
		t.Fatal("timeout ignorado")
	}
}

// Opt-in: baixa e executa duas distribuições oficiais em um diretório temporário.
func TestOfficialNodeInstallation(t *testing.T) {
	if os.Getenv("VERMAN_INTEGRATION") != "1" {
		t.Skip("defina VERMAN_INTEGRATION=1 para validar downloads oficiais")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("alvo do MVP: windows/amd64")
	}
	isolatedHome(t)
	for _, version := range []string{"22.11.0", "22.12.0"} {
		if err := InstallContext(context.Background(), "node", version, io.Discard); err != nil {
			t.Fatal(err)
		}
		if !IsInstalled("node", version) {
			t.Fatalf("versão %s não instalada", version)
		}
		if err := InstallContext(context.Background(), "node", version, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	versions, err := ListInstalled("node")
	if err != nil || len(versions) != 2 || versions[0] != "22.12.0" {
		t.Fatalf("instalações oficiais: %v %v", versions, err)
	}
}
