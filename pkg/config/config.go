package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version             string               `yaml:"version"`
	Source              string               `yaml:"source"`
	Target              string               `yaml:"target"`
	Options             Options              `yaml:"options"`
	Root                Root                 `yaml:"root"`
	ImplicitForeignKeys []ImplicitForeignKey `yaml:"implicit_foreign_keys"`
	Transformations     map[string][]Rule    `yaml:"transformations"`
}

type Options struct {
	MaxDepth  int `yaml:"max_depth"`
	BatchSize int `yaml:"batch_size"`
	Workers   int `yaml:"workers"`
}

type Root struct {
	Table string `yaml:"table"`
	Where string `yaml:"where"`
}

type ImplicitForeignKey struct {
	Table         string `yaml:"table"`
	Column        string `yaml:"column"`
	ForeignTable  string `yaml:"foreign_table"`
	ForeignColumn string `yaml:"foreign_column"`
}

type Rule struct {
	Column string `yaml:"column"`
	Rule   string `yaml:"rule"`
	Salt   string `yaml:"salt,omitempty"`
}

// LoadConfig reads, expands env variables, and unmarshals the YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables like ${STAGING_SNAPSHOT_URL}
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return &cfg, nil
}
