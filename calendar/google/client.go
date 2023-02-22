package google

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"github.com/guilherme-santos/synccalendar/internal"
)

//go:embed credentials.json
var credentials []byte

type Client struct {
	oauthCfg *oauth2.Config

	Verbose bool
}

func NewClient(credJSON []byte) (*Client, error) {
	if credJSON == nil {
		credJSON = credentials
	}
	oauthCfg, err := google.ConfigFromJSON(credJSON, calendar.CalendarEventsScope)
	if err != nil {
		return nil, fmt.Errorf("google: parsing credentials file: %v", err)
	}

	return &Client{
		oauthCfg: oauthCfg,
	}, nil
}

const defaultSleep = 5 * time.Second

func (c Client) Events(ctx context.Context, cal *internal.Calendar, from internal.Date) (internal.Iterator, error) {
	svc, err := c.calendarSvc(ctx, cal)
	if err != nil {
		return nil, err
	}
	eventsCall := svc.Events.List(cal.ProviderID).Context(ctx).ShowDeleted(false)
	if !from.IsZero() {
		eventsCall.TimeMin(from.Format(time.RFC3339))
	}

	it := newEventIterator()
	go c.events(ctx, svc, cal, eventsCall, it.events)
	return it, nil
}

func (c Client) NewEventsFrom(ctx context.Context, cal *internal.Calendar, from internal.Date) (internal.Iterator, error) {
	svc, err := c.calendarSvc(ctx, cal)
	if err != nil {
		return nil, err
	}
	eventsCall := svc.Events.
		List(cal.ProviderID).
		Context(ctx).
		ShowDeleted(true).
		SingleEvents(true)
	if !from.IsZero() {
		eventsCall = eventsCall.TimeMin(from.Format(time.RFC3339))
	}

	it := newEventIterator()
	go c.events(ctx, svc, cal, eventsCall, it.events)
	return it, nil
}

func (c Client) NewEventsSince(ctx context.Context, cal *internal.Calendar, lastSync string) (internal.Iterator, error) {
	svc, err := c.calendarSvc(ctx, cal)
	if err != nil {
		return nil, err
	}
	eventsCall := svc.Events.
		List(cal.ProviderID).
		Context(ctx).
		ShowDeleted(true).
		SingleEvents(true)
	if lastSync != "" {
		eventsCall = eventsCall.SyncToken(lastSync)
	}

	it := newEventIterator()
	go c.events(ctx, svc, cal, eventsCall, it.events)
	return it, nil
}

func (c Client) events(
	ctx context.Context,
	svc *calendar.Service,
	cal *internal.Calendar,
	call *calendar.EventsListCall,
	eventCh chan eventOrError,
) {
	c.logf(cal, "checking for events")

	defer close(eventCh)

	var (
		nextPageToken string
		hasEvents     bool
	)

	for {
		events, err := call.PageToken(nextPageToken).Do()
		if err != nil {
			if shouldRetry(err) {
				time.Sleep(defaultSleep)
				continue
			}
			c.logf(cal, "unable to get list of events: %v", err)
			eventCh <- eventOrError{err: err}
			return
		}

		if !hasEvents {
			hasEvents = len(events.Items) > 0
		}

		for _, item := range events.Items {
			eventCh <- eventOrError{
				e:        newEvent(item),
				lastSync: events.NextSyncToken,
			}
		}
		nextPageToken = events.NextPageToken
		if nextPageToken == "" {
			break
		}
	}
	if !hasEvents {
		c.logf(cal, "no changes, events are up to date!")
	}
}

func (c Client) CreateEvent(ctx context.Context, cal *internal.Calendar, req *internal.Event) (*internal.Event, error) {
	msg := fmt.Sprintf("creating event: %q on %s... ", req.Summary, req.StartsAt)
	defer func() {
		c.logf(cal, msg)
	}()

	svc, err := c.calendarSvc(ctx, cal)
	if err != nil {
		msg += "❌"
		return nil, err
	}
	prefix := fmt.Sprintf("[%s] ", cal.Name)

	var res *internal.Event
	for {
		gevent, err := svc.Events.Insert(cal.ProviderID, newGoogleEvent(prefix, req)).Context(ctx).Do()
		if err == nil {
			res = newEvent(gevent)
			msg += "✅"
			break
		}
		if shouldRetry(err) {
			time.Sleep(defaultSleep)
			continue
		}
		msg += "❌"
		return nil, err
	}
	return res, nil
}

func (c Client) UpdateEvent(ctx context.Context, cal *internal.Calendar, req *internal.Event) error {
	msg := fmt.Sprintf("updating event: %q on %s... ", req.Summary, req.StartsAt)
	defer func() {
		c.logf(cal, msg)
	}()

	svc, err := c.calendarSvc(ctx, cal)
	if err != nil {
		msg += "❌"
		return err
	}
	prefix := fmt.Sprintf("[%s] ", cal.Name)

	for {
		_, err := svc.Events.Update(cal.ProviderID, req.ID, newGoogleEvent(prefix, req)).Context(ctx).Do()
		if err == nil {
			msg += "✅"
			break
		}
		if shouldRetry(err) {
			time.Sleep(defaultSleep)
			continue
		}
		msg += "❌"
		return err
	}
	return nil
}

func (c Client) DeleteEvent(ctx context.Context, cal *internal.Calendar, id string) error {
	msg := fmt.Sprintf("deleting event %s... ", id)
	defer func() {
		c.logf(cal, msg)
	}()

	svc, err := c.calendarSvc(ctx, cal)
	if err != nil {
		msg += "❌"
		return err
	}
	for {
		err = svc.Events.Delete(cal.ProviderID, id).Context(ctx).Do()
		if err == nil || alreadyDeleted(err) {
			msg += "✅"
			break
		}
		if shouldRetry(err) {
			time.Sleep(defaultSleep)
			continue
		}
		msg += "❌"
		return err
	}
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

func (c Client) calendarSvc(ctx context.Context, cal *internal.Calendar) (*calendar.Service, error) {
	var tok *oauth2.Token
	err := json.Unmarshal([]byte(cal.Account.Auth), &tok)
	if err != nil {
		return nil, err
	}
	httpClient, err := c.oauthCfg.Client(ctx, tok), nil
	if err != nil {
		return nil, err
	}
	return calendar.NewService(ctx, option.WithHTTPClient(httpClient))
}

func (c Client) logf(cal *internal.Calendar, format string, a ...any) {
	if c.Verbose {
		internal.Logf(os.Stdout, "google:", cal, format, a...)
	}
}

func shouldRetry(err error) bool {
	return errIsReason(err, "rateLimitExceeded")
}

func alreadyDeleted(err error) bool {
	return errIsReason(err, "deleted")
}

func errIsReason(err error, reason string) bool {
	var gErr *googleapi.Error
	if !errors.As(err, &gErr) {
		return false
	}

	for _, err := range gErr.Errors {
		switch err.Reason {
		case reason:
			return true
		}
	}
	return false
}
