GO?=go

.PHONY: build install

default: build

build:
	$(GO) build ./cmd/synccalendar

install:
	$(GO) install ./cmd/synccalendar
