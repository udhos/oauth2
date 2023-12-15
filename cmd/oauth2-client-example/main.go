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
	"time"

	"github.com/udhos/oauth2/clientcredentials"
)

type application struct {
	tokenURL          string
	clientID          string
	clientSecret      string
	scope             string
	targetURL         string
	targetMethod      string
	targetBody        string
	count             int
	softExpireSeconds int
	interval          time.Duration
}

func main() {

	app := application{}

	flag.StringVar(&app.tokenURL, "tokenURL", "http://localhost:8080/oauth/token", "token URL")
	flag.StringVar(&app.clientID, "clientID", "admin", "client ID")
	flag.StringVar(&app.clientSecret, "clientSecret", "admin", "client secret")
	flag.StringVar(&app.scope, "scope", "", "scope")
	flag.StringVar(&app.targetURL, "targetURL", "https://httpbin.org/get", "target URL")
	flag.StringVar(&app.targetMethod, "targetMethod", "GET", "target method")
	flag.StringVar(&app.targetBody, "targetBody", "targetBody", "target body")
	flag.IntVar(&app.count, "count", 2, "how many requests to send")
	flag.IntVar(&app.softExpireSeconds, "softExpireSeconds", 10, "token soft expire in seconds")
	flag.DurationVar(&app.interval, "interval", 2*time.Second, "interval bewteen sends")

	flag.Parse()

	options := clientcredentials.Options{
		TokenURL:            app.tokenURL,
		ClientID:            app.clientID,
		ClientSecret:        app.clientSecret,
		Scope:               app.scope,
		HTTPClient:          http.DefaultClient,
		SoftExpireInSeconds: app.softExpireSeconds,
	}

	client := clientcredentials.New(options)

	for i := 1; i <= app.count; i++ {

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
}
