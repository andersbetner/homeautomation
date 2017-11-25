package ica

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ParseHTML parses html
func ParseHTML(resp *http.Response, ica *Ica) (*Ica, error) {
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return ica, err
	}
	var available string
	doc.Find("div[class=\"account-container account-loaded active\"] dd").EachWithBreak(func(i int, s *goquery.Selection) bool {
		// We want the first dd
		available = strings.Replace(s.Text(), ",", ".", 1)
		available = strings.Replace(available, " kr", "", 1)
		return false
	})
	amount, err := strconv.ParseFloat(available, 64)
	if err != nil {
		return ica, err
	}
	ica.AvailableAmount = amount

	return ica, nil
}

// ParseJSON parses json
func ParseJSON(resp *http.Response, ica *Ica) (*Ica, error) {
	defer resp.Body.Close()
	jsonBlob, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ica, err
	}
	err = json.Unmarshal(jsonBlob, &ica)
	if err != nil {
		return ica, err
	}

	ica.JSON = string(jsonBlob)

	return ica, nil
}
