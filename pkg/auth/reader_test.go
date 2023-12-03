//go:build utest

package auth

import (
	"testing"

	"github.com/kxplxn/goteam/pkg/assert"
)

// TestBearerTokenReader tests the Read method of BearerTokenReader.
func TestBearerTokenReader(t *testing.T) {
	sut := NewBearerTokenReader()

	for _, c := range []struct {
		name       string
		authHeader string
		wantToken  string
		wantErr    error
	}{
		{name: "InvalidHeader1", authHeader: "Basic ABCDEFGH", wantToken: ""},
		{name: "InvalidHeader2", authHeader: "Bear ABCDEFGH", wantToken: ""},
		{name: "InvalidHeader3", authHeader: "Bearer ABCD EFGH", wantToken: ""},
		{name: "EmptyToken", authHeader: "Bearer ", wantToken: ""},
		{name: "Success", authHeader: "Bearer ABCDEFGH", wantToken: "ABCDEFGH"},
	} {
		t.Run(c.name, func(t *testing.T) {
			token := sut.Read(c.authHeader)

			assert.Equal(t.Error, token, c.wantToken)
		})
	}
}