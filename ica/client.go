package ica

import (
	"bytes"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/publicsuffix"
)

// Client connects to ICA via the app api
type Client struct {
	AuthenticationTicket string
	Client               http.Client
}

// Login performs a login for the json api
func (c *Client) Login(user string, password string) (err error) {
	client := &http.Client{
		CheckRedirect: nil,
		Timeout:       time.Second * 10,
	}
	request, _ := http.NewRequest("GET", "https://handla.api.ica.se/api/login/", nil)
	request.SetBasicAuth(user, password)
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	c.AuthenticationTicket = resp.Header.Get("AuthenticationTicket")

	return nil
}

func (c *Client) request(url string) (resp *http.Response, err error) {
	request, _ := http.NewRequest("GET", url, nil)
	request.Header.Add("AuthenticationTicket", c.AuthenticationTicket)
	resp, err = c.Client.Do(request)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// GetAccount returns accounts
func (c *Client) GetAccount() (resp *http.Response, err error) {

	return c.request("https://handla.api.ica.se/api/user/cardaccounts")
}

// GetTransactions returns all transactions
func (c *Client) GetTransactions() (resp *http.Response, err error) {

	return c.request("https://handla.api.ica.se/api/user/minbonustransaction")
}

// GetHTML fetches the page for a user and returns a http.Response
func (c *Client) GetHTML(user string, password string) (resp *http.Response, err error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return resp, err
	}

	client := &http.Client{
		Jar: jar,
	}
	// The page we want
	resp, err = client.Get("https://www.ica.se/templates/ajaxresponse.aspx?ajaxFunction=DashboardAccountInfo&callerPageId=446575&_=1511639168567")
	if err != nil {
		return resp, err
	}
	resp, err = client.Get("https://www.ica.se/logga-in/sso/?returnurl=https://www.ica.se/templates/ajaxresponse.aspx?ajaxFunction=DashboardAccountInfo&callerPageId=446575&_=1511639168567")
	if err != nil {
		return resp, err
	}
	referer := resp.Request.URL.String()
	post := url.Values{}
	post.Add("userName", user)
	post.Add("password", password)

	r, _ := http.NewRequest("POST", "https://ims.icagruppen.se/authn/authenticate/IcaCustomers", bytes.NewBufferString(post.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Referer", referer)
	r.Header.Set("Origin", "https://ims.icagruppen.se")
	r.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	r.Header.Set("Upgrade-Insecure-Requests", "1")
	r.Header.Set("Sec-Fetch-Dest", "document")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	r.Header.Set("Sec-Fetch-Mode", "navigate")
	r.Header.Set("Sec-Fetch-User", "?1")
	r.Header.Set("Accept-Encoding", "gzip, deflate, br")
	r.Header.Set("Accept-Language", "en,sv;q=0.9,no;q=0.8,nn;q=0.7,nb;q=0.6,en-US;q=0.5,pl;q=0.4")

	resp, err = client.Do(r)
	if err != nil {
		return resp, err
	}
	// Extra form with tokens to post
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return resp, err
	}
	post = url.Values{}
	doc.Find("input[type=\"hidden\"]").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		value, _ := s.Attr("value")
		post.Add(name, value)
	})
	resp, err = client.PostForm("https://ims.icagruppen.se/oauth/v2/authorize?client_id=ica.se&forceAuthN=true", post)
	if err != nil {
		return resp, err
	}

	return resp, nil
}
