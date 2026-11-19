package durable

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Write sincroniza o temporário antes da substituição no mesmo diretório.
func Write(path string, data []byte, mode os.FileMode) error {
	return WriteContext(context.Background(), path, data, mode)
}
func WriteContext(ctx context.Context, path string, data []byte, mode os.FileMode) error {
	return writeContext(ctx, path, data, mode, func(f *os.File, data []byte) (int, error) { return f.Write(data) })
}

func writeContext(ctx context.Context, path string, data []byte, mode os.FileMode, write func(*os.File, []byte) (int, error)) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// O chamador deve deter o lock deste destino. Só então os temporários
	// deste arquivo podem ser abandonos de um processo anterior.
	prefix := ".verman-write-" + filepath.Base(path) + "-"
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer root.Close()
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) && strings.HasSuffix(entry.Name(), ".tmp") && entry.Type().IsRegular() {
			if err := root.Remove(entry.Name()); err != nil {
				return err
			}
		}
	}
	f, err := os.CreateTemp(filepath.Dir(path), prefix+"*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := f.Chmod(mode); err != nil {
		return err
	}
	if _, err := write(f, data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := replace(f.Name(), path); err != nil {
		return fmt.Errorf("substituir %s: %w", path, err)
	}
	return nil
}
