package internal

type Account struct {
	Platform string
	Name     string
	Auth     string
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
