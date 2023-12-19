// Package main implements the tool.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/udhos/oauth2/cache"
	"github.com/udhos/oauth2/clientcredentials"
)

type application struct {
	tokenURL            string
	clientID            string
	clientSecret        string
	scope               string
	targetURL           string
	targetMethod        string
	targetBody          string
	count               int
	softExpireSeconds   int
	interval            time.Duration
	cache               string
	disableSingleflight bool
	concurrent          bool
	debug               bool
}

func main() {

	app := application{}

	flag.StringVar(&app.tokenURL, "tokenURL", "http://localhost:8080/oauth/token", "token URL")
	flag.StringVar(&app.clientID, "clientID", "admin", "client ID")
	flag.StringVar(&app.clientSecret, "clientSecret", "admin", "client secret")
	flag.StringVar(&app.scope, "scope", "", "space-delimited list of scopes")
	flag.StringVar(&app.targetURL, "targetURL", "https://httpbin.org/get", "target URL")
	flag.StringVar(&app.targetMethod, "targetMethod", "GET", "target method")
	flag.StringVar(&app.targetBody, "targetBody", "targetBody", "target body")
	flag.IntVar(&app.count, "count", 2, "how many requests to send")
	flag.IntVar(&app.softExpireSeconds, "softExpireSeconds", 10, "token soft expire in seconds")
	flag.DurationVar(&app.interval, "interval", 2*time.Second, "interval between sends")
	flag.StringVar(&app.cache, "cache", "", "empty means default memory cache\n'file:<path>' means filecache (example: file:/tmp/cache)\n'error' means errorcache\nredis format: 'redis:<host>:<port>:<password>:<key>' (example: redis:localhost:6379::oauth2-client-example)")
	flag.BoolVar(&app.disableSingleflight, "disableSingleflight", false, "disable singleflight")
	flag.BoolVar(&app.concurrent, "concurrent", false, "concurrent requests")
	flag.BoolVar(&app.debug, "debug", false, "enable debug logging")

	flag.Parse()

	cache, errCache := cache.New(app.cache)
	if errCache != nil {
		log.Fatalf("cache error: %s: %v", app.cache, errCache)
	}

	options := clientcredentials.Options{
		TokenURL:            app.tokenURL,
		ClientID:            app.clientID,
		ClientSecret:        app.clientSecret,
		Scope:               app.scope,
		HTTPClient:          http.DefaultClient,
		SoftExpireInSeconds: app.softExpireSeconds,
		Cache:               cache,
		DisableSingleFlight: app.disableSingleflight,
		Debug:               app.debug,
	}

	client := clientcredentials.New(options)

	if app.concurrent {
		//
		// concurrent requests
		//
		var wg sync.WaitGroup
		for i := 1; i <= app.count; i++ {
			j := i
			wg.Add(1)
			go func() {
				send(&app, client, j)
				wg.Done()
			}()
		}
		wg.Wait()
		return
	}

	//
	// non-concurrent requests
	//
	for i := 1; i <= app.count; i++ {
		send(&app, client, i)
	}
}

func send(app *application, client *clientcredentials.Client, i int) {
	label := fmt.Sprintf("request %d/%d", i, app.count)

	req, errReq := http.NewRequestWithContext(context.TODO(), app.targetMethod, app.targetURL, bytes.NewBufferString(app.targetBody))
	if errReq != nil {
		log.Fatalf("%s: request: %v", label, errReq)
	}

	resp, errDo := client.Do(req)
	if errDo != nil {
		log.Fatalf("%s: do: %v", label, errDo)
	}
	defer resp.Body.Close()

	log.Printf("%s: status: %d", label, resp.StatusCode)

	body, errBody := io.ReadAll(resp.Body)
	if errBody != nil {
		log.Fatalf("%s: body: %v", label, errBody)
	}

	log.Printf("%s: body:", label)
	fmt.Println(string(body))

	if i < app.count && app.interval != 0 {
		log.Printf("%s: sleeping for interval=%v", label, app.interval)
		time.Sleep(app.interval)
	}
}
