package server_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/avenga/couper/internal/test"
)

func TestEndpoints_OAuth2(t *testing.T) {
	helper := test.New(t)

	for i := range []int{0, 1, 2} {
		var seenCh, tokenSeenCh chan struct{}

		retries := 0

		oauthOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/oauth2" {
				reqBody, _ := ioutil.ReadAll(req.Body)
				authorization := req.Header.Get("Authorization")

				if i == 0 {
					exp := `client_id=user&client_secret=pass+word&grant_type=client_credentials&scope=scope1+scope2`
					if exp != string(reqBody) {
						t.Errorf("want\n%s\ngot\n%s", exp, reqBody)
					}
					exp = ""
					if exp != authorization {
						t.Errorf("want\n%s\ngot\n%s", exp, authorization)
					}
				} else {
					exp := `grant_type=client_credentials`
					if exp != string(reqBody) {
						t.Errorf("want\n%s\ngot\n%s", exp, reqBody)
					}
					exp = "Basic dXNlcjpwYXNz"
					if exp != authorization {
						t.Errorf("want\n%s\ngot\n%s", exp, authorization)
					}
				}

				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusOK)

				body := []byte(`{
					"access_token": "abcdef0123456789",
					"token_type": "bearer",
					"expires_in": 100
				}`)
				_, werr := rw.Write(body)
				helper.Must(werr)

				// retries must be equal with the number of retries in the `testdata/oauth2/XXX_retries_couper.hcl`
				if retries == i {
					close(tokenSeenCh)
				}

				return
			}
			rw.WriteHeader(http.StatusBadRequest)
		}))
		defer oauthOrigin.Close()

		ResourceOrigin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/resource" {
				// retries must be equal with the number of retries in the `testdata/oauth2/XXX_retries_couper.hcl`
				if req.Header.Get("Authorization") == "Bearer abcdef0123456789" && retries == i {
					rw.WriteHeader(http.StatusNoContent)
					close(seenCh)
					return
				}

				retries++

				rw.WriteHeader(http.StatusUnauthorized)
				return
			}

			rw.WriteHeader(http.StatusNotFound)
		}))
		defer ResourceOrigin.Close()

		confPath := fmt.Sprintf("testdata/oauth2/%d_retries_couper.hcl", i)
		shutdown, hook := newCouper(confPath, test.New(t))
		defer func() {
			if t.Failed() {
				for _, e := range hook.Entries {
					println(e.String())
				}
			}
			shutdown()
		}()

		req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080/", nil)
		helper.Must(err)

		req.Header.Set("X-Token-Endpoint", oauthOrigin.URL)
		req.Header.Set("X-Origin", ResourceOrigin.URL)

		for _, p := range []string{"/", "/2nd"} {
			hook.Reset()

			seenCh = make(chan struct{})
			tokenSeenCh = make(chan struct{})

			req.URL.Path = p
			res, err := newClient().Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusNoContent {
				t.Errorf("expected status NoContent, got: %d", res.StatusCode)
				return
			}

			timer := time.NewTimer(time.Second * 2)
			select {
			case <-timer.C:
				t.Error("OAuth2 request failed")
			case <-tokenSeenCh:
				<-seenCh
			}
		}

		oauthOrigin.Close()
		ResourceOrigin.Close()
		shutdown()
	}
}
