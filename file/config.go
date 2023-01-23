package file

import (
	"os"

	"github.com/guilherme-santos/synccalendar"

	"gopkg.in/yaml.v3"
)

type Config struct {
	filename string
	cfg      *synccalendar.Config
}

func LoadConfig(filename string) (*Config, error) {
	cfg, err := readFile(filename)
	if err != nil {
		return nil, err
	}
	return &Config{
		filename: filename,
		cfg:      cfg,
	}, nil
}

func (c Config) Get() *synccalendar.Config {
	return c.cfg
}

func (c *Config) Set(cfg *synccalendar.Config) {
	c.cfg = cfg
}

func (c Config) Flush() error {
	b, err := yaml.Marshal(c.cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(c.filename, b, 0644)
}

func readFile(filename string) (*synccalendar.Config, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg *synccalendar.Config
	return cfg, yaml.Unmarshal(b, &cfg)
}
