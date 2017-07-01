package util

import "time"

// Opac holds library loans
type Opac struct {
	Name    string    `json:"name"`
	Fee     float64   `json:"fee"`
	Updated time.Time `json:"updated"`
	Books   []struct {
		Title       string    `json:"title"`
		DateDue     time.Time `json:"date_due"`
		LibraryName string    `json:"library_name"`
		Renewable   bool      `json:"renewable"`
	} `json:"books"`
	Reservations []struct {
		Title        string    `json:"title"`
		QuePosition  int       `json:"que_position"`
		BooksTotal   int       `json:"books_total"`
		PickupDue    time.Time `json:"pickup_due"`
		PickupNumber int       `json:"pickup_number"`
	} `json:"reservations"`
}
