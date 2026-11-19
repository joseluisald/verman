//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func embedIcon(exe string, data []byte) error {
	group, frames, err := iconResources(data)
	if err != nil {
		return err
	}
	path, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	kernel := syscall.NewLazyDLL("kernel32.dll")
	begin := kernel.NewProc("BeginUpdateResourceW")
	update := kernel.NewProc("UpdateResourceW")
	end := kernel.NewProc("EndUpdateResourceW")
	handle, _, err := begin.Call(uintptr(unsafe.Pointer(path)), 0)
	if handle == 0 {
		return fmt.Errorf("abrir recursos do desktop: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			end.Call(handle, 1)
		}
	}()
	write := func(kind, id uintptr, bytes []byte) error {
		ok, _, err := update.Call(handle, kind, id, 0, uintptr(unsafe.Pointer(&bytes[0])), uintptr(len(bytes)))
		if ok == 0 {
			return fmt.Errorf("gravar recurso %d/%d: %w", kind, id, err)
		}
		return nil
	}
	for i, frame := range frames {
		if err := write(3, uintptr(i+1), frame); err != nil {
			return err
		}
	}
	// Wails v2 carrega IDI_APPLICATION (32512) para a janela e a barra de tarefas.
	if err := write(14, 32512, group); err != nil {
		return err
	}
	ok, _, err := end.Call(handle, 0)
	committed = true
	if ok == 0 {
		return fmt.Errorf("salvar ícone do desktop: %w", err)
	}
	return nil
}
