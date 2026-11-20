package environment

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
)

var managedVariables = []string{"PHPRC", "PHP_INI_SCAN_DIR", "PYTHONHOME", "PYTHONPATH", "GOROOT", "GOTOOLCHAIN", "CARGO_HOME", "RUSTUP_HOME", "RUSTUP_TOOLCHAIN", "VCPKG_ROOT", "CHOCOLATEYINSTALL", "DOTNET_ROOT", "DOTNET_ROOT_X64", "DOTNET_MULTILEVEL_LOOKUP"}

func envKey(key string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(key)
	}
	return key
}

func envMap(environ []string) map[string]string {
	values := map[string]string{}
	for _, item := range environ {
		pair := strings.SplitN(item, "=", 2)
		if len(pair) == 2 {
			values[envKey(pair[0])] = pair[1]
		}
	}
	return values
}

func allowedVariable(key string) bool {
	for _, name := range managedVariables {
		if name == key {
			return true
		}
	}
	return false
}

// Compute cria o ambiente do filho sem modificar o processo nem executar código
// armazenado em marcadores ou configurações. É usado por exec e shells POSIX.
func Compute(dir string, environ []string) ([]string, error) {
	paths, nextVars, err := selectedEnvironment(dir)
	if err != nil {
		return nil, err
	}
	return applySelection(paths, nextVars, environ)
}

func applySelection(paths []string, nextVars map[string]*string, environ []string) ([]string, error) {
	values := envMap(environ)
	oldPaths := []string{}
	if raw := values["VERMAN_PATH_ENTRIES"]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &oldPaths); err != nil {
			return nil, fmt.Errorf("VERMAN_PATH_ENTRIES inválido: %w", err)
		}
	}
	oldVars := map[string]*string{}
	if raw := values["VERMAN_ENV_BACKUP"]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &oldVars); err != nil {
			return nil, fmt.Errorf("VERMAN_ENV_BACKUP inválido: %w", err)
		}
		if oldVars == nil {
			return nil, fmt.Errorf("VERMAN_ENV_BACKUP inválido: objeto esperado")
		}
		for key := range oldVars {
			if !allowedVariable(key) {
				return nil, fmt.Errorf("variável não permitida no backup: %s", key)
			}
		}
	}
	removed := map[string]bool{}
	for _, path := range append(append([]string{}, oldPaths...), paths...) {
		removed[envKey(path)] = true
	}
	resultPaths := append([]string{}, paths...)
	if current, exists := values["PATH"]; exists {
		for _, path := range strings.Split(current, string(os.PathListSeparator)) {
			if !removed[envKey(path)] {
				resultPaths = append(resultPaths, path)
			}
		}
	}
	values["PATH"] = strings.Join(resultPaths, string(os.PathListSeparator))
	for name, value := range oldVars {
		if value == nil {
			delete(values, name)
		} else {
			values[name] = *value
		}
	}
	backup := map[string]*string{}
	for name, value := range nextVars {
		if previous, exists := values[name]; exists {
			copy := previous
			backup[name] = &copy
		} else {
			backup[name] = nil
		}
		if value == nil {
			delete(values, name)
		} else {
			values[name] = *value
		}
	}
	if len(paths) > 0 {
		data, _ := json.Marshal(paths)
		values["VERMAN_PATH_ENTRIES"] = string(data)
	} else {
		delete(values, "VERMAN_PATH_ENTRIES")
	}
	if len(backup) > 0 {
		data, _ := json.Marshal(backup)
		values["VERMAN_ENV_BACKUP"] = string(data)
	} else {
		delete(values, "VERMAN_ENV_BACKUP")
	}
	result := make([]string, 0, len(values))
	for key, value := range values {
		result = append(result, key+"="+value)
	}
	sort.Strings(result)
	return result, nil
}

func ShellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }

func POSIX(dir string) (string, error) {
	child, err := Compute(dir, os.Environ())
	if err != nil {
		return "", err
	}
	return renderPOSIX(child), nil
}

func ClearPOSIX() (string, error) {
	child, err := applySelection(nil, map[string]*string{}, os.Environ())
	if err != nil {
		return "", err
	}
	return renderPOSIX(child), nil
}

func renderPOSIX(child []string) string {
	values := envMap(child)
	var code strings.Builder
	if runtime.GOOS == "windows" {
		code.WriteString("if _verman_path=$(command cygpath -up -- " + ShellQuote(values["PATH"]) + "); then\n")
		code.WriteString("export PATH=\"$_verman_path\"\n")
	} else {
		code.WriteString("export PATH=" + ShellQuote(values["PATH"]) + "\n")
	}
	for _, key := range append(append([]string{}, managedVariables...), "VERMAN_PATH_ENTRIES", "VERMAN_ENV_BACKUP") {
		if value, exists := values[key]; exists {
			code.WriteString("export " + key + "=" + ShellQuote(value) + "\n")
		} else {
			code.WriteString("unset " + key + "\n")
		}
	}
	if runtime.GOOS == "windows" {
		code.WriteString("else\nprintf '%s\\n' 'Verman: cygpath indisponivel; ambiente preservado.' >&2\nfi\nunset _verman_path\n")
	}
	return code.String()
}
