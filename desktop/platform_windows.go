//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/joseluisald/verman/docs"
	"github.com/joseluisald/verman/internal/application"
	"github.com/joseluisald/verman/internal/shimmeta"
	"github.com/joseluisald/verman/internal/tool"
)

type SetupInfo struct {
	Version   string          `json:"version"`
	CLIPath   string          `json:"cli_path"`
	CLIExists bool            `json:"cli_exists"`
	Shims     map[string]bool `json:"shims"`
	Message   string          `json:"message"`
}

func (a *App) SetupInfo() SetupInfo {
	exe, _ := os.Executable()
	cli := filepath.Join(filepath.Dir(exe), "verman.exe")
	info := SetupInfo{Version: Version, CLIPath: cli, Shims: map[string]bool{}, Message: "Reabra outros terminais e o editor uma vez depois de configurar o PATH."}
	if stat, err := os.Stat(cli); err == nil && stat.Mode().IsRegular() {
		info.CLIExists = true
	}
	for _, lang := range tool.Languages() {
		path, err := application.FindExecutable(lang, os.Environ())
		info.Shims[lang] = err == nil && shimmeta.Matches(path)
	}
	return info
}
func (a *App) Documentation(topic string) (string, error) { return docs.Read(topic) }
func (a *App) OpenTerminal(dir string) error {
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("selecione uma pasta absoluta")
	}
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("o projeto não é uma pasta")
	}
	childEnv, err := application.TerminalEnvironment(dir, os.Environ())
	if err != nil {
		return err
	}
	// A pasta vai em Dir, nunca em uma expressão PowerShell. Sem scripts do projeto.
	shell := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	process := exec.Command(shell, "-NoLogo", "-NoProfile", "-NoExit")
	process.Dir = dir
	process.Env = childEnv
	process.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000010}
	if err := process.Start(); err != nil {
		return err
	}
	go process.Wait()
	return nil
}
