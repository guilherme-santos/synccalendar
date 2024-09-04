package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"github.com/guilherme-santos/synccalendar/calendar"
	"github.com/guilherme-santos/synccalendar/calendar/google"
	"github.com/guilherme-santos/synccalendar/internal"
	"github.com/guilherme-santos/synccalendar/internal/sqlite"
	"github.com/guilherme-santos/synccalendar/internal/syncer"
)

var SyncCommand = _syncCommand{
	Name:        "sync",
	Description: "Sync calendars previosly configured",
}

type _syncCommand struct {
	Name        string
	Description string
}

func (s _syncCommand) Run(ctx context.Context, dbFilename string, verbose bool, args []string) error {
	db, err := sql.Open(sqlite.DriverName, dbFilename)
	if err != nil {
		return err
	}

	storage := sqlite.NewStorage(db)
	mux, err := newMux(verbose)
	if err != nil {
		return err
	}

	syncer := syncer.New(flag.CommandLine.Output(), mux, storage)

	var (
		force     bool
		forceFrom internal.Date
		calIDs    Strings
	)

	fs := flag.NewFlagSet(s.Name, flag.ExitOnError)
	fs.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "Usage of %s %s:\n", os.Args[0], fs.Name())
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Options:\n")
		fs.PrintDefaults()
	}
	fs.BoolVar(&force, "force", false, "delete all events and insert then again")
	fs.Var(&forceFrom, "force-from", "force events since the date (e.g. 2022-08-12)")
	fs.Var(&calIDs, "calendar-id", "calendar-id to be synced")
	fs.BoolVar(&syncer.IgnoreDeclinedEvents, "ignore-declined-events", false, "ignore events that were declined")
	fs.BoolVar(&syncer.IgnoreMyEventsAlone, "ignore-my-events-alone", false, "ignore events that I'm alone")
	fs.BoolVar(&syncer.IgnoreOutOfOfficeEvent, "ignore-out-of-office-alone", false, "ignore out of office events")
	fs.BoolVar(&syncer.IgnoreFocusTimeEvent, "ignore-focus-time-alone", false, "ignore focus time events")

	if err := fs.Parse(args); err != nil {
		return err
	}
	return syncer.Sync(ctx, calIDs, force, forceFrom)
}

func newMux(verbose bool) (internal.Mux, error) {
	googleCal, err := google.NewClient(nil)
	if err != nil {
		return nil, err
	}
	googleCal.Verbose = verbose

	mux := calendar.NewMux()
	mux.Register(googleProvider, googleCal)
	return mux, nil
}
