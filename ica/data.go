package ica

import "time"

// Ica holds data
type Ica struct {
	Balance      float64
	Available    float64
	Transactions []Transaction
}

// Transaction for ica account
type Transaction struct {
	Date     time.Time
	Location string
	Discount float64
	Amount   float64
}

// New returns a new Ica
func New() *Ica {
	ica := &Ica{}

	return ica
}
