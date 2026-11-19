//go:build !windows

package installer

func guardRemoval(root string) (func(), error) { return func() {}, nil }
