package installer

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/fault"
	"github.com/joseluisald/verman/internal/tool"
)

func InstallContext(ctx context.Context, language, version string, out io.Writer) error {
	return InstallWithEvents(ctx, language, version, legacyEvents(out))
}

func InstallWithEvents(ctx context.Context, language, version string, sink EventSink) error {
	return installRuntime(ctx, language, version, sink, false)
}
func RepairWithEvents(ctx context.Context, language, version string, sink EventSink) error {
	return installRuntime(ctx, language, version, sink, true)
}
func installRuntime(ctx context.Context, language, version string, sink EventSink, repair bool) error {
	if err := tool.Validate(language, version); err != nil {
		return err
	}
	p, err := providerFor(language)
	if err != nil {
		return err
	}
	base, err := config.BaseDir()
	if err != nil {
		return err
	}
	platform := Platform{runtime.GOOS, runtime.GOARCH}
	if err := windowsTarget(language, platform); err != nil {
		return err
	}
	destination, err := config.InstallDir(language, version)
	if err != nil {
		return err
	}
	return installOperation(ctx, p, platform, filepath.Join(base, "cache"), destination, language, version, sink, clientFor(language), validateRuntime, repair)
}

type runtimeValidator func(ctx context.Context, staged, destination string, artifact Artifact) error

func installWith(ctx context.Context, p Provider, platform Platform, base, language, version string, out io.Writer, client *http.Client, validate runtimeValidator) error {
	return installWithEvents(ctx, p, platform, base, language, version, legacyEvents(out), client, validate)
}

func installWithEvents(ctx context.Context, p Provider, platform Platform, base, language, version string, sink EventSink, client *http.Client, validate runtimeValidator) error {
	return installOperation(ctx, p, platform, filepath.Join(base, "cache"), filepath.Join(base, "installs", language, version), language, version, sink, client, validate, false)
}
func installOperation(ctx context.Context, p Provider, platform Platform, cacheDir, destination, language, version string, sink EventSink, client *http.Client, validate runtimeValidator, repair bool) (resultErr error) {
	if err := tool.Validate(language, version); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	parent := filepath.Dir(destination)
	parentRoot, err := managedParent(filepath.Dir(parent), language, true)
	if err != nil {
		return err
	}
	defer parentRoot.Close()
	unlock, err := lockVersion(parentRoot, language, version)
	if err != nil {
		return err
	}
	defer unlock()
	if err := recoverTransaction(parentRoot, version); err != nil {
		return err
	}
	if _, err := readMetadataAt(destination, language, version, platform); err == nil && !repair {
		return emit(sink, Event{Kind: EventAlreadyInstalled, Language: language, Version: version, Directory: destination})
	}
	catalog, err := loadCatalog(ctx, p, platform, filepath.Join(cacheDir, language+"-"+platform.OS+"-"+platform.Arch+".json"), false)
	if err != nil {
		return err
	}
	if catalog.Warning != "" {
		if err := emit(sink, Event{Kind: EventWarning, Language: language, Version: version, Message: catalog.Warning}); err != nil {
			return err
		}
	}
	var selected *Artifact
	for _, artifact := range catalog.Catalog.Artifacts {
		if artifact.Version == version {
			copy := artifact
			selected = &copy
			break
		}
	}
	if selected == nil {
		if language == "php" || language == "python" {
			p = withHistory(p)
			historical, err := loadCatalog(ctx, p, platform, filepath.Join(cacheDir, language+"-"+platform.OS+"-"+platform.Arch+"-history.json"), false)
			if err != nil {
				return err
			}
			if historical.Warning != "" {
				if err := emit(sink, Event{Kind: EventWarning, Message: historical.Warning}); err != nil {
					return err
				}
			}
			for _, a := range historical.Catalog.Artifacts {
				if a.Version == version {
					copy := a
					selected = &copy
					break
				}
			}
		}
	}
	if selected == nil {
		if resolver, ok := p.(interface {
			ResolveExact(context.Context, string, Platform) (Artifact, error)
		}); ok {
			a, err := resolver.ResolveExact(ctx, version, platform)
			if err != nil {
				return err
			}
			selected = &a
		}
	}
	if selected == nil {
		return fmt.Errorf("%s %s não disponível para %s/%s; consulte list %s --available --refresh", language, version, platform.OS, platform.Arch, language)
	}
	if policy := Describe(*selected); !policy.Installable {
		return fault.New("artifact_unverified", 4, "Escolha uma versão com checksum oficial em list --available --history.", fmt.Errorf("%s %s: %s", language, version, policy.UnavailableReason))
	}
	if policy := Describe(*selected); policy.Support == "eol" {
		if err := emit(sink, Event{Kind: EventWarning, Language: language, Version: version, Message: "ramo em fim de suporte (EOL); instalação explícita para compatibilidade"}); err != nil {
			return err
		}
	}
	artifact, err := p.Resolve(ctx, *selected)
	if err != nil {
		return err
	}
	if artifact.Language != language || artifact.Version != version || artifact.Platform != platform || (artifact.Format != "zip" && artifact.Format != "tar.gz" && artifact.Format != "file") {
		return fmt.Errorf("artefato incompatível com a instalação solicitada")
	}
	tx, err := beginTransaction(parentRoot, version, "install")
	if err != nil {
		return err
	}
	defer tx.onExit(&resultErr)
	stage := filepath.Join(parent, stageName(version))
	archive := filepath.Join(stage, "download.zip")
	payload := filepath.Join(stage, "runtime")
	if err := os.Mkdir(payload, 0700); err != nil {
		return err
	}
	if err := emit(sink, Event{Kind: EventDownloading, Language: language, Version: version}); err != nil {
		return err
	}
	if err := downloadEvents(ctx, client, artifact, archive, sink); err != nil {
		return err
	}
	if artifact.SHA256 == "" {
		artifact.SHA256, err = hashFile(archive)
		if err != nil {
			return err
		}
	}
	if err := emit(sink, Event{Kind: EventExtracting, Language: language, Version: version}); err != nil {
		return err
	}
	if err := extractArchive(ctx, archive, payload, artifact); err != nil {
		return fmt.Errorf("extrair pacote: %w", err)
	}
	if err := emit(sink, Event{Kind: EventValidating, Language: language, Version: version}); err != nil {
		return err
	}
	if err := validate(ctx, payload, destination, artifact); err != nil {
		return fmt.Errorf("validar runtime: %w", err)
	}
	meta := Metadata{SchemaVersion: 1, Language: language, Version: version, OS: platform.OS, Arch: platform.Arch, Source: artifact.URL, SHA256: artifact.SHA256, Executables: artifact.Executables, Variant: artifact.Variant, Build: artifact.Build}
	meta.TransactionID = tx.record.ID
	meta.SHA512 = artifact.SHA512
	if repair {
		if err := preserveMutable(ctx, destination, payload, language); err != nil {
			return err
		}
	}
	if err := fingerprint(payload, &meta); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(payload, MetadataFileName), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if _, err := readMetadataAt(payload, language, version, platform); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := tx.publish(ctx); err != nil {
		return err
	}
	if err := tx.finish(); err != nil {
		return err
	}
	return emit(sink, Event{Kind: EventInstalled, Language: language, Version: version, Directory: destination})
}

func download(ctx context.Context, client *http.Client, artifact Artifact, path string) error {
	return downloadEvents(ctx, client, artifact, path, nil)
}
func downloadEvents(ctx context.Context, client *http.Client, artifact Artifact, path string, sink EventSink) error {
	if err := requireNetwork(ctx); err != nil {
		return err
	}
	var hash hash.Hash = sha256.New()
	expectedHash := artifact.SHA256
	if expectedHash == "" && validSHA512(artifact.SHA512) {
		hash = sha512.New()
		expectedHash = artifact.SHA512
	}
	expected, err := hex.DecodeString(expectedHash)
	if err != nil || len(expected) != hash.Size() {
		return fmt.Errorf("checksum oficial inválido")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artifact.URL, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(req)
	if err != nil {
		return fault.New("network_error", 3, "Verifique a conexão e tente novamente.", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return httpFailure(artifact.URL, response)
	}
	const max = int64(512 << 20)
	if response.ContentLength > max {
		return fmt.Errorf("download excede 512 MiB")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	progress := &downloadProgress{sink: sink, language: artifact.Language, version: artifact.Version, total: response.ContentLength}
	written, err := io.Copy(io.MultiWriter(f, hash, progress), io.LimitReader(response.Body, max+1))
	if err != nil {
		return fmt.Errorf("download interrompido: %w", err)
	}
	if written > max {
		return fmt.Errorf("download excede 512 MiB")
	}
	if err := progress.send(); err != nil {
		return err
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), expectedHash) {
		return fault.New("integrity_error", 4, "O pacote não foi instalado. Repita com catálogo atualizado; reporte se persistir.", fmt.Errorf("checksum oficial não confere"))
	}
	return f.Close()
}

type downloadProgress struct {
	sink              EventSink
	language, version string
	bytes, total      int64
	last              time.Time
}

func (p *downloadProgress) Write(data []byte) (int, error) {
	p.bytes += int64(len(data))
	if time.Since(p.last) >= 250*time.Millisecond {
		if err := p.send(); err != nil {
			return 0, err
		}
	}
	return len(data), nil
}
func (p *downloadProgress) send() error {
	p.last = time.Now()
	return emit(p.sink, Event{Kind: EventProgress, Language: p.language, Version: p.version, Bytes: p.bytes, Total: p.total})
}

type limitedOutput struct{ data []byte }

func (b *limitedOutput) Write(p []byte) (int, error) {
	if len(b.data)+len(p) > 64<<10 {
		return 0, fmt.Errorf("saída do runtime excede 64 KiB")
	}
	b.data = append(b.data, p...)
	return len(p), nil
}

func validateNode(ctx context.Context, root, _ string, artifact Artifact) error {
	for _, relative := range artifact.Executables {
		if _, err := executablePath(root, relative); err != nil {
			return err
		}
	}
	node := filepath.Join(root, "node.exe")
	checks := [][]string{{"--version"}, {"node_modules/npm/bin/npm-cli.js", "--version"}, {"node_modules/npm/bin/npx-cli.js", "--version"}}
	for i, args := range checks {
		if i > 0 {
			if _, err := executablePath(root, args[0]); err != nil {
				return err
			}
			args[0] = filepath.Join(root, filepath.FromSlash(args[0]))
		}
		checkCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		cmd := exec.CommandContext(checkCtx, node, args...)
		cmd.Dir = root
		cmd.WaitDelay = time.Second
		for _, env := range os.Environ() {
			key := strings.ToUpper(strings.SplitN(env, "=", 2)[0])
			if key != "NODE_OPTIONS" && key != "NODE_PATH" && !strings.HasPrefix(key, "NPM_CONFIG_") {
				cmd.Env = append(cmd.Env, env)
			}
		}
		var stdout, stderr limitedOutput
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		cancel()
		if err != nil {
			return fmt.Errorf("executar %v: %w (%s)", args, err, stderr.data)
		}
		result := strings.TrimSpace(string(stdout.data))
		if i == 0 && result != "v"+artifact.Version {
			return fmt.Errorf("versão executada %q difere de %s", result, artifact.Version)
		}
		if result == "" {
			return fmt.Errorf("runtime retornou versão vazia")
		}
	}
	return nil
}
