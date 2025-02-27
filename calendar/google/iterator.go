package google

import (
	"time"

	"github.com/guilherme-santos/synccalendar/internal"
	"google.golang.org/api/calendar/v3"
)

type eventOrError struct {
	e        *internal.Event
	lastSync string
	err      error
}

type eventIterator struct {
	events   chan eventOrError
	current  eventOrError
	lastSync string
}

func newEventIterator() *eventIterator {
	return &eventIterator{
		events: make(chan eventOrError),
	}
}

func (it *eventIterator) Next() (ok bool) {
	it.current, ok = <-it.events
	if it.current.err != nil {
		return false
	}
	if ok && it.current.lastSync != "" {
		it.lastSync = it.current.lastSync
	}
	return ok
}

func (it *eventIterator) Event() *internal.Event {
	c := it.current
	if c.e == nil && c.err == nil {
		panic("google: Event() called before Next()")
	}
	return c.e
}

func (it *eventIterator) LastSync() string {
	return it.lastSync
}

func (it *eventIterator) Err() error {
	return it.current.err
}

const statusCanceled = "cancelled"

func newEvent(event *calendar.Event) *internal.Event {
	if event.Status == statusCanceled {
		return &internal.Event{
			ID:             event.Id,
			ResponseStatus: internal.Cancelled,
		}
	}

	var responseStatus internal.ResponseStatus
	for _, attendees := range event.Attendees {
		if attendees.Self {
			responseStatus = internal.ResponseStatus(attendees.ResponseStatus)
		}
	}

	startsAt, _ := time.Parse(time.RFC3339, event.Start.DateTime)
	endsAt, _ := time.Parse(time.RFC3339, event.End.DateTime)
	return &internal.Event{
		ID:             event.Id,
		Type:           internal.EventType(event.EventType),
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

var eventTypeFromGmail internal.EventType = "fromGmail"

func newGoogleEvent(prefix string, event *internal.Event) *calendar.Event {
	eventType := event.Type
	if event.Type == eventTypeFromGmail {
		eventType = internal.EventTypeDefault
	}
	return &calendar.Event{
		EventType:   eventType.String(),
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
