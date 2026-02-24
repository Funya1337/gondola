package cmd

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Project ProjectConfig `yaml:"project"`
	Build   BuildConfig   `yaml:"build"`
	Test    TestConfig    `yaml:"test"`
	Deploy  DeployConfig  `yaml:"deploy"`
}

type ServiceConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Restart     string `yaml:"restart"`
}

type DeployConfig struct {
	Host       string        `yaml:"host"`
	Port       int           `yaml:"port"`
	User       string        `yaml:"user"`
	KeyPath    string        `yaml:"key_path"`
	RemotePath string        `yaml:"remote_path"`
	Service    ServiceConfig `yaml:"service"`
	PreDeploy  []string      `yaml:"pre_deploy"`
	PostDeploy []string      `yaml:"post_deploy"`
}

type TestConfig struct {
	Commands []string `yaml:"commands"`
	Skip     bool     `yaml:"skip"`
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
