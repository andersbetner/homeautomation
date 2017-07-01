package util

import "time"

// Otraf holds bus card data
type Otraf struct {
	Name        string    `json:"name"`
	Cardend     time.Time `json:"cardend"`
	Amount      int       `json:"amount"`
	Updated     time.Time `json:"updated"`
	CardUpdated time.Time `json:"card_updated"`
}
