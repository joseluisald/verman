package tool

import "testing"

func TestRegistryCommandsHaveUniqueOwners(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range Definitions() {
		if err := Validate(d.ID, "1.2.3"); err != nil {
			t.Fatal(err)
		}
		if len(d.Commands) == 0 || d.Commands[0] != d.ID {
			t.Fatalf("comando principal ausente: %s", d.ID)
		}
		for _, name := range d.Commands {
			if seen[name] {
				t.Fatalf("comando duplicado: %s", name)
			}
			seen[name] = true
			owner, ok := Owner(name)
			if !ok || owner.ID != d.ID {
				t.Fatalf("owner %s", name)
			}
		}
	}
}
