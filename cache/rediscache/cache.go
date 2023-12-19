// Package rediscache implements a cache.
package rediscache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/udhos/oauth2/token"
)

// Cache holds cache client.
type Cache struct {
	key         string
	redisClient *redis.Client
}

// Options define redis options.
type Options struct {
	// Format: RedisString = <host>:<port>:<password>:<key>
	// Example: RedisString = localhost:6379::oauth2-client-example
	// Leave <key> empty for auto generation.
	RedisString string
	TokenURL    string // only used if key is empty for auto generation
	ClientID    string // only used if key is empty for auto generation
}

// New creates a new cache client.
func New(options Options) (*Cache, error) {
	fields := strings.SplitN(options.RedisString, ":", 4)
	if len(fields) != 4 {
		return nil, fmt.Errorf("4 fields are required, but got: %d", len(fields))
	}
	host := fields[0]
	port := fields[1]
	password := fields[2]
	key := fields[3]

	if host == "" {
		host = "localhost"
	}

	if port == "" {
		port = "6379"
	}

	if key == "" {
		key = "github.com/udhos/oauth2|" + options.TokenURL + "|" + options.ClientID + "|token"
	}

	c := Cache{
		redisClient: redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", host, port),
			Password: password,
			DB:       0,
		}),
		key: key,
	}
	return &c, nil
}

var errRedisCacheKeyNotFound = errors.New("redis cache error: key not found")

// getKey gets the redis key for storing the token.
func (c *Cache) getKey() string {
	return c.key
}

// Get retrieves token from cache.
func (c *Cache) Get() (token.Token, error) {

	var t token.Token

	cmdID := c.redisClient.Get(context.TODO(), c.getKey())
	errID := cmdID.Err()
	if errID == redis.Nil {
		return t, errRedisCacheKeyNotFound
	}
	if errID != nil {
		return t, cmdID.Err()
	}

	buf, _ := cmdID.Bytes()

	return token.NewTokenFromJSON(buf)
}

// Put inserts token into cache.
func (c *Cache) Put(t token.Token) error {

	buf, errJSON := t.ExportJSON()
	if errJSON != nil {
		return errJSON
	}

	var expiration time.Duration

	if t.Expirable {
		expiration = time.Until(t.Deadline) + time.Minute // token remaining TTL + 1 minute
	}

	errSet := c.redisClient.Set(context.TODO(), c.getKey(), buf, expiration)

	return errSet.Err()
}

// Expire invalidates token in cache.
func (c *Cache) Expire() error {

	t, errGet := c.Get()
	if errGet != nil {
		return errGet
	}

	t.Expire()

	return c.Put(t)
}
