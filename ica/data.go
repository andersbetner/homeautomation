package ica

import "time"

// Ica holds data
type Ica struct {
	Balance      float64
	Available    float64
	Transactions []IcaTransaction
}

type IcaTransaction struct {
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
