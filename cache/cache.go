// Package cache provides cache implementations.
package cache

import (
	"fmt"
	"strings"

	"github.com/udhos/oauth2/cache/errorcache"
	"github.com/udhos/oauth2/cache/filecache"
	"github.com/udhos/oauth2/clientcredentials"
)

// New creates cache from string.
func New(s string) (clientcredentials.TokenCache, error) {
	switch {
	case s == "":
		return nil, nil
	case s == "error":
		return errorcache.New()
	case strings.HasPrefix(s, "file:"):
		return filecache.New(strings.TrimPrefix(s, "file:"))
	}
	return nil, fmt.Errorf("unknown cache: %s", s)
}
