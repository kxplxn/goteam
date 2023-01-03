package login

import "golang.org/x/crypto/bcrypt"

// Comparer represents a type that compares the given bytes to the given
// string value. The first return value should be true if they are a match and
// no errors occur during the comparison process.
type Comparer interface {
	Compare([]byte, string) (bool, error)
}

// ComparerHash is used to compare a given hashed bytes with a plaintext string.
// If the hash was originally created from the plaintext value and no errors
// occured during comparison, the first return value is true.
type ComparerHash struct{}

// NewComparerHash is the constructor for ComparerHash.
func NewComparerHash() *ComparerHash { return &ComparerHash{} }

// Compare compares the given hashed bytes with the given plaintext string. The
// first return value communicates whether it was a match. The second return
// value is for any errors that may ocur during comparison.
func (c *ComparerHash) Compare(hash []byte, plaintext string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(hash, []byte(plaintext))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}