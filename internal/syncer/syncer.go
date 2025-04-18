package syncer

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/guilherme-santos/synccalendar/internal"
)

var ErrSyncing = errors.New("an error occoured while syncing, check the logs")

type (
	Mux      = internal.Mux
	Calendar = internal.Calendar
	Event    = internal.Event
)

type Storage interface {
	DestinationCalendars(_ context.Context, calIDs []string) ([]*Calendar, error)
	SourceCalendars(_ context.Context, dstCalID string) ([]*Calendar, error)

	DestinationEventID(_ context.Context, _ *Calendar, srcEventID string) (string, error)
	CreateEvent(_ context.Context, _ *Calendar, dstEventID, srcEventID string) error
	DeleteEvent(_ context.Context, _ *Calendar, eventID string) error
	SaveLastSync(_ context.Context, _ *Calendar, lastSync string) error
}

type Syncer struct {
	output  io.Writer
	mux     Mux
	storage Storage

	IgnoreDeclinedEvents   bool
	IgnoreMyEventsAlone    bool
	IgnoreOutOfOfficeEvent bool
	IgnoreFocusTimeEvent   bool
}

func New(output io.Writer, providers Mux, storage Storage) *Syncer {
	if output == nil {
		output = os.Stdout
	}
	return &Syncer{
		output:  output,
		mux:     providers,
		storage: storage,
	}
}

func (s Syncer) Sync(ctx context.Context, calIDs []string, force bool, forceFrom internal.Date) error {
	dstcals, err := s.storage.DestinationCalendars(ctx, calIDs)
	if err != nil {
		return err
	}
	for _, dstcal := range dstcals {
		if err := ctx.Err(); err != nil {
			return err
		}

		if force {
			err := s.DeleteEvents(ctx, dstcal, forceFrom)
			if err != nil {
				return err
			}
		}

		srccals, err := s.storage.SourceCalendars(ctx, dstcal.ID)
		if err != nil {
			return err
		}
		for _, srccal := range srccals {
			if err := ctx.Err(); err != nil {
				return err
			}

			err := s.SyncCalendar(ctx, dstcal, srccal, forceFrom)
			if err != nil && !errors.Is(err, ErrSyncing) {
				return err
			}
		}
	}
	return nil
}

func (s Syncer) DeleteEvents(ctx context.Context, cal *Calendar, from internal.Date) error {
	logf(s.output, cal, "Removing events since: %s", relativeDate(from))

	provider, err := s.mux.Get(cal.Account.Platform)
	if err != nil {
		logf(s.output, cal, "Unable to load provider: %v", err)
		return ErrSyncing
	}

	it, err := provider.Events(ctx, cal, from)
	if err != nil {
		logf(s.output, cal, "Unable to get list of events: %v", err)
		return ErrSyncing
	}
	var (
		eventsDeleted uint64
		foundErr      bool
	)
	for it.Next() {
		err := s.deleteEvent(ctx, provider, cal, it.Event())
		if err != nil {
			foundErr = true
			continue
		}
		eventsDeleted++
	}

	if err := it.Err(); err != nil {
		logf(s.output, cal, "Unable to get list of events: %v", err)
		return ErrSyncing
	}
	if foundErr {
		logf(s.output, cal, "Some events couldn't be deleted, %d deleted succesfully", eventsDeleted)
	} else if eventsDeleted == 0 {
		logf(s.output, cal, "No events found to be deleted")
	} else {
		logf(s.output, cal, "%d event(s) deleted succesfully", eventsDeleted)
	}
	return nil
}

func (s Syncer) SyncCalendar(ctx context.Context, dst, src *Calendar, from internal.Date) error {
	logf(s.output, dst, "Syncing calendar with %s...", src)

	dstProvider, err := s.mux.Get(dst.Account.Platform)
	if err != nil {
		logf(s.output, dst, "Unable to load destination provider: %v", err)
		return ErrSyncing
	}
	srcProvider, err := s.mux.Get(src.Account.Platform)
	if err != nil {
		logf(s.output, dst, "Unable to load source provider: %v", err)
		return ErrSyncing
	}

	var it internal.Iterator
	if !from.IsZero() || src.LastSync == "" {
		it, err = srcProvider.NewEventsFrom(ctx, src, from)
	} else {
		it, err = srcProvider.NewEventsSince(ctx, src, src.LastSync)
	}
	if err != nil {
		logf(s.output, dst, "Unable to get new events from %s: %v", src, err)
		return ErrSyncing
	}
	var foundErr bool
	for it.Next() {
		event := it.Event()
		ignoreEvent := s.ignoreEvent(event)
		srcProviderID := event.ID

		// We don't care about the id from the source, but the id
		// from the destination.
		event.ID, err = s.storage.DestinationEventID(ctx, dst, srcProviderID)
		if err != nil {
			logf(s.output, dst, "Unable to get destination event id %s: %v", event.ID, err)
			return ErrSyncing
		}

		if event.ResponseStatus == internal.Cancelled || ignoreEvent {
			if event.ID != "" {
				s.deleteEvent(ctx, dstProvider, dst, event)
			}
		} else if event.ID == "" {
			err = s.createEvent(ctx, dstProvider, dst, srcProviderID, event)
		} else {
			err = s.updateEvent(ctx, dstProvider, dst, event)
		}
		if err != nil {
			foundErr = true
		}
	}

	if err := it.Err(); err != nil {
		logf(s.output, dst, "Unable to get list of events: %v", err)
		return ErrSyncing
	}
	if foundErr {
		logf(s.output, dst, "Sync complete with error!")
	} else {
		if lastSync := it.LastSync(); lastSync != "" {
			err = s.storage.SaveLastSync(ctx, src, lastSync)
			if err != nil {
				logf(s.output, dst, "Unable to save last sync: %v", err)
			}
		}
		logf(s.output, dst, "Sync complete!")
	}
	return nil
}

func (s Syncer) deleteEvent(ctx context.Context, provider internal.Provider, cal *Calendar, event *Event) error {
	logf(s.output, cal, "Deleting event %s: %q on %s", event.ID, event.Summary, formatDateTime(event.StartsAt))

	err := provider.DeleteEvent(ctx, cal, event.ID)
	if err != nil {
		logf(s.output, cal, "Unable to delete event from provider %s: %v", event.ID, err)
		return err
	}
	err = s.storage.DeleteEvent(ctx, cal, event.ID)
	if err != nil {
		logf(s.output, cal, "Unable to delete event from storage %s: %v", event.ID, err)
		return err
	}
	return nil
}

func (s Syncer) createEvent(ctx context.Context, provider internal.Provider, cal *Calendar, srcProviderID string, event *Event) error {
	logf(s.output, cal, "Creating event: %q on %s", event.Summary, formatDateTime(event.StartsAt))

	newEvent, err := provider.CreateEvent(ctx, cal, event)
	if err != nil {
		logf(s.output, cal, "Unable to create event on the provider: %v", err)
		return err
	}
	logf(s.output, cal, "Map event id %s to %s", srcProviderID, newEvent.ID)

	err = s.storage.CreateEvent(ctx, cal, newEvent.ID, srcProviderID)
	if err != nil {
		logf(s.output, cal, "Unable to create event on the storage: %v", err)

		// Let's try to remove from the provider as we couldn't save on our db
		// this avoid that the next time we duplicate the event in the provider.
		_ = provider.DeleteEvent(ctx, cal, event.ID)
		return err
	}
	return nil
}

func (s Syncer) updateEvent(ctx context.Context, provider internal.Provider, cal *Calendar, event *Event) error {
	logf(s.output, cal, "Updating event %s: %q on %s", event.ID, event.Summary, formatDateTime(event.StartsAt))

	err := provider.UpdateEvent(ctx, cal, event)
	if err != nil {
		logf(s.output, cal, "Unable to update event on the provider %s: %v", event.ID, err)
		return err
	}
	return nil
}

func (s Syncer) ignoreEvent(e *Event) bool {
	if s.IgnoreDeclinedEvents && e.ResponseStatus == internal.Declined {
		return true
	}
	if s.IgnoreMyEventsAlone && e.CreatedByMe && e.NumAttendees == 0 {
		return true
	}
	if s.IgnoreOutOfOfficeEvent && e.Type == internal.EventTypeOutOfOffice {
		return true
	}
	if s.IgnoreFocusTimeEvent && e.Type == internal.EventTypeFocusTime {
		return true
	}
	return false
}
