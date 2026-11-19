package installer

import (
	"context"
	"os"
	"runtime"
	"testing"
)

func TestOfficialHistoricalCatalogs(t *testing.T) {
	if os.Getenv("VERMAN_INTEGRATION") != "1" || runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("opt-in Windows/amd64 com rede")
	}
	isolatedHome(t)
	ctx := context.Background()
	php, err := AvailableHistory(ctx, "php", true, true)
	if err != nil {
		t.Fatal(err)
	}
	archived, unverified := 0, 0
	for _, a := range php.Catalog.Artifacts {
		if a.Archived {
			archived++
			if !Describe(a).Installable {
				unverified++
			}
		}
	}
	if archived == 0 || unverified == 0 {
		t.Fatal("histórico PHP sem arquivos antigos/bloqueados")
	}
	t.Logf("PHP: %d entradas, %d de arquivo, %d sem checksum oficial (bloqueadas)", len(php.Catalog.Artifacts), archived, unverified)
	latest, err := Available(ctx, "python", true)
	if err != nil {
		t.Fatal(err)
	}
	history, err := AvailableHistory(ctx, "python", true, true)
	if err != nil {
		t.Fatal(err)
	}
	current := map[string]bool{}
	for _, a := range latest.Catalog.Artifacts {
		current[a.Version] = true
	}
	var selected Artifact
	for _, a := range history.Catalog.Artifacts {
		if !current[a.Version] {
			selected = a
			break
		}
	}
	if selected.Version == "" {
		t.Fatal("nenhuma versão adicional encontrada nas 30 releases consultadas")
	}
	t.Logf("Python histórico: %s build %s", selected.Version, selected.Build)
	if err := InstallWithEvents(ctx, "python", selected.Version, nil); err != nil {
		t.Fatal(err)
	}
	if err := Verify("python", selected.Version); err != nil {
		t.Fatal(err)
	}
	metadata, err := ReadMetadata("python", selected.Version)
	if err != nil || metadata.Build != selected.Build {
		t.Fatalf("build publicado: %#v %v", metadata, err)
	}
}
