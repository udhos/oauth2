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
	{"not-json", "not-json", expectFailure, "", 0},
	{"no fields", `{}`, expectFailure, "", 0},
	{"missing access_token", `{"other":"field"}`, expectFailure, "", 0},
	{"empty access_token", `{"access_token":""}`, expectFailure, "", 0},
	{"only good access token", `{"access_token":"abc"}`, expectSucess, "abc", 0},
	{"wrong access token type int", `{"access_token":123}`, expectFailure, "", 0},
	{"wrong access token type bool", `{"access_token":true}`, expectFailure, "", 0},
	{"wrong access token type float", `{"access_token":1.1}`, expectFailure, "", 0},
	{"expire integer", `{"access_token":"abc","expires_in":300}`, expectSucess, "abc", 300 * time.Second},
	{"expire float", `{"access_token":"abc","expires_in":300.0}`, expectSucess, "abc", 300 * time.Second},
	{"expire string", `{"access_token":"abc","expires_in":"300"}`, expectSucess, "abc", 300 * time.Second},
	{"expire broken string", `{"access_token":"abc","expires_in":"TTT"}`, expectFailure, "", 0},
	{"expire empty string", `{"access_token":"abc","expires_in":""}`, expectFailure, "", 0},
	{"expire broken bool", `{"access_token":"abc","expires_in":true}`, expectFailure, "", 0},
}

func TestParseToken(t *testing.T) {
	for _, data := range parseTokenTestTable {
		buf := []byte(data.token)
		info, errParse := parseToken(buf)
		success := errParse == nil
		if success != bool(data.expect) {
			t.Errorf("%s: expectedSuccess=%t gotSuccess=%t error:%v", data.name, data.expect, success, errParse)
			continue
		}

		if !success {
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
	expireIn := 0
	softExpire := 0
	timeSource := (func() time.Time)(nil)

	tokenServerStat := serverStat{}
	serverStat := serverStat{}

	ts := newTokenServer(&tokenServerStat, clientID, clientSecret, token, expireIn)
	defer ts.Close()

	validToken := func(t string) bool { return t == token }

	srv := newServer(&serverStat, validToken)
	defer srv.Close()

	client := newClient(ts.URL, clientID, clientSecret, softExpire, timeSource)

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

func TestClientCredentialsExpiration(t *testing.T) {

	clientID := "clientID"
	clientSecret := "clientSecret"
	token := "abc"
	expireIn := 1
	softExpire := -1 // disable soft expire

	tokenServerStat := serverStat{}
	serverStat := serverStat{}

	ts := newTokenServer(&tokenServerStat, clientID, clientSecret, token, expireIn)
	defer ts.Close()

	validToken := func(t string) bool { return t == token }

	srv := newServer(&serverStat, validToken)
	defer srv.Close()

	clock := time.Now()
	timeSource := func() time.Time {
		return clock
	}

	client := newClient(ts.URL, clientID, clientSecret, softExpire, timeSource)

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

	{
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

	clock = clock.Add(time.Second * time.Duration(expireIn+1))

	// send 3

	_, errSend3 := send(client, srv.URL)
	if errSend3 != nil {
		t.Errorf("send: %v", errSend3)
	}
	if tokenServerStat.count != 2 {
		t.Errorf("unexpected token server access count: %d", tokenServerStat.count)
	}
	if serverStat.count != 3 {
		t.Errorf("unexpected server access count: %d", serverStat.count)
	}
}

func TestForcedExpiration(t *testing.T) {

	clientID := "clientID"
	clientSecret := "clientSecret"
	token := "abc"
	expireIn := 60
	softExpire := -1 // disable soft expire

	tokenServerStat := serverStat{}
	serverStat := serverStat{}

	ts := newTokenServer(&tokenServerStat, clientID, clientSecret, token, expireIn)
	defer ts.Close()

	validToken := func(t string) bool { return t == token }

	srv := newServer(&serverStat, validToken)
	defer srv.Close()

	clock := time.Now()
	timeSource := func() time.Time {
		return clock
	}

	client := newClient(ts.URL, clientID, clientSecret, softExpire, timeSource)

	// send 1: get first token

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

	// send 2: get cached token

	{
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

	// send 3: break cached token

	token = "broken"

	{
		result, errSend3 := send(client, srv.URL)
		if errSend3 == nil {
			t.Errorf("unexpected send sucesss")
		}
		if result.status != 401 {
			t.Errorf("unexpected status: %d", result.status)
		}
		if tokenServerStat.count != 1 {
			t.Errorf("unexpected token server access count: %d", tokenServerStat.count)
		}
		if serverStat.count != 3 {
			t.Errorf("unexpected server access count: %d", serverStat.count)
		}
	}

	// send 4: fix token

	token = "abc"

	{
		_, errSend3 := send(client, srv.URL)
		if errSend3 != nil {
			t.Errorf("send: %v", errSend3)
		}
		if tokenServerStat.count != 2 {
			t.Errorf("unexpected token server access count: %d", tokenServerStat.count)
		}
		if serverStat.count != 4 {
			t.Errorf("unexpected server access count: %d", serverStat.count)
		}
	}

}

func TestTokenServerBrokenURL(t *testing.T) {

	clientID := "clientID"
	clientSecret := "clientSecret"
	token := "abc"
	softExpire := 0
	timeSource := (func() time.Time)(nil)

	serverStat := serverStat{}

	validToken := func(t string) bool { return t == token }

	srv := newServer(&serverStat, validToken)
	defer srv.Close()

	client := newClient("broken-url", clientID, clientSecret, softExpire, timeSource)

	// send 1

	_, errSend := send(client, srv.URL)
	if errSend == nil {
		t.Errorf("unexpected send success")
	}
}

func TestBrokenTokenServer(t *testing.T) {

	clientID := "clientID"
	clientSecret := "clientSecret"
	token := "abc"
	softExpire := 0
	timeSource := (func() time.Time)(nil)

	tokenServerStat := serverStat{}
	serverStat := serverStat{}

	ts := newTokenServerBroken(&tokenServerStat)
	defer ts.Close()

	validToken := func(t string) bool { return t == token }

	srv := newServer(&serverStat, validToken)
	defer srv.Close()

	client := newClient(ts.URL, clientID, clientSecret, softExpire, timeSource)

	// send 1

	{
		_, errSend := send(client, srv.URL)
		if errSend == nil {
			t.Errorf("unexpected success with broken token server")
		}
		if tokenServerStat.count != 1 {
			t.Errorf("unexpected token server access count: %d", tokenServerStat.count)
		}
		if serverStat.count != 0 {
			t.Errorf("unexpected server access count: %d", serverStat.count)
		}
	}

	// send 2

	{
		_, errSend := send(client, srv.URL)
		if errSend == nil {
			t.Errorf("unexpected success with broken token server")
		}
		if tokenServerStat.count != 2 {
			t.Errorf("unexpected token server access count: %d", tokenServerStat.count)
		}
		if serverStat.count != 0 {
			t.Errorf("unexpected server access count: %d", serverStat.count)
		}
	}

}

func TestLockedTokenServer(t *testing.T) {

	clientID := "clientID"
	clientSecret := "clientSecret"
	token := "abc"
	expireIn := 60
	softExpire := 0
	timeSource := (func() time.Time)(nil)

	tokenServerStat := serverStat{}
	serverStat := serverStat{}

	ts := newTokenServer(&tokenServerStat, clientID, "WRONG-SECRET", token, expireIn)
	defer ts.Close()

	validToken := func(t string) bool { return t == token }

	srv := newServer(&serverStat, validToken)
	defer srv.Close()

	client := newClient(ts.URL, clientID, clientSecret, softExpire, timeSource)

	// send 1

	{
		_, errSend := send(client, srv.URL)
		if errSend == nil {
			t.Errorf("unexpected success with locked token server")
		}
		if tokenServerStat.count != 1 {
			t.Errorf("unexpected token server access count: %d", tokenServerStat.count)
		}
		if serverStat.count != 0 {
			t.Errorf("unexpected server access count: %d", serverStat.count)
		}
	}

	// send 2

	{
		_, errSend := send(client, srv.URL)
		if errSend == nil {
			t.Errorf("unexpected success with locked token server")
		}
		if tokenServerStat.count != 2 {
			t.Errorf("unexpected token server access count: %d", tokenServerStat.count)
		}
		if serverStat.count != 0 {
			t.Errorf("unexpected server access count: %d", serverStat.count)
		}
	}
}

type sendResult struct {
	body   string
	status int
}

func send(client *Client, serverURL string) (sendResult, error) {

	var result sendResult

	req, errReq := http.NewRequestWithContext(context.TODO(), "GET", serverURL, nil)
	if errReq != nil {
		return result, fmt.Errorf("request: %v", errReq)
	}

	resp, errDo := client.Do(req)
	if errDo != nil {
		return result, fmt.Errorf("do: %v", errDo)
	}
	defer resp.Body.Close()

	body, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		return result, fmt.Errorf("body: %v", errBody)
	}

	bodyStr := string(body)

	result.body = bodyStr
	result.status = resp.StatusCode

	if resp.StatusCode != 200 {
		return result, fmt.Errorf("bad status:%d body:%v", resp.StatusCode, bodyStr)
	}

	return result, nil
}

func formParam(r *http.Request, key string) string {
	v := r.Form[key]
	if v == nil {
		return ""
	}
	return v[0]
}

func newServer(stat *serverStat, validToken func(token string) bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stat.count++
		h := r.Header.Get("Authorization")
		t := strings.TrimPrefix(h, "Bearer ")
		if !validToken(t) {
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

func newTokenServer(serverInfo *serverStat, clientID, clientSecret, token string, expireIn int) *httptest.Server {
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

		var t string

		if expireIn > 0 {
			t = fmt.Sprintf(`{"access_token":"%s","expires_in":%d}`, token, expireIn)
		} else {
			t = fmt.Sprintf(`{"access_token":"%s"}`, token)
		}

		httpJSON(w, t, http.StatusOK)
	}))
}

func newTokenServerBroken(serverInfo *serverStat) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverInfo.count++
		httpJSON(w, "broken-token", http.StatusOK)
	}))
}

func newClient(tokenURL, clientID, clientSecret string, softExpire int, timeSource func() time.Time) *Client {
	options := Options{
		TokenURL:            tokenURL,
		ClientID:            clientID,
		ClientSecret:        clientSecret,
		Scope:               "scope1 scope2",
		HTTPClient:          http.DefaultClient,
		SoftExpireInSeconds: softExpire,
		TimeSource:          timeSource,
	}

	client := New(options)

	return client
}
