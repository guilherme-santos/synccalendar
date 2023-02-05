package internal

import "time"

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
