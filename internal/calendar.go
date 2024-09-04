package internal

type Account struct {
	Platform string
	Name     string
	Auth     string
}

func (a Account) ID() string {
	return a.Platform + "/" + a.Name
}

type Calendar struct {
	ID         string
	Name       string
	ProviderID string
	Account    Account
	LastSync   string
}

func (c Calendar) String() string {
	return c.ID
}
