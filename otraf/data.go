package otraf

import "time"

// Otraf holds bus card data
type Otraf struct {
	Name        string    `json:"name"`
	CardStart   time.Time `json:"cardstart"`
	CardEnd     time.Time `json:"cardend"`
	Amount      int       `json:"amount"`
	Updated     time.Time `json:"updated"`
	CardUpdated time.Time `json:"card_updated"`
}

// New returns a new Opac
func New(name string) *Otraf {
	o := &Otraf{}
	o.Name = name
	o.Updated = time.Now()

	return o
}
