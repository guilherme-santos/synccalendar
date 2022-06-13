package synccalendar

import (
	"context"
	"time"
)

const DateFormat = "2006-01-02"

type Account struct {
	Platform string
	Name     string
	Auth     string
	LastSync string `yaml:"last_sync,omitempty"`
}

type Config struct {
	DestinationAccount Account `yaml:"destination_account"`
	Calendars          []*Calendar
}

func (c Config) AccountByName(name string) *Account {
	if c.DestinationAccount.Name == name {
		acc := c.DestinationAccount
		return &acc
	}
	for _, cal := range c.Calendars {
		if cal.Account.Name == name {
			acc := cal.Account
			return &acc
		}
	}
	return nil
}

func (c *Config) SetAccountLastSync(name string, lastSync string) {
	for i, cal := range c.Calendars {
		if cal.Account.Name == name {
			c.Calendars[i].Account.LastSync = lastSync
		}
	}
}

type ConfigStorage interface {
	Read(context.Context) (*Config, error)
	Write(context.Context, *Config) error
}

type Mux interface {
	Get(platform string) (Provider, error)
}

type Provider interface {
	Login(context.Context) ([]byte, error)
	HasNewEvents(_ context.Context, _ *Calendar) (bool, error)
	Events(_ context.Context, _ *Calendar, from, to time.Time) ([]*Event, error)
	DeleteEventsPeriod(_ context.Context, _ *Calendar, from, to time.Time) error
	CreateEvents(_ context.Context, _ *Calendar, prefix string, _ []*Event) error
}

type Calendar struct {
	ID            string
	DstCalendarID string `yaml:"destination_calendar_id"`
	DstPrefix     string `yaml:"destination_prefix"`
	Account       Account
}

type Event struct {
	ID             string
	Type           string
	Summary        string
	Description    string
	StartsAt       time.Time
	EndsAt         time.Time
	CreatedBy      string
	CreatedByMe    bool
	ResponseStatus ResponseStatus
	NumAttendees   int
}

type ResponseStatus string

var (
	NeedsAction ResponseStatus = "needsAction"
	Declined    ResponseStatus = "declined"
	Tentative   ResponseStatus = "tentative"
	Accepted    ResponseStatus = "accepted"
)
