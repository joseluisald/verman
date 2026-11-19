package installer

import (
	"context"
	"fmt"
	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/fslock"
	"os"
	"path/filepath"
)

// O mesmo bloqueio por versão coordena instalação e remoção entre processos.
func lockVersion(parent *os.Root, language, version string) (func(), error) {
	name := ".lock-" + version
	if _, err := parent.Lstat(name); err == nil {
		return nil, fmt.Errorf("não foi possível bloquear: bloqueio legado %s presente; não foi removido automaticamente", name)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	unlock, err := fslock.Acquire(context.Background(), parent.Name(), name+".lck", false, false)
	if err != nil {
		return nil, fmt.Errorf("não foi possível bloquear %s %s: %w", language, version, err)
	}
	return unlock, nil
}

// Lease protege a execução de runtimes gerenciados contra uninstall/repair.
func Lease(ctx context.Context, language, version string) (func(), error) {
	root, err := config.InstallDir(language, version)
	if err != nil {
		return nil, err
	}
	if _, err := os.Lstat(filepath.Join(filepath.Dir(root), ".lock-"+version)); err == nil {
		return nil, fmt.Errorf("bloqueio legado presente; execução recusada")
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return fslock.Acquire(ctx, filepath.Dir(root), ".lock-"+version+".lck", true, false)
}
