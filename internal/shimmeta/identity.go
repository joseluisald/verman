package shimmeta

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

func Matches(executable string) bool {
	file, err := os.Open(filepath.Join(filepath.Dir(executable), ".verman-shims.json"))
	if err != nil {
		return false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 4097))
	if err != nil || len(data) > 4096 {
		return false
	}
	var manifest struct {
		Schema int    `json:"schema"`
		SHA256 string `json:"sha256"`
	}
	if json.Unmarshal(data, &manifest) != nil || manifest.Schema != 1 {
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
