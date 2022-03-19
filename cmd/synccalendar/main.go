package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"time"

	"github.com/guilherme-santos/synccalendar"
	"github.com/guilherme-santos/synccalendar/calendar"
	"github.com/guilherme-santos/synccalendar/calendar/google"
	"github.com/guilherme-santos/synccalendar/file"
)

var cfg struct {
	Google struct {
		CredentialsFile string
	}
}

func init() {
	flag.StringVar(&cfg.Google.CredentialsFile, "google-cred", "credentials.json", "credentials file for google")
}

func main() {
	flag.Parse()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		<-ch
		cancel()
	}()

	credFile, err := ioutil.ReadFile(cfg.Google.CredentialsFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to read credentials file:", err)
		os.Exit(1)
	}

	googleCal, err := google.NewClient(credFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to create google client:", err)
		os.Exit(1)
	}

	mux := calendar.NewMux()
	mux.Register("google", googleCal)

	cfg := file.NewConfig()
	syncer := synccalendar.NewSyncer(cfg, mux)

	from := time.Now().UTC().AddDate(0, 0, -7)
	to := time.Now().UTC().AddDate(0, 0, 30)

	err = syncer.Sync(ctx, from, to)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Sync failed:", err)
		os.Exit(1)
	}
}
