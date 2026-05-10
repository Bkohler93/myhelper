package cmd

import (
	"testing"

	"github.com/bkohler93/myhelper/internal/config"
)

func TestApplyFlagOverrides(t *testing.T) {
	t.Run("positive token-limit overrides threshold", func(t *testing.T) {
		tokenLimitFlag = 8000
		defer func() { tokenLimitFlag = 0 }()
		cfg := config.Config{TokenThreshold: 4100}
		ApplyFlagOverrides(&cfg)
		if cfg.TokenThreshold != 8000 {
			t.Errorf("expected threshold 8000, got %d", cfg.TokenThreshold)
		}
	})

	t.Run("zero token-limit leaves threshold unchanged", func(t *testing.T) {
		tokenLimitFlag = 0
		cfg := config.Config{TokenThreshold: 4100}
		ApplyFlagOverrides(&cfg)
		if cfg.TokenThreshold != 4100 {
			t.Errorf("zero flag must not override threshold, got %d", cfg.TokenThreshold)
		}
	})

	t.Run("negative token-limit is ignored", func(t *testing.T) {
		tokenLimitFlag = -1
		defer func() { tokenLimitFlag = 0 }()
		cfg := config.Config{TokenThreshold: 4100}
		ApplyFlagOverrides(&cfg)
		if cfg.TokenThreshold != 4100 {
			t.Errorf("negative flag must not override threshold, got %d", cfg.TokenThreshold)
		}
	})
}
