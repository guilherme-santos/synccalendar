package main

import "context"

var CalendarCommand = _calendarCommand{
	Name:        "sync",
	Description: "Sync calendars previosly configured",
}

type _calendarCommand struct {
	Name        string
	Description string
}

func (s _calendarCommand) Run(ctx context.Context, args []string) error {
	return nil
}
