package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCLIJSONContractsAndExitCodes(t *testing.T) {
	binary, project, home := step9Setup(t)
	for _, test := range []struct {
		args []string
		exit int
		code string
	}{
		{[]string{"list", "node", "--json"}, 0, ""},
		{[]string{"use", "node", "22.11.0", "--json", "--offline"}, 0, ""},
		{[]string{"global", "node", "22.12.0", "--json", "--offline"}, 0, ""},
		{[]string{"global", "--json"}, 0, ""},
		{[]string{"install", "node", "22.11.0", "--json", "--offline"}, 0, ""},
		{[]string{"doctor", "node", "--json"}, 1, "diagnostic_issue"},
		{[]string{"use", "bad", "1.0.0", "--json"}, 2, "invalid_argument"},
		{[]string{"install", "node", "--json"}, 2, "invalid_argument"},
		{[]string{"list", "node", "--refresh", "--json"}, 2, "invalid_argument"},
		{[]string{"env", "--json"}, 2, "invalid_argument"},
		{[]string{"--json", "unknown-command"}, 2, "invalid_argument"},
		{[]string{"list", "--unknown-flag", "--json"}, 2, "invalid_argument"},
		{[]string{"list", "node", "--available", "--offline", "--json"}, 3, "offline"},
		{[]string{"support", "--json"}, 0, ""},
	} {
		cmd := exec.Command(binary, test.args...)
		cmd.Dir = project
		var out, diagnostics bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &diagnostics
		err := cmd.Run()
		exit := 0
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				exit = e.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if exit != test.exit {
			t.Fatalf("%v: exit=%d, stdout=%s stderr=%s", test.args, exit, &out, &diagnostics)
		}
		var response struct {
			Schema  int             `json:"schema"`
			Success bool            `json:"success"`
			Data    json.RawMessage `json:"data"`
			Error   *errorResponse  `json:"error"`
		}
		decoder := json.NewDecoder(&out)
		if err := decoder.Decode(&response); err != nil {
			t.Fatalf("%v: JSON inválido: %v %s", test.args, err, &out)
		}
		if response.Schema != 2 || response.Success != (test.exit == 0) {
			t.Fatalf("%v: contrato inválido %#v", test.args, response)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			t.Fatalf("%v: stdout tem conteúdo extra: %v", test.args, err)
		}
		if test.code != "" && (response.Error == nil || response.Error.Code != test.code || response.Error.Hint == "") {
			t.Fatalf("%v: erro inválido %#v", test.args, response.Error)
		}
		if test.args[0] == "doctor" && bytes.Equal(response.Data, []byte("null")) {
			t.Fatal("diagnóstico perdido em erro")
		}
		if test.args[0] == "support" && (bytes.Contains(response.Data, []byte(home)) || bytes.Contains(response.Data, []byte(project))) {
			t.Fatal("support expôs diretórios")
		}
	}
	if err := os.WriteFile(filepath.Join(project, ".verman.toml"), []byte("[broken"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "list", "--json")
	cmd.Dir = project
	out, err := cmd.Output()
	if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 6 {
		t.Fatalf("TOML inválido: %s %v", out, err)
	}
	if !bytes.Contains(out, []byte(`"code":"configuration_invalid"`)) {
		t.Fatalf("classificação: %s", out)
	}
}
