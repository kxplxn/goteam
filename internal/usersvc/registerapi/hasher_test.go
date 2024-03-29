//go:build utest

package registerapi

import (
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/kxplxn/goteam/pkg/assert"
)

// TestPasswordHasher tests the Hash method of the password hasher. It uses
// bcrypt.CompareHashAndPassword to assert that the result was generated from
// the given plaintext and doesn't match another plaintext string.
func TestPasswordHasher(t *testing.T) {
	sut := NewPasswordHasher()

	for _, c := range []struct {
		name           string
		inPlaintext    string
		matchPlaintext string
		wantErr        error
	}{
		{
			name:           "NoMatch",
			inPlaintext:    "password",
			matchPlaintext: "differentPassword",
			wantErr:        bcrypt.ErrMismatchedHashAndPassword,
		},
		{
			name:           "Match",
			inPlaintext:    "password",
			matchPlaintext: "password",
			wantErr:        nil,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			hash, err := sut.Hash(c.inPlaintext)
			assert.Nil(t.Fatal, err)

			err = bcrypt.CompareHashAndPassword(hash, []byte(c.matchPlaintext))
			assert.Equal(t.Error, err, c.wantErr)
		})
	}
}
