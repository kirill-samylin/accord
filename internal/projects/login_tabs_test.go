package projects

import (
	"encoding/json"
	"net/url"
	"testing"
)

// Two login tabs share one browser cookie jar: opening the form again must not
// invalidate the form already open in the first tab.
func TestSecondLoginPageKeepsFirstFormValid(t *testing.T) {
	f := setup(t)
	csrf := func(body []byte) string {
		var page struct {
			Page struct {
				CSRF string `json:"csrf"`
			}
		}
		if e := json.Unmarshal(body, &page); e != nil {
			t.Fatal(e)
		}
		return page.Page.CSRF
	}
	first := f.request("GET", "/web/login", nil, nil)
	cookies := first.Result().Cookies()
	firstCSRF := csrf(first.Body.Bytes())
	second := f.request("GET", "/web/login", cookies, nil)
	if second.Code != 200 || csrf(second.Body.Bytes()) != firstCSRF {
		t.Fatalf("second tab rotated the login form: %d", second.Code)
	}
	jar := second.Result().Cookies()
	if len(jar) == 0 {
		jar = cookies
	}
	out := f.request("POST", "/web/login", jar, url.Values{"csrf": {firstCSRF}, "token": {f.token.Secret}})
	if out.Code != 200 {
		t.Fatalf("login from the first tab after opening a second one: %d %s", out.Code, out.Body.String())
	}
	// A consumed preauth is not reused: the next login page issues a new form.
	again := f.request("GET", "/web/login", cookies, nil)
	if again.Code != 200 || csrf(again.Body.Bytes()) == firstCSRF {
		t.Fatal("consumed preauth was reused")
	}
}
