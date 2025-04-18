package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"

	"github.com/guilherme-santos/synccalendar/calendar/google"
	"github.com/guilherme-santos/synccalendar/internal"
	"github.com/guilherme-santos/synccalendar/internal/sqlite"
)

const (
	googleProvider = "google"
	primaryEmail   = "guilherme@giox.com.br"
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
	db, err := sql.Open(sqlite.DriverName, dbFilename)
	if err != nil {
		return err
	}
	storage := sqlite.NewStorage(db)

	w := flag.CommandLine.Output()

	fmt.Fprintf(w, "Select a calendar provider:\n")
	fmt.Fprintf(w, "1. Google\n")

	var providerChoice int
	fmt.Scanf("%d", &providerChoice)

	var providerName string
	var provider interface {
		Login(ctx context.Context, fn func(string)) (*oauth2.Token, error)
		Email(ctx context.Context, token *oauth2.Token) (string, error)
	}
	switch providerChoice {
	case 1:
		providerName = googleProvider
		googleCal, err := google.NewClient(nil)
		if err != nil {
			return fmt.Errorf("creating Google client: %v", err)
		}
		googleCal.Verbose = verbose
		provider = googleCal
	default:
		return fmt.Errorf("invalid choice: %d", providerChoice)
	}

	authToken, err := provider.Login(ctx, func(authURL string) {
		fmt.Fprintf(w, "Go to the following link in your browser\n%s\n", authURL)
	})
	if err != nil {
		return fmt.Errorf("google: logging in: %v", err)
	}
	userEmail, err := provider.Email(ctx, authToken)
	if err != nil {
		return fmt.Errorf("google: getting email: %v", err)
	}

	acc := internal.Account{
		Platform: providerName,
		Name:     userEmail,
		Auth: func() string {
			v, _ := json.Marshal(authToken)
			return string(v)
		}(),
	}
	fmt.Fprintf(w, "Saving account %q for %q provider...\n", acc.Name, acc.Platform)
	err = storage.AddAccount(ctx, &acc)
	if err != nil {
		return fmt.Errorf("saving account: %v", err)
	}

	destinationCalendar := &internal.Calendar{
		ID: googleProvider + "/" + primaryEmail,
		// Name
		// ProviderID
		Account: internal.Account{
			Platform: googleProvider,
			Name:     primaryEmail,
		},
	}

	fmt.Fprint(w, "Name of the new calendar: ")
	fmt.Scanln(&destinationCalendar.Name)
	fmt.Fprintf(w, "Calendar ID of the Destination on %q: ", googleProvider)
	fmt.Scanln(&destinationCalendar.ProviderID)

	sourceCalendar := &internal.Calendar{
		ID:   acc.ID(),
		Name: destinationCalendar.Name,
		// We only sync with the primary calendar.
		ProviderID: "primary",
		Account:    acc,
	}

	err = storage.LinkCalendar(ctx, sourceCalendar, destinationCalendar)
	if err != nil {
		return fmt.Errorf("linking calendars: %v", err)
	}
	return nil
}
