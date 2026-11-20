package installer

import (
	"github.com/joseluisald/verman/internal/tool"
	"strings"
	"time"
)

type DistributionInfo struct {
	Provider          string `json:"provider"`
	Trust             string `json:"trust"`
	Installable       bool   `json:"installable"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	Support           string `json:"support"`
	EOL               string `json:"eol,omitempty"`
	PolicySource      string `json:"policy_source"`
	PolicyReviewed    string `json:"policy_reviewed"`
}

func Describe(a Artifact) DistributionInfo { return describeAt(a, time.Now().UTC()) }
func describeAt(a Artifact, now time.Time) DistributionInfo {
	info := DistributionInfo{Installable: true, Support: "unknown", PolicyReviewed: "2026-11-18", Trust: "publisher_sha256_https"}
	parts := strings.Split(a.Version, ".")
	branch := ""
	if len(parts) >= 2 {
		branch = parts[0] + "." + parts[1]
	}
	if info.Provider == "" {
		if d, ok := tool.Lookup(a.Language); ok {
			info.Provider = d.Name + " oficial"
		}
		info.PolicySource = a.ChecksumURL
	}
	if (a.Language == "yarn" || a.Language == "dotnet") && validSHA512(a.SHA512) {
		info.Trust = "publisher_sha512_https"
		return info
	}
	if a.Language == "dotnet" {
		info.Installable = false
		info.Trust = "unverified"
		info.UnavailableReason = "checksum oficial SHA-512 ausente"
		return info
	}
	switch a.Language {
	case "node":
		info.Provider = "Node.js oficial"
		info.PolicySource = "https://github.com/nodejs/Release/blob/main/schedule.json"
		if len(parts) > 0 {
			info.EOL = map[string]string{"18": "2025-04-30", "19": "2023-06-01", "20": "2026-04-30", "21": "2024-06-01", "22": "2027-04-30", "23": "2025-06-01", "24": "2028-04-30", "25": "2026-06-01", "26": "2029-04-30"}[parts[0]]
		}
	case "php":
		info.Provider = "PHP oficial NTS x64"
		info.PolicySource = "https://www.php.net/supported-versions.php"
		info.EOL = map[string]string{"8.2": "2026-12-31", "8.3": "2027-12-31", "8.4": "2028-12-31", "8.5": "2029-12-31"}[branch]
		if len(parts) >= 2 && versionGreater("8.2.0", a.Version) {
			info.Support = "eol"
		}
	case "python":
		info.Provider = "Astral python-build-standalone (CPython)"
		info.PolicySource = "https://devguide.python.org/versions/"
		info.EOL = map[string]string{"3.8": "2024-10", "3.9": "2025-10", "3.10": "2026-10", "3.11": "2027-10", "3.12": "2028-10", "3.13": "2029-10", "3.14": "2030-10", "3.15": "2031-10"}[branch]
	}
	if info.EOL != "" {
		date := now.Format("2006-01-02")
		if len(info.EOL) == 7 {
			date = now.Format("2006-01")
		}
		info.Support = "supported_branch"
		if date > info.EOL {
			info.Support = "eol"
		}
	}
	if a.Language != "node" && !validSHA256(a.SHA256) {
		info.Installable = false
		info.Trust = "unverified"
		info.UnavailableReason = "checksum oficial SHA-256 ausente"
	}
	return info
}
