package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
)

func main() {
	log.SetFlags(0) // log.LstdFlags | log.Lshortfile | log.Ltime)
	log.SetOutput(os.Stdout)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		<-ch
		cancel()
	}()

	if err := runMain(ctx); err != nil {
		log.Fatal("Error: ", err)
	}
}

func runMain(ctx context.Context) (err error) {
	var (
		verbose    bool
		dbFilename string
	)

	flag.BoolVar(&verbose, "v", false, "verbose mode")
	flag.BoolVar(&verbose, "verbose", false, "verbose mode")
	flag.StringVar(&dbFilename, "db", "./synccalendar.db", "database file")
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "%s syncs two or more calendars with another calendar\n", os.Args[0])
		fmt.Fprintln(w)
		fmt.Fprint(w, "Global defaults:\n")
		flag.PrintDefaults()
		fmt.Fprintln(w)
		fmt.Fprint(w, "Commands:")
		fmt.Fprintf(w, "  %-4s    %s\n", SyncCommand.Name, SyncCommand.Description)
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Use \"%s <command> --help\" for more information about a given command.", os.Args[0])
		fmt.Fprintln(w)
	}
	flag.Parse()

	switch flag.Arg(0) {
	case "":
		flag.Usage()
		os.Exit(2)

	case SyncCommand.Name:
		err = SyncCommand.Run(ctx, dbFilename, verbose, flag.Args()[1:])

	case CalendarCommand.Name:
		err = CalendarCommand.Run(ctx, flag.Args()[1:])

	default:
		fmt.Fprintf(os.Stderr, "unknown command %q", flag.Arg(0))
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}

	return err
}
