# oauth2

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
```

# References

[Cache token / transport confusion](https://github.com/golang/oauth2/issues/84)
