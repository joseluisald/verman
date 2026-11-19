package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/joseluisald/verman/internal/fault"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPHPHistoryRefusesUnverifiedArchives(t *testing.T) {
	hash := strings.Repeat("a", 64)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases.json":
			fmt.Fprintf(w, `{"8.5":{"version":"8.5.12","nts-vs17-x64":{"zip":{"path":"php-8.5.12-nts-Win32-vs17-x64.zip","sha256":"%s"}}}}`, hash)
		case "/archives/":
			io.WriteString(w, `<a href="php-7.4.33-nts-Win32-vc15-x64.zip">old</a><a href="php-8.3.1-nts-Win32-vs16-x64.zip">unverified</a><a href="../php-8.3.2-nts-Win32-vs16-x64.zip">escape</a><a href="php-8.3.1-Win32-vs16-x64.zip">TS</a>`)
		case "/sha256sum.txt":
			fmt.Fprintf(w, "%s *php-7.4.33-nts-Win32-vc15-x64.zip\n", hash)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	p := &phpProvider{baseURL: server.URL, client: server.Client(), history: true}
	items, err := p.List(context.Background(), windowsAMD64)
	if err != nil || len(items) != 3 {
		t.Fatalf("histórico: %#v %v", items, err)
	}
	for _, a := range items {
		policy := Describe(a)
		if a.Version == "8.3.1" {
			if policy.Installable || policy.Trust != "unverified" {
				t.Fatal("arquivo sem checksum anunciado como instalável")
			}
			if _, err := p.Resolve(context.Background(), a); err == nil {
				t.Fatal("arquivo sem checksum resolvido")
			}
		}
		if a.Version == "7.4.33" {
			a.SHA256 = strings.Repeat("b", 64)
			resolved, err := p.Resolve(context.Background(), a)
			if err != nil || resolved.SHA256 != hash || !strings.Contains(resolved.URL, "/archives/") {
				t.Fatalf("revalidação: %#v %v", resolved, err)
			}
		}
	}
}

func TestPythonHistoryMergeAndPinnedBuild(t *testing.T) {
	var p *pythonProvider
	hash := strings.Repeat("a", 64)
	release := func(version, build string) map[string]any {
		a := p.artifact(version, build, hash, windowsAMD64)
		return map[string]any{"tag_name": build, "assets": []map[string]string{{"name": "cpython-" + version + "+" + build + "-x86_64-pc-windows-msvc-install_only.tar.gz", "browser_download_url": a.URL, "digest": "sha256:" + hash}}}
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			json.NewEncoder(w).Encode(release("3.14.10", "20261110"))
		case "/releases":
			json.NewEncoder(w).Encode([]any{release("3.14.9", "20261010"), release("3.14.10", "20261010"), release("3.13.11", "20260910"), map[string]any{"draft": true}})
		case "/releases/tags/20261010":
			json.NewEncoder(w).Encode(release("3.14.9", "20261010"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	p = &pythonProvider{apiURL: server.URL + "/releases", downloadURL: server.URL + "/download", client: server.Client(), history: true}
	items, err := p.List(context.Background(), windowsAMD64)
	if err != nil || len(items) != 3 {
		t.Fatalf("histórico: %#v %v", items, err)
	}
	if items[0].Build != "20261110" {
		t.Fatal("build antigo sobrepôs atual")
	}
	for _, a := range items {
		if a.Version == "3.14.9" {
			resolved, err := p.Resolve(context.Background(), a)
			if err != nil || resolved.Build != "20261010" {
				t.Fatalf("build não fixado: %#v %v", resolved, err)
			}
		}
	}
}

func TestRateLimitAndOfflineCache(t *testing.T) {
	var calls atomic.Int32
	var limited atomic.Bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if limited.Load() {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", "9999999999")
			w.WriteHeader(403)
			return
		}
		io.WriteString(w, `[{"version":"v22.11.0","files":["win-x64-zip"]}]`)
	}))
	defer server.Close()
	p := &nodeProvider{baseURL: server.URL, client: server.Client()}
	path := filepath.Join(t.TempDir(), "catalog.json")
	result, err := loadCatalog(context.Background(), p, windowsAMD64, path, false)
	if err != nil {
		t.Fatal(err)
	}
	result.Catalog.FetchedAt = time.Now().Add(-48 * time.Hour)
	data, _ := json.Marshal(result.Catalog)
	if err := writeCache(path, data); err != nil {
		t.Fatal(err)
	}
	limited.Store(true)
	_, err = get(context.Background(), server.Client(), server.URL, 1024)
	if failure := fault.Classify(err); failure.Code != "rate_limited" || !strings.Contains(failure.Hint, "9999999999") {
		t.Fatalf("limite: %#v", failure)
	}
	result, err = loadCatalog(context.Background(), p, windowsAMD64, path, true)
	if err != nil || !result.Stale || !result.FromCache || result.Warning == "" {
		t.Fatalf("fallback: %#v %v", result, err)
	}
	before := calls.Load()
	result, err = loadCatalog(WithOffline(context.Background()), p, windowsAMD64, path, false)
	if err != nil || !result.Stale || calls.Load() != before {
		t.Fatalf("offline acessou rede: %#v %v", result, err)
	}
	if _, err := loadCatalog(WithOffline(context.Background()), p, windowsAMD64, path, true); err == nil {
		t.Fatal("refresh offline aceito")
	}
}
