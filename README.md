# Sync Calendar

Sync one or more calendar Google (for now) Calendar, to other Calendar.

For example, all events created in the calendar A and B will show up on calendar C.

## Build

### Standalone

```sh
$ go install ./cmd/synccalendar
```

The previous command will install the binary in your `$GOPATH/bin`.

### Docker

```sh
$ docker build -t xguiga/synccalendar:latest .
```

## Running

The command line accept two flags `-config` to specify the config file and `-google-cred` to specify the crendentials of your Google app. More details [here](https://developers.google.com/workspace/guides/create-project) and [here](https://developers.google.com/workspace/guides/create-credentials).

To generate the config file you try the following:

```sh
$ synccalendar configure
```

### Standalone

Assuming that your `PATH` is correctly configured and pointing to your `$GOPATH/bin`, you can simply type:

```sh
$ synccalendar
```

### Docker

```sh
$ docker run -it \
    -v `pwd`/config.yml:/config.yml \
    -v `pwd`/credentials.json:/credentials.json \
    xguiga/synccalendar:latest
```
