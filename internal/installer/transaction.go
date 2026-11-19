package installer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/durable"
	"github.com/joseluisald/verman/internal/tool"
)

type journal struct {
	Schema    int    `json:"schema"`
	ID        string `json:"id"`
	Version   string `json:"version"`
	Operation string `json:"operation"`
	Phase     string `json:"phase"`
}
type transaction struct {
	root   *os.Root
	record journal
	active bool
	hook   func(string)
}

func journalName(version string) string { return config.InstallJournalName(version) }
func stageName(version string) string   { return config.InstallStageName(version) }

func managedParent(installRoot, language string, create bool) (*os.Root, error) {
	base := filepath.Dir(installRoot)
	if create {
		if err := os.MkdirAll(base, 0700); err != nil {
			return nil, err
		}
	}
	if info, err := os.Lstat(base); err != nil {
		return nil, err
	} else if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("base gerenciada é link ou não é diretório")
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return nil, err
	}
	for _, name := range []string{filepath.Base(installRoot), language} {
		if create {
			if err := root.Mkdir(name, 0700); err != nil && !os.IsExist(err) {
				root.Close()
				return nil, err
			}
		}
		info, err := root.Lstat(name)
		if err != nil {
			root.Close()
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			root.Close()
			return nil, fmt.Errorf("diretório gerenciado é link: %s", name)
		}
		next, err := root.OpenRoot(name)
		root.Close()
		if err != nil {
			return nil, err
		}
		root = next
	}
	return root, nil
}

func beginTransaction(root *os.Root, version, operation string) (*transaction, error) {
	if err := recoverTransaction(root, version); err != nil {
		return nil, err
	}
	if _, err := root.Lstat(stageName(version)); err == nil {
		return nil, fmt.Errorf("área temporária sem journal; preservada para inspeção: %s", stageName(version))
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	tx := &transaction{root: root, record: journal{Schema: 1, ID: hex.EncodeToString(nonce[:]), Version: version, Operation: operation, Phase: "building"}, active: true}
	if err := tx.phase("building"); err != nil {
		return nil, err
	}
	if err := root.Mkdir(stageName(version), 0700); err != nil {
		return nil, errors.Join(err, recoverTransaction(root, version))
	}
	return tx, nil
}

func (tx *transaction) phase(value string) error {
	tx.record.Phase = value
	data, err := json.Marshal(tx.record)
	if err != nil {
		return err
	}
	return durable.Write(filepath.Join(tx.root.Name(), journalName(tx.record.Version)), data, 0600)
}
func (tx *transaction) publish(ctx context.Context) error {
	version := tx.record.Version
	stage := stageName(version)
	if err := tx.phase("prepared"); err != nil {
		return err
	}
	if tx.hook != nil {
		tx.hook("prepared")
	}
	if _, err := tx.root.Lstat(version); err == nil {
		if err := tx.root.Rename(version, filepath.Join(stage, "previous")); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := tx.phase("old_moved"); err != nil {
		return err
	}
	if tx.hook != nil {
		tx.hook("old_moved")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := tx.root.Rename(filepath.Join(stage, "runtime"), version); err != nil {
		return err
	}
	if err := tx.phase("published"); err != nil {
		return err
	}
	if tx.hook != nil {
		tx.hook("published")
	}
	return nil
}
func (tx *transaction) finish() error {
	tx.active = false
	return recoverTransaction(tx.root, tx.record.Version)
}
func (tx *transaction) onExit(result *error) {
	if tx.active {
		tx.active = false
		if err := recoverTransaction(tx.root, tx.record.Version); err != nil {
			*result = errors.Join(*result, err)
		}
	}
}

func exists(root *os.Root, name string) (bool, error) {
	_, err := root.Lstat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// Só deve ser chamada com o lock exclusivo da versão. Caminhos são derivados
// da versão validada, nunca de caminhos arbitrários do journal.
func recoverTransaction(root *os.Root, version string) error {
	file, err := root.Open(journalName(version))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, 8193))
	if err := errors.Join(readErr, file.Close()); err != nil {
		return err
	}
	var record journal
	if len(data) > 8192 || json.Unmarshal(data, &record) != nil || record.Schema != 1 || record.Version != version || len(record.ID) != 32 || (record.Operation != "install" && record.Operation != "remove") {
		return fmt.Errorf("journal inválido; arquivos preservados: %s", journalName(version))
	}
	if _, err := hex.DecodeString(record.ID); err != nil {
		return fmt.Errorf("identificador de transação inválido")
	}
	switch record.Phase {
	case "building", "prepared", "old_moved", "published", "detaching":
	default:
		return fmt.Errorf("fase de transação desconhecida; arquivos preservados")
	}
	stage := stageName(version)
	if info, err := root.Lstat(stage); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return fmt.Errorf("área de transação é link/não diretório; preservada")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	destination, err := exists(root, version)
	if err != nil {
		return err
	}
	previous, err := exists(root, filepath.Join(stage, "previous"))
	if err != nil {
		return err
	}
	payload, err := exists(root, filepath.Join(stage, "runtime"))
	if err != nil {
		return err
	}
	if record.Operation == "install" {
		if previous && !destination {
			if err := root.Rename(filepath.Join(stage, "previous"), version); err != nil {
				return fmt.Errorf("restaurar instalação anterior: %w", err)
			}
		}
		if previous && destination {
			language := filepath.Base(root.Name())
			meta, err := readMetadataAt(filepath.Join(root.Name(), version), language, version, Platform{runtime.GOOS, runtime.GOARCH})
			if err != nil || meta.TransactionID != record.ID {
				return fmt.Errorf("destino e backup presentes sem publicação válida; preservados para inspeção")
			}
			if meta.SchemaVersion == 2 {
				if err := verifyFiles(filepath.Join(root.Name(), version), meta); err != nil {
					if err := root.Rename(version, filepath.Join(stage, "failed")); err != nil {
						return err
					}
					if err := root.Rename(filepath.Join(stage, "previous"), version); err != nil {
						return fmt.Errorf("restaurar após corrupção: %w", err)
					}
				}
			}
		}
	} else {
		if destination && payload {
			return fmt.Errorf("remoção ambígua: destino e quarentena presentes; preservados")
		}
		// Após detach, concluir a limpeza é o commit da remoção autorizada.
	}
	if err := root.RemoveAll(stage); err != nil {
		return fmt.Errorf("limpeza pendente; execute verman recover %s %s: %w", filepath.Base(root.Name()), version, err)
	}
	if err := root.Remove(journalName(version)); err != nil {
		return err
	}
	return nil
}

func Recover(ctx context.Context, language, version string) error {
	if err := tool.Validate(language, version); err != nil {
		return err
	}
	destination, err := config.InstallDir(language, version)
	if err != nil {
		return err
	}
	parent, err := managedParent(filepath.Dir(filepath.Dir(destination)), language, false)
	if err != nil {
		return err
	}
	defer parent.Close()
	unlock, err := lockVersion(parent, language, version)
	if err != nil {
		return err
	}
	defer unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	return recoverTransaction(parent, version)
}
