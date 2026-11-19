//go:build windows

package installer

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

// PHP carrega DLLs com LoadLibraryA. Um extension_dir UTF-8 com caracteres fora
// de ASCII não é interpretado como UTF-8 nesse ponto. Prefira o caminho curto
// do pai já existente, e codifique o resultado exatamente na página ANSI local.
func phpExtensionDir(final string) ([]byte, error) {
	parent := filepath.Dir(final)
	ptr, err := syscall.UTF16PtrFromString(parent)
	if err != nil {
		return nil, err
	}
	if n, err := syscall.GetShortPathName(ptr, nil, 0); err == nil && n > 0 {
		buffer := make([]uint16, n)
		if written, err := syscall.GetShortPathName(ptr, &buffer[0], n); err == nil && written < n {
			parent = syscall.UTF16ToString(buffer)
		}
	}
	path := filepath.ToSlash(filepath.Join(parent, filepath.Base(final), "ext"))
	wide, err := syscall.UTF16FromString(path)
	if err != nil {
		return nil, err
	}
	kernel := syscall.NewLazyDLL("kernel32.dll")
	codepage, _, _ := kernel.NewProc("GetACP").Call()
	convert := kernel.NewProc("WideCharToMultiByte")
	flags := uintptr(0x400) // WC_NO_BEST_FIT_CHARS: nunca substitua caracteres do caminho.
	var substituted int32
	usedDefault := &substituted
	if codepage == 65001 {
		flags = 0x80
		usedDefault = nil
	} // WC_ERR_INVALID_CHARS para UTF-8.
	n, _, callErr := convert.Call(codepage, flags, uintptr(unsafe.Pointer(&wide[0])), uintptr(len(wide)), 0, 0, 0, uintptr(unsafe.Pointer(usedDefault)))
	if n == 0 {
		return nil, fmt.Errorf("codificar extension_dir: %w", callErr)
	}
	result := make([]byte, n)
	written, _, callErr := convert.Call(codepage, flags, uintptr(unsafe.Pointer(&wide[0])), uintptr(len(wide)), uintptr(unsafe.Pointer(&result[0])), n, 0, uintptr(unsafe.Pointer(usedDefault)))
	if written == 0 {
		return nil, fmt.Errorf("codificar extension_dir: %w", callErr)
	}
	if substituted != 0 {
		return nil, fmt.Errorf("caminho PHP não representável na página ANSI do Windows e sem nome curto utilizável; escolha um diretório pessoal compatível")
	}
	return result[:written-1], nil
}
