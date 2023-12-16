[![license](http://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/udhos/oauth2/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/udhos/oauth2)](https://goreportcard.com/report/github.com/udhos/oauth2)
[![Go Reference](https://pkg.go.dev/badge/github.com/udhos/oauth2.svg)](https://pkg.go.dev/github.com/udhos/oauth2)

# oauth2

# Features

- [X] oauth2 client_credentials flow.
- [X] plugable cache.
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

req, errReq := http.NewRequestWithContext(context.TODO(), "GET" "https://server/resource", nil)
if errReq != nil {
    log.Fatalf("%s: request: %v", label, errReq)
}

resp, errDo := client.Do(req)
if errDo != nil {
    log.Fatalf("%s: do: %v", label, errDo)
}
defer resp.Body.Close()
```

# Test

https://oauth.tools/collection/1599045253169-GHF

```bash
go install ./...

oauth2-client-example -tokenURL https://login-demo.curity.io/oauth/v2/oauth-token -clientID demo-backend-client -clientSecret MJlO3binatD9jk1

oauth2-client-example -tokenURL https://login-demo.curity.io/oauth/v2/oauth-token -clientID demo-backend-client -clientSecret MJlO3binatD9jk1 -cache file:/tmp/cache
```

# References

[Cache token / transport confusion](https://github.com/golang/oauth2/issues/84)
