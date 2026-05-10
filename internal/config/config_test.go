package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Run("default TokenThreshold is 4100", func(t *testing.T) {
		// Ensure no env var interference
		t.Setenv("MYHELPER_TOKEN_LIMIT", "")
		t.Setenv("MYHELPER_ENDPOINT", "")
		t.Setenv("MYHELPER_MODEL", "")

		// Use a temp dir with no config file
		dir := t.TempDir()
		orig, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(orig) }()

		cfg := Load()
		if cfg.TokenThreshold != 4100 {
			t.Errorf("expected TokenThreshold 4100, got %d", cfg.TokenThreshold)
		}
	})

	t.Run("MYHELPER_TOKEN_LIMIT env var overrides default", func(t *testing.T) {
		t.Setenv("MYHELPER_TOKEN_LIMIT", "2000")
		t.Setenv("MYHELPER_ENDPOINT", "")
		t.Setenv("MYHELPER_MODEL", "")

		dir := t.TempDir()
		orig, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(orig) }()

		cfg := Load()
		if cfg.TokenThreshold != 2000 {
			t.Errorf("expected TokenThreshold 2000, got %d", cfg.TokenThreshold)
		}
	})

	t.Run("config file .myhelper/config.json sets TokenThreshold", func(t *testing.T) {
		t.Setenv("MYHELPER_TOKEN_LIMIT", "")
		t.Setenv("MYHELPER_ENDPOINT", "")
		t.Setenv("MYHELPER_MODEL", "")

		dir := t.TempDir()
		orig, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(orig) }()

		// Create .myhelper/config.json
		if err := os.MkdirAll(".myhelper", 0755); err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(map[string]interface{}{"token_threshold": 3000})
		if err := os.WriteFile(filepath.Join(".myhelper", "config.json"), data, 0644); err != nil {
			t.Fatal(err)
		}

		cfg := Load()
		if cfg.TokenThreshold != 3000 {
			t.Errorf("expected TokenThreshold 3000, got %d", cfg.TokenThreshold)
		}
	})

	t.Run("localConfigPath returns .myhelper/config.json", func(t *testing.T) {
		got := localConfigPath()
		want := ".myhelper/config.json"
		if got != want {
			t.Errorf("localConfigPath() = %q, want %q", got, want)
		}
	})

	t.Run("MYHELPER_TOKEN_LIMIT env overrides .myhelper/config.json file value", func(t *testing.T) {
		t.Setenv("MYHELPER_TOKEN_LIMIT", "1500")
		t.Setenv("MYHELPER_ENDPOINT", "")
		t.Setenv("MYHELPER_MODEL", "")

		dir := t.TempDir()
		orig, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(orig) }()

		// Create .myhelper/config.json with a different value
		if err := os.MkdirAll(".myhelper", 0755); err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(map[string]interface{}{"token_threshold": 3000})
		if err := os.WriteFile(filepath.Join(".myhelper", "config.json"), data, 0644); err != nil {
			t.Fatal(err)
		}

		cfg := Load()
		if cfg.TokenThreshold != 1500 {
			t.Errorf("expected TokenThreshold 1500 (env overrides file), got %d", cfg.TokenThreshold)
		}
	})

	t.Run("model and endpoint are empty when no config or env set", func(t *testing.T) {
		t.Setenv("MYHELPER_MODEL", "")
		t.Setenv("MYHELPER_ENDPOINT", "")
		t.Setenv("MYHELPER_TOKEN_LIMIT", "")

		dir := t.TempDir()
		orig, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.Chdir(orig) }()

		cfg := Load()
		if cfg.Model != "" {
			t.Errorf("expected empty Model, got %q (CFG-01)", cfg.Model)
		}
		if cfg.Endpoint != "" {
			t.Errorf("expected empty Endpoint, got %q (CFG-02)", cfg.Endpoint)
		}
	})
}
