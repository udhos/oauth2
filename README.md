[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/oauth2/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/oauth2)](https://goreportcard.com/report/github.com/udhos/oauth2)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/oauth2.svg)](https://pkg.go.dev/github.com/udhos/oauth2)

# oauth2

* [Features](#features)
* [Usage](#usage)
* [Example client](#example-client)
* [Test with example client](#test-with-example-client)
* [Test caches](#test-caches)
* [Development](#development)
* [References](#references)

Created by [gh-md-toc](https://github.com/ekalinin/github-markdown-toc.go)

# Features

- [X] oauth2 client_credentials flow.
- [X] plugable cache.
- [X] default memory cache.
- [X] filesystem cache.
- [X] testing-only error cache.
- [X] redis cache.
- [ ] singleflight.

# Usage

```golang
import "github.com/udhos/oauth2/clientcredentials"

options := clientcredentials.Options{
    TokenURL:            "https://token-server/token",
    ClientID:            "client-id",
    ClientSecret:        "client-secret",
    Scope:               "scope1 scope2",
    HTTPClient:          http.DefaultClient,
}

client := clientcredentials.New(options)

req, errReq := http.NewRequestWithContext(context.TODO(), "GET", "https://server/resource", nil)
if errReq != nil {
    log.Fatalf("%s: request: %v", label, errReq)
}

resp, errDo := client.Do(req)
if errDo != nil {
    log.Fatalf("%s: do: %v", label, errDo)
}
defer resp.Body.Close()
```

# Example client

See [cmd/oauth2-client-example/main.go](cmd/oauth2-client-example/main.go).

# Test with example client

Test using this token server: https://oauth.tools/collection/1599045253169-GHF

```bash
go install github.com/udhos/oauth2/cmd/oauth2-client-example@latest

oauth2-client-example -tokenURL https://login-demo.curity.io/oauth/v2/oauth-token -clientID demo-backend-client -clientSecret MJlO3binatD9jk1

oauth2-client-example -tokenURL https://login-demo.curity.io/oauth/v2/oauth-token -clientID demo-backend-client -clientSecret MJlO3binatD9jk1 -cache file:/tmp/cache

oauth2-client-example -tokenURL https://login-demo.curity.io/oauth/v2/oauth-token -clientID demo-backend-client -clientSecret MJlO3binatD9jk1 -cache error

oauth2-client-example -tokenURL https://login-demo.curity.io/oauth/v2/oauth-token -clientID demo-backend-client -clientSecret MJlO3binatD9jk1 -cache redis:localhost:6379::oauth2-client-example
```

# Test caches

Set the cache with the env var `CACHE`, then run the tests.

```bash
export CACHE=file:/tmp/cache
go test -race ./...

./run-redis-local.sh
export CACHE=redis:localhost:6379::oauth2-client-example
go test -race ./...
```

# Development

```bash
git clone https://github.com/udhos/oauth2
cd oauth2
./build.sh
```

# References

[Cache token / transport confusion](https://github.com/golang/oauth2/issues/84)
