package syncer

import (
	"io"
	"time"

	"github.com/guilherme-santos/synccalendar/internal"
)

func relativeDate(d internal.Date) string {
	if !d.IsZero() {
		return d.String()
	}
	return "always"
}

func formatDateTime(d time.Time) string {
	return d.In(time.Local).Format("02 Jan 06 15:04")
}

func logf(w io.Writer, cal *Calendar, format string, a ...any) {
	internal.Logf(w, "", cal, format, a...)
}
