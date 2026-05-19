package tui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"commit_craft_reborn/internal/config"
)

// UpdateLocalConfigVersion sets release_config.version inside the repo's
// .commitcraft.toml. The file is created from the default template if it
// doesn't exist yet, so the user doesn't need to bootstrap it manually.
func UpdateLocalConfigVersion(version string) error {
	if err := config.CreateLocalConfigTomlTmpl(); err != nil {
		return fmt.Errorf("ensuring local config exists: %w", err)
	}

	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	configPath := filepath.Join(workDir, ".commitcraft.toml")

	var cfg config.Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		return fmt.Errorf("decoding %s: %w", configPath, err)
	}
	cfg.ReleaseConfig.Version = version

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(configPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", configPath, err)
	}
	return nil
}

// UpdateLocalConfigRelease writes the four user-facing release fields
// (repository, branch, version, binary assets path) into the repo's
// `.commitcraft.toml`. GH_TOKEN is never serialized here — it lives in
// ~/.config/CommitCraft/.env via SaveGhTokenToEnv. The file is created
// from the default template on first call so the user doesn't have to
// bootstrap it manually.
func UpdateLocalConfigRelease(
	repository, branch, version, assetsPath string,
) error {
	if err := config.CreateLocalConfigTomlTmpl(); err != nil {
		return fmt.Errorf("ensuring local config exists: %w", err)
	}

	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	configPath := filepath.Join(workDir, ".commitcraft.toml")

	var cfg config.Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		return fmt.Errorf("decoding %s: %w", configPath, err)
	}
	cfg.ReleaseConfig.Repository = repository
	cfg.ReleaseConfig.Branch = branch
	cfg.ReleaseConfig.Version = version
	cfg.ReleaseConfig.BinaryAssetsPath = assetsPath

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(configPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", configPath, err)
	}
	return nil
}

// UpdateConfigTheme persists the chosen TUI theme to the global config
// (~/.config/CommitCraft/config.toml). The theme is a user-level
// preference; per-repo `.commitcraft.toml` files can still override it at
// read time but are never modified here.
func UpdateConfigTheme(theme string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	globalDir := filepath.Join(home, config.GlobalConfigDir)
	globalPath := filepath.Join(globalDir, "config.toml")
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		return err
	}

	var cfg config.Config
	if _, err := os.Stat(globalPath); err == nil {
		if _, err := toml.DecodeFile(globalPath, &cfg); err != nil {
			return fmt.Errorf("decoding %s: %w", globalPath, err)
		}
	} else {
		cfg = config.GetDefaultConfigWithTypes()
	}
	cfg.TUI.Theme = theme

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(globalPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", globalPath, err)
	}
	return nil
}

// globalEnvPath returns the absolute path of the global `.env` file,
// creating the parent directory at mode 0o755 if it doesn't exist yet.
// All credential-bearing keys (GROQ_API_KEY, GH_TOKEN, ...) live here so
// they never get checked in next to a per-repo `.commitcraft.toml`.
func globalEnvPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "CommitCraft")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, ".env"), nil
}

// saveEnvVar upserts a single KEY=VALUE pair in the global `.env` file
// while preserving the order and contents of any other keys already
// stored there. The file is written at mode 0o600. Empty `value` removes
// the key.
func saveEnvVar(name, value string) error {
	envPath, err := globalEnvPath()
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
		// Drop name from order while keeping the rest stable.
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
	return os.WriteFile(envPath, buf.Bytes(), 0o600)
}

// saveAPIKeyToEnv persists the user's Groq API key in the global `.env`
// file (created at mode 0o600). Wrapper around saveEnvVar so other call
// sites can use the original ergonomic helper name.
func saveAPIKeyToEnv(key string) error {
	return saveEnvVar("GROQ_API_KEY", key)
}

// SaveGhTokenToEnv persists the GitHub personal-access token in the
// global `.env`. Exported because the release config popup lives in a
// different file and needs to call it after the user finishes the form.
func SaveGhTokenToEnv(token string) error {
	return saveEnvVar("GH_TOKEN", token)
}
