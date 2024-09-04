package internal

import (
	"context"

	"golang.org/x/oauth2"
)

type Mux interface {
	Get(platform string) (Provider, error)
}

type Provider interface {
	Login(context.Context, func(authURL string)) (*oauth2.Token, error)
	Events(_ context.Context, _ *Calendar, from Date) (Iterator, error)
	NewEventsFrom(_ context.Context, _ *Calendar, from Date) (Iterator, error)
	NewEventsSince(_ context.Context, _ *Calendar, token string) (Iterator, error)
	CreateEvent(_ context.Context, _ *Calendar, _ *Event) (*Event, error)
	UpdateEvent(_ context.Context, _ *Calendar, _ *Event) error
	DeleteEvent(_ context.Context, _ *Calendar, id string) error
}

type Iterator interface {
	Next() bool
	Event() *Event
	LastSync() string
	Err() error
}
