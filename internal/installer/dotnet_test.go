package installer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDotnetCatalogRevalidatesOfficialSDK(t *testing.T) {
	var provider dotnetProvider
	hash := strings.Repeat("a", 128)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encode := json.NewEncoder(w)
		switch r.URL.Path {
		case "/release-metadata/releases-index.json":
			_ = encode.Encode(map[string]any{"releases-index": []map[string]string{
				{"channel-version": "10.0", "releases.json": provider.channelURL("10.0")},
				{"channel-version": "9.0", "releases.json": "https://example.invalid/releases.json"},
			}})
		case "/release-metadata/10.0/releases.json":
			makeSDK := func(version string) dotnetSDK {
				sdk := dotnetSDK{Version: version}
				sdk.Files = append(sdk.Files, struct{ RID, URL, Hash string }{
					RID: "win-x64", URL: provider.artifact(version, hash, Platform{}).URL, Hash: hash,
				})
				return sdk
			}
			forged := makeSDK("10.0.999")
			forged.Files[0].URL = "https://example.invalid/package.zip"
			_ = encode.Encode(map[string]any{
				"channel-version": "10.0",
				"releases": []map[string]any{
					{"sdk": makeSDK("10.0.102"), "sdks": []dotnetSDK{makeSDK("10.0.102"), makeSDK("10.0.201")}},
					{"sdk": makeSDK("10.0.300-preview.1"), "sdks": []dotnetSDK{forged}},
				},
			})
		default:
			t.Errorf("rota inesperada: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	provider = dotnetProvider{baseURL: server.URL, client: server.Client()}
	ctx := context.Background()
	items, err := provider.List(ctx, Platform{"windows", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Version != "10.0.201" || items[1].Version != "10.0.102" {
		t.Fatalf("catálogo incorreto: %+v", items)
	}
	a := items[0]
	if info := Describe(a); !info.Installable || info.Trust != "publisher_sha512_https" {
		t.Fatalf("política incorreta: %+v", info)
	}
	a.SHA512 = strings.Repeat("b", 128)
	resolved, err := provider.Resolve(ctx, a)
	if err != nil || resolved.SHA512 != hash {
		t.Fatalf("SHA-512 oficial não revalidado: %+v (%v)", resolved, err)
	}
	a.URL = "https://example.invalid/package.zip"
	if err := provider.ValidateArtifact(a); err == nil {
		t.Fatal("URL externa aceita")
	}
}
