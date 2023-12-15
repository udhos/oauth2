package clientcredentials

import (
	"log"
	"time"
)

// Token holds a token.
type Token struct {
	Value     string
	Deadline  time.Time
	Expirable bool
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
