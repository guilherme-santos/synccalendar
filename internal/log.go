package internal

import (
	"fmt"
	"io"
	"strings"
)

func Logf(w io.Writer, prefix string, cal *Calendar, format string, a ...any) {
	parts := []string{}
	if prefix != "" {
		parts = append(parts, prefix)
	}
	if cal != nil {
		parts = append(parts, fmt.Sprintf("Calendar %s:", cal))
	}
	parts = append(parts, fmt.Sprintf(format, a...))
	fmt.Fprintln(w, strings.Join(parts, " "))
}
