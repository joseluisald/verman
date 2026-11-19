package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDownloadProgressAndCancellation(t *testing.T) {
	body := strings.Repeat("x", 1<<20)
	digest := sha256.Sum256([]byte(body))
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for offset := 0; offset < len(body); offset += 4096 {
			select {
			case <-r.Context().Done():
				return
			default:
			}
			if _, err := w.Write([]byte(body[offset : offset+4096])); err != nil {
				return
			}
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()
	a := Artifact{Language: "node", Version: "22.11.0", URL: server.URL, SHA256: hex.EncodeToString(digest[:])}
	for _, cancelEarly := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		var progress []int64
		err := downloadEvents(ctx, server.Client(), a, filepath.Join(t.TempDir(), "archive.zip"), func(event Event) error {
			if event.Kind == EventProgress {
				progress = append(progress, event.Bytes)
				if cancelEarly {
					cancel()
				}
			}
			return nil
		})
		cancel()
		if len(progress) == 0 {
			t.Fatal("progresso não emitido")
		}
		if cancelEarly {
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelamento perdido: %v", err)
			}
		} else {
			if err != nil || progress[len(progress)-1] != int64(len(body)) {
				t.Fatalf("download completo: %v %v", progress, err)
			}
			for i := 1; i < len(progress); i++ {
				if progress[i] < progress[i-1] {
					t.Fatal("progresso regressivo")
				}
			}
		}
	}
}

func TestDistributionPolicy(t *testing.T) {
	now := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct{ lang, version, expected string }{{"node", "22.11.0", "supported_branch"}, {"node", "20.0.0", "eol"}, {"php", "8.5.12", "supported_branch"}, {"php", "7.4.33", "eol"}, {"python", "3.10.19", "eol"}, {"python", "3.14.10", "supported_branch"}, {"python", "3.99.0", "unknown"}} {
		policy := describeAt(Artifact{Language: test.lang, Version: test.version, SHA256: strings.Repeat("a", 64)}, now)
		if policy.Support != test.expected || policy.PolicySource == "" || policy.PolicyReviewed == "" {
			t.Fatalf("%#v: %#v", test, policy)
		}
	}
}
