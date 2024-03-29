// Package cookie contains code for generating, validating, and decoding JWTs
// into/from http cookies.
package cookie

import (
	"errors"
	"net/http"
)

// Encoder defines a type that can be used to encode a JWT.
type Encoder[T any] interface{ Encode(T) (http.Cookie, error) }

// Decoder defines a type that can be used to decode a JWT.
type Decoder[T any] interface{ Decode(http.Cookie) (T, error) }

// StringDecoder defines a type that can be used to decode a JWT from a string.
type StringDecoder[T any] interface{ Decode(string) (T, error) }

// ErrInvalid means that the given cookie was invalid.
var ErrInvalid = errors.New("invalid cookie")
