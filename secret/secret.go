package secret

import (
  "errors"
	"crypto/rand"
	"io"
  "encoding/base64"
)

const (
  RECOMMENDED_CLIENT_SECRET_ENTROPY_IN_BYTES = 32 // 256 bits
  MIN_CLIENT_SECRET_ENTROPY_IN_BYTES = 16 //  128 bits
)

// Configure base64 encoding to use url encode (+ becomes escaped) and no padding
var b64 = base64.URLEncoding.WithPadding(base64.NoPadding)

// RandomBytes returns n random bytes by reading from crypto/rand.Reader
func RandomBytes(n int) ([]byte, error) {
	bytes := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return []byte{}, err
	}
	return bytes, nil
}

// @SecurityConsideration
// In order to mitigate risk of guessing attacks the level of entropy should be >=128 bits long and
// constructed from a cryptographically strong random or pseudo-random number sequence
func CreateClientSecret(entropyInBytes int) (string, error) {
  if entropyInBytes < MIN_CLIENT_SECRET_ENTROPY_IN_BYTES {
    return "", errors.New("Not enough entropy. At least 128 bits required")
  }
  secret, err := RandomBytes(entropyInBytes)
  if err != nil {
    return "", err
  }
  return b64.EncodeToString(secret), nil
}