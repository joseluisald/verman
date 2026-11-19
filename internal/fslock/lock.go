// Package fslock mantém locks entre processos, liberados pelo SO ao fechar/morrer.
// O arquivo de lock permanece: removê-lo criaria uma corrida entre identidades.
package fslock

import (
	"context"
	"fmt"
	"github.com/joseluisald/verman/internal/fault"
	"os"
	"time"
)

func Acquire(ctx context.Context, dir, name string, shared, wait bool) (func(), error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	if info, err := root.Lstat(name); err == nil && (info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular()) {
		return nil, fmt.Errorf("lock não é arquivo regular: %s", name)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	flags := os.O_CREATE | os.O_RDWR
	if shared {
		flags = os.O_RDONLY
	}
	f, err := root.OpenFile(name, flags, 0600)
	if shared && os.IsNotExist(err) {
		f, err = root.OpenFile(name, os.O_CREATE|os.O_RDWR, 0600)
	}
	if err != nil {
		return nil, err
	}
	for {
		if err := ctx.Err(); err != nil {
			f.Close()
			return nil, err
		}
		locked, err := tryLock(f, shared)
		if err != nil {
			f.Close()
			return nil, err
		}
		if locked {
			return func() { f.Close() }, nil
		}
		if !wait {
			f.Close()
			return nil, fault.New("operation_busy", 5, "Aguarde a operação ou feche o runtime em uso. Não apague arquivos de lock.", fmt.Errorf("outra operação está usando o lock %s", name))
		}
		select {
		case <-ctx.Done():
			f.Close()
			return nil, ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
}
