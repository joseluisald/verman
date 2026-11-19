package shimmeta

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

const FileName = ".verman-shims.json"

type Manifest struct {
	Schema int    `json:"schema"`
	SHA256 string `json:"sha256"`
	AppDir string `json:"app_dir,omitempty"`
}

func Read(dir string) (Manifest, bool) {
	file, err := os.Open(filepath.Join(dir, FileName))
	if err != nil {
		return Manifest{}, false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 4097))
	if err != nil || len(data) > 4096 {
		return Manifest{}, false
	}
	var manifest Manifest
	if json.Unmarshal(data, &manifest) != nil || manifest.Schema != 1 || len(manifest.SHA256) != 64 {
		return Manifest{}, false
	}
	return manifest, true
}

func Matches(executable string) bool {
	manifest, ok := Read(filepath.Dir(executable))
	if !ok {
		return false
	}
	binary, err := os.Open(executable)
	if err != nil {
		return false
	}
	defer binary.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, binary); err != nil {
		return false
	}
	return hex.EncodeToString(hash.Sum(nil)) == manifest.SHA256
}
