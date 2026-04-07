package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	DefaultEndpoint = "192.168.0.9:11434"
	DefaultModel    = "qwen2.5-coder:7b"
)

// Config holds the resolved endpoint and model name.
type Config struct {
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`
}

// Load resolves config with the following precedence (highest to lowest):
//  1. Environment variables: MYHELPER_ENDPOINT, MYHELPER_MODEL
//  2. Config file: .myhelper.json in CWD, then ~/.config/myhelper/config.json
//  3. Hardcoded defaults
func Load() Config {
	cfg := Config{
		Endpoint: DefaultEndpoint,
		Model:    DefaultModel,
	}

	// Layer 2: config files (CWD takes precedence over home dir)
	if loaded, ok := loadFile(localConfigPath()); ok {
		if loaded.Endpoint != "" {
			cfg.Endpoint = loaded.Endpoint
		}
		if loaded.Model != "" {
			cfg.Model = loaded.Model
		}
	} else if loaded, ok := loadFile(homeConfigPath()); ok {
		if loaded.Endpoint != "" {
			cfg.Endpoint = loaded.Endpoint
		}
		if loaded.Model != "" {
			cfg.Model = loaded.Model
		}
	}

	// Layer 1: environment variables (highest priority)
	if v := os.Getenv("MYHELPER_ENDPOINT"); v != "" {
		cfg.Endpoint = v
	}
	if v := os.Getenv("MYHELPER_MODEL"); v != "" {
		cfg.Model = v
	}

	return cfg
}

func localConfigPath() string {
	return ".myhelper.json"
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
