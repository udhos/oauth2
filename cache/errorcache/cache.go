// Package errorcache implements a cache.
package errorcache

import (
	"errors"

	"github.com/udhos/oauth2/token"
)

// Cache holds cache client.
type Cache struct {
}

// New creates a new cache client.
func New() (*Cache, error) {
	return &Cache{}, nil
}

var errAlways = errors.New("errorcache error always")

// Get retrieves token from cache.
func (c *Cache) Get() (token.Token, error) {
	return token.Token{}, errAlways
}

// Put inserts token into cache.
func (c *Cache) Put(_ token.Token) error {
	return errAlways
}

// Expire invalidates token in cache.
func (c *Cache) Expire() error {
	return errAlways
}
