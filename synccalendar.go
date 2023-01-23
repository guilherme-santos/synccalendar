package synccalendar

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/guilherme-santos/golib/xtime"
)

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

func (c Config) calendar(accountName string) *Calendar {
	for i, cal := range c.Calendars {
		if cal.Account.Name == accountName {
			return c.Calendars[i]
		}
	}
	return nil
}

func (c *Config) SetAccountLastSync(accountName string, lastSync string) {
	cal := c.calendar(accountName)
	if cal != nil {
		cal.Account.LastSync = lastSync
	}
}

func (c *Config) AddEventMapping(accountName string, m EventMapping) {
	cal := c.calendar(accountName)
	if cal != nil {
		cal.Events = append(cal.Events, m)
	}
}

func (c *Config) EventMapping(accountName, srcID string) (destID string) {
	cal := c.calendar(accountName)
	if cal != nil {
		for _, m := range cal.Events {
			if srcID == m.SourceID {
				return m.DestinationID
			}
		}
	}
	return
}

func (c *Config) DeleteEventMapping(accountName, srcID string) {
	cal := c.calendar(accountName)
	if cal != nil {
		for i, m := range cal.Events {
			if srcID == m.SourceID {
				cal.Events = append(cal.Events[:i], cal.Events[i+1:]...)
				break
			}
		}
	}
}

type ConfigStorage interface {
	Get() *Config
	Set(*Config)
	Flush() error
}

type Mux interface {
	Get(platform string) (Provider, error)
	Providers() []string
}

type Iterator interface {
	Next() bool
	Event() *Event
	Err() error
}

type Provider interface {
	Login(context.Context) ([]byte, error)
	Changes(_ context.Context, _ *Calendar, from xtime.Date) (Iterator, error)
	CreateEvent(_ context.Context, _ *Calendar, prefix string, _ *Event) (*Event, error)
	UpdateEvent(_ context.Context, _ *Calendar, prefix string, _ *Event) (*Event, error)
	DeleteEvent(_ context.Context, _ *Calendar, id string) error
}

type Calendar struct {
	ID            string
	DstCalendarID string `yaml:"destination_calendar_id"`
	DstPrefix     string `yaml:"destination_prefix"`
	Account       Account
	Events        []EventMapping
}

func (c Calendar) String() string {
	return fmt.Sprintf("%s/%s/%s", c.Account.Platform, c.Account.Name, c.ID)
}

type EventMapping struct {
	SourceID      string `yaml:"source_id"`
	DestinationID string `yaml:"destination_id"`
}

type Event struct {
	ID             string
	Type           EventType
	Summary        string
	Description    string
	StartsAt       time.Time
	EndsAt         time.Time
	CreatedBy      string
	CreatedByMe    bool
	ResponseStatus ResponseStatus
	NumAttendees   int
}

type EventType string

func (s EventType) String() string {
	return string(s)
}

var (
	EventTypeDefault     EventType = "default"
	EventTypeOutOfOffice EventType = "outOfOffice"
	EventTypeFocusTime   EventType = "focusTime"
)

type ResponseStatus string

func (s ResponseStatus) String() string {
	return string(s)
}

var (
	NeedsAction ResponseStatus = "needsAction"
	Declined    ResponseStatus = "declined"
	Tentative   ResponseStatus = "tentative"
	Accepted    ResponseStatus = "accepted"
	Cancelled   ResponseStatus = "cancelled"
)
