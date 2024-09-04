package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/guilherme-santos/synccalendar/internal"
	"github.com/jmoiron/sqlx"
)

const DriverName = "sqlite3"

type Storage struct {
	db *sqlx.DB
}

func NewStorage(db *sql.DB) *Storage {
	s := &Storage{
		db: sqlx.NewDb(db, DriverName),
	}
	err := s.RunMigrations()
	if err != nil {
		panic(fmt.Sprintf("sqlite: running migrations: %v", err))
	}
	return s
}

func (s Storage) AddAccount(ctx context.Context, account *internal.Account) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (id, auth) VALUES (?, ?)
		ON CONFLICT(id) DO UPDATE SET auth=?;
	`, account.ID(), account.Auth, account.Auth)
	return err
}

func (s Storage) LinkCalendar(ctx context.Context, src, dst *internal.Calendar) error {
	dstCalendarID := dst.Account.ID() + "/" + dst.Name

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO calendars (account_id, name, provider_id, dst_calendar_id)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(account_id, name) DO UPDATE
			SET dst_calendar_id = ?;
	`, src.Account.ID(), src.Name, src.ProviderID, dstCalendarID, dstCalendarID)
	if err != nil {
		return fmt.Errorf("source calendar: %v", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO calendars (account_id, name, provider_id)
		VALUES (?, ?, ?)
		ON CONFLICT(account_id, name) DO UPDATE
			SET provider_id = ?;
	`, dst.Account.ID(), dst.Name, dst.ProviderID, dst.ProviderID)
	if err != nil {
		return fmt.Errorf("destination calendar: %v", err)
	}
	return tx.Commit()
}

func (s Storage) DestinationCalendars(ctx context.Context, calIDs []string) ([]*internal.Calendar, error) {
	orWhere := []string{}
	var args []interface{}
	if len(calIDs) > 0 {
		for _, id := range calIDs {
			orWhere = append(orWhere, `c.account_id || "/" || c.name = ?`)
			args = append(args, id)
		}
	}
	if len(orWhere) == 0 {
		orWhere = append(orWhere, "1 = 1")
	}

	var cals []Calendar

	err := s.db.SelectContext(ctx, &cals, `
		SELECT c.account_id, c.name, c.provider_id, a.auth
		FROM calendars c
		INNER JOIN accounts a ON a.id = c.account_id
		WHERE dst_calendar_id IS NULL
			AND `+strings.Join(orWhere, " OR "), args...)
	if err != nil {
		return nil, err
	}

	res := make([]*internal.Calendar, len(cals))
	for i, c := range cals {
		res[i] = c.Convert()
	}
	return res, nil
}

func (s Storage) SourceCalendars(ctx context.Context, dstCalID string) ([]*internal.Calendar, error) {
	var cals []Calendar

	err := s.db.SelectContext(ctx, &cals, `
		SELECT c.account_id, c.name, c.provider_id, c.last_sync, a.auth
		FROM calendars c
		INNER JOIN accounts a ON a.id = c.account_id
		WHERE dst_calendar_id  = ?
	`, dstCalID)
	if err != nil {
		return nil, err
	}

	res := make([]*internal.Calendar, len(cals))
	for i, c := range cals {
		res[i] = c.Convert()
	}
	return res, nil
}

func (s Storage) DestinationEventID(ctx context.Context, cal *internal.Calendar, srcEventID string) (string, error) {
	var providerID string
	err := s.db.GetContext(ctx, &providerID, `
		SELECT provider_id
		FROM events
		WHERE calendar_id = ? AND src_provider_id = ?
	`, cal.ID, srcEventID)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return providerID, err
}

func (s Storage) CreateEvent(ctx context.Context, cal *internal.Calendar, dstEventID, srcEventID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO events (calendar_id, provider_id, src_provider_id)
		VALUES (?, ?, ?)
	`, cal.ID, dstEventID, srcEventID)
	return err
}

func (s Storage) DeleteEvent(ctx context.Context, cal *internal.Calendar, eventID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM events WHERE calendar_id = ? AND provider_id = ?
	`, cal.ID, eventID)
	return err
}

func (s Storage) SaveLastSync(ctx context.Context, cal *internal.Calendar, lastSync string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE calendars SET last_sync = ? WHERE account_id = ? AND name = ?
	`, lastSync, cal.Account.ID(), cal.Name)
	return err
}
