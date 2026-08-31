package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Nox1KCL/Arbitrage/internal/logger"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	WorkersNum   int                 `toml:"workers_num"`
	AdminID      int                 `toml:"admin_id"`
	LogsDir      string              `toml:"logs_dir"`
	LumberConfig logger.LumberConfig `toml:"logger"`
}

func GetConfig(path string) (*Config, error) {
	var doc []byte
	var err error
	if filepath.IsAbs(filepath.Clean(path)) {
		return nil, fmt.Errorf("path %s is not absolute", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("path is not exist: %s", path)
		}
		return nil, err
	}

	if !info.IsDir() {
		doc, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading config file %q: %w", path, err)
		}
	} else {
		return nil, fmt.Errorf("path %s leads to dir", path)
	}

	var cfg Config

	if filepath.Ext(path) != ".toml" {
		return nil, fmt.Errorf("unexpected file type (accepting only toml, got %s)", filepath.Ext(path))
	}
	if err := toml.Unmarshal(doc, &cfg); err != nil {
		return nil, fmt.Errorf("reading toml doc %q: %w", path, err)
	}

	return &cfg, nil
}
