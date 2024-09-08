GO?=go
MAKE?=make
sqlitedb?=synccalendar.db

.PHONY: build install

default: build

build:
	$(GO) build ./cmd/synccalendar

install:
	$(GO) install ./cmd/synccalendar

sqlite:
	sqlite3 -column -header $(sqlitedb)

accounts:
	@sqlite3 -list $(sqlitedb) "SELECT id FROM accounts;"

calendars:
	@sqlite3 -header -box $(sqlitedb) "SELECT * FROM calendars WHERE dst_calendar_id IS NOT NULL;"

destinations:
	@sqlite3 -header -box $(sqlitedb) "SELECT account_id, name, provider_id FROM calendars WHERE dst_calendar_id IS NULL;"
