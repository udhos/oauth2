// Package cache provides cache implementations.
package cache

import (
	"fmt"
	"strings"

	"github.com/udhos/oauth2/cache/errorcache"
	"github.com/udhos/oauth2/cache/filecache"
	"github.com/udhos/oauth2/cache/rediscache"
	"github.com/udhos/oauth2/token"
)

// New creates cache from string.
func New(s string) (token.TokenCache, error) {
	switch {
	case s == "":
		return nil, nil
	case s == "error":
		return errorcache.New()
	case strings.HasPrefix(s, "file:"):
		return filecache.New(strings.TrimPrefix(s, "file:"))
	case strings.HasPrefix(s, "redis:"):
		return rediscache.New(strings.TrimPrefix(s, "redis:"))
	}
	return nil, fmt.Errorf("unknown cache: %s", s)
}
