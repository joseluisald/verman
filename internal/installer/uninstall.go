package installer

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/tool"
)

// Uninstall não muda o TOML e não procura referências em outros projetos.
func Uninstall(ctx context.Context, dir, language, version string, allowSelected bool) error {
	if err := tool.Validate(language, version); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	cfg, err := config.LoadProjectConfig(dir)
	if err != nil {
		return err
	}
	global, err := config.LoadGlobalConfig()
	if err != nil {
		return err
	}
	if (cfg.Tools[language] == version || global.Tools[language] == version) && !allowSelected {
		return fmt.Errorf("%s %s está selecionado pelo projeto atual ou globalmente; remover deixará sua seleção sem instalação. Use --allow-selected para permitir; as configurações serão mantidas", language, version)
	}
	base, err := config.BaseDir()
	if err != nil {
		return err
	}
	return uninstallAt(ctx, base, language, version)
}

func uninstallAt(ctx context.Context, base, language, version string) (resultErr error) {
	if err := tool.Validate(language, version); err != nil {
		return err
	}
	// Verificação lexical e resolvida antes de qualquer remoção recursiva.
	rootPath, err := filepath.Abs(filepath.Join(base, "installs"))
	if err != nil {
		return err
	}
	target := filepath.Join(rootPath, language, version)
	rel, err := filepath.Rel(rootPath, target)
	if err != nil || !filepath.IsLocal(rel) {
		return fmt.Errorf("destino fora das instalações gerenciadas")
	}
	for _, path := range []string{base, rootPath, filepath.Join(rootPath, language), target} {
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("consultar instalação %s %s: %w", language, version, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("remoção recusada: diretório gerenciado é link ou não é diretório: %s", path)
		}
	}
	resolvedRoot, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		return err
	}
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return err
	}
	resolvedRel, err := filepath.Rel(resolvedRoot, resolvedTarget)
	if err != nil || resolvedRel != rel {
		return fmt.Errorf("destino resolvido fora do caminho gerenciado")
	}
	// Operações ancoradas no diretório aberto também impedem escapes por links
	// alterados entre a verificação acima e a exclusão.
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return err
	}
	defer root.Close()
	parent, err := root.OpenRoot(language)
	if err != nil {
		return err
	}
	defer parent.Close()
	unlock, err := lockVersion(parent, language, version)
	if err != nil {
		return err
	}
	defer unlock()
	if err := recoverTransaction(parent, version); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := readMetadataAt(target, language, version, Platform{runtime.GOOS, runtime.GOARCH}); err != nil {
		return fmt.Errorf("remoção recusada: instalação sem metadados ou executáveis válidos: %w", err)
	}
	if language == "chocolatey" {
		for _, folder := range []string{"lib", "lib-bad", "lib-bkp"} {
			entries, err := fs.ReadDir(parent.FS(), version+"/tools/chocolateyInstall/"+folder)
			if err != nil && !os.IsNotExist(err) {
				return err
			}
			for _, entry := range entries {
				if entry.Name() != "chocolatey" {
					return fmt.Errorf("Chocolatey contém pacotes em %s; remova ou migre os pacotes antes de desinstalar esta versão", folder)
				}
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	guards, err := guardRemoval(target)
	if err != nil {
		return err
	}
	defer func() { guards() }()
	tx, err := beginTransaction(parent, version, "remove")
	if err != nil {
		return err
	}
	defer tx.onExit(&resultErr)
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := tx.phase("detaching"); err != nil {
		return err
	}
	// O Windows pode recusar rename de diretório com handles de filhos abertos.
	// O lease exclusivo continua impedindo novas execuções pelos shims/exec.
	guards()
	guards = func() {}
	if err := parent.Rename(version, filepath.Join(stageName(version), "runtime")); err != nil {
		return err
	}
	// Depois do detach, a remoção está comprometida. Libere os handles e
	// conclua a limpeza; uma interrupção será retomada por recover.
	return tx.finish()
}
