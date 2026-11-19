package installer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Apenas arquivos regulares e diretórios são aceitos. ZIPs oficiais de Windows
// não precisam de links; rejeitá-los evita escapes e ambiguidades na extração.
func extractZIP(ctx context.Context, archive, destination, prefix string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer r.Close()
	if len(r.File) > 100000 {
		return fmt.Errorf("arquivo ZIP com entradas demais")
	}
	var total uint64
	for _, file := range r.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := strings.TrimSuffix(file.Name, "/")
		if name == prefix && file.FileInfo().IsDir() {
			continue
		}
		if prefix != "" && !strings.HasPrefix(name, prefix+"/") {
			return fmt.Errorf("entrada ZIP fora da raiz esperada: %q", name)
		}
		relative := name
		if prefix != "" {
			relative = strings.TrimPrefix(name, prefix+"/")
		}
		if strings.ContainsAny(relative, `\:`) || !filepath.IsLocal(relative) {
			return fmt.Errorf("caminho ZIP inválido: %q", name)
		}
		for _, component := range strings.Split(relative, "/") {
			if component == ".." || component == "." || component == "" || strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") {
				return fmt.Errorf("componente ZIP inválido: %q", name)
			}
		}
		mode := file.Mode()
		if !mode.IsRegular() && !mode.IsDir() {
			return fmt.Errorf("link ou tipo ZIP não suportado: %q", name)
		}
		if file.UncompressedSize64 > 2<<30 || total > (2<<30)-file.UncompressedSize64 {
			return fmt.Errorf("ZIP expandido excede 2 GiB")
		}
		total += file.UncompressedSize64
		path := filepath.Join(destination, filepath.FromSlash(relative))
		if mode.IsDir() {
			if err := os.MkdirAll(path, 0700); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		if err := extractFile(ctx, file, path); err != nil {
			return err
		}
	}
	return nil
}

func extractArchive(ctx context.Context, archive, destination string, a Artifact) error {
	switch a.Format {
	case "file":
		if !safeRelative(a.ArchiveRoot) || strings.Contains(a.ArchiveRoot, "/") || a.ArchiveRoot == "." {
			return fmt.Errorf("nome de arquivo inválido")
		}
		input, err := os.Open(archive)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.OpenFile(filepath.Join(destination, a.ArchiveRoot), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
		if err != nil {
			return err
		}
		_, err = io.Copy(output, &contextReader{ctx: ctx, reader: input})
		closeErr := output.Close()
		if err != nil {
			return err
		}
		return closeErr
	case "zip":
		return extractZIP(ctx, archive, destination, a.ArchiveRoot)
	case "tar.gz":
		return extractTarGzip(ctx, archive, destination, a.ArchiveRoot)
	default:
		return fmt.Errorf("formato não suportado: %s", a.Format)
	}
}

func extractTarGzip(ctx context.Context, archive, destination, prefix string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(&contextReader{ctx: ctx, reader: f})
	if err != nil {
		return err
	}
	defer gz.Close()
	r := tar.NewReader(gz)
	var total int64
	for count := 0; ; count++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if count >= 100000 {
			return fmt.Errorf("TAR com entradas demais")
		}
		name := strings.TrimSuffix(header.Name, "/")
		if name == prefix && header.Typeflag == tar.TypeDir {
			continue
		}
		if prefix != "" && !strings.HasPrefix(name, prefix+"/") {
			return fmt.Errorf("entrada TAR fora da raiz: %q", name)
		}
		relative := name
		if prefix != "" {
			relative = strings.TrimPrefix(name, prefix+"/")
		}
		if strings.ContainsAny(relative, `\:`) || !filepath.IsLocal(relative) {
			return fmt.Errorf("caminho TAR inválido: %q", name)
		}
		for _, component := range strings.Split(relative, "/") {
			if component == ".." || component == "." || component == "" || strings.HasSuffix(component, ".") || strings.HasSuffix(component, " ") {
				return fmt.Errorf("componente TAR inválido: %q", name)
			}
		}
		if header.Size < 0 || header.Size > 2<<30 || total > (2<<30)-header.Size {
			return fmt.Errorf("TAR expandido excede 2 GiB")
		}
		total += header.Size
		path := filepath.Join(destination, filepath.FromSlash(relative))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0700); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				return err
			}
			dst, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
			if err != nil {
				return err
			}
			written, copyErr := io.Copy(dst, &contextReader{ctx: ctx, reader: r})
			closeErr := dst.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			if written != header.Size {
				return fmt.Errorf("TAR truncado: %s", name)
			}
		default:
			return fmt.Errorf("link ou tipo TAR não suportado: %q", name)
		}
	}
	// Drene o restante limitado para verificar também o checksum do gzip.
	n, err := io.Copy(io.Discard, io.LimitReader(gz, 1<<20))
	if err != nil {
		return err
	}
	if n >= 1<<20 {
		return fmt.Errorf("dados adicionais excessivos no gzip")
	}
	return nil
}

func extractFile(ctx context.Context, file *zip.File, path string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	defer dst.Close()
	// Leitura até EOF verifica também o CRC do ZIP; o tamanho é limitado.
	written, err := io.Copy(dst, io.LimitReader(&contextReader{ctx: ctx, reader: src}, int64(file.UncompressedSize64)+1))
	if err != nil {
		return err
	}
	if uint64(written) != file.UncompressedSize64 {
		return fmt.Errorf("tamanho ZIP inválido: %s", file.Name)
	}
	return dst.Close()
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
