package ica

// Ica holds data
type Ica struct {
	Accounts []IcaAccount
	JSON     string
}

type IcaAccount struct {
	AvailableAmount float64
}

// New returns a new Opac
func New() *Ica {
	ica := &Ica{}

	return ica
}
