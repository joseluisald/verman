//go:build windows

package installer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// Mantém handles com acesso DELETE/WRITE durante o preflight.
// Arquivos em execução/abertos falham antes de mover ou excluir qualquer arquivo.
func guardRemoval(root string) (func(), error) {
	handles := []syscall.Handle{}
	closeAll := func() {
		for _, h := range handles {
			syscall.CloseHandle(h)
		}
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		wide, err := syscall.UTF16PtrFromString(path)
		if err != nil {
			return err
		}
		attributes, err := syscall.GetFileAttributes(wide)
		if err != nil {
			return err
		}
		if attributes&syscall.FILE_ATTRIBUTE_READONLY != 0 {
			return fmt.Errorf("arquivo somente leitura: %s", path)
		}
		if attributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
			return nil
		}
		// GENERIC_WRITE também detecta imagens executáveis mapeadas: acesso
		// DELETE sozinho pode abrir um EXE em execução, mas DeleteFile falharia.
		h, err := syscall.CreateFile(wide, 0x10000|syscall.GENERIC_WRITE, syscall.FILE_SHARE_DELETE|syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
		if err != nil {
			return fmt.Errorf("arquivo ocupado/sem permissão; instalação preservada: %s: %w", path, err)
		}
		handles = append(handles, h)
		return nil
	})
	if err != nil {
		closeAll()
		return nil, err
	}
	return closeAll, nil
}
