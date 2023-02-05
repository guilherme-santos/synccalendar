package main

// func configure(cfgStorage synccalendar.ConfigStorage, mux synccalendar.Mux) {
// 	providers := mux.Providers()

// 	fmt.Fprintln(os.Stdout, "Let's configure your calendars")
// 	fmt.Fprintln(os.Stdout, "\nCalendar destination")

// 	var cfg synccalendar.Config

// 	configurePlatform(&cfg.DestinationAccount.Platform, "platform", providers)
// 	configureField(&cfg.DestinationAccount.Name, "Account Name (your e-mail)")

// 	calAPI, err := mux.Get(cfg.DestinationAccount.Platform)
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Unable to communicate with platform: %v\n", err)
// 		os.Exit(1)
// 	}

// 	ctx := context.Background()
// 	auth, err := calAPI.Login(ctx)
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Unable to authenticate with platform: %v\n", err)
// 		os.Exit(1)
// 	}
// 	cfg.DestinationAccount.Auth = string(auth)

// 	for i := 0; ; i++ {
// 		if i > 0 {
// 			var newCal bool

// 			configureField(&newCal, "New calendar source? (true/false)")
// 			if !newCal {
// 				break
// 			}
// 		}

// 		fmt.Fprintln(os.Stdout, "")
// 		fmt.Fprintf(os.Stdout, "Calendar source #%d\n", i+1)

// 		var cal synccalendar.Calendar

// 		configurePlatform(&cal.Account.Platform, "platform", providers)
// 		configureField(&cal.Account.Name, "Account Name (your e-mail)")

// 		calAPI, err := mux.Get(cal.Account.Platform)
// 		if err != nil {
// 			fmt.Fprintf(os.Stderr, "Unable to communicate with platform: %v\n", err)
// 			os.Exit(1)
// 		}

// 		auth, err := calAPI.Login(ctx)
// 		if err != nil {
// 			fmt.Fprintf(os.Stderr, "Unable to authenticate with platform: %v\n", err)
// 			os.Exit(1)
// 		}
// 		cal.Account.Auth = string(auth)

// 		configureField(&cal.ID, "Calendar ID (empty for primary)")
// 		if cal.ID == "" {
// 			cal.ID = "primary"
// 		}
// 		fmt.Fprintln(os.Stdout, `IMPORTANT: For the Destination Calendar ID, if you use "primary" all your events will be deleted`)
// 		configureField(&cal.DstCalendarID, "Calendar ID on the destination account")
// 		configureField(&cal.DstPrefix, `Event prefix (e.g. "[MyCompany] ")`)
// 		if cal.DstPrefix != "" {
// 			cal.DstPrefix += " "
// 		}

// 		cfg.Calendars = append(cfg.Calendars, &cal)
// 	}

// 	cfgStorage.Set(&cfg)
// 	err = cfgStorage.Flush()
// 	if err != nil {
// 		fmt.Fprintln(os.Stderr, "Unable to save config:", err)
// 		os.Exit(1)
// 	}
// 	fmt.Fprintln(os.Stdout, "Config saved!")
// }

// func configurePlatform(a *string, field string, providers []string) {
// 	configureField(a, fmt.Sprintf("Platform (%v)", strings.Join(providers, ",")))

// 	for _, p := range providers {
// 		if a != nil && *a == p {
// 			return
// 		}
// 	}
// 	configurePlatform(a, field, providers)
// }

// func configureField(a any, label string) {
// 	fmt.Fprintf(os.Stdout, "%s: ", label)
// 	if _, err := fmt.Scan(a); err != nil {
// 		fmt.Fprintf(os.Stderr, "Unable to read field: %v\n", err)
// 		os.Exit(1)
// 	}
// }
