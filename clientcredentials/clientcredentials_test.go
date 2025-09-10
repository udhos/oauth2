package clientcredentials

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/udhos/oauth2/cache"
	"github.com/udhos/oauth2/token"
)

func TestClientCredentials(t *testing.T) {

	clientID := "clientID"
	clientSecret := "clientSecret"
	token := "abc"
	expireIn := 0
	softExpire := 0
	timeSource := (func() time.Time)(nil)
	disableSingleflight := false

	tokenServerStat := serverStat{}
	serverStat := serverStat{}

	ts := newTokenServer(&tokenServerStat, clientID, clientSecret, token, expireIn)
	defer ts.Close()

	validToken := func(t string) bool { return t == token }

	srv := newServer(&serverStat, validToken)
	defer srv.Close()

	client := newClient(t, ts.URL, clientID, clientSecret, softExpire, timeSource, disableSingleflight)

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

func TestConcurrency(t *testing.T) {

	clientID := "clientID"
	clientSecret := "clientSecret"
	token := "abc"
	expireIn := 0
	softExpire := 0
	timeSource := (func() time.Time)(nil)
	disableSingleflight := false

	tokenServerStat := serverStat{}
	serverStat := serverStat{}

	ts := newTokenServer(&tokenServerStat, clientID, clientSecret, token, expireIn)
	defer ts.Close()

	validToken := func(t string) bool { return t == token }

	srv := newServer(&serverStat, validToken)
	defer srv.Close()

	client := newClient(t, ts.URL, clientID, clientSecret, softExpire, timeSource, disableSingleflight)

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {

			for j := 0; j < 100; j++ {
				_, errSend := send(client, srv.URL)
				if errSend != nil {
					t.Errorf("send1: %v", errSend)
				}
			}

			wg.Done()
		}()
	}

	wg.Wait()
}

// go test -run TestSingleFlight -count 1 ./clientcredentials
func TestSingleFlight(t *testing.T) {

	clientID := "clientID"
	clientSecret := "clientSecret"
	token := "abc"
	expireIn := 0
	softExpire := 0
	timeSource := (func() time.Time)(nil)
	disableSingleflight := false

	tokenServerStat := serverStat{}
	serverStat := serverStat{}

	ts := newTokenServer(&tokenServerStat, clientID, clientSecret, token, expireIn)
	defer ts.Close()

	validToken := func(t string) bool { return t == token }

	srv := newServer(&serverStat, validToken)
	defer srv.Close()

	//
	// error cache forces token retrieval for every request
	//
	oldCache := os.Getenv("CACHE")
	os.Setenv("CACHE", "error")
	defer os.Setenv("CACHE", oldCache)

	client := newClient(t, ts.URL, clientID, clientSecret, softExpire, timeSource, disableSingleflight)

	//
	// fire concurrent requests
	//

	goroutines := 100
	requestsPerGoroutine := 100
	total := goroutines * requestsPerGoroutine

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {

			for j := 0; j < requestsPerGoroutine; j++ {
				_, errSend := send(client, srv.URL)
				if errSend != nil {
					t.Errorf("send1: %v", errSend)
				}
			}

			wg.Done()
		}()
	}
	wg.Wait()

	//
	// check requests count
	//

	t.Logf("requests: total=%d tokens=%d server=%d", total, tokenServerStat.count, serverStat.count)

	if total <= tokenServerStat.count {
		t.Errorf("singleflight didnt save token requests: total=%d tokens=%d server=%d",
			total, tokenServerStat.count, serverStat.count)
	}
}

// go test -run TestDisableSingleFlight -count 1 ./clientcredentials
func TestDisableSingleFlight(t *testing.T) {

	clientID := "clientID"
	clientSecret := "clientSecret"
	token := "abc"
	expireIn := 0
	softExpire := 0
	timeSource := (func() time.Time)(nil)
	disableSingleflight := true

	tokenServerStat := serverStat{}
	serverStat := serverStat{}

	ts := newTokenServer(&tokenServerStat, clientID, clientSecret, token, expireIn)
	defer ts.Close()

	validToken := func(t string) bool { return t == token }

	srv := newServer(&serverStat, validToken)
	defer srv.Close()

	//
	// error cache forces token retrieval for every request
	//
	oldCache := os.Getenv("CACHE")
	os.Setenv("CACHE", "error")
	defer os.Setenv("CACHE", oldCache)

	client := newClient(t, ts.URL, clientID, clientSecret, softExpire, timeSource, disableSingleflight)

	//
	// fire concurrent requests
	//

	goroutines := 100
	requestsPerGoroutine := 100
	total := goroutines * requestsPerGoroutine

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {

			for j := 0; j < requestsPerGoroutine; j++ {
				_, errSend := send(client, srv.URL)
				if errSend != nil {
					t.Errorf("send1: %v", errSend)
				}
			}

			wg.Done()
		}()
	}
	wg.Wait()

	//
	// check requests count
	//

	t.Logf("requests: total=%d tokens=%d server=%d", total, tokenServerStat.count, serverStat.count)

	if total != tokenServerStat.count {
		t.Errorf("unexpected different request count: total=%d tokens=%d server=%d",
			total, tokenServerStat.count, serverStat.count)
	}
}

func TestClientCredentialsExpiration(t *testing.T) {

	clientID := "clientID"
	clientSecret := "clientSecret"
	token := "abc"
	expireIn := 1
	softExpire := -1 // disable soft expire
	disableSingleflight := false

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

	client := newClient(t, ts.URL, clientID, clientSecret, softExpire, timeSource, disableSingleflight)

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
	disableSingleflight := false

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

	client := newClient(t, ts.URL, clientID, clientSecret, softExpire, timeSource, disableSingleflight)

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

func TestServerBrokenURL(t *testing.T) {

	clientID := "clientID"
	clientSecret := "clientSecret"
	token := "abc"
	expireIn := 0
	softExpire := 0
	timeSource := (func() time.Time)(nil)
	disableSingleflight := false

	tokenServerStat := serverStat{}
	serverStat := serverStat{}

	ts := newTokenServer(&tokenServerStat, clientID, clientSecret, token, expireIn)
	defer ts.Close()

	client := newClient(t, ts.URL, clientID, clientSecret, softExpire, timeSource, disableSingleflight)

	// send

	{
		_, errSend := send(client, "broken-url")
		if errSend == nil {
			t.Errorf("unexpected success from broken server")
		}
		if tokenServerStat.count != 1 {
			t.Errorf("unexpected token server access count: %d", tokenServerStat.count)
		}
		if serverStat.count != 0 {
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
	disableSingleflight := false

	serverStat := serverStat{}

	validToken := func(t string) bool { return t == token }

	srv := newServer(&serverStat, validToken)
	defer srv.Close()

	client := newClient(t, "broken-url", clientID, clientSecret, softExpire, timeSource, disableSingleflight)

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
	disableSingleflight := false

	tokenServerStat := serverStat{}
	serverStat := serverStat{}

	ts := newTokenServerBroken(&tokenServerStat)
	defer ts.Close()

	validToken := func(t string) bool { return t == token }

	srv := newServer(&serverStat, validToken)
	defer srv.Close()

	client := newClient(t, ts.URL, clientID, clientSecret, softExpire, timeSource, disableSingleflight)

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
	disableSingleflight := false

	tokenServerStat := serverStat{}
	serverStat := serverStat{}

	ts := newTokenServer(&tokenServerStat, clientID, "WRONG-SECRET", token, expireIn)
	defer ts.Close()

	validToken := func(t string) bool { return t == token }

	srv := newServer(&serverStat, validToken)
	defer srv.Close()

	client := newClient(t, ts.URL, clientID, clientSecret, softExpire, timeSource, disableSingleflight)

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
		stat.inc()
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
	mutex sync.Mutex
}

func (stat *serverStat) inc() {
	stat.mutex.Lock()
	stat.count++
	stat.mutex.Unlock()
}

func newTokenServer(serverInfo *serverStat, clientID, clientSecret, token string, expireIn int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		serverInfo.inc()

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
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		serverInfo.inc()
		httpJSON(w, "broken-token", http.StatusOK)
	}))
}

func newClient(t *testing.T, tokenURL, clientID, clientSecret string, softExpire int, timeSource func() time.Time, disableSingleflight bool) *Client {

	var c token.TokenCache

	if cacheStr := os.Getenv("CACHE"); cacheStr != "" {
		t.Logf("cache: CACHE=%s", cacheStr)
		cc, errCache := cache.New(cacheStr, tokenURL, clientID)
		if errCache != nil {
			t.Fatalf("test: newClient: %v", errCache)
			return nil
		}
		c = cc
	}

	options := Options{
		TokenURL:            tokenURL,
		ClientID:            clientID,
		ClientSecret:        clientSecret,
		Scope:               "scope1 scope2",
		HTTPClient:          http.DefaultClient,
		SoftExpireInSeconds: softExpire,
		TimeSource:          timeSource,
		Cache:               c,
		DisableSingleFlight: disableSingleflight,
	}

	client := New(options)

	return client
}
