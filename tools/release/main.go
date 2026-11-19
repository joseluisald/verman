// Command release gera um pacote apenas para a plataforma validada do produto.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func main() {
	version := flag.String("version", "", "Versão exata de release, por exemplo 0.1.0")
	output := flag.String("out", "dist", "Diretório de saída")
	flag.Parse()
	if err := release(strings.TrimPrefix(*version, "v"), *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func release(version, output string) error {
	if !regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`).MatchString(version) {
		return fmt.Errorf("informe -version X.Y.Z")
	}
	if err := os.MkdirAll(output, 0700); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(output, ".build-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	binary := filepath.Join(stage, "verman.exe")
	command := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-ldflags", "-s -w -buildid= -X github.com/joseluisald/verman/cmd.Version="+version, "-o", binary, ".")
	command.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return err
	}
	name := "verman_" + version + "_windows_amd64.zip"
	archive := filepath.Join(output, name)
	if err := pack(archive, map[string]string{"verman.exe": binary, "README.md": "README.md", "PLANEJAMENTO.md": "PLANEJAMENTO.md", "ROADMAP_PRODUTO.md": "ROADMAP_PRODUTO.md", "docs/CLI_JSON.md": "docs/CLI_JSON.md", "docs/SUPORTE.md": "docs/SUPORTE.md", "docs/PROVEDORES.md": "docs/PROVEDORES.md", "install.ps1": "scripts/install.ps1", "uninstall.ps1": "scripts/uninstall.ps1", "setup-path.ps1": "scripts/setup-path.ps1"}); err != nil {
		return err
	}
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(output, "SHA256SUMS"), []byte(hex.EncodeToString(hash.Sum(nil))+"  "+name+"\n"), 0600); err != nil {
		return err
	}
	fmt.Println(archive)
	return nil
}

func pack(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	// Ordem e timestamps fixos evitam diferenças artificiais entre builds.
	for _, name := range []string{"verman.exe", "README.md", "PLANEJAMENTO.md", "ROADMAP_PRODUTO.md", "docs/CLI_JSON.md", "docs/SUPORTE.md", "docs/PROVEDORES.md", "install.ps1", "uninstall.ps1", "setup-path.ps1"} {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetModTime(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))
		if name == "verman.exe" {
			header.SetMode(0755)
		} else {
			header.SetMode(0644)
		}
		entry, err := w.CreateHeader(header)
		if err != nil {
			w.Close()
			return err
		}
		source, err := os.Open(files[name])
		if err != nil {
			w.Close()
			return err
		}
		_, copyErr := io.Copy(entry, source)
		closeErr := source.Close()
		if copyErr != nil {
			w.Close()
			return copyErr
		}
		if closeErr != nil {
			w.Close()
			return closeErr
		}
	}
	if err := w.Close(); err != nil {
		return err
	}
	return f.Close()
}
