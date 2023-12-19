[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/oauth2/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/oauth2)](https://goreportcard.com/report/github.com/udhos/oauth2)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/oauth2.svg)](https://pkg.go.dev/github.com/udhos/oauth2)

# oauth2

https://github.com/udhos/oauth2 implements the oauth2 client_credentials flow with singleflight and plugable cache interface.

* [Features](#features)
* [Usage](#usage)
* [Example client](#example-client)
* [Test with example client](#test-with-example-client)
* [Test singleflight with example client](#test-singleflight-with-example-client)
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
- [X] singleflight.
- [X] debug logs.

# Usage

```golang
import "github.com/udhos/oauth2/clientcredentials"
import "github.com/udhos/oauth2/cache/rediscache"

cache, errRedis := rediscache.New("localhost:6379::my-cache-key")
if errRedis != nil {
    log.Fatalf("redis: %v", errRedis)
}

options := clientcredentials.Options{
    TokenURL:     "https://token-server/token",
    ClientID:     "client-id",
    ClientSecret: "client-secret",
    Scope:        "scope1 scope2",
    HTTPClient:   http.DefaultClient,
    Cache:        cache,
}

client := clientcredentials.New(options)

req, errReq := http.NewRequestWithContext(context.TODO(), "GET", "https://server/resource", nil)
if errReq != nil {
    log.Fatalf("request: %v", errReq)
}

resp, errDo := client.Do(req)
if errDo != nil {
    log.Fatalf("do: %v", errDo)
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

oauth2-client-example -tokenURL https://login-demo.curity.io/oauth/v2/oauth-token -clientID demo-backend-client -clientSecret MJlO3binatD9jk1 -cache redis:localhost:6379::

oauth2-client-example -tokenURL https://login-demo.curity.io/oauth/v2/oauth-token -clientID demo-backend-client -clientSecret MJlO3binatD9jk1 -cache redis:localhost:6379::oauth2-client-example
```

# Test singleflight with example client

Run token server at: http://localhost:8080/oauth/token

Run server at: http://localhost:8000/v1/hello

Cache error makes sure every request retrieves a new token: `-cache error`.

1. Send requests with singlefligh:

```bash
oauth2-client-example -tokenURL http://localhost:8080/oauth/token -targetURL http://localhost:8000/v1/hello -cache error -interval 0 -concurrent -count 10
```

2. Send requests WITHOUT singlefligh:

```bash
oauth2-client-example -tokenURL http://localhost:8080/oauth/token -targetURL http://localhost:8000/v1/hello -cache error -interval 0 -concurrent -count 10 -disableSingleflight
```

# Test caches

Set the cache with the env var `CACHE`, then run the tests.

```bash
# Test file cache
export CACHE=file:/tmp/cache
go test -race ./...

# Test redis cache
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
