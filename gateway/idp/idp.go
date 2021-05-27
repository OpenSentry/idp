package idp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"io"
	"time"
)

type RecoverChallenge struct {
	Id         string
	Code       string
	Expire     int64
	RedirectTo string
}

type DeleteChallenge struct {
	Id         string
	Code       string
	Expire     int64
	RedirectTo string
}

type ChallengeCode struct {
	Code string
}

func ValidatePassword(storedPassword string, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err != nil {
		return false, err
	}
	return true, nil
}

func CreatePassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func ValidateOtp(otp string, secret string) (bool, error) {
	valid := totp.Validate(otp, secret)
	return valid, nil
}

func CreateDeleteChallenge(url string, identity Human, challengeTimeoutInSeconds int64) (DeleteChallenge, error) {
	code, err := GenerateRandomDigits(6)
	if err != nil {
		return DeleteChallenge{}, err
	}

	timeout := time.Duration(challengeTimeoutInSeconds)
	expirationTime := time.Now().Add(timeout * time.Second)
	expiresAt := expirationTime.Unix()
	redirectTo := url

	return DeleteChallenge{
		Id:         identity.Id,
		Code:       code,
		Expire:     expiresAt,
		RedirectTo: redirectTo,
	}, nil
}

func CreateRecoverChallenge(url string, identity Human, challengeTimeoutInSeconds int64) (RecoverChallenge, error) {
	code, err := GenerateRandomDigits(6)
	if err != nil {
		return RecoverChallenge{}, err
	}

	timeout := time.Duration(challengeTimeoutInSeconds)
	expirationTime := time.Now().Add(timeout * time.Second)
	expiresAt := expirationTime.Unix()
	redirectTo := url

	return RecoverChallenge{
		Id:         identity.Id,
		Code:       code,
		Expire:     expiresAt,
		RedirectTo: redirectTo,
	}, nil
}

func CreateChallengeCode() (ChallengeCode, error) {
	code, err := GenerateRandomDigits(6)
	if err != nil {
		return ChallengeCode{}, err
	}
	return ChallengeCode{Code: code}, nil
}

var table = [...]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}

func GenerateRandomDigits(max int) (string, error) {
	b := make([]byte, max)
	n, err := io.ReadAtLeast(rand.Reader, b, max)
	if n != max {
		return "", err
	}
	for i := 0; i < len(b); i++ {
		b[i] = table[int(b[i])%len(table)]
	}
	return string(b), nil
}

// Enforce AES-256 by using 32 byte string as key param
func Encrypt(str string, key string) (string, error) {

	bKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}

	bStr := []byte(str)
	bEncryptedStr, err := encrypt(bStr, bKey)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(bEncryptedStr), nil
}

// Enforce AES-256 by using 32 byte string as key param
func Decrypt(str string, key string) (string, error) {

	bKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}

	bStr, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}

	bDecryptedStr, err := decrypt(bStr, bKey)
	if err != nil {
		return "", err
	}
	return string(bDecryptedStr), nil
}

// The key argument should be 32 bytes to use AES-256
func encrypt(plaintext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// The key argument should be 32 bytes to use AES-256
func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
