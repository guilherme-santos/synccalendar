# Build image
FROM golang:1.18-alpine AS builder

RUN apk update \
    && apk upgrade \
    && apk add --update \
    ca-certificates \
    gcc \
    git \
    libc-dev \
    make \
    && update-ca-certificates

WORKDIR ${GOPATH}/src/github.com/guilherme-santos/synccalendar

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 go install -v -ldflags "-s -w" -a -installsuffix cgo ./cmd/synccalendar

# Final image
FROM alpine:3.11

LABEL maintainer="Guilherme Santos <xguiga@gmail.com>"

# set up nsswitch.conf for Go's "netgo" implementation
RUN [ ! -e /etc/nsswitch.conf ] && echo 'hosts: files dns' > /etc/nsswitch.conf

COPY --from=builder /go/bin/synccalendar /usr/bin/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

ENTRYPOINT [ "synccalendar" ]

CMD [ "-config", "/config.yml", "-google-cred", "/credentials.json" ]
