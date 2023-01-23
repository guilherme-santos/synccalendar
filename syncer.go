package synccalendar

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gitlab.com/guilherme-santos/golib/xtime"
)

type Syncer struct {
	cfgStorage ConfigStorage
	mux        Mux

	IgnoreDeclinedEvents   bool
	IgnoreMyEventsAlone    bool
	IgnoreOutOfOfficeEvent bool
	IgnoreFocusTimeEvent   bool
	Clockwise              struct {
		SyncFocusTime bool
		SyncLunch     bool
	}
}

func NewSyncer(cfgStorage ConfigStorage, mux Mux) *Syncer {
	return &Syncer{
		cfgStorage: cfgStorage,
		mux:        mux,
	}
}

func (s Syncer) Sync(ctx context.Context, from xtime.Date, force bool) error {
	fmt.Fprintf(os.Stdout, "Syncing calendar from %s...\n", from.String())

	cfg := s.cfgStorage.Get()
	dstCalAPI, err := s.mux.Get(cfg.DestinationAccount.Platform)
	if err != nil {
		return err
	}

	for _, cal := range cfg.Calendars {
		srcCalAPI, err := s.mux.Get(cal.Account.Platform)
		if err != nil {
			return err
		}
		if force {
			cal.Account.LastSync = ""
		}

		it, err := srcCalAPI.Changes(ctx, cal, from)
		if err != nil {
			return fmt.Errorf("unable to check for changes on %s: %w", cal, err)
		}

		for it.Next() {
			event := it.Event()
			ignoreEvent := s.ignoreEvent(event)
			srcEventID := event.ID

			// dstCal is cal on the destination side
			dstCal := new(Calendar)
			dstCal.Account = cfg.DestinationAccount
			dstCal.ID = cal.DstCalendarID

			dstEventID := cfg.EventMapping(cal.Account.Name, srcEventID)
			if dstEventID != "" {
				if event.ResponseStatus == Cancelled || ignoreEvent {
					err := dstCalAPI.DeleteEvent(ctx, dstCal, dstEventID)
					if err != nil {
						return fmt.Errorf("unable to delete event for %s: %w", cal, err)
					}

					cfg.DeleteEventMapping(cal.Account.Name, srcEventID)
				} else {
					event.ID = dstEventID
					_, err := dstCalAPI.UpdateEvent(ctx, dstCal, cal.DstPrefix, event)
					if err != nil {
						return fmt.Errorf("unable to update event for %s: %w", cal, err)
					}
				}
				continue
			}

			if event.ResponseStatus == Cancelled || ignoreEvent {
				// canceled and we do not have the destination id
				continue
			}

			event, err := dstCalAPI.CreateEvent(ctx, dstCal, cal.DstPrefix, event)
			if err != nil {
				return fmt.Errorf("unable to create event for %s: %w", cal, err)
			}

			cfg.AddEventMapping(cal.Account.Name, EventMapping{
				SourceID:      srcEventID,
				DestinationID: event.ID,
			})
		}
		if err := it.Err(); err != nil {
			return fmt.Errorf("unable to fetch event changes for %s: %w", cal, err)
		}
	}

	return nil
}

func (s Syncer) ignoreEvent(e *Event) bool {
	if s.IgnoreDeclinedEvents && e.ResponseStatus == Declined {
		return true
	}
	if s.IgnoreMyEventsAlone && e.CreatedByMe && e.NumAttendees == 0 {
		return true
	}
	if s.IgnoreOutOfOfficeEvent && e.Type == EventTypeOutOfOffice {
		return true
	}
	if s.IgnoreFocusTimeEvent && e.Type == EventTypeFocusTime {
		return true
	}
	if !s.Clockwise.SyncFocusTime && strings.EqualFold(e.Summary, "❇️ Focus Time (via Clockwise)") {
		return true
	}
	if !s.Clockwise.SyncLunch && strings.EqualFold(e.Summary, "❇️ Lunch (via Clockwise)") {
		return true
	}
	return false
}
