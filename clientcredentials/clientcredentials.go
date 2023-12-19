// Package clientcredentials helps with oauth2 client-credentials flow.
package clientcredentials

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/udhos/oauth2/token"
	"golang.org/x/sync/singleflight"
)

// Options define client options.
type Options struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scope        string
	HTTPClient   *http.Client

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
}

// Client is context for invokations with client-credentials flow.
type Client struct {
	options Options
	group   singleflight.Group
}

// New creates a client.
func New(options Options) *Client {
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

	if resp.StatusCode == 401 {
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

	form := url.Values{}
	form.Add("grant_type", "client_credentials")
	form.Add("client_id", c.options.ClientID)
	form.Add("client_secret", c.options.ClientSecret)
	if c.options.Scope != "" {
		form.Add("scope", c.options.Scope)
	}

	req, errReq := http.NewRequestWithContext(context.TODO(), "POST", c.options.TokenURL, strings.NewReader(form.Encode()))
	if errReq != nil {
		return "", errReq
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, errDo := c.options.HTTPClient.Do(req)
	if errDo != nil {
		return "", errDo
	}
	defer resp.Body.Close()

	body, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		return "", errBody
	}

	elap := time.Since(begin)

	c.debugf("fetchToken: elapsed:%v token: %s", elap, string(body))

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status:%d body:%v", resp.StatusCode, string(body))
	}

	info, errParse := parseToken(body, c.debugf)
	if errParse != nil {
		return "", fmt.Errorf("parse token: %v", errParse)
	}

	newToken := token.Token{
		Value: info.accessToken,
	}

	if info.expiresIn != 0 {
		newToken.SetExpiration(time.Now().Add(info.expiresIn))
	}

	c.debugf("saving new token")
	if err := c.options.Cache.Put(newToken); err != nil {
		c.errorf("cache put error: %v", err)
	}

	return newToken.Value, nil
}

type tokenInfo struct {
	accessToken string
	expiresIn   time.Duration
}

func parseToken(buf []byte, debugf func(format string, v ...any)) (tokenInfo, error) {
	var info tokenInfo

	var data map[string]interface{}

	errJSON := json.Unmarshal(buf, &data)
	if errJSON != nil {
		return info, errJSON
	}

	accessToken, foundToken := data["access_token"]
	if !foundToken {
		return info, fmt.Errorf("missing access_token field in token response")
	}

	tokenStr, isStr := accessToken.(string)
	if !isStr {
		return info, fmt.Errorf("non-string value for access_token field in token response")
	}

	if tokenStr == "" {
		return info, fmt.Errorf("empty access_token in token response")
	}

	info.accessToken = tokenStr

	expire, foundExpire := data["expires_in"]
	if foundExpire {
		switch expireVal := expire.(type) {
		case float64:
			debugf("found expires_in field with %f seconds", expireVal)
			info.expiresIn = time.Second * time.Duration(expireVal)
		case string:
			debugf("found expires_in field with %s seconds", expireVal)
			exp, errConv := strconv.Atoi(expireVal)
			if errConv != nil {
				return info, fmt.Errorf("error converting expires_in field from string='%s' to int: %v", expireVal, errConv)
			}
			info.expiresIn = time.Second * time.Duration(exp)
		default:
			return info, fmt.Errorf("unexpected type %T for expires_in field in token response", expire)
		}
	}

	return info, nil
}
