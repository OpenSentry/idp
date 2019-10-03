package client

import (
  "bytes"
  "encoding/json"
)

type Human struct {
  Id                   string `json:"id"                      validate:"required,uuid"`
  Username             string `json:"username"                validate:"required"`
  Password             string `json:"password"                validate:"required"`
  Name                 string `json:"name"                    validate:"required`
  Email                string `json:"email"                   validate:"required,email"`
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

// Endpoints

type CreateHumansRequest struct {
  Password   string `json:"password"           validate:"required"`
  Username   string `json:"username,omitempty" validate:"required"`
  Email      string `json:"email,omitempty"    validate:"required,email"`
  Name       string `json:"name,omitempty"     validate:"required"`
  AllowLogin bool   `json:"allow_login"`
}

type CreateHumansResponse struct {
  BulkResponse
  Ok Human `json:"ok,omitempty" validate:"dive"`
}

type ReadHumansRequest struct {
  Id       string `json:"id,omitempty"        validate:"omitempty,uuid"`
  Email    string `json:"email,omitempty"     validate:"omitempty,email"`
  Username string `json:"username,omitempty"`
}

type ReadHumansResponse struct {
  BulkResponse
  Ok []Human `json:"ok,omitempty" validate:"dive"`
}

type UpdateHumansRequest struct {
  Id    string `json:"id" validate:"required,uuid"`
  Email string `json:"email,omitempty" validate:"email"`
  Name  string `json:"name,omitempty"`
}

type UpdateHumansResponse struct {
  BulkResponse
  Ok Human `json:"ok,omitempty" validate:"dive"`
}

type DeleteHumansRequest struct {
  Id string `json:"id" validate:"required,uuid"`
}

type DeleteHumansResponse struct {
  BulkResponse
  Ok HumanRedirect `json:"ok,omitempty" validate:"dive"`
}

type UpdateHumansDeleteVerifyRequest struct {
  Id         string `json:"id"          validate:"required,uuid"`
  Code       string `json:"code"        validate:"required"`
  RedirectTo string `json:"redirect_to" validate:"required,uri"`
}

type UpdateHumansDeleteVerifyResponse struct {
  BulkResponse
  Ok HumanVerification `json:"ok,omitempty" validate:"dive"`
}

type UpdateHumansPasswordRequest struct {
  Id       string `json:"id"       validate:"required,uuid"`
  Password string `json:"password" validate:"required"`
}

type UpdateHumansPasswordResponse struct {
  BulkResponse
  Ok Human `json:"ok,omitempty" validate:"dive"`
}

type UpdateHumansTotpRequest struct {
  Id           string `json:"id"            validate:"required,uuid"`
  TotpRequired bool   `json:"totp_required"`
  TotpSecret   string `json:"totp_secret"   validate:"required"`
}

type UpdateHumansTotpResponse struct {
  BulkResponse
  Ok Human `json:"ok,omitempty" validate:"dive"`
}

type CreateHumansAuthenticateRequest struct {
  Challenge    string `json:"challenge"                validate:"required"`
  Id           string `json:"id,omitempty"             validate:"omitempty,uuid"`
  Password     string `json:"password,omitempty"`
  OtpChallenge string `json:"otp_challenge,omitempty"`
}

type CreateHumansAuthenticateResponse struct {
  BulkResponse
  Ok HumanAuthentication `json:"ok,omitempty" validate:"dive"`
}

type CreateHumansRecoverRequest struct {
  Id string `json:"id" validate:"required,uuid"`
}

type CreateHumansRecoverResponse struct {
  BulkResponse
  Ok HumanRedirect `json:"ok,omitempty" validate:"dive"`
}

type UpdateHumansRecoverVerifyRequest struct {
  Id         string `json:"id"          validate:"required,uuid"`
  Code       string `json:"code"        validate:"required"`
  Password   string `json:"password"    validate:"required"`
  RedirectTo string `json:"redirect_to" validate:"required,uri"`
}

type UpdateHumansRecoverVerifyResponse struct {
  BulkResponse
  Ok HumanVerification `json:"ok,omitempty" validate:"dive"`
}

type CreateHumansLogoutRequest struct {
  Challenge string `json:"challenge" validate:"required,uuid"`
}

type CreateHumansLogoutResponse struct {
  BulkResponse
  Ok HumanRedirect `json:"ok,omitempty" validate:"dive"`
}

func CreateHumans(client *IdpClient, url string, requests []CreateHumansRequest) (int, []CreateHumansResponse, error) {
  var response []CreateHumansResponse

  body, err := json.Marshal(requests)
  if err != nil {
    return 999, nil, err // Client system was unable marshal request
  }

  status, responseData, err := callService(client, "POST", url, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(responseData, &response)
  if err != nil {
    return 666, nil, err // Client system was unable to unmarshal request, but server already executed
  }

  return status, response, nil
}

func ReadHumans(client *IdpClient, url string, requests []ReadHumansRequest) (int, []ReadHumansResponse, error) {
  var response []ReadHumansResponse

  body, err := json.Marshal(requests)
  if err != nil {
    return 999, nil, err // Client system was unable marshal request
  }

  status, responseData, err := callService(client, "GET", url, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(responseData, &response)
  if err != nil {
    return 666, nil, err // Client system was unable to unmarshal request, but server already executed
  }

  return status, response, nil
}

func UpdateHumans(client *IdpClient, url string, requests []UpdateHumansRequest) (int, []UpdateHumansResponse, error) {
  var response []UpdateHumansResponse

  body, err := json.Marshal(requests)
  if err != nil {
    return 999, nil, err // Client system was unable marshal request
  }

  status, responseData, err := callService(client, "PUT", url, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(responseData, &response)
  if err != nil {
    return 666, nil, err // Client system was unable to unmarshal request, but server already executed
  }

  return status, response, nil
}

func DeleteHumans(client *IdpClient, url string, requests []DeleteHumansRequest) (int, []DeleteHumansResponse, error) {
  var response []DeleteHumansResponse

  body, err := json.Marshal(requests)
  if err != nil {
    return 999, nil, err // Client system was unable marshal request
  }

  status, responseData, err := callService(client, "DELETE", url, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(responseData, &response)
  if err != nil {
    return 666, nil, err // Client system was unable to unmarshal request, but server already executed
  }

  return status, response, nil
}

func DeleteHumansVerify(client *IdpClient, url string, requests []UpdateHumansDeleteVerifyRequest) (int, []UpdateHumansDeleteVerifyResponse, error) {
  var response []UpdateHumansDeleteVerifyResponse

  body, err := json.Marshal(requests)
  if err != nil {
    return 999, nil, err // Client system was unable marshal request
  }

  status, responseData, err := callService(client, "PUT", url, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(responseData, &response)
  if err != nil {
    return 666, nil, err // Client system was unable to unmarshal request, but server already executed
  }

  return status, response, nil
}

func UpdateHumansPassword(client *IdpClient, url string, requests []UpdateHumansPasswordRequest) (int, []UpdateHumansPasswordResponse, error) {
  var response []UpdateHumansPasswordResponse

  body, err := json.Marshal(requests)
  if err != nil {
    return 999, nil, err // Client system was unable marshal request
  }

  status, responseData, err := callService(client, "PUT", url, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(responseData, &response)
  if err != nil {
    return 666, nil, err // Client system was unable to unmarshal request, but server already executed
  }

  return status, response, nil
}

func UpdateHumansTotp(client *IdpClient, url string, requests []UpdateHumansTotpRequest) (int, []UpdateHumansTotpResponse, error) {
  var response []UpdateHumansTotpResponse

  body, err := json.Marshal(requests)
  if err != nil {
    return 999, nil, err // Client system was unable marshal request
  }

  status, responseData, err := callService(client, "PUT", url, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(responseData, &response)
  if err != nil {
    return 666, nil, err // Client system was unable to unmarshal request, but server already executed
  }

  return status, response, nil
}

func CreateHumansAuthenticate(client *IdpClient, url string, requests []CreateHumansAuthenticateRequest) (int, []CreateHumansAuthenticateResponse, error) {
  var response []CreateHumansAuthenticateResponse

  body, err := json.Marshal(requests)
  if err != nil {
    return 999, nil, err // Client system was unable marshal request
  }

  status, responseData, err := callService(client, "POST", url, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(responseData, &response)
  if err != nil {
    return 666, nil, err // Client system was unable to unmarshal request, but server already executed
  }

  return status, response, nil
}

func RecoverHumans(client *IdpClient, url string, requests []CreateHumansRecoverRequest) (int, []CreateHumansRecoverResponse, error) {
  var response []CreateHumansRecoverResponse

  body, err := json.Marshal(requests)
  if err != nil {
    return 999, nil, err // Client system was unable marshal request
  }

  status, responseData, err := callService(client, "POST", url, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(responseData, &response)
  if err != nil {
    return 666, nil, err // Client system was unable to unmarshal request, but server already executed
  }

  return status, response, nil
}

func RecoverHumansVerify(client *IdpClient, url string, requests []UpdateHumansRecoverVerifyRequest) (int, []UpdateHumansRecoverVerifyResponse, error) {
  var response []UpdateHumansRecoverVerifyResponse

  body, err := json.Marshal(requests)
  if err != nil {
    return 999, nil, err // Client system was unable marshal request
  }

  status, responseData, err := callService(client, "PUT", url, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(responseData, &response)
  if err != nil {
    return 666, nil, err // Client system was unable to unmarshal request, but server already executed
  }

  return status, response, nil
}

func LogoutHumans(client *IdpClient, url string, requests []CreateHumansLogoutRequest) (int, []CreateHumansLogoutResponse, error) {
  var response []CreateHumansLogoutResponse

  body, err := json.Marshal(requests)
  if err != nil {
    return 999, nil, err // Client system was unable marshal request
  }

  status, responseData, err := callService(client, "POST", url, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(responseData, &response)
  if err != nil {
    return 666, nil, err // Client system was unable to unmarshal request, but server already executed
  }

  return status, response, nil
}

