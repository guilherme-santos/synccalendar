package main

import (
	"context"
	"flag"
	"fmt"

	_ "github.com/mattn/go-sqlite3"

	"github.com/guilherme-santos/synccalendar/calendar/google"
)

var ConfigureCommand = _configureCommand{
	Name:        "configure",
	Description: "Give access to the application",
}

type _configureCommand struct {
	Name        string
	Description string
}

func (s _configureCommand) Run(ctx context.Context, dbFilename string, verbose bool, args []string) error {
	googleCal, err := google.NewClient(nil)
	if err != nil {
		return err
	}
	googleCal.Verbose = verbose

	token, err := googleCal.Login(ctx)
	if err != nil {
		return err
	}

	w := flag.CommandLine.Output()
	fmt.Fprintf(w, "Your token:\n%s", string(token))
	return nil
}
