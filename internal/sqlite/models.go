package sqlite

import (
	"strings"

	"github.com/guilherme-santos/synccalendar/internal"
)

type Calendar struct {
	AccountID   string `db:"account_id"`
	Name        string
	ProviderID  string `db:"provider_id"`
	LastSync    string `db:"last_sync"`
	AccountAuth string `db:"auth"`
}

func (c Calendar) Convert() *internal.Calendar {
	acc := internal.Account{
		Auth: c.AccountAuth,
	}
	acc.Platform, acc.Name, _ = strings.Cut(c.AccountID, "/")
	return &internal.Calendar{
		ID:         c.AccountID + "/" + c.Name,
		Name:       c.Name,
		ProviderID: c.ProviderID,
		Account:    acc,
		LastSync:   c.LastSync,
	}
}
