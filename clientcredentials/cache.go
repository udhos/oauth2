package clientcredentials

import "sync"

// TokenCache defines a cache interface for storing tokens.
type TokenCache interface {
	Get() (Token, error)
	Put(t Token) error
	Expire() error
}

type memoryCache struct {
	t     Token
	mutex sync.Mutex
}

func (mc *memoryCache) Get() (Token, error) {
	mc.mutex.Lock()
	t := mc.t
	mc.mutex.Unlock()
	return t, nil
}

func (mc *memoryCache) Put(t Token) error {
	mc.mutex.Lock()
	mc.t = t
	mc.mutex.Unlock()
	return nil
}

func (mc *memoryCache) Expire() error {
	mc.mutex.Lock()
	mc.t.Expire()
	mc.mutex.Unlock()
	return nil
}

// DefaultTokenCache provides default implementation for token cache.
var DefaultTokenCache = &memoryCache{}
