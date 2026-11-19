//go:build !windows

package installer

import "path/filepath"

func phpExtensionDir(final string) ([]byte, error) {
	return []byte(filepath.ToSlash(filepath.Join(final, "ext"))), nil
}
