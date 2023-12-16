package clientcredentials

import "sync"

// TokenCache defines a cache interface for storing tokens.
type TokenCache interface {
	Get() Token
	Put(t Token)
	Expire()
}

type memoryCache struct {
	t     Token
	mutex sync.Mutex
}

func (mc *memoryCache) Get() Token {
	mc.mutex.Lock()
	t := mc.t
	mc.mutex.Unlock()
	return t
}

func (mc *memoryCache) Put(t Token) {
	mc.mutex.Lock()
	mc.t = t
	mc.mutex.Unlock()
}

func (mc *memoryCache) Expire() {
	mc.mutex.Lock()
	mc.t.Expire()
	mc.mutex.Unlock()
}

// DefaultTokenCache provides default implementation for token cache.
var DefaultTokenCache = &memoryCache{}
