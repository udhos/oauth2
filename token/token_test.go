package token

import (
	"testing"
	"time"
)

func TestToken(t *testing.T) {
	tk := Token{
		Value:    "abc",
		Deadline: time.Now(),
	}

	buf, errJSON := tk.ExportJSON()
	if errJSON != nil {
		t.Errorf("export: %v", errJSON)
	}

	tk2, errNew := NewTokenFromJSON(buf)
	if errNew != nil {
		t.Errorf("import: %v", errNew)
	}

	if tk.Value != tk2.Value {
		t.Errorf("value: '%s' != '%s'", tk.Value, tk2.Value)
	}

	if tk.Expirable != tk2.Expirable {
		t.Errorf("expirable: %t != %t'", tk.Expirable, tk2.Expirable)
	}

	if !tk.Deadline.Equal(tk2.Deadline) {
		t.Errorf("deadline: %v != %v'", tk.Deadline, tk2.Deadline)
	}
}
