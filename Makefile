GO?=go
GOFLAGS=-ldflags "-extldflags '-static -pthread'

.PHONY: build install

default: build

build:
	@GCO_ENABLED=0 $(GO) build -a $(GOFLAGS) ./cmd/synccalendar

install:
	$(GO) install ./cmd/synccalendar
