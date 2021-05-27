package client

import (
	bulky "github.com/charmixer/bulky/client"
)

type ConfirmationType int

const (
	ConfirmIdentity ConfirmationType = iota + 1 // Start from one so required input validation will work.
	ConfirmIdentityDeletion
	ConfirmIdentityRecovery
	ConfirmIdentityControlOfEmail
	ConfirmIdentityControlOfEmailDuringChange
)

func (d ConfirmationType) String() string {
	return [...]string{"ConfirmIdentity", "ConfirmIdentityDeletion", "ConfirmIdentityRecovery", "ConfirmIdentityControlOfEmail", "ConfirmIdentityControlOfEmailDuringChange"}[d]
}

type OTPType int

const (
	OTP OTPType = OTPType(iota)
	TOTP
)

func (d OTPType) String() string {
	return [...]string{"OTP", "TOTP"}[d]
}

type Challenge struct {
	OtpChallenge     string `json:"otp_challenge"  validate:"required"`
	ConfirmationType int    `json:"confirmation_type" validate:"numeric"`

	Subject    string `json:"sub"            validate:"required,uuid"`
	Audience   string `json:"aud"            validate:"required"`
	IssuedAt   int64  `json:"iat"            validate:"required"`
	ExpiresAt  int64  `json:"exp"            validate:"required"`
	TTL        int64  `json:"ttl"            validate:"required"`
	RedirectTo string `json:"redirect_to"    validate:"required,url"`
	CodeType   int64  `json:"code_type"`
	Code       string `json:"code,omitempty"`

	VerifiedAt int64 `json:"verified_at"`

	Data string `json:"data,omitempty"`
}

type ChallengeVerification struct {
	OtpChallenge string `json:"otp_challenge" validate:"required"`
	Verified     bool   `json:"verified"      `
	RedirectTo   string `json:"redirect_to"   validate:"required,url"`
}

type CreateChallengesResponse Challenge
type CreateChallengesRequest struct {
	ConfirmationType int `json:"confirmation_type" validate:"numeric"`

	Subject    string `json:"sub"         validate:"required,uuid"`
	Audience   string `json:"aud"         validate:"required"`
	TTL        int64  `json:"ttl"         validate:"required"`
	RedirectTo string `json:"redirect_to" validate:"required,url"`
	CodeType   int64  `json:"code_type"   validate:"required"`
	Code       string `json:"code"        validate:"required"`

	Email string `json:"email,omitempty" validate:"omitempty,email"`
}

type ReadChallengesResponse []Challenge
type ReadChallengesRequest struct {
	OtpChallenge string `json:"otp_challenge" validate:"required"`
}

type UpdateChallengesVerifyResponse ChallengeVerification
type UpdateChallengesVerifyRequest struct {
	OtpChallenge string `json:"otp_challenge" validate:"required"`
	Code         string `json:"code"          validate:"required"`
}

func ReadChallenges(client *IdpClient, url string, requests []ReadChallengesRequest) (status int, responses bulky.Responses, err error) {
	status, err = handleRequest(client, requests, "GET", url, &responses)

	if err != nil {
		return status, nil, err
	}

	return status, responses, nil
}

func CreateChallenges(client *IdpClient, url string, requests []CreateChallengesRequest) (status int, responses bulky.Responses, err error) {
	status, err = handleRequest(client, requests, "POST", url, &responses)

	if err != nil {
		return status, nil, err
	}

	return status, responses, nil
}

func VerifyChallenges(client *IdpClient, url string, requests []UpdateChallengesVerifyRequest) (status int, responses bulky.Responses, err error) {
	status, err = handleRequest(client, requests, "PUT", url, &responses)

	if err != nil {
		return status, nil, err
	}

	return status, responses, nil
}
