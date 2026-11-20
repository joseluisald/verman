package installer

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/joseluisald/verman/internal/fault"
	"github.com/joseluisald/verman/internal/tool"
)

type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

type Artifact struct {
	Language    string            `json:"language"`
	Version     string            `json:"version"`
	Platform    Platform          `json:"platform"`
	URL         string            `json:"url"`
	Format      string            `json:"format"`
	SHA256      string            `json:"sha256,omitempty"`
	ChecksumURL string            `json:"checksum_url"`
	ArchiveRoot string            `json:"archive_root"`
	Executables map[string]string `json:"executables"`
	Variant     string            `json:"variant,omitempty"`
	Build       string            `json:"build,omitempty"`
	LTS         bool              `json:"lts,omitempty"`
	Archived    bool              `json:"archived,omitempty"`
	SHA512      string            `json:"sha512,omitempty"`
}

// Provider separa o catálogo e o layout da linguagem do ciclo de instalação.
type Provider interface {
	Source() string
	List(context.Context, Platform) ([]Artifact, error)
	Resolve(context.Context, Artifact) (Artifact, error)
	ValidateArtifact(Artifact) error
}

func providerFor(language string) (Provider, error) {
	if err := tool.ValidateLanguage(language); err != nil {
		return nil, err
	}
	if spec, ok := githubToolSpecs[language]; ok {
		return &githubToolProvider{spec: spec, apiURL: "https://api.github.com/repos/" + spec.repo + "/releases", downloadURL: "https://github.com/" + spec.repo + "/releases/download", client: clientFor(language)}, nil
	}
	switch language {
	case "node":
		return &nodeProvider{baseURL: "https://nodejs.org/dist", client: officialClient()}, nil
	case "php":
		return &phpProvider{baseURL: "https://downloads.php.net/~windows/releases", client: clientFor("php")}, nil
	case "python":
		return &pythonProvider{apiURL: "https://api.github.com/repos/astral-sh/python-build-standalone/releases", downloadURL: "https://github.com/astral-sh/python-build-standalone/releases/download", client: clientFor("python")}, nil
	case "go":
		return &goProvider{indexURL: "https://go.dev/dl/?mode=json&include=all", baseURL: "https://go.dev/dl", client: clientFor(language)}, nil
	case "dotnet":
		return &dotnetProvider{baseURL: "https://builds.dotnet.microsoft.com/dotnet", client: clientFor(language)}, nil
	case "composer":
		return &composerProvider{baseURL: "https://getcomposer.org", client: clientFor(language)}, nil
	case "yarn":
		return &yarnProvider{baseURL: "https://registry.npmjs.org", client: clientFor(language)}, nil
	case "rust":
		return &rustProvider{baseURL: "https://static.rust-lang.org", apiURL: "https://api.github.com/repos/rust-lang/rust/releases?per_page=30", client: clientFor(language)}, nil
	}
	return nil, fmt.Errorf("provedor desconhecido")
}

func officialClient() *http.Client {
	return clientFor("node")
}

func clientFor(language string) *http.Client {
	hosts := map[string]bool{}
	if _, ok := githubToolSpecs[language]; ok {
		for _, host := range []string{"api.github.com", "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com"} {
			hosts[host] = true
		}
	}
	switch language {
	case "node":
		hosts["nodejs.org"] = true
	case "php":
		hosts["windows.php.net"] = true
		hosts["downloads.php.net"] = true
	case "python":
		for _, host := range []string{"api.github.com", "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com"} {
			hosts[host] = true
		}
	case "go":
		hosts["go.dev"] = true
		hosts["dl.google.com"] = true
	case "dotnet":
		hosts["builds.dotnet.microsoft.com"] = true
	case "composer":
		hosts["getcomposer.org"] = true
	case "yarn":
		hosts["registry.npmjs.org"] = true
	case "rust":
		hosts["static.rust-lang.org"] = true
		hosts["api.github.com"] = true
	}
	return &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("excesso de redirecionamentos")
		}
		if req.URL.Scheme != "https" || !hosts[req.URL.Host] {
			return fmt.Errorf("redirecionamento fora da origem oficial")
		}
		return nil
	}}
}

func windowsTarget(language string, platform Platform) error {
	if platform.OS != "windows" || platform.Arch != "amd64" {
		return fault.New("unsupported_platform", 2, "Consulte docs/PROVEDORES.md para a matriz verificada.", fmt.Errorf("%s: plataforma %s/%s não suportada; suporte atual: windows/amd64", language, platform.OS, platform.Arch))
	}
	return nil
}

func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func get(ctx context.Context, client *http.Client, url string, max int64) ([]byte, error) {
	if err := requireNetwork(ctx); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "verman")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fault.New("network_error", 3, "Verifique a conexão ou consulte um catálogo local com --offline.", fmt.Errorf("consultar %s: %w", url, err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, httpFailure(url, resp)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("resposta excede o limite de tamanho")
	}
	return data, nil
}

func httpFailure(url string, resp *http.Response) error {
	hint := "Tente novamente mais tarde ou use --offline com um cache existente."
	code := "http_error"
	if resp.StatusCode == 429 || (resp.StatusCode == 403 && resp.Header.Get("X-RateLimit-Remaining") == "0") {
		code = "rate_limited"
		hint = fmt.Sprintf("Limite da API; Retry-After=%s; X-RateLimit-Reset=%s. Aguarde ou use --offline.", resp.Header.Get("Retry-After"), resp.Header.Get("X-RateLimit-Reset"))
	}
	return fault.New(code, 3, hint, fmt.Errorf("consultar %s: HTTP %d", url, resp.StatusCode))
}
