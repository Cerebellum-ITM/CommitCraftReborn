package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// Env var names for the dual Groq API key slots. GROQ_API_KEY keeps its
// historical name (the "user" slot) so existing single-key setups are
// untouched. GROQ_ACTIVE_KEY points at whichever slot is live.
const (
	EnvGroqUserKey = "GROQ_API_KEY"
	EnvGroqAIKey   = "GROQ_API_KEY_AI"
	EnvGroqActive  = "GROQ_ACTIVE_KEY"
	KeySlotUser    = "user"
	KeySlotAI      = "ai"
	defaultKeySlot = KeySlotUser
	envFileName    = ".env"
	envDirMode     = 0o755
	envFileMode    = 0o600
)

// GlobalEnvPath returns the absolute path of the global `.env` file,
// creating the parent directory at mode 0o755 if it doesn't exist yet.
// All credential-bearing keys (GROQ_API_KEY, GROQ_API_KEY_AI, GH_TOKEN, …)
// live here so they never get checked in next to a per-repo
// `.commitcraft.toml`. Path matches the one LoadConfigs reads from.
func GlobalEnvPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, GlobalConfigDir)
	if err := os.MkdirAll(dir, envDirMode); err != nil {
		return "", err
	}
	return filepath.Join(dir, envFileName), nil
}

// SaveEnvVar upserts a single KEY=VALUE pair in the global `.env` file while
// preserving the order and contents of any other keys already stored there.
// The file is written at mode 0o600. An empty value removes the key. This is
// the single source of truth for `.env` mutation — the TUI delegates here.
func SaveEnvVar(name, value string) error {
	envPath, err := GlobalEnvPath()
	if err != nil {
		return err
	}

	existing := map[string]string{}
	order := []string{}
	if raw, err := os.ReadFile(envPath); err == nil {
		for _, line := range bytes.Split(raw, []byte("\n")) {
			s := string(line)
			if s == "" || strings.HasPrefix(s, "#") {
				continue
			}
			eq := strings.IndexByte(s, '=')
			if eq <= 0 {
				continue
			}
			k := strings.TrimSpace(s[:eq])
			v := s[eq+1:]
			if _, seen := existing[k]; !seen {
				order = append(order, k)
			}
			existing[k] = v
		}
	}

	if _, exists := existing[name]; !exists && value != "" {
		order = append(order, name)
	}
	if value == "" {
		delete(existing, name)
		filtered := order[:0]
		for _, k := range order {
			if k != name {
				filtered = append(filtered, k)
			}
		}
		order = filtered
	} else {
		existing[name] = value
	}

	var buf bytes.Buffer
	for _, k := range order {
		fmt.Fprintf(&buf, "%s=%s\n", k, existing[k])
	}
	return os.WriteFile(envPath, buf.Bytes(), envFileMode)
}

// NormalizeKeySlot maps arbitrary input to a valid slot, defaulting to the
// user slot for anything that isn't exactly "ai".
func NormalizeKeySlot(slot string) string {
	if strings.ToLower(strings.TrimSpace(slot)) == KeySlotAI {
		return KeySlotAI
	}
	return defaultKeySlot
}

// EnvVarForSlot returns the `.env` variable name backing the given slot.
func EnvVarForSlot(slot string) string {
	if NormalizeKeySlot(slot) == KeySlotAI {
		return EnvGroqAIKey
	}
	return EnvGroqUserKey
}

// ReadEnvFile parses the global `.env` directly (independent of the process
// environment, so it reflects exactly what's on disk after a SaveEnvVar). A
// missing file yields an empty map, not an error.
func ReadEnvFile() (map[string]string, error) {
	p, err := GlobalEnvPath()
	if err != nil {
		return nil, err
	}
	m, err := godotenv.Read(p)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	return m, nil
}

// KeyState is the disk-truth view of the two Groq key slots, used by the
// `commitcraft ai key` subcommand. It never carries the key values, only
// whether each slot is populated.
type KeyState struct {
	ActiveSlot string
	UserKeySet bool
	AIKeySet   bool
}

// SlotSet reports whether the given slot currently holds a key.
func (s KeyState) SlotSet(slot string) bool {
	if NormalizeKeySlot(slot) == KeySlotAI {
		return s.AIKeySet
	}
	return s.UserKeySet
}

// LoadKeyState reads the current slot configuration straight from the `.env`
// file on disk.
func LoadKeyState() (KeyState, error) {
	m, err := ReadEnvFile()
	if err != nil {
		return KeyState{}, err
	}
	return KeyState{
		ActiveSlot: NormalizeKeySlot(m[EnvGroqActive]),
		UserKeySet: strings.TrimSpace(m[EnvGroqUserKey]) != "",
		AIKeySet:   strings.TrimSpace(m[EnvGroqAIKey]) != "",
	}, nil
}
