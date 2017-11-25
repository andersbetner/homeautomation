package ica

// Ica holds data
type Ica struct {
	AvailableAmount float64
	JSON            string
}

// New returns a new Opac
func New() *Ica {
	ica := &Ica{}

	return ica
}
