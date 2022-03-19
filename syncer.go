package synccalendar

import (
	"context"
	"fmt"
	"time"
)

type Syncer struct {
	cfg Config
	mux Mux
}

func NewSyncer(cfg Config, mux Mux) *Syncer {
	return &Syncer{
		cfg: cfg,
		mux: mux,
	}
}

func (s Syncer) Sync(ctx context.Context, from, to time.Time) error {
	srcCals, err := s.cfg.Sources(ctx)
	if err != nil {
		return fmt.Errorf("unable to get source calendars: %w", err)
	}

	dstCal, err := s.cfg.Destination(ctx)
	if err != nil {
		return fmt.Errorf("unable to get destination calendar: %w", err)
	}

	dstCalAPI, err := s.mux.Get(dstCal.Platform)
	if err != nil {
		return err
	}

	for _, cal := range srcCals {
		calAPI, err := s.mux.Get(cal.Platform)
		if err != nil {
			return err
		}

		events, err := calAPI.Events(ctx, cal, from, to)
		if err != nil {
			return fmt.Errorf("unable to get events from %s/%s: %w", cal.Platform, cal.ID, err)
		}

		err = dstCalAPI.DeleteEventsPeriod(ctx, dstCal, cal.DstCalendarID, time.Time{}, to)
		if err != nil {
			return fmt.Errorf("unable to remove events from %s/%s.%s: %w", cal.Platform, cal.ID, cal.DstCalendarID, err)
		}

		err = dstCalAPI.CreateEvents(ctx, dstCal, cal, events)
		if err != nil {
			return fmt.Errorf("unable to create events on %s/%s.%s: %w", cal.Platform, cal.ID, cal.DstCalendarID, err)
		}
	}

	return nil
}
