package opac

import "time"

// Book yada
type Book struct {
	Title       string    `json:"title"`
	DateDue     time.Time `json:"date_due"`
	LibraryName string    `json:"library_name"`
	Renewable   bool      `json:"renewable"`
}

// Reservation yada
type Reservation struct {
	Title        string    `json:"title"`
	QuePosition  int       `json:"que_position"`
	BooksTotal   int       `json:"books_total"`
	PickupDue    time.Time `json:"pickup_due"`
	PickupNumber int       `json:"pickup_number"`
}

// Opac holds library loans
type Opac struct {
	Name         string        `json:"name"`
	Fee          float64       `json:"fee"`
	Updated      time.Time     `json:"updated"`
	Books        []Book        `json:"books"`
	Reservations []Reservation `json:"reservations"`
}

// New returns a new Opac
func New(name string) *Opac {
	o := &Opac{}
	o.Name = name
	o.Updated = time.Now()

	return o
}

// FirstDue returns the first book due for return
func (o *Opac) FirstDue() (book Book) {
	minDue := time.Date(9999, 0, 0, 0, 0, 0, 0, time.Now().Location())
	for _, b := range o.Books {
		if b.DateDue.Before(minDue) {
			minDue = b.DateDue
			book = b
		}
	}
	return book
}

// ReservationPickup returns true if any book reserved is due for pickup
func (o *Opac) ReservationPickup() bool {
	for _, r := range o.Reservations {
		if r.PickupNumber > 0 {
			return true
		}
	}
	return false
}
