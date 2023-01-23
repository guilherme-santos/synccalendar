package google

import (
	"time"

	"github.com/guilherme-santos/synccalendar"

	"google.golang.org/api/calendar/v3"
)

type eventOrError struct {
	e   *synccalendar.Event
	err error
}

type eventIterator struct {
	events  chan eventOrError
	current eventOrError
}

func (it *eventIterator) Next() (ok bool) {
	it.current, ok = <-it.events
	if it.current.err != nil {
		return false
	}
	return ok
}

func (it *eventIterator) Event() *synccalendar.Event {
	c := it.current
	if c.e == nil && c.err == nil {
		panic("google: Event() called before Next()")
	}
	return c.e
}

func (it *eventIterator) Err() error {
	return it.current.err
}

func newEvent(event *calendar.Event) *synccalendar.Event {
	if event.Status == "cancelled" {
		return &synccalendar.Event{
			ID:             event.Id,
			ResponseStatus: synccalendar.Cancelled,
		}
	}

	var responseStatus synccalendar.ResponseStatus
	for _, attendees := range event.Attendees {
		if attendees.Self {
			responseStatus = synccalendar.ResponseStatus(attendees.ResponseStatus)
		}
	}

	startsAt, _ := time.Parse(time.RFC3339, event.Start.DateTime)
	endsAt, _ := time.Parse(time.RFC3339, event.End.DateTime)
	return &synccalendar.Event{
		ID:             event.Id,
		Type:           synccalendar.EventType(event.EventType),
		Summary:        event.Summary,
		Description:    event.Description,
		StartsAt:       startsAt,
		EndsAt:         endsAt,
		CreatedBy:      event.Creator.Email,
		CreatedByMe:    event.Creator.Self,
		ResponseStatus: responseStatus,
		NumAttendees:   len(event.Attendees),
	}
}

func newGoogleEvent(prefix string, event *synccalendar.Event) *calendar.Event {
	return &calendar.Event{
		EventType:   event.Type.String(),
		Summary:     prefix + event.Summary,
		Description: event.Description,
		Start: &calendar.EventDateTime{
			DateTime: event.StartsAt.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: event.EndsAt.Format(time.RFC3339),
		},
		Reminders: &calendar.EventReminders{
			UseDefault: true,
		},
	}
}
