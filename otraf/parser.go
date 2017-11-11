package otraf

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func parseDateTime(str string) (time.Time, error) {
	t, error := time.Parse("2006-01-02 15:04", str)
	if error != nil {
		return t, error
	}
	ret := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, time.Now().Location())

	return ret, nil
}

func parseDate(str string) (time.Time, error) {
	t, error := time.Parse("2006-01-02", str)
	if error != nil {
		return t, error
	}
	ret := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Now().Location())

	return ret, nil
}

// Parse parses a page from otraf
func Parse(resp *http.Response, o *Otraf) (*Otraf, error) {
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return o, err
	}
	page := doc.Text()

	// Check cash on card
	cash := 0
	cashRex := regexp.MustCompile(`(\d+) kr`)
	match := cashRex.FindAllStringSubmatch(page, 2)
	if match != nil {
		for _, m := range match {
			amount, err := strconv.Atoi(m[1])
			if err == nil {
				cash += amount
			}
		}
	}
	o.Amount = cash

	// Check monthly card
	// Fr&#229;n  2017-01-23 Till 2017-02-22
	cardRex := regexp.MustCompile(`Fr&#229;n (\d{4}-\d\d-\d\d).*? Till (\d{4}-\d\d-\d\d)`)
	match = cardRex.FindAllStringSubmatch(page, 1)
	if match != nil {
		o.CardStart, err = parseDate(match[0][1])
		if err != nil {
			return o, err
		}
		o.CardEnd, err = parseDate(match[0][2])
		if err != nil {
			return o, err
		}
	}

	// Get card updated
	updatedRex := regexp.MustCompile(`\(Senast uppdaterat.*(\d{4}.*?)\)`)
	match = updatedRex.FindAllStringSubmatch(page, 1)
	if match != nil {
		o.CardUpdated, err = parseDateTime(match[0][1])
		if err != nil {
			return o, err
		}
	}
	return o, nil
}
