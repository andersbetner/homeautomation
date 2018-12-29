package opac

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/publicsuffix"
)

// Client holds a connection to opac
type Client struct {
	user     string
	password string
	client   *http.Client
	baseURL  string
}

// New creates a new Client
func NewClient(user string, password string) (*Client, error) {
	s := &Client{}
	s.baseURL = "https://www.gotabiblioteken.se/web/arena/"
	s.user = user
	s.password = password

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return s, err
	}

	s.client = &http.Client{
		Jar: jar,
	}

	return s, nil
}

// Login performs a login
func (s *Client) Login() error {
	_, err := s.client.Get(s.baseURL + "protected/profile")
	if err != nil {
		return err
	}
	post := url.Values{}
	post.Set("id__patronLogin__WAR__arenaportlets____3_hf_0=", "")
	post.Set("openTextUsernameContainer:openTextUsername", s.user)
	post.Set("textPassword", s.password)

	resp, err := s.client.PostForm(
		s.baseURL+"welcome"+
			"?p_p_id=patronLogin_WAR_arenaportlets&p_p_lifecycle=1&p_p_state=normal&p_p_mode=view&p_p_col_id=column-1&p_p_col_pos=4&p_p_col_count=7&_patronLogin_WAR_arenaportlets__wu=/patronLogin/?wicket:interface=:0:signInPanel:signInFormPanel:signInForm::IFormSubmitListener::",
		post)
	if err != nil {
		return err
	}
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return err
	}
	html, err := doc.Html()
	if err != nil {
		return err
	}
	if !strings.Contains(html, "Logga ut") {
		return errors.New("Can't find text Logga ut on page")
	}

	return nil

}

// Loans returns all loans
func (s *Client) Loans() (*http.Response, error) {

	return s.client.Get(s.baseURL + "protected/loans")
}

// Fee returns fees
func (s *Client) Fee() (*http.Response, error) {

	return s.client.Get(s.baseURL + "protected/debts")
}

// Reservations returns tada
func (s *Client) Reservations() (*http.Response, error) {

	return s.client.Get(s.baseURL + "protected/reservations")
}
