package google

import (
	"context"
	"encoding/json"
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
	config *oauth2.Config
	svcs   map[string]*calendar.Service
}

func NewClient(credJSON []byte) (*Client, error) {
	config, err := google.ConfigFromJSON(credJSON, calendar.CalendarEventsScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	return &Client{
		config: config,
		svcs:   make(map[string]*calendar.Service),
	}, nil
}

func (c Client) calendarSvc(ctx context.Context, owner string) (*calendar.Service, error) {
	svc, ok := c.svcs[owner]
	if ok {
		return svc, nil
	}

	httpClient := httpClient(c.config, owner)

	svc, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	c.svcs[owner] = svc
	return svc, err
}

func (c Client) Events(ctx context.Context, cal *synccalendar.Calendar, from, to time.Time) ([]*synccalendar.Event, error) {
	svc, err := c.calendarSvc(ctx, cal.Owner)
	if err != nil {
		return nil, err
	}

	events, err := svc.Events.List(cal.ID).
		Context(ctx).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(from.Format(time.RFC3339)).
		TimeMax(to.Format(time.RFC3339)).
		OrderBy("startTime").
		Do()
	if err != nil {
		return nil, err
	}

	scEvents := make([]*synccalendar.Event, len(events.Items))

	for i, evt := range events.Items {
		startsAt, _ := time.Parse(time.RFC3339, evt.Start.DateTime)
		endsAt, _ := time.Parse(time.RFC3339, evt.End.DateTime)

		scEvents[i] = &synccalendar.Event{
			ID:          evt.Id,
			Type:        evt.EventType,
			Summary:     evt.Summary,
			Description: evt.Description,
			StartsAt:    startsAt,
			EndsAt:      endsAt,
		}
	}

	return scEvents, nil
}

func (c Client) DeleteEventsPeriod(ctx context.Context, cal *synccalendar.Calendar, calID string, from, to time.Time) error {
	events, err := c.Events(ctx, &synccalendar.Calendar{
		Platform: cal.Platform,
		Owner:    cal.Owner,
		ID:       calID,
	}, from, to)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Deleting %d events\n", len(events))

	svc, err := c.calendarSvc(ctx, cal.Owner)
	if err != nil {
		return err
	}

	for _, evt := range events {
		err := svc.Events.Delete(calID, evt.ID).
			Context(ctx).
			Do()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c Client) CreateEvents(ctx context.Context, dstCal, srcCal *synccalendar.Calendar, events []*synccalendar.Event) error {
	svc, err := c.calendarSvc(ctx, dstCal.Owner)
	if err != nil {
		return err
	}

	for _, evt := range events {
		fmt.Fprintf(os.Stdout, "Creating %q event\n", evt.Summary)

		_, err = svc.Events.Insert(srcCal.DstCalendarID, &calendar.Event{
			EventType:   evt.Type,
			Summary:     srcCal.DstPrefix + evt.Summary,
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
	return nil
}

func httpClient(config *oauth2.Config, owner string) *http.Client {
	tokFile := "token-" + owner + ".json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config, owner)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config, owner string) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	fmt.Fprintln(os.Stdout, "Account", owner)
	fmt.Fprintf(os.Stdout, "Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		fmt.Fprintln(os.Stderr, "Unable to read authorization code:", err)
		os.Exit(1)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to retrieve token from web:", err)
		os.Exit(1)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var tok *oauth2.Token
	err = json.NewDecoder(f).Decode(&tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to cache oauth token:", err)
		return
	}
	defer f.Close()

	json.NewEncoder(f).Encode(token)
}
