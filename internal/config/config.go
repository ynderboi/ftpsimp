package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

type Config struct {
	Root     string `json:"root"`
	Port     int    `json:"port"`
	ReadOnly bool   `json:"readOnly"`
	// Open disables PIN auth (trusted LAN only). Default false = auth on.
	Open bool `json:"open"`
	// PIN is optional fixed PIN; empty means generate a new one each start.
	PIN string `json:"pin,omitempty"`
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ftpsimp", "config.json"), nil
}

func Load() Config {
	cfg := Config{Port: 8080}
	p, err := Path()
	if err != nil {
		return cfg
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return cfg
	}
	data = bytes.TrimPrefix(data, utf8BOM)
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return cfg
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{Port: 8080}
	}
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	return cfg
}

func Save(cfg Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	// Never persist ephemeral empty PIN marker; omit generated runtime PIN.
	out := cfg
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(p, data, 0o644)
}

func DefaultRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Documents"), nil
}
