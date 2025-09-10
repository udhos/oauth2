// Package clientcredentials helps with oauth2 client-credentials flow.
package clientcredentials

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/udhos/oauth2/token"
	cc "github.com/udhos/oauth2clientcredentials/clientcredentials"
	"golang.org/x/sync/singleflight"
)

// HTTPDoer is interface for http client.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Options define client options.
type Options struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scope        string

	// HTTPClient is the HTTP client to use to make requests.
	// If nil, http.DefaultClient is used.
	HTTPClient HTTPDoer

	// IsTokenStatusCodeOk defines custom function to check whether the
	// token server response status is OK.
	// If undefined, defaults to nil, which means any 2xx status is OK.
	IsTokenStatusCodeOk func(status int) bool

	// 0 defaults to 10 seconds. Set to -1 to no soft expire.
	//
	// Example: consider expire_in = 30 seconds and soft expire = 10 seconds.
	// The token will hard expire after 30 seconds, but we will consider it
	// expired after (30-10) = 20 seconds, in order to attempt renewal before
	// hard expiration.
	//
	SoftExpireInSeconds int

	Cache token.TokenCache

	// Time source used to check token expiration.
	// If unspecified, defaults to time.Now().
	TimeSource func() time.Time

	DisableSingleFlight bool

	// Logging function, if undefined defaults to log.Printf
	Logf func(format string, v ...any)

	// Enable debug logging.
	Debug bool

	// IsBadTokenStatus defines custom function to check whether the
	// server response status is bad token.
	// If undefined, defaults to DefaulIsBadTokenStatus that just checks
	// for status 401.
	IsBadTokenStatus func(status int) bool
}

// DefaulIsBadTokenStatus is used as default function when option IsBadTokenStatus
// is left undefined. DefaulIsBadTokenStatus just checks for status 401.
func DefaulIsBadTokenStatus(status int) bool {
	return status == 401
}

// Client is context for invokations with client-credentials flow.
type Client struct {
	options Options
	group   singleflight.Group
}

// New creates a client.
func New(options Options) *Client {
	if options.HTTPClient == nil {
		options.HTTPClient = http.DefaultClient
	}
	switch options.SoftExpireInSeconds {
	case 0:
		options.SoftExpireInSeconds = 10
	case -1:
		options.SoftExpireInSeconds = 0
	}
	if options.Cache == nil {
		options.Cache = token.DefaultTokenCache
	}
	if options.TimeSource == nil {
		options.TimeSource = time.Now
	}
	if options.Logf == nil {
		options.Logf = log.Printf
	}
	if options.IsBadTokenStatus == nil {
		options.IsBadTokenStatus = DefaulIsBadTokenStatus
	}
	options.Cache.Expire()
	return &Client{
		options: options,
	}
}

func (c *Client) errorf(format string, v ...any) {
	c.options.Logf("ERROR: "+format, v...)
}

func (c *Client) debugf(format string, v ...any) {
	if c.options.Debug {
		c.options.Logf("DEBUG: "+format, v...)
	}
}

// Do sends an HTTP request.
func (c *Client) Do(req *http.Request) (*http.Response, error) {

	accessToken, errToken := c.getToken()
	if errToken != nil {
		return nil, errToken
	}

	resp, errResp := c.send(req, accessToken)
	if errResp != nil {
		return resp, errResp
	}

	if c.options.IsBadTokenStatus(resp.StatusCode) {
		//
		// the server refused our token, so we expire it in order to
		// renew it at the next invokation.
		//
		if err := c.options.Cache.Expire(); err != nil {
			c.errorf("cache expire error: %v", err)
		}
	}

	return resp, errResp
}

func (c *Client) send(req *http.Request, accessToken string) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	return c.options.HTTPClient.Do(req)
}

func (c *Client) getToken() (string, error) {
	t, errCache := c.options.Cache.Get()
	if errCache != nil {
		c.errorf("cache get error: %v", errCache)
		return c.fetchToken()
	}
	softExpire := time.Duration(c.options.SoftExpireInSeconds) * time.Second
	now := c.options.TimeSource()
	if t.IsValid(now, softExpire, c.debugf) {
		c.debugf("found valid cached token")
		return t.Value, nil
	}
	c.debugf("NO valid cached token")
	return c.fetchToken()
}

// fetchTokens retrieves new token and saves into cache, guarded with singleflight.
func (c *Client) fetchToken() (string, error) {

	if c.options.DisableSingleFlight {
		return c.fetchTokenRaw()
	}

	key := ""

	f := func() (interface{}, error) {
		return c.fetchTokenRaw()
	}

	result, errFetch, _ := c.group.Do(key, f)
	if errFetch != nil {
		return "", errFetch
	}

	str, isStr := result.(string)
	if !isStr {
		return "", fmt.Errorf("non-string result: type:%[1]T value:%[1]v", result)
	}

	return str, nil
}

// fetchTokensRaw retrieves new token and saves into cache.
func (c *Client) fetchTokenRaw() (string, error) {

	begin := time.Now()

	reqOptions := cc.RequestOptions{
		TokenURL:       c.options.TokenURL,
		ClientID:       c.options.ClientID,
		ClientSecret:   c.options.ClientSecret,
		Scope:          c.options.Scope,
		HTTPClient:     c.options.HTTPClient,
		IsStatusCodeOK: c.options.IsTokenStatusCodeOk,
	}

	if c.options.HTTPClient != nil {
		// do not assign nil to interface
		reqOptions.HTTPClient = c.options.HTTPClient
	}

	resp, errSend := cc.SendRequest(context.TODO(), reqOptions)
	if errSend != nil {
		return "", errSend
	}

	elap := time.Since(begin)

	c.debugf("fetchToken: elapsed:%v token:%v", elap, resp)

	if resp.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	newToken := token.Token{
		Value: resp.AccessToken,
	}

	if resp.ExpiresIn != 0 {
		newToken.SetExpiration(time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second))
	}

	c.debugf("saving new token")
	if err := c.options.Cache.Put(newToken); err != nil {
		c.errorf("cache put error: %v", err)
	}

	return newToken.Value, nil
}
