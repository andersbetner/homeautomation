package ica

import (
	"net/http"
	"time"
)

type IcaClient struct {
	AuthenticationTicket string
	Client               http.Client
}

// IcaClient connects to ICA via the app api
func (c *IcaClient) Login(user string, password string) (err error) {
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

func (c *IcaClient) request(url string) (resp *http.Response, err error) {
	request, _ := http.NewRequest("GET", url, nil)
	request.Header.Add("AuthenticationTicket", c.AuthenticationTicket)
	resp, err = c.Client.Do(request)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// GetJSON returns accounts
func (c *IcaClient) GetAccount() (resp *http.Response, err error) {

	return c.request("https://handla.api.ica.se/api/user/cardaccounts")
}

// GetTransactions returns all transactions
func (c *IcaClient) GetTransactions() (resp *http.Response, err error) {

	return c.request("https://handla.api.ica.se/api/user/minbonustransaction")
}
