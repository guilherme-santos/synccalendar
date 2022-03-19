package file

import (
	"context"
	"os"

	"github.com/guilherme-santos/synccalendar"

	"gopkg.in/yaml.v3"
)

type Config struct {
	filename string
}

func NewConfig(filename string) *Config {
	return &Config{
		filename: filename,
	}
}

func (c Config) Read(ctx context.Context) (*synccalendar.Config, error) {
	b, err := os.ReadFile(c.filename)
	if err != nil {
		return nil, err
	}

	var cfg *synccalendar.Config
	return cfg, yaml.Unmarshal(b, &cfg)
}

func (c Config) Write(ctx context.Context, cfg *synccalendar.Config) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(c.filename, b, 0644)
}
