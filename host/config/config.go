package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DeviceName string `yaml:"device_name" json:"device_name"`
	DeviceAddr string `yaml:"device_addr" json:"device_addr"`
	OutputPath string `yaml:"output_path" json:"output_path"`
	Verbose    bool   `yaml:"verbose" json:"verbose"`
}

func Default() Config {
	home, _ := os.UserHomeDir()
	return Config{
		DeviceName: "Corne Keyboard",
		OutputPath: filepath.Join(home, ".cache", "corne-layer.json"),
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		path = Path()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func Path() string {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "corned", "config.yaml")
}

func CachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "corne-layer.json")
}
