package client

import (
  bulky "github.com/charmixer/bulky/client"
)

type Human struct {
  Id                   string `json:"id"                      validate:"required,uuid"`
  Username             string `json:"username"                validate:"required"`
  Password             string `json:"password,omitempty"      validate:"omitempty"`
  Name                 string `json:"name"                    validate:"required`
  Email                string `json:"email"                   validate:"required,email"`
  EmailConfirmedAt     int64  `json:"email_confirmed_at"`
  AllowLogin           bool   `json:"allow_login"             validate:"required"`
  TotpRequired         bool   `json:"totp_required"           `
  TotpSecret           string `json:"totp_secret"             `
  OtpRecoverCode       string `json:"otp_recover_code"        `
  OtpRecoverCodeExpire int64  `json:"otp_recover_code_expire" `
  OtpDeleteCode        string `json:"otp_delete_code"         `
  OtpDeleteCodeExpire  int64  `json:"otp_delete_code_expire"  `
}

type HumanAuthentication struct {
  Id                 string `json:"id"            validate:"omitempty,uuid"`
  Authenticated      bool   `json:"authenticated"`
  RedirectTo         string `json:"redirect_to"   validate:"omitempty,uri"`
  TotpRequired       bool   `json:"totp_required"`
  IsPasswordInvalid  bool `json:"is_password_invalid"`
  IdentityExists     bool `json:"identity_exists"`
}

type HumanRedirect struct {
  Id         string `json:"id"          validate:"required,uuid"`
  RedirectTo string `json:"redirect_to" validate:"required,uri"`
}

type HumanVerification struct {
  Id         string `json:"id"          validate:"required,uuid"`
  RedirectTo string `json:"redirect_to" validate:"required,uri"`
  Verified   bool   `json:"verified"`
}

type HumanLogout struct {
  SessionId string `json:"sid"`
  InitiatedByRelayingParty bool `json:"rp_initiated"`
  Id string `json:"id" validate:"required,uuid"`
  RequestUrl string `json:"request_url" validate:"required,uri"`
}

type Logout struct {
  RedirectTo string `json:"redirect_to" validate:"required,uri"`
}

// Endpoints

type CreateHumansResponse Human
type CreateHumansRequest struct {
  Id               string `json:"id"                 validate:"required,uuid"`
  Password         string `json:"password"           validate:"required"`
  Username         string `json:"username,omitempty" validate:"required"`
  Email            string `json:"email,omitempty"    validate:"required,email"`
  Name             string `json:"name,omitempty"     validate:"required"`
  AllowLogin       bool   `json:"allow_login"`
  EmailConfirmedAt int64  `json:"email_confirmed_at"`
}

type ReadHumansResponse []Human
type ReadHumansRequest struct {
  Id       string `json:"id,omitempty"        validate:"omitempty,uuid"`
  Email    string `json:"email,omitempty"     validate:"omitempty,email"`
  Username string `json:"username,omitempty"`
}

type UpdateHumansResponse Human
type UpdateHumansRequest struct {
  Id    string `json:"id" validate:"required,uuid"`
  Name  string `json:"name,omitempty"`
}

type DeleteHumansResponse HumanRedirect
type DeleteHumansRequest struct {
  Id string `json:"id" validate:"required,uuid"`
}

type UpdateHumansDeleteVerifyResponse HumanVerification
type UpdateHumansDeleteVerifyRequest struct {
  Id         string `json:"id"          validate:"required,uuid"`
  Code       string `json:"code"        validate:"required"`
  RedirectTo string `json:"redirect_to" validate:"required,uri"`
}

type UpdateHumansPasswordResponse Human
type UpdateHumansPasswordRequest struct {
  Id       string `json:"id"       validate:"required,uuid"`
  Password string `json:"password" validate:"required"`
}

type UpdateHumansTotpResponse Human
type UpdateHumansTotpRequest struct {
  Id           string `json:"id"            validate:"required,uuid"`
  TotpRequired bool   `json:"totp_required"`
  TotpSecret   string `json:"totp_secret"   validate:"required"`
}

type CreateHumansAuthenticateResponse HumanAuthentication
type CreateHumansAuthenticateRequest struct {
  Challenge    string `json:"challenge"                validate:"required"`
  Id           string `json:"id,omitempty"             validate:"omitempty,uuid"`
  Password     string `json:"password,omitempty"`
  OtpChallenge string `json:"otp_challenge,omitempty"     validate:"omitempty,uuid"`
  EmailChallenge string `json:"email_challenge,omitempty" validate:"omitempty,uuid"`
}

type CreateHumansRecoverResponse HumanRedirect
type CreateHumansRecoverRequest struct {
  Id string `json:"id" validate:"required,uuid"`
}

type UpdateHumansRecoverVerifyResponse HumanVerification
type UpdateHumansRecoverVerifyRequest struct {
  Id         string `json:"id"          validate:"required,uuid"`
  Code       string `json:"code"        validate:"required"`
  Password   string `json:"password"    validate:"required"`
  RedirectTo string `json:"redirect_to" validate:"required,uri"`
}

type CreateHumansLogoutResponse Logout
type CreateHumansLogoutRequest struct {
  IdToken    string `json:"id_token"              validate:"required"`
  State      string `json:"state"                 validate:"required"`
  RedirectTo string `json:"redirect_to,omitempty" validate:"omitempty,uri"`
}

type ReadHumansLogoutResponse HumanLogout
type ReadHumansLogoutRequest struct {
  Challenge string `json:"challenge" validate:"required"`
}

type UpdateHumansLogoutAcceptResponse HumanRedirect
type UpdateHumansLogoutAcceptRequest struct {
  Challenge string `json:"challenge" validate:"required"`
}

func CreateHumans(client *IdpClient, url string, requests []CreateHumansRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "POST", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}


func ReadHumans(client *IdpClient, url string, requests []ReadHumansRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "GET", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}


func UpdateHumans(client *IdpClient, url string, requests []UpdateHumansRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "PUT", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}


func DeleteHumans(client *IdpClient, url string, requests []DeleteHumansRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "DELETE", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}


func DeleteHumansVerify(client *IdpClient, url string, requests []UpdateHumansDeleteVerifyRequest)  (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "PUT", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}


func UpdateHumansPassword(client *IdpClient, url string, requests []UpdateHumansPasswordRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "PUT", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}


func UpdateHumansTotp(client *IdpClient, url string, requests []UpdateHumansTotpRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "PUT", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}


func CreateHumansAuthenticate(client *IdpClient, url string, requests []CreateHumansAuthenticateRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "POST", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}


func RecoverHumans(client *IdpClient, url string, requests []CreateHumansRecoverRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "PUT", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}


func RecoverHumansVerify(client *IdpClient, url string, requests []UpdateHumansRecoverVerifyRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "POST", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}

func CreateHumansLogout(client *IdpClient, url string, requests []CreateHumansLogoutRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "POST", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}

func ReadHumansLogout(client *IdpClient, url string, requests []ReadHumansLogoutRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "GET", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}


func UpdateHumansLogoutAccept(client *IdpClient, url string, requests []UpdateHumansLogoutAcceptRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "PUT", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}

