// Package filecache implements a cache.
package filecache

import (
	"os"
	"sync"

	"github.com/udhos/oauth2/token"
)

// Cache holds cache client.
type Cache struct {
	filename string
	mutex    sync.Mutex
}

// New creates a new cache client.
func New(filename string) (*Cache, error) {
	return &Cache{filename: filename}, nil
}

// Get retrieves token from cache.
func (c *Cache) Get() (token.Token, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return tokenFromFile(c.filename)
}

func tokenFromFile(filename string) (token.Token, error) {
	buf, errRead := os.ReadFile(filename)
	if errRead != nil {
		return token.Token{}, errRead
	}
	return token.NewTokenFromJSON(buf)
}

// Put inserts token into cache.
func (c *Cache) Put(t token.Token) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return saveToken(t, c.filename)
}

func saveToken(t token.Token, filename string) error {
	out, errOpen := os.Create(filename)
	if errOpen != nil {
		return errOpen
	}
	buf, errJSON := t.ExportJSON()
	if errJSON != nil {
		return errJSON
	}
	_, errWrite := out.Write(buf)
	return errWrite
}

// Expire invalidates token in cache.
func (c *Cache) Expire() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	t, errGet := tokenFromFile(c.filename)
	if errGet != nil {
		return errGet
	}
	t.Expire()
	return saveToken(t, c.filename)
}
