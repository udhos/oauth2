package token

import (
	"sync"
)

// TokenCache defines a cache interface for storing tokens.
type TokenCache interface {
	Get() (Token, error)
	Put(t Token) error
	Expire() error
}

// memoryCache implements a memory cache.
type memoryCache struct {
	t     Token
	mutex sync.Mutex
}

// Get retrieves token from cache.
func (mc *memoryCache) Get() (Token, error) {
	mc.mutex.Lock()
	t := mc.t
	mc.mutex.Unlock()
	return t, nil
}

// Put inserts token into cache.
func (mc *memoryCache) Put(t Token) error {
	mc.mutex.Lock()
	mc.t = t
	mc.mutex.Unlock()
	return nil
}

// Expire invalidates token in cache.
func (mc *memoryCache) Expire() error {
	mc.mutex.Lock()
	mc.t.Expire()
	mc.mutex.Unlock()
	return nil
}

// DefaultTokenCache provides default implementation for token cache.
var DefaultTokenCache = &memoryCache{}
