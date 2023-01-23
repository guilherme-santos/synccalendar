package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"

	"github.com/guilherme-santos/synccalendar"
	"github.com/guilherme-santos/synccalendar/calendar"
	"github.com/guilherme-santos/synccalendar/calendar/google"
	"github.com/guilherme-santos/synccalendar/file"
)

var cfg Config

func init() {
	flag.StringVar(&cfg.ConfigFile, "config", "./config.yml", "config file to be used")
	flag.StringVar(&cfg.Google.CredentialsFile, "google-cred", "credentials.json", "credentials file for google")
	flag.Var(&cfg.SyncFrom, "from", "events since (e.g. 2022-08-12)")
	flag.BoolVar(&cfg.Force, "force", false, "force update")
	flag.BoolVar(&cfg.IgnoreDeclinedEvents, "ignore-declined-events", false, "ignore events that were declined")
	flag.BoolVar(&cfg.IgnoreMyEventsAlone, "ignore-my-events-alone", false, "ignore events that I'm alone")
	flag.BoolVar(&cfg.IgnoreOutOfOfficeEvent, "ignore-out-of-office-alone", false, "ignore out of office events")
	flag.BoolVar(&cfg.IgnoreFocusTimeEvent, "ignore-focus-time-alone", false, "ignore focus time events")
	flag.BoolVar(&cfg.Clockwise.SyncFocusTime, "clockwise-sync-focus-time", false, "sync clockwise focus time")
	flag.BoolVar(&cfg.Clockwise.SyncLunch, "clockwise-sync-lunch", false, "sync clockwise lunch")
}

func main() {
	flag.Parse()

	cfgStorage, err := file.LoadConfig(cfg.ConfigFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to read config file:", err)
		os.Exit(1)
	}

	credFile, err := ioutil.ReadFile(cfg.Google.CredentialsFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to read credentials file:", err)
		os.Exit(1)
	}

	googleCal, err := google.NewClient(credFile, cfgStorage)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to create google client:", err)
		os.Exit(1)
	}

	mux := calendar.NewMux()
	mux.Register("google", googleCal)

	if flag.Arg(0) == "configure" {
		configure(cfgStorage, mux)
		return
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		<-ch
		cancel()
	}()

	syncFrom := cfg.SyncFrom
	if syncFrom.IsZero() {
		syncFrom = synccalendar.Today().AddDate(0, 0, -7)
	}

	syncer := synccalendar.NewSyncer(cfgStorage, mux)
	syncer.IgnoreDeclinedEvents = cfg.IgnoreDeclinedEvents
	syncer.IgnoreMyEventsAlone = cfg.IgnoreMyEventsAlone
	syncer.IgnoreOutOfOfficeEvent = cfg.IgnoreOutOfOfficeEvent
	syncer.IgnoreFocusTimeEvent = cfg.IgnoreFocusTimeEvent
	syncer.Clockwise.SyncFocusTime = cfg.Clockwise.SyncFocusTime
	syncer.Clockwise.SyncLunch = cfg.Clockwise.SyncLunch

	err = syncer.Sync(ctx, syncFrom, cfg.Force)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Sync failed:", err)
	}
	// flush even though we have an error
	err = cfgStorage.Flush()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to save config:", err)
		os.Exit(1)
	}
}

func configure(cfgStorage synccalendar.ConfigStorage, mux synccalendar.Mux) {
	providers := mux.Providers()

	fmt.Fprintln(os.Stdout, "Let's configure your calendars")
	fmt.Fprintln(os.Stdout, "\nCalendar destination")

	var cfg synccalendar.Config

	configurePlatform(&cfg.DestinationAccount.Platform, "platform", providers)
	configureField(&cfg.DestinationAccount.Name, "Account Name (your e-mail)")

	calAPI, err := mux.Get(cfg.DestinationAccount.Platform)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to communicate with platform: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	auth, err := calAPI.Login(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to authenticate with platform: %v\n", err)
		os.Exit(1)
	}
	cfg.DestinationAccount.Auth = string(auth)

	for i := 0; ; i++ {
		if i > 0 {
			var newCal bool

			configureField(&newCal, "New calendar source? (true/false)")
			if !newCal {
				break
			}
		}

		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintf(os.Stdout, "Calendar source #%d\n", i+1)

		var cal synccalendar.Calendar

		configurePlatform(&cal.Account.Platform, "platform", providers)
		configureField(&cal.Account.Name, "Account Name (your e-mail)")

		calAPI, err := mux.Get(cal.Account.Platform)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to communicate with platform: %v\n", err)
			os.Exit(1)
		}

		auth, err := calAPI.Login(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to authenticate with platform: %v\n", err)
			os.Exit(1)
		}
		cal.Account.Auth = string(auth)

		configureField(&cal.ID, "Calendar ID (empty for primary)")
		if cal.ID == "" {
			cal.ID = "primary"
		}
		fmt.Fprintln(os.Stdout, `IMPORTANT: For the Destination Calendar ID, if you use "primary" all your events will be deleted`)
		configureField(&cal.DstCalendarID, "Calendar ID on the destination account")
		configureField(&cal.DstPrefix, `Event prefix (e.g. "[MyCompany] ")`)
		if cal.DstPrefix != "" {
			cal.DstPrefix += " "
		}

		cfg.Calendars = append(cfg.Calendars, &cal)
	}

	cfgStorage.Set(&cfg)
	err = cfgStorage.Flush()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to save config:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "Config saved!")
}

func configurePlatform(a *string, field string, providers []string) {
	configureField(a, fmt.Sprintf("Platform (%v)", strings.Join(providers, ",")))

	for _, p := range providers {
		if a != nil && *a == p {
			return
		}
	}
	configurePlatform(a, field, providers)
}

func configureField(a any, label string) {
	fmt.Fprintf(os.Stdout, "%s: ", label)
	if _, err := fmt.Scan(a); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read field: %v\n", err)
		os.Exit(1)
	}
}

type Config struct {
	ConfigFile string
	Google     struct {
		CredentialsFile string
	}
	SyncFrom               synccalendar.Date
	Force                  bool
	IgnoreDeclinedEvents   bool
	IgnoreMyEventsAlone    bool
	IgnoreOutOfOfficeEvent bool
	IgnoreFocusTimeEvent   bool
	Clockwise              struct {
		SyncFocusTime bool
		SyncLunch     bool
	}
}
