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
)

// TokenCache defines a cache interface for storing tokens.
type TokenCache interface {
	Get() Token
	Put(t Token)
	Expire()
}

// Options define client options.
type Options struct {
	TokenURL                string
	ClientID                string
	ClientSecret            string
	Scope                   string
	HTTPClient              *http.Client
	ExpireTolerationSeconds int // 0 defaults to 10 seconds. Set to -1 to no toleration.
	Cache                   TokenCache
}

type memoryCache struct {
	t Token
}

func (mc *memoryCache) Get() Token {
	return mc.t
}

func (mc *memoryCache) Put(t Token) {
	mc.t = t
}

var expired = time.Time{}

func (mc *memoryCache) Expire() {
	mc.t.Deadline = &expired
}

// DefaultTokenCache provides default implementation for token cache.
var DefaultTokenCache = &memoryCache{}

// Client is context for invokations with client-credentials flow.
type Client struct {
	options Options
}

// Token holds a token.
type Token struct {
	Value    string
	Deadline *time.Time // nil deadline is always valid
}

// IsValid checks whether token is valid.
func (t Token) IsValid(toleration time.Duration) bool {
	log.Printf("token toleration: %v", toleration)
	if t.Deadline == nil {
		return true
	}
	return t.Deadline.After(time.Now().Add(toleration))
}

// New creates a client.
func New(options Options) *Client {
	switch options.ExpireTolerationSeconds {
	case 0:
		options.ExpireTolerationSeconds = 10
	case -1:
		options.ExpireTolerationSeconds = 0
	}
	if options.Cache == nil {
		options.Cache = DefaultTokenCache
	}
	options.Cache.Expire()
	return &Client{
		options: options,
	}
}

// Do sends an HTTP request.
func (c *Client) Do(req *http.Request) (*http.Response, error) {

	accessToken, errToken := c.getToken()
	if errToken != nil {
		return nil, errToken
	}

	resp, errResp := c.send(req, accessToken)

	if resp.StatusCode == 401 {
		c.options.Cache.Expire()
	}

	return resp, errResp
}

func (c *Client) send(req *http.Request, accessToken string) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	return c.options.HTTPClient.Do(req)
}

func (c *Client) getToken() (string, error) {
	t := c.options.Cache.Get()
	toleration := time.Duration(c.options.ExpireTolerationSeconds) * time.Second
	if t.IsValid(toleration) {
		log.Printf("found valid cached token")
		return t.Value, nil
	}
	log.Printf("NO valid cached token")
	return c.fetchToken()
}

// fetchTokens retrieves new token and saves into cache.
func (c *Client) fetchToken() (string, error) {

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

	log.Printf("fetchToken: elapsed:%v token: %s", elap, string(body))

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status:%d body:%v", resp.StatusCode, string(body))
	}

	var data map[string]interface{}

	errJSON := json.Unmarshal(body, &data)
	if errJSON != nil {
		return "", errJSON
	}

	accessToken, foundToken := data["access_token"]
	if !foundToken {
		return "", fmt.Errorf("missing access_token field in token response")
	}

	tokenStr, isStr := accessToken.(string)
	if !isStr {
		return "", fmt.Errorf("non-string value for access_token field in token response")
	}

	newToken := Token{
		Value: tokenStr,
	}

	expire, foundExpire := data["expires_in"]
	if foundExpire {
		switch expireVal := expire.(type) {
		case float64:
			log.Printf("found expires_in field with %f seconds", expireVal)
			deadline := time.Now().Add(time.Second * time.Duration(expireVal))
			newToken.Deadline = &deadline
		case string:
			log.Printf("found expires_in field with %s seconds", expireVal)
			exp, errConv := strconv.Atoi(expireVal)
			if errConv != nil {
				return "", fmt.Errorf("error converting expires_in field from string='%s' to int: %v", expireVal, errConv)
			}
			deadline := time.Now().Add(time.Second * time.Duration(exp))
			newToken.Deadline = &deadline
		default:
			return "", fmt.Errorf("unexpected type %T for expires_in field in token response", expire)
		}
	}

	log.Printf("saving new token")
	c.options.Cache.Put(newToken)

	return tokenStr, nil
}
