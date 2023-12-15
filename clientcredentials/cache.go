package clientcredentials

// TokenCache defines a cache interface for storing tokens.
type TokenCache interface {
	Get() Token
	Put(t Token)
	Expire()
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

func (mc *memoryCache) Expire() {
	mc.t.Expire()
}

// DefaultTokenCache provides default implementation for token cache.
var DefaultTokenCache = &memoryCache{}
