package otraf

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"golang.org/x/net/publicsuffix"
)

// GetHTML fetches the page for a user and returns a http.Response
func GetHTML(user string, password string, tab string) (resp *http.Response, err error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return resp, err
	}

	c := &http.Client{
		Jar: jar,
	}

	// POSTDATA=={"authSource":10,"keepMeLimitedLoggedIn":true,
	// "userName":"XXX","password":"XXX","impersonateUserName":""}
	post := `={"authSource":10,"keepMeLimitedLoggedIn":true,`
	post += fmt.Sprintf(`"userName":"%s","password":"%s","impersonateUserName":""}`, user, password)
	poster := strings.NewReader(post)
	_, err = c.Post("https://www.ostgotatrafiken.se/ajax/Login/Attempt",
		"application/x-www-form-urlencoded;charset=UTF-8", poster)
	if err != nil {
		return resp, err
	}
	// resp, err = c.Get("https://www.ostgotatrafiken.se/ajax2/store/cardclient/getcards")
	resp, err = c.Get("https://webtick.ostgotatrafiken.se/webtick/user/pages/CardOverview.iface")
	if tab == "" {
		return resp, err
	}

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return resp, err
	}
	form := url.Values{}
	doc.Find("#formLinkedCardRequests").Find("input").Each(func(i int, s *goquery.Selection) {
		// Should be these inputs
		// formLinkedCardRequests formLinkedCardRequests
		// javax.faces.ViewState -8263062689401194327:5324554437178885862
		// ice.window 2tj9t0t3da
		// ice.view v3gtxnu7xki
		// icefacesCssUpdates
		name, ok := s.Attr("name")
		if !ok {
			return
		}
		value, ok := s.Attr("value")
		if !ok {
			return
		}
		form.Add(name, value)
	})
	var tabID string

	doc.Find("#form1cardOverviewTabs").Find("span").Each(func(i int, s *goquery.Selection) {
		if strings.TrimSpace(s.Text()) == tab {
			tabID = s.AttrOr("id", "")
		}
	})
	if tabID == "" {
		return resp, errors.New("Can't find span id for tab:" + tab)
	}
	tabID += "Link"
	// form.Add("ice.event.target", tabID)
	form.Add("ice.focus", tabID)
	form.Add("ice.event.captured", tabID)
	form.Add(tabID, tabID)
	form.Add("javax.faces.source", tabID)
	form.Add("form1cardOverviewTabs:j_idcl", tabID)

	form.Add("name", "formLinkedCardRequests")
	form.Add("form1cardOverviewTabs", "form1cardOverviewTabs")
	form.Add("ice.event.alt", "false")
	form.Add("ice.event.ctrl", "false")
	form.Add("ice.event.left", "true")
	form.Add("ice.event.meta", "false")
	form.Add("ice.event.right", "false")
	form.Add("ice.event.shift", "false")
	form.Add("ice.event.type", "onclick")
	form.Add("ice.event.x", "644")
	form.Add("ice.event.y", "372")
	form.Add("ice.submit.serialization", "form")
	form.Add("ice.submit.type", "ice.s")
	form.Add("javax.faces.partial.ajax", "true")
	form.Add("javax.faces.partial.event", "click")
	form.Add("javax.faces.partial.execute", "@all")
	form.Add("javax.faces.partial.render", "@all")
	resp, err = c.PostForm("https://webtick.ostgotatrafiken.se/webtick/user/pages/CardOverview.iface", form)
	if err != nil {
		return resp, err
	}

	return resp, err
}
