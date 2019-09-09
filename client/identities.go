package client

import (
  "net/http"
  "bytes"
  "encoding/json"
)

type IdentitiesResponse struct {
  Id                   string `json:"id" binding:"required"`
  Subject              string `json:"sub" binding:"required"`
  Password             string `json:"password" binding:"required"`
  Name                 string `json:"name" binding:"required`
  Email                string `json:"email" binding:"required"`
  AllowLogin           bool   `json:"allow_login" binding:"required"`
  TotpRequired         bool   `json:"totp_required" binding:"required"`
  TotpSecret           string `json:"totp_secret" binding:"required"`
  OtpRecoverCode       string `json:"otp_recover_code" binding:"required"`
  OtpRecoverCodeExpire int64  `json:"otp_recover_code_expire" binding:"required"`
  OtpDeleteCode        string `json:"otp_delete_code" binding:"required"`
  OtpDeleteCodeExpire  int64  `json:"otp_delete_code_expire" binding:"required"`
}

// CRUD

type IdentitiesCreateRequest struct {
  Id       string `json:"id" binding:"required"`
  Password string `json:"password" binding:"required"`
  Subject  string `json:"sub" binding:"required"`
  Name     string `json:"name,omitempty"`
  Email    string `json:"email,omitempty"`
}

type IdentitiesCreateResponse struct {
  *IdentitiesResponse
}

type IdentitiesReadRequest struct {
  Id string `form:"id" json:"id"`
  Subject string `form:"email" json:"id"`
  Email string `form:"email" json:"email"`
}

type IdentitiesReadResponse struct {
  *IdentitiesResponse
}

type IdentitiesUpdateRequest struct {
  Id            string          `json:"id" binding:"required"`
  Name          string          `json:"name,omitempty"`
  Email         string          `json:"email,omitempty"`
}

type IdentitiesUpdateResponse struct {
  *IdentitiesResponse
}

type IdentitiesDeleteRequest struct {
  Id string `json:"id" binding:"required"`
}

type IdentitiesDeleteResponse struct {
  Id         string `json:"id" binding:"required"`
  RedirectTo string `json:"redirect_to" binding:"required"`
}

type IdentitiesDeleteVerificationRequest struct {
  Id               string `json:"id" binding:"required"`
  VerificationCode string `json:"verification_code" binding:"required"`
  RedirectTo       string `json:"redirect_to" binding:"required"`
}

type IdentitiesDeleteVerificationResponse struct {
  Id         string `json:"id" binding:"required"`
  Verified   bool   `json:"verified" binding:"required"`
  RedirectTo string `json:"redirect_to" binding:"required"`
}

type IdentitiesPasswordRequest struct {
  Id       string `json:"id" binding:"required"`
  Password string `json:"password" binding:"required"`
}

type IdentitiesPasswordResponse struct {
  *IdentitiesResponse
}

type IdentitiesTotpRequest struct {
  Id           string `json:"id" binding:"required"`
  TotpRequired bool   `json:"totp_required" binding:"required"`
  TotpSecret   string `json:"totp_secret" binding:"required"`
}

type IdentitiesTotpResponse struct {
  Id string `json:"id" binding:"required"`
}

// Actions

type IdentitiesAuthenticateRequest struct {
  Challenge    string `json:"challenge" binding:"required"`
  Id           string `json:"id"`
  Subject      string `json:"sub"`
  Password     string `json:"password"`
  OtpChallenge string `json:"otp_challenge"`
}

type IdentitiesAuthenticateResponse struct {
  Id            string `json:"id" binding:"required"`
  NotFound      bool   `json:"not_found" binding:"required"`
  Authenticated bool   `json:"authenticated" binding:"required"`
  TotpRequired  bool   `json:"totp_required" binding:"required"`
  RedirectTo    string `json:"redirect_to" binding:"required"`
}

type IdentitiesRecoverRequest struct {
  Id string `json:"id" binding:"required"`
}

type IdentitiesRecoverResponse struct {
  Id         string `json:"id" binding:"required"`
  RedirectTo string `json:"redirect_to" binding:"required"`
}

type IdentitiesRecoverVerificationRequest struct {
  Id               string `json:"id" binding:"required"`
  VerificationCode string `json:"verification_code" binding:"required"`
  Password         string `json:"password" binding:"required"`
  RedirectTo       string `json:"redirect_to" binding:"required"`
}

type IdentitiesRecoverVerificationResponse struct {
  Id         string `json:"id" binding:"required"`
  Verified   bool   `json:"verified" binding:"required"`
  RedirectTo string `json:"redirect_to" binding:"required"`
}

type IdentitiesLogoutRequest struct {
  Challenge string `json:"challenge" binding:"required"`
}

type IdentitiesLogoutResponse struct {
  RedirectTo string `json:"redirect_to" binding:"required"`
}

func CreateIdentity(client *IdpClient, identitiesUrl string, request *IdentitiesCreateRequest) (*IdentitiesCreateResponse, error) {
  var response IdentitiesCreateResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", identitiesUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func ReadIdentity(client *IdpClient, identitiesUrl string, request *IdentitiesReadRequest) (*IdentitiesReadResponse, error) {
  var response IdentitiesReadResponse

  req, err := http.NewRequest("GET", identitiesUrl, nil)
  if err != nil {
    return nil, err
  }

  // TODO: Can we marshal this somehow?
  query := req.URL.Query()
  query.Add("id", request.Id)
  req.URL.RawQuery = query.Encode()

  res, err := client.Do(req)
  if err != nil {
    return nil, err
  }

  result, err := parseResponse(res)
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func UpdateIdentity(client *IdpClient, identitiesUrl string, request *IdentitiesUpdateRequest) (*IdentitiesUpdateResponse, error) {
  var response IdentitiesUpdateResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "PUT", identitiesUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func DeleteIdentity(client *IdpClient, identitiesUrl string, request *IdentitiesDeleteRequest) (*IdentitiesDeleteResponse, error) {
  var response IdentitiesDeleteResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "DELETE", identitiesUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func DeleteIdentityVerification(client *IdpClient, verificationUrl string, request *IdentitiesDeleteVerificationRequest) (*IdentitiesDeleteVerificationResponse, error) {
  var response IdentitiesDeleteVerificationResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", verificationUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func UpdateIdentityPassword(client *IdpClient, passwordUrl string, request *IdentitiesPasswordRequest) (*IdentitiesPasswordResponse, error) {
  var response IdentitiesPasswordResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "PUT", passwordUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func UpdateIdentityTotp(client *IdpClient, totpUrl string, request *IdentitiesTotpRequest) (*IdentitiesTotpResponse, error) {
  var response IdentitiesTotpResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "PUT", totpUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func AuthenticateIdentity(client *IdpClient, authenticateUrl string, request *IdentitiesAuthenticateRequest) (*IdentitiesAuthenticateResponse, error) {
  var response IdentitiesAuthenticateResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", authenticateUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func RecoverIdentity(client *IdpClient, recoverUrl string, request *IdentitiesRecoverRequest) (*IdentitiesRecoverResponse, error) {
  var response IdentitiesRecoverResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", recoverUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func RecoverIdentityVerification(client *IdpClient, verificationUrl string, request *IdentitiesRecoverVerificationRequest) (*IdentitiesRecoverVerificationResponse, error) {
  var response IdentitiesRecoverVerificationResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", verificationUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func LogoutIdentity(client *IdpClient, logoutUrl string, request *IdentitiesLogoutRequest) (*IdentitiesLogoutResponse, error) {
  var response IdentitiesLogoutResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", logoutUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}
