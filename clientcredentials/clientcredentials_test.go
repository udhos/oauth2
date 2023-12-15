package clientcredentials

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	expectSucess  = true
	expectFailure = false
)

type expectResult bool

type parseTokenTestCase struct {
	name             string
	token            string
	expect           expectResult
	expectAcessToken string
	expectExpire     time.Duration
}

var parseTokenTestTable = []parseTokenTestCase{
	{"empty", "", expectFailure, "", 0},
	{"no fields", `{}`, expectFailure, "", 0},
	{"missing access_token", `{"other":"field"}`, expectFailure, "", 0},
	{"simple", `{"access_token":"abc"}`, expectSucess, "abc", 0},
	{"expire integer", `{"access_token":"abc","expires_in":300}`, expectSucess, "abc", 300 * time.Second},
	{"expire float", `{"access_token":"abc","expires_in":300.0}`, expectSucess, "abc", 300 * time.Second},
	{"expire string", `{"access_token":"abc","expires_in":"300"}`, expectSucess, "abc", 300 * time.Second},
}

func TestParseToken(t *testing.T) {
	for _, data := range parseTokenTestTable {
		buf := []byte(data.token)
		info, errParse := parseToken(buf)
		success := errParse == nil
		if success != bool(data.expect) {
			t.Errorf("%s: expectedError=%t gotError=%t error:%v", data.name, data.expect, success, errParse)
			continue
		}

		if info.accessToken != data.expectAcessToken {
			t.Errorf("%s: expectedAccessToken=%s gotAccessToken=%s", data.name, data.expectAcessToken, info.accessToken)
		}

		if info.expiresIn != data.expectExpire {
			t.Errorf("%s: expectedExpire=%v gotExpire=%v", data.name, data.expectExpire, info.expiresIn)
		}

		if !t.Failed() {
			t.Logf("%s: ok", data.name)
		}
	}
}

func TestClientCredentials(t *testing.T) {

	clientID := "clientID"
	clientSecret := "clientSecret"
	token := "abc"

	tokenServerStat := serverStat{}
	serverStat := serverStat{}

	ts := newTokenServer(&tokenServerStat, clientID, clientSecret, token)
	defer ts.Close()

	srv := newServer(&serverStat, token)
	defer srv.Close()

	client := newClient(ts.URL, clientID, clientSecret)

	// send 1

	{
		_, errSend := send(client, srv.URL)
		if errSend != nil {
			t.Errorf("send: %v", errSend)
		}
		if tokenServerStat.count != 1 {
			t.Errorf("unexpected token server access count: %d", tokenServerStat.count)
		}
		if serverStat.count != 1 {
			t.Errorf("unexpected server access count: %d", serverStat.count)
		}
	}

	// send 2

	_, errSend2 := send(client, srv.URL)
	if errSend2 != nil {
		t.Errorf("send: %v", errSend2)
	}
	if tokenServerStat.count != 1 {
		t.Errorf("unexpected token server access count: %d", tokenServerStat.count)
	}
	if serverStat.count != 2 {
		t.Errorf("unexpected server access count: %d", serverStat.count)
	}
}

func send(client *Client, serverURL string) (string, error) {

	req, errReq := http.NewRequestWithContext(context.TODO(), "GET", serverURL, nil)
	if errReq != nil {
		return "", fmt.Errorf("request: %v", errReq)
	}

	resp, errDo := client.Do(req)
	if errDo != nil {
		return "", fmt.Errorf("do: %v", errDo)
	}
	defer resp.Body.Close()

	body, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		return "", fmt.Errorf("body: %v", errBody)
	}

	bodyStr := string(body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("bad status:%d body:%v", resp.StatusCode, bodyStr)
	}

	return bodyStr, nil
}

func formParam(r *http.Request, key string) string {
	v := r.Form[key]
	if v == nil {
		return ""
	}
	return v[0]
}

func newServer(stat *serverStat, token string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stat.count++
		h := r.Header.Get("Authorization")
		t := strings.TrimPrefix(h, "Bearer ")
		if t != token {
			httpJSON(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		httpJSON(w, `{"message":"ok"}`, http.StatusOK)
	}))
}

// httpJSON replies to the request with the specified error message and HTTP code.
// It does not otherwise end the request; the caller should ensure no further
// writes are done to w.
// The message should be JSON.
func httpJSON(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprintln(w, message)
}

type serverStat struct {
	count int
}

func newTokenServer(serverInfo *serverStat, clientID, clientSecret, token string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		serverInfo.count++

		r.ParseForm()
		formGrantType := formParam(r, "grant_type")
		formClientID := formParam(r, "client_id")
		formClientSecret := formParam(r, "client_secret")

		if formGrantType != "client_credentials" || formClientID != clientID || formClientSecret != clientSecret {
			httpJSON(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		t := fmt.Sprintf(`{"access_token":"%s"}`, token)

		httpJSON(w, t, http.StatusOK)
	}))
}

func newClient(tokenURL, clientID, clientSecret string) *Client {
	options := Options{
		TokenURL:     tokenURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		HTTPClient:   http.DefaultClient,
	}

	client := New(options)

	return client
}
