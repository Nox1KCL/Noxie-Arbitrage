package config

import (
	"fmt"
	"github.com/Nox1KCL/Arbitrage/internal/logger"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	WorkersNum int                 `toml:"workers_num"`
	AdminID    int                 `toml:"admin_id"`
	LogsDir    string              `toml:"logs_dir"`
	LumberConfig     logger.LumberConfig `toml:"logger"`
}

func GetConfig(path string) (*Config, error) {
	var doc []byte
	var err error

	if path != "" || filepath.IsAbs(filepath.Clean(path)) {
		doc, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading config file %q: %w", path, err)
		}
	} else {
		return nil, fmt.Errorf("no path provided or not absolute path")
	}

	var cfg Config

	if err := toml.Unmarshal(doc, &cfg); err != nil {
		return nil, fmt.Errorf("reading toml doc %q: %w", path, err)
	}

	return &cfg, nil
}
