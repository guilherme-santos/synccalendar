package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/guilherme-santos/synccalendar"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Client struct {
	cfgStorage synccalendar.ConfigStorage
	oauthCfg   *oauth2.Config
	svcs       map[string]*calendar.Service // map[account_name]calendar.Service
}

func NewClient(credJSON []byte, cfgStorage synccalendar.ConfigStorage) (*Client, error) {
	oauthCfg, err := google.ConfigFromJSON(credJSON, calendar.CalendarEventsScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	return &Client{
		cfgStorage: cfgStorage,
		oauthCfg:   oauthCfg,
		svcs:       make(map[string]*calendar.Service),
	}, nil
}

func (c Client) calendarSvc(ctx context.Context, accName string) (*calendar.Service, error) {
	svc, ok := c.svcs[accName]
	if ok {
		return svc, nil
	}

	httpClient, err := c.httpClient(ctx, accName)
	if err != nil {
		return nil, err
	}

	svc, err = calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	c.svcs[accName] = svc
	return svc, err
}

func (c Client) HasNewEvents(ctx context.Context, cal *synccalendar.Calendar) (bool, error) {
	acc, err := c.account(ctx, cal.Account.Name)
	if err != nil {
		return false, err
	}

	svc, err := c.calendarSvc(ctx, cal.Account.Name)
	if err != nil {
		return false, err
	}

	fmt.Fprintf(os.Stdout, "Check new events for %s/%s/%s... ", cal.Account.Platform, cal.Account.Name, cal.ID)

	nextPageToken := ""
	nextSyncToken := acc.LastSync
	changes := false

	for {
		events, err := svc.Events.List(cal.ID).
			Context(ctx).
			SingleEvents(true).
			PageToken(nextPageToken).
			SyncToken(nextSyncToken).
			Do()
		if err != nil {
			return false, err
		}

		if len(events.Items) > 0 {
			changes = true
		}

		nextSyncToken = events.NextSyncToken

		if events.NextPageToken == "" {
			cfg, err := c.cfgStorage.Read(ctx)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Unable to read account:", err)
			} else {
				cfg.SetAccountLastSync(cal.Account.Name, events.NextSyncToken)

				err := c.cfgStorage.Write(ctx, cfg)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Unable to save NextSyncToken:", err)
				}
			}
			break
		}
		nextPageToken = events.NextPageToken
	}

	if changes {
		fmt.Fprintln(os.Stdout, "found!")
		return true, nil
	}

	fmt.Fprintln(os.Stdout, "up to date!")
	return false, nil
}

func (c Client) Events(ctx context.Context, cal *synccalendar.Calendar, from, to time.Time) ([]*synccalendar.Event, error) {
	svc, err := c.calendarSvc(ctx, cal.Account.Name)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(os.Stdout, "Getting events from %s/%s/%s... ", cal.Account.Platform, cal.Account.Name, cal.ID)

	var (
		scEvents      []*synccalendar.Event
		nextPageToken string
	)

	for {
		events, err := svc.Events.List(cal.ID).
			Context(ctx).
			ShowDeleted(false).
			SingleEvents(true).
			TimeMin(from.Format(time.RFC3339)).
			TimeMax(to.Format(time.RFC3339)).
			OrderBy("startTime").
			PageToken(nextPageToken).
			Do()
		if err != nil {
			return nil, err
		}

		for _, evt := range events.Items {
			startsAt, _ := time.Parse(time.RFC3339, evt.Start.DateTime)
			endsAt, _ := time.Parse(time.RFC3339, evt.End.DateTime)

			scEvents = append(scEvents, &synccalendar.Event{
				ID:          evt.Id,
				Type:        evt.EventType,
				Summary:     evt.Summary,
				Description: evt.Description,
				StartsAt:    startsAt,
				EndsAt:      endsAt,
			})
		}

		if events.NextPageToken == "" {
			break
		}
		nextPageToken = events.NextPageToken
	}

	fmt.Fprintf(os.Stdout, "%d event(s) found\n", len(scEvents))

	return scEvents, nil
}

func (c Client) DeleteEventsPeriod(ctx context.Context, cal *synccalendar.Calendar, from, to time.Time) error {
	events, err := c.Events(ctx, cal, from, to)
	if err != nil {
		return err
	}

	svc, err := c.calendarSvc(ctx, cal.Account.Name)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Deleting events from %s/%s/%s... ", cal.Account.Platform, cal.Account.Name, cal.ID)

	for _, evt := range events {
		err := svc.Events.Delete(cal.ID, evt.ID).
			Context(ctx).
			Do()
		if err != nil {
			return err
		}
	}

	fmt.Fprintln(os.Stdout, "OK!")

	return nil
}

func (c Client) CreateEvents(ctx context.Context, cal *synccalendar.Calendar, prefix string, events []*synccalendar.Event) error {
	svc, err := c.calendarSvc(ctx, cal.Account.Name)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Creating events on %s/%s/%s... ", cal.Account.Platform, cal.Account.Name, cal.ID)

	for _, evt := range events {
		_, err = svc.Events.Insert(cal.ID, &calendar.Event{
			EventType:   evt.Type,
			Summary:     prefix + evt.Summary,
			Description: evt.Description,
			Start: &calendar.EventDateTime{
				DateTime: evt.StartsAt.Format(time.RFC3339),
			},
			End: &calendar.EventDateTime{
				DateTime: evt.EndsAt.Format(time.RFC3339),
			},
		}).
			Context(ctx).
			Do()
		if err != nil {
			return err
		}
	}

	fmt.Fprintln(os.Stdout, "OK!")

	return nil
}

func (c Client) Login(ctx context.Context) ([]byte, error) {
	state := fmt.Sprintf("synccalendar-%d", time.Now().UTC().Nanosecond())
	authURL := c.oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Fprintf(os.Stdout, "\nGo to the following link in your browser\n%s\n", authURL)

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	var (
		token   *oauth2.Token
		authErr error
	)

	mux.HandleFunc("/synccalendar", func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			go server.Shutdown(ctx)
		}()

		query := req.URL.Query()
		if query.Get("state") != state {
			authErr = errors.New("oauth link is not valid")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		token, authErr = c.oauthCfg.Exchange(context.TODO(), query.Get("code"))
		if authErr != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "Unable to retrieve token:", authErr)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "All good, you can close this window!")
	})

	serverCh := make(chan struct{})
	var svrErr error
	go func() {
		svrErr = server.ListenAndServe()
		close(serverCh)
	}()

	<-serverCh
	if svrErr != nil && svrErr != http.ErrServerClosed {
		return nil, svrErr
	}
	if authErr != nil {
		return nil, authErr
	}
	return json.Marshal(token)
}

func (c Client) account(ctx context.Context, accName string) (*synccalendar.Account, error) {
	cfg, err := c.cfgStorage.Read(ctx)
	if err != nil {
		return nil, err
	}

	acc := cfg.AccountByName(accName)
	if acc == nil {
		return nil, errors.New("account not found")
	}
	return acc, nil
}

func (c Client) httpClient(ctx context.Context, accName string) (*http.Client, error) {
	acc, err := c.account(ctx, accName)
	if err != nil {
		return nil, err
	}

	var tok *oauth2.Token
	err = json.Unmarshal([]byte(acc.Auth), &tok)
	if err != nil {
		return nil, err
	}
	return c.oauthCfg.Client(ctx, tok), nil
}
