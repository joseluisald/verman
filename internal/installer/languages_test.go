package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPHPCatalogVariantsAndRevisions(t *testing.T) {
	var response string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, response) }))
	defer server.Close()
	p := &phpProvider{baseURL: server.URL, client: server.Client()}
	checksum := strings.Repeat("a", 64)
	response = fmt.Sprintf(`{"8.4":{"version":"8.4.27-1","nts-vs17-x64":{"zip":{"path":"php-8.4.27-1-nts-Win32-vs17-x64.zip","sha256":"%s"}},"ts-vs17-x64":{"zip":{"path":"ignore.zip"}}},"8.3":{"version":"8.3.36","nts-vs16-x64":{"zip":{"path":"php-8.3.36-nts-Win32-vs16-x64.zip","sha256":"%s"}}},"8.6":{"version":"8.6.0RC1"}}`, checksum, checksum)
	items, err := p.List(context.Background(), windowsAMD64)
	if err != nil || len(items) != 2 || items[0].Version != "8.4.27" || items[0].Build != "8.4.27-1" || items[0].Variant != "nts-vs17-x64" {
		t.Fatalf("catálogo PHP: %#v %v", items, err)
	}
	tampered := items[0]
	tampered.SHA256 = strings.Repeat("b", 64)
	resolved, err := p.Resolve(context.Background(), tampered)
	if err != nil || resolved.SHA256 != checksum {
		t.Fatalf("checksum local não substituído pela fonte confiável: %#v %v", resolved, err)
	}
	tampered.URL = "https://evil.test/php.zip"
	if _, err := p.Resolve(context.Background(), tampered); err == nil {
		t.Fatal("URL arbitrária aceita")
	}
	for _, body := range []string{`{}`, `[]`, `{"8.4":{}}`, strings.ReplaceAll(response, checksum, "bad"), strings.ReplaceAll(response, "php-8.4.27-1-nts-Win32-vs17-x64.zip", "../evil.zip")} {
		response = body
		if _, err := p.List(context.Background(), windowsAMD64); err == nil {
			t.Fatalf("catálogo inválido aceito: %s", body)
		}
	}
	if _, err := p.List(context.Background(), Platform{"linux", "amd64"}); err == nil {
		t.Fatal("plataforma inválida aceita")
	}
}

func TestPythonCatalogPinningAndIntegrity(t *testing.T) {
	var body []byte
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(body) }))
	defer server.Close()
	p := &pythonProvider{apiURL: server.URL + "/releases", downloadURL: server.URL + "/download", client: server.Client()}
	checksum := strings.Repeat("a", 64)
	asset := p.artifact("3.14.10", "20261110", checksum, windowsAMD64)
	release := map[string]any{"tag_name": "20261110", "assets": []map[string]string{
		{"name": "cpython-3.14.10+20261110-x86_64-pc-windows-msvc-install_only.tar.gz", "browser_download_url": asset.URL, "digest": "sha256:" + checksum},
		{"name": "cpython-3.16.0a2+20261110-x86_64-pc-windows-msvc-install_only.tar.gz"},
		{"name": "cpython-3.14.10+20261110-aarch64-pc-windows-msvc-install_only.tar.gz"},
	}}
	var err error
	body, err = json.Marshal(release)
	if err != nil {
		t.Fatal(err)
	}
	items, err := p.List(context.Background(), windowsAMD64)
	if err != nil || len(items) != 1 || items[0].Version != "3.14.10" {
		t.Fatalf("catálogo Python: %#v %v", items, err)
	}
	tampered := items[0]
	tampered.SHA256 = strings.Repeat("b", 64)
	resolved, err := p.Resolve(context.Background(), tampered)
	if err != nil || resolved.SHA256 != checksum {
		t.Fatalf("checksum local aceito: %#v %v", resolved, err)
	}
	for _, mutate := range []func(*Artifact){func(a *Artifact) { a.URL = "https://evil.test/python.tar.gz" }, func(a *Artifact) { a.Build = "../../escape" }, func(a *Artifact) { a.Platform = Platform{"linux", "amd64"} }} {
		a := asset
		mutate(&a)
		if _, err := p.Resolve(context.Background(), a); err == nil {
			t.Fatal("artefato inválido aceito")
		}
	}
	for _, bad := range []string{`{}`, `null`, `{"tag_name":"20261110","assets":null}`, strings.ReplaceAll(string(body), "sha256:"+checksum, ""), strings.ReplaceAll(string(body), asset.URL, "https://evil.test/python.tar.gz")} {
		body = []byte(bad)
		if _, err := p.List(context.Background(), windowsAMD64); err == nil {
			t.Fatalf("release inválido aceito: %s", bad)
		}
	}
}

func tarFixture(t *testing.T, name string, kind byte) []byte {
	t.Helper()
	var result bytes.Buffer
	gz := gzip.NewWriter(&result)
	writer := tar.NewWriter(gz)
	header := &tar.Header{Name: name, Mode: 0700, Typeflag: kind}
	if kind == tar.TypeReg {
		header.Size = 7
	}
	if kind == tar.TypeSymlink || kind == tar.TypeLink {
		header.Linkname = "../../outside"
	}
	if err := writer.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if kind == tar.TypeReg {
		if _, err := io.WriteString(writer, "fixture"); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return result.Bytes()
}

func TestTarGzipExtraction(t *testing.T) {
	for _, test := range []struct {
		name  string
		kind  byte
		valid bool
	}{
		{"python/python.exe", tar.TypeReg, true},
		{"python/../outside", tar.TypeReg, false},
		{"python/C:/escape", tar.TypeReg, false},
		{`python/..\outside`, tar.TypeReg, false},
		{"outside", tar.TypeReg, false},
		{"python/link", tar.TypeSymlink, false},
		{"python/link", tar.TypeLink, false},
		{"python/device", tar.TypeChar, false},
	} {
		t.Run(test.name+fmt.Sprint(test.kind), func(t *testing.T) {
			archive := filepath.Join(t.TempDir(), "package.tar.gz")
			if err := os.WriteFile(archive, tarFixture(t, test.name, test.kind), 0600); err != nil {
				t.Fatal(err)
			}
			destination := t.TempDir()
			err := extractTarGzip(context.Background(), archive, destination, "python")
			if (err == nil) != test.valid {
				t.Fatalf("extração: %v", err)
			}
			if test.valid {
				data, err := os.ReadFile(filepath.Join(destination, "python.exe"))
				if err != nil || string(data) != "fixture" {
					t.Fatal("arquivo extraído incorreto")
				}
			}
		})
	}
	corrupt := tarFixture(t, "python/python.exe", tar.TypeReg)
	corrupt[len(corrupt)-8] ^= 0xff
	archive := filepath.Join(t.TempDir(), "corrupt.tar.gz")
	if err := os.WriteFile(archive, corrupt, 0600); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGzip(context.Background(), archive, t.TempDir(), "python"); err == nil {
		t.Fatal("checksum gzip corrompido aceito")
	}
	flat := filepath.Join(t.TempDir(), "php.zip")
	if err := os.WriteFile(flat, zipFixture(t, map[string]string{"php.exe": "fixture", "ext/php_curl.dll": "fixture"}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := extractZIP(context.Background(), flat, t.TempDir(), ""); err != nil {
		t.Fatalf("ZIP sem pasta raiz: %v", err)
	}
}
