GO?=go

.PHONY: build install

default: build

build:
	$(GO) build ./cmd/synccalendar

install:
	$(GO) install ./cmd/synccalendar

sqlite:
	sqlite3 synccalendar.db ".read ./internal/sqlite/config.sql"
