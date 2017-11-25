package ica

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/publicsuffix"
)

// GetHTML fetches the page for a user and returns a http.Response
func GetHTML(user string, password string) (resp *http.Response, err error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return resp, err
	}

	c := &http.Client{
		Jar: jar,
	}
	resp, err = c.Get("https://www.ica.se/logga-in/?returnurl=https://www.ica.se/mittica#:mittica=inkopslistor")
	if err != nil {
		return resp, err
	}
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return resp, err
	}
	g := doc.Find("input[name=\"__RequestVerificationToken\"]").AttrOr("value", "")
	post := url.Values{}
	post.Add("LoginModel.Username", user)
	post.Add("LoginModel.Password", password)
	post.Add("initiator", "")
	post.Add("returnhash", "")
	post.Add("returnurl", "https://www.ica.se/mittica/#:mittica=inkopslistor")
	post.Add("__RequestVerificationToken", g)
	_, err = c.PostForm("https://www.ica.se/logga-in?ReturnUrl=https://www.ica.se/mittica/#:mittica=inkopslistor",
		post)
	if err != nil {
		return resp, err
	}
	resp, err = c.Get("https://www.ica.se/templates/ajaxresponse.aspx?ajaxFunction=DashboardAccountInfo&callerPageId=446575&_=1511639168567")
	if err != nil {
		return resp, err
	}

	return resp, err
}

// GetJSON returns json
func GetJSON(user string, password string) (resp *http.Response, err error) {
	client := &http.Client{
		CheckRedirect: nil,
		Timeout:       time.Second * 10,
	}
	request, _ := http.NewRequest("GET", "https://api.ica.se/api/login/", nil)
	request.SetBasicAuth(user, password)
	resp, err = client.Do(request)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	authTicket := resp.Header.Get("AuthenticationTicket")
	request, _ = http.NewRequest("GET", "https://api.ica.se/api/user/minasidor/", nil)
	request.Header.Add("AuthenticationTicket", authTicket)
	resp, err = client.Do(request)
	if err != nil {
		return resp, err
	}

	return resp, nil

}
