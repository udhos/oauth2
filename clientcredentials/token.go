package clientcredentials

import (
	"log"
	"time"
)

// Token holds a token.
type Token struct {
	Value    string
	Deadline *time.Time // nil deadline is always valid
}

// IsValid checks whether token is valid.
func (t Token) IsValid(toleration time.Duration) bool {
	log.Printf("token toleration: %v", toleration)
	if t.Deadline == nil {
		return true
	}
	return t.Deadline.After(time.Now().Add(toleration))
}
