package opac

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func parseDate(str string) (time.Time, error) {
	t, error := time.Parse("2006-01-02", str)
	ret := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Now().Location())
	return ret, error
}
func parseBook(s *goquery.Selection, renewable bool) Book {
	book := Book{}
	book.Renewable = renewable
	book.Title = strings.TrimSpace(s.Find(".arena-record-title").Text())
	tmp := s.Find(".arena-renewal-branch").Find(".arena-value").Text()
	re := regexp.MustCompile(`\d{4}-\d\d-\d\d`)
	book.LibraryName = strings.TrimSpace(re.ReplaceAllString(tmp, ""))
	tmp = s.Find(".arena-renewal-date").Find(".arena-renewal-date-value").Text()
	var err error
	book.DateDue, err = parseDate(tmp)
	if err != nil {
		// log error here
	}

	return book
}

func parseLoans(client *Client) ([]Book, error) {
	var books []Book
	resp, err := client.Loans()
	if err != nil {
		return books, err
	}
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return books, err
	}

	doc.Find(".arena-renewal-true").Each(func(i int, s *goquery.Selection) {
		books = append(books, parseBook(s, true))
	})
	doc.Find(".arena-renewal-false").Each(func(i int, s *goquery.Selection) {
		books = append(books, parseBook(s, false))
	})

	return books, nil
}

func parseReservations(client *Client) ([]Reservation, error) {
	var reservations []Reservation
	resp, err := client.Reservations()
	if err != nil {
		return reservations, err
	}
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return reservations, err
	}
	doc.Find(".arena-record").Each(func(i int, s *goquery.Selection) {
		reservation := Reservation{}
		reservation.Title = strings.TrimSpace(s.Find(".arena-record-title").Text())

		que := strings.TrimSpace(s.Find(".arena-record-queue").Find(".arena-value").Text())
		re := regexp.MustCompile(`(\d+).*av (\d+) exemplar`)
		if m := re.FindStringSubmatch(que); m != nil {
			pos, err := strconv.Atoi(m[1])
			if err == nil {
				reservation.QuePosition = pos
			}
			total, err := strconv.Atoi(m[2])
			if err == nil {
				reservation.BooksTotal = total
			}
		}

		dueDate := strings.TrimSpace(s.Find(".arena-record-expire").Find(".arena-value").Text())
		reservation.PickupDue, err = parseDate(dueDate)

		reservation.PickupNumber, err = strconv.Atoi(s.Find(".arena-record-pickup").Find(".arena-value").Text())

		reservations = append(reservations, reservation)
	})

	return reservations, nil
}

func parseFee(client *Client) (float64, error) {
	var fee float64
	var err error
	resp, err := client.Fee()
	if err != nil {
		return fee, err
	}
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return fee, err
	}
	doc.Find(".arena-debts-amount").EachWithBreak(func(i int, s *goquery.Selection) bool {
		str := strings.TrimSpace(strings.Replace(s.Text(), ",", ".", 1))
		if str != "" {
			var val float64
			val, err = strconv.ParseFloat(str, 64)
			if err != nil {
				return false
			}
			fee += val
		}
		return true
	})

	if err != nil {
		return fee, err
	}

	return fee, nil
}

// Parse parses opac loan data
func Parse(client *Client, opac *Opac) (*Opac, error) {
	var err error
	opac.Books, err = parseLoans(client)
	if err != nil {
		return opac, err
	}
	opac.Reservations, err = parseReservations(client)
	if err != nil {
		return opac, err
	}
	opac.Fee, err = parseFee(client)
	if err != nil {
		return opac, err
	}

	return opac, nil
}
