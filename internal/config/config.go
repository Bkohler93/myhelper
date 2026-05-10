package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const (
	DefaultTokenThreshold = 4100
)

// Config holds the resolved endpoint, model name, and token threshold.
type Config struct {
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`
	// TokenThreshold is the maximum token count before conversation history is
	// summarized. The --token-limit CLI flag override is applied by the caller
	// (ApplyFlagOverrides in cmd/root.go) after Load() returns.
	TokenThreshold int `json:"token_threshold"`
}

// Load resolves config with the following precedence (highest to lowest):
//  1. Environment variables: MYHELPER_ENDPOINT, MYHELPER_MODEL, MYHELPER_TOKEN_LIMIT
//  2. Config file: .myhelper/config.json in CWD, then ~/.config/myhelper/config.json
//  3. Hardcoded defaults
func Load() Config {
	cfg := Config{
		TokenThreshold: DefaultTokenThreshold,
	}

	// Layer 2: config files (CWD takes precedence over home dir)
	if loaded, ok := loadFile(localConfigPath()); ok {
		if loaded.Endpoint != "" {
			cfg.Endpoint = loaded.Endpoint
		}
		if loaded.Model != "" {
			cfg.Model = loaded.Model
		}
		if loaded.TokenThreshold != 0 {
			cfg.TokenThreshold = loaded.TokenThreshold
		}
	} else if loaded, ok := loadFile(homeConfigPath()); ok {
		if loaded.Endpoint != "" {
			cfg.Endpoint = loaded.Endpoint
		}
		if loaded.Model != "" {
			cfg.Model = loaded.Model
		}
		if loaded.TokenThreshold != 0 {
			cfg.TokenThreshold = loaded.TokenThreshold
		}
	}

	// Layer 1: environment variables (highest priority)
	if v := os.Getenv("MYHELPER_ENDPOINT"); v != "" {
		cfg.Endpoint = v
	}
	if v := os.Getenv("MYHELPER_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("MYHELPER_TOKEN_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.TokenThreshold = n
		} else {
			fmt.Fprintf(os.Stderr, "warning: MYHELPER_TOKEN_LIMIT %q is not a valid integer; using default\n", v)
		}
	}

	return cfg
}

func localConfigPath() string {
	return ".myhelper/config.json"
}

func homeConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "myhelper", "config.json")
}

func loadFile(path string) (Config, bool) {
	if path == "" {
		return Config{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, false
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, false
	}
	return c, true
}
