# Sync Calendar

Sync one or more calendar Google (for now) Calendar, to other Calendar.

For example, all events created in the calendar A and B will show up on calendar C.

## Build

### Standalone

In order to run the binary we need to install two packages on Ubuntu:

```sh
$ apt install -y build-essential libsqlite3-dev
```

After that you can install, like so:

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

### Cron

Add the following in your cron:

```
@hourly flock -x /var/lock/synccalendar ~/go/bin/synccalendar -v -db ~/synccalendar/synccalendar.db sync --ignore-declined-events --ignore-focus-time-alone --ignore-my-events-alone --ignore-out-of-office-alone >> ~/synccalendar/logs/$(date +\%F).log 2>&1
# Delete logs older than a month
0 0 * * * find ~/synccalendar/logs/ -type f -name "*.log" -mtime +30 -delete
```

## SQLite

- `.tables` - List all tables
- `.schema` - List the schema of a table
