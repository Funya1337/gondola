package cmd

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Project ProjectConfig `yaml:"project"`
	Build   BuildConfig   `yaml:"build"`
}

type ProjectConfig struct {
	Name string `yaml:"name"`
}

type BuildConfig struct {
	Entry    string   `yaml:"entry"`
	Output   string   `yaml:"output"`
	GOOS     string   `yaml:"goos"`
	GOARCH   string   `yaml:"goarch"`
	LDFlags  string   `yaml:"ldflags"`
	ExtraEnv []string `yaml:"extra_env"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	return &cfg, nil
}
