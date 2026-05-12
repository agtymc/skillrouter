package skillrouter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ConfigStore interface {
	Path() (string, error)
	Load(path string) (Config, error)
	Save(path string, cfg Config) error
}

type FileConfigStore struct{}

func (FileConfigStore) Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".skillrouter")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func (FileConfigStore) Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (FileConfigStore) Save(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func AddOrUpdatePreset(cfg *Config, name, path string) {
	for i := range cfg.Presets {
		if strings.EqualFold(cfg.Presets[i].Name, name) {
			cfg.Presets[i].Path = path
			return
		}
	}

	cfg.Presets = append(cfg.Presets, Preset{Name: name, Path: path})
	sort.Slice(cfg.Presets, func(i, j int) bool {
		return strings.ToLower(cfg.Presets[i].Name) < strings.ToLower(cfg.Presets[j].Name)
	})
}

func DeletePresetByName(cfg *Config, name string) bool {
	for i := range cfg.Presets {
		if strings.EqualFold(cfg.Presets[i].Name, name) {
			cfg.Presets = append(cfg.Presets[:i], cfg.Presets[i+1:]...)
			return true
		}
	}
	return false
}

func EnsureDir(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("path does not exist: %s", path)
		}
		return err
	}
	if !st.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}
	return nil
}
