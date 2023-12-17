// Package token implements Token.
package token

import (
	"encoding/json"
	"log"
	"time"
)

// Token holds a token.
type Token struct {
	Value    string    `json:"value"`
	Deadline time.Time `json:"deadline"`

	// By default Token is non-expirable.
	// It becomes expirable when either Expire() or SetExpiration() is applied.
	// SetExpiration() is used to take explicit expires_in field into effect.
	// Expire() is used to invalidate the Token, since the server
	// refused it and a new one must be retrieved.
	//
	Expirable bool `json:"expirable"`
}

// NewTokenFromJSON creates token from json.
func NewTokenFromJSON(buf []byte) (Token, error) {
	var t Token
	err := json.Unmarshal(buf, &t)
	if err != nil {
		return t, err
	}
	return t, nil
}

// ExportJSON exports token as json.
func (t Token) ExportJSON() ([]byte, error) {
	return json.Marshal(t)
}

// IsValid checks whether token is valid.
func (t *Token) IsValid(now time.Time, softExpire time.Duration) bool {
	remain := t.Deadline.Sub(now)
	valid := !t.Expirable || t.Deadline.After(now.Add(softExpire))
	log.Printf("token softExpire=%v remain=%v expirable=%v valid=%v",
		softExpire, remain, t.Expirable, valid)
	return valid
}

// Expire expires the token.
func (t *Token) Expire() {
	t.Expirable = true
	t.Deadline = expired
}

// SetExpiration schedules token expiration time.
func (t *Token) SetExpiration(deadline time.Time) {
	t.Expirable = true
	t.Deadline = deadline
}

var expired = time.Time{}
