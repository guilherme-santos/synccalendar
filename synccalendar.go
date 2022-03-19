package synccalendar

import (
	"context"
	"time"
)

type Config interface {
	Sources(context.Context) ([]*Calendar, error)
	Destination(context.Context) (*Calendar, error)
}

type Mux interface {
	Get(platform string) (Storage, error)
}

type Storage interface {
	Events(_ context.Context, _ *Calendar, from, to time.Time) ([]*Event, error)
	DeleteEventsPeriod(_ context.Context, _ *Calendar, calID string, from, to time.Time) error
	CreateEvents(_ context.Context, dst, src *Calendar, _ []*Event) error
}

type Calendar struct {
	Platform      string
	Owner         string
	ID            string
	DstCalendarID string
	DstPrefix     string
}

type Event struct {
	ID          string
	Type        string
	Summary     string
	Description string
	StartsAt    time.Time
	EndsAt      time.Time
}
