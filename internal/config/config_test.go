package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nox1KCL/Arbitrage/internal/config"
)

func TestGetConfig(t *testing.T) {
	t.Run("Happy path", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "config.toml")
		err := os.WriteFile(filePath, []byte(
			`workers_num = 5
            admin_id = 123456789
            logs_dir = "/var/log/arbitrage"

            [logger]
            max_size = 10
            max_age = 28
            max_backups = 4
            compress = true`), 0o644)
		if err != nil {
			t.Fatalf("expected write on file, got %s", err.Error())
		}

		cfg, err := config.GetConfig(filePath)
		if err != nil {
			t.Fatalf("unexpected error, got %s", err.Error())
		}
		if cfg.WorkersNum != 5 {
			t.Errorf("got %d, want 5", cfg.WorkersNum)
		}
	})

	t.Run("Dir path", func(t *testing.T) {
		tmpDir := t.TempDir()
		dirPath := filepath.Join(tmpDir, "directory/")
		err := os.Mkdir(dirPath, 0o755)
		if err != nil {
			t.Fatalf("expected creating directory, got %s", err.Error())
		}

		_, err = config.GetConfig(dirPath)
		if !strings.Contains(err.Error(), "path leads to dir") {
			t.Errorf("expected path leads to dir, got %s", err.Error())
		}
	})

	t.Run("Wrong extension", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir + "config.txt")
		err := os.WriteFile(filePath, []byte(
			`workers_num = 5
            admin_id = 123456789
            logs_dir = "/var/log/arbitrage"

            [logger]
            max_size = 10
            max_age = 28
            max_backups = 4
            compress = true`), 0o644)
		if err != nil {
			t.Fatalf("expected write on file, got %s", err.Error())
		}

		cfg, err := config.GetConfig(filePath)
		if cfg != nil {
			t.Fatalf("expected config == nil, got %s", err.Error())
		}
	})

	t.Run("Path is not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "directory/config.toml")

		_, err := config.GetConfig(path)
		if !strings.Contains(err.Error(), "path is not exist") {
			t.Errorf("expected path leads to dir, got %s", err.Error())
		}
	})

	t.Run("Not absolute path", func(t *testing.T) {
		notAbsPath := "./config.toml"
		_, err := config.GetConfig(notAbsPath)
		if !strings.Contains(err.Error(), "not absolute path") {
			t.Errorf("expected not abs path, got %s", err.Error())
		}
	})

	t.Run("Invalid toml syntax", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "config.toml")
		err := os.WriteFile(filePath, []byte(
			`workers_num = 5
            admin_id = 123456789
            logs_dir  "/var/log/arbitrage"

            [logger
            max_size = 10
            max_age = 28
            max_backups = 4
            compress = true`), 0o644)
		if err != nil {
			t.Fatalf("expected write on file, got %s", err.Error())
		}

		_, err = config.GetConfig(filePath)
		if !strings.Contains(err.Error(), "reading toml doc") {
			t.Fatalf("expected reading toml doc, got %s", err.Error())
		}
	})
}
