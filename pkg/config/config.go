package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version             string               `yaml:"version"`
	Source              SourceConfig         `yaml:"source"`
	Target              TargetConfig         `yaml:"target"`
	Options             Options              `yaml:"options"`
	Root                Root                 `yaml:"root"`
	ImplicitForeignKeys []ImplicitForeignKey `yaml:"implicit_foreign_keys"`
	Transformations     map[string][]Rule    `yaml:"transformations"`
}

type SourceConfig struct {
	Driver string `yaml:"driver"`
	URL    string `yaml:"url"`
}

type TargetConfig struct {
	Driver string `yaml:"driver"`
	URL    string `yaml:"url"`
}

type Options struct {
	Workers   int `yaml:"workers"`
	BatchSize int `yaml:"batch_size"`
	MaxDepth  int `yaml:"max_depth"`
	Limit     int `yaml:"limit"` // Global default row limit per entity for subsetting
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

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return &cfg, nil
}
