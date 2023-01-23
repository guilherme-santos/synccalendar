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
	"google.golang.org/api/googleapi"
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

const defaultSleep = 2 * time.Second

func (c Client) Changes(ctx context.Context, cal *synccalendar.Calendar, from synccalendar.Date) (synccalendar.Iterator, error) {
	it := &eventIterator{}

	svc, err := c.calendarSvc(ctx, cal.Account.Name)
	if err != nil {
		return it, err
	}

	it.events = make(chan eventOrError)
	go c.changes(ctx, cal, svc, from, it.events)

	return it, nil
}

func (c Client) changes(ctx context.Context, cal *synccalendar.Calendar, svc *calendar.Service, from synccalendar.Date, eventCh chan eventOrError) {
	fmt.Fprintf(os.Stdout, "Checking for new events on %s... ", cal)
	defer close(eventCh)

	var changes bool

	nextPageToken := ""
	nextSyncToken := cal.Account.LastSync
	timeMax := time.Now().AddDate(0, 1, 0).Format(time.RFC3339)

	for {
		eventsCall := svc.Events.List(cal.ID).
			Context(ctx).
			ShowDeleted(true).
			SingleEvents(true).
			PageToken(nextPageToken)
		if nextSyncToken != "" {
			eventsCall.SyncToken(nextSyncToken)
		} else {
			eventsCall.TimeMax(timeMax)
			if !from.IsZero() {
				eventsCall.TimeMin(from.Format(time.RFC3339))
			}
		}

		events, err := eventsCall.Do()
		if err != nil {
			if shouldRetry(err) {
				time.Sleep(defaultSleep)
				continue
			}
			fmt.Fprintln(os.Stdout, "error!")
			eventCh <- eventOrError{err: err}
			return
		}

		if len(events.Items) > 0 {
			if !changes {
				fmt.Println()
			}
			changes = true
		}
		for _, item := range events.Items {
			eventCh <- eventOrError{e: newEvent(item)}
		}

		nextPageToken = events.NextPageToken
		if nextPageToken == "" {
			c.cfgStorage.Get().SetAccountLastSync(cal.Account.Name, events.NextSyncToken)
			break
		}
	}

	if !changes {
		fmt.Fprintln(os.Stdout, "up to date!")
	}
}

func (c Client) CreateEvent(ctx context.Context, cal *synccalendar.Calendar, prefix string, req *synccalendar.Event) (*synccalendar.Event, error) {
	svc, err := c.calendarSvc(ctx, cal.Account.Name)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(os.Stdout, "Creating event %q on %s... ", req.Summary, cal)

	var res *synccalendar.Event

	for i := 0; i < 3; i++ {
		gevent, err := svc.Events.Insert(cal.ID, newGoogleEvent(prefix, req)).Context(ctx).Do()
		if err == nil {
			res = newEvent(gevent)
			break
		}
		if shouldRetry(err) {
			time.Sleep(defaultSleep)
			continue
		}
		fmt.Fprintln(os.Stdout, "error!")
		return nil, err
	}

	fmt.Fprintln(os.Stdout, "OK!")
	return res, nil
}

func (c Client) UpdateEvent(ctx context.Context, cal *synccalendar.Calendar, prefix string, req *synccalendar.Event) (*synccalendar.Event, error) {
	svc, err := c.calendarSvc(ctx, cal.Account.Name)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(os.Stdout, "Updating event %q on %s... ", req.Summary, cal)

	var res *synccalendar.Event

	for i := 0; i < 3; i++ {
		gevent, err := svc.Events.Update(cal.ID, req.ID, newGoogleEvent(prefix, req)).Context(ctx).Do()
		if err == nil {
			res = newEvent(gevent)
			break
		}
		if shouldRetry(err) {
			time.Sleep(defaultSleep)
			continue
		}
		fmt.Fprintln(os.Stdout, "error!")
		return nil, err
	}

	fmt.Fprintln(os.Stdout, "OK!")
	return res, nil
}

func (c Client) DeleteEvent(ctx context.Context, cal *synccalendar.Calendar, id string) error {
	svc, err := c.calendarSvc(ctx, cal.Account.Name)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Deleting event on %s... ", cal)

	for i := 0; i < 3; i++ {
		err := svc.Events.Delete(cal.ID, id).Context(ctx).Do()
		if err == nil {
			break
		}
		if alreadyDeleted(err) {
			break
		}
		if shouldRetry(err) {
			time.Sleep(defaultSleep)
			continue
		}
		fmt.Fprintln(os.Stdout, "error!")
		return err
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
	cfg := c.cfgStorage.Get()
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

func shouldRetry(err error) bool {
	var gErr *googleapi.Error
	if !errors.As(err, &gErr) {
		return false
	}

	for _, err := range gErr.Errors {
		switch err.Reason {
		case "rateLimitExceeded":
			return true
		}
	}
	return false
}

func alreadyDeleted(err error) bool {
	var gErr *googleapi.Error
	if !errors.As(err, &gErr) {
		return false
	}

	for _, err := range gErr.Errors {
		switch err.Reason {
		case "deleted":
			return true
		}
	}
	return false
}
