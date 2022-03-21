package synccalendar

import (
	"context"
	"fmt"
	"time"
)

type Syncer struct {
	cfgStorage ConfigStorage
	mux        Mux
}

func NewSyncer(cfgStorage ConfigStorage, mux Mux) *Syncer {
	return &Syncer{
		cfgStorage: cfgStorage,
		mux:        mux,
	}
}

// sleepBetweenCals is used to avoid rate limit
const sleepBetweenCals = time.Second

func (s Syncer) Sync(ctx context.Context, from, to time.Time) error {
	cfg, err := s.cfgStorage.Read(ctx)
	if err != nil {
		return fmt.Errorf("unable to get configuration: %w", err)
	}

	dstCalAPI, err := s.mux.Get(cfg.DestinationAccount.Platform)
	if err != nil {
		return err
	}

	for _, cal := range cfg.Calendars {
		calAPI, err := s.mux.Get(cal.Account.Platform)
		if err != nil {
			return err
		}

		changes, err := calAPI.HasNewEvents(ctx, cal)
		if err != nil {
			return fmt.Errorf("unable to check if there're new events available for %s/%s/%s: %w", cal.Account.Platform, cal.Account.Name, cal.ID, err)
		}

		if cal.Account.LastSync == "" {
			// this means it was the first sync, it can be really heavy, so let's wait a bit to avoid rate limit
			time.Sleep(sleepBetweenCals)
		}

		if !changes {
			continue
		}

		// Get events from the source
		events, err := calAPI.Events(ctx, cal, from, to)
		if err != nil {
			return fmt.Errorf("unable to get events from %s/%s/%s: %w", cal.Account.Platform, cal.Account.Name, cal.ID, err)
		}

		// cal2 is cal on the destination side
		cal2 := new(Calendar)
		cal2.Account = cfg.DestinationAccount
		cal2.ID = cal.DstCalendarID

		// Clear events form the DstCalendarID on the destination
		err = dstCalAPI.DeleteEventsPeriod(ctx, cal2, time.Time{}, to)
		if err != nil {
			return fmt.Errorf("unable to remove events from %s/%s/%s: %w", cal2.Account.Platform, cal2.Account.Name, cal2.ID, err)
		}

		// Create events
		err = dstCalAPI.CreateEvents(ctx, cal2, cal.DstPrefix, events)
		if err != nil {
			return fmt.Errorf("unable to create events on %s/%s/%s: %w", cal2.Account.Platform, cal2.Account.Name, cal2.ID, err)
		}

		// To avoid rate limit
		time.Sleep(sleepBetweenCals)
	}

	return nil
}
