//go:build !windows

package main

import "fmt"

func embedIcon(string, []byte) error { return fmt.Errorf("incorporar ícone ao EXE exige Windows") }
