package file

import (
	"context"

	"github.com/guilherme-santos/synccalendar"
)

type Config struct{}

func NewConfig() *Config {
	return &Config{}
}

func (c Config) Sources(context.Context) ([]*synccalendar.Calendar, error) {
	cals := []*synccalendar.Calendar{}

	return cals, nil
}

func (c Config) Destination(context.Context) (*synccalendar.Calendar, error) {
	return &synccalendar.Calendar{
		Platform: "google",
		Owner:    "guilherme@giox.com.br",
	}, nil
}
