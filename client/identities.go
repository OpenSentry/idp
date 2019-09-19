package client

import (
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

type IdentitiesInviteResponse struct {
  Id string `json:"id" binding:"required"`
  Email string `json:"email" binding:"required"`
  Username string `json:"username" binding:"required"`
  GrantedScopes string `json:"granted_scopes" binding:"required"`
  FollowIdentities string `json:"follow_identities" binding:"required"`
  TTL int64 `json:"ttl" binding:"required"`
  IssuedAt int64 `json:"iat" binding:"required"`
  ExpiresAt int64 `json:"exp" binding:"required"`
  InviterId string `json:"inviter_id" binding:"required"`
  InvitedId string `json:"invited_id" binding:"required"`
}

// CRUD

type IdentitiesCreateRequest struct {
  Password string `json:"password" binding:"required"`
  Subject  string `json:"sub"`
  Email    string `json:"email,omitempty"`
  Name     string `json:"name,omitempty"`
}

type IdentitiesCreateResponse struct {
  *IdentitiesResponse
}

type IdentitiesReadRequest struct {
  Id      string `form:"id"    json:"id"`
  Subject string `form:"sub"   json:"sub"`
  Email   string `form:"email" json:"email"`
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
  Id           string `json:"id,omitempty"`
  Password     string `json:"password,omitempty"`
  OtpChallenge string `json:"otp_challenge,omitempty"`
}

// We try and limit the amount of information returned by the endpoint.
type IdentitiesAuthenticateResponse struct {
  Id                 string `json:"id" binding:"required"`
  Authenticated      bool   `json:"authenticated" binding:"required"`
  RedirectTo         string `json:"redirect_to" binding:"required"`
  TotpRequired       bool   `json:"totp_required" binding:"required"`

  IsPasswordInvalid  bool `json:"is_password_invalid" binding:"required"`
  IdentityExists     bool `json:"identity_exists" binding:"required"`
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

type IdentitiesInviteCreateRequest struct {
  Id string `json:"id" binding:"required"`
  Email string `json:"email" binding:"required"`
  Username string `json:"username"`
  GrantedScopes []string `json:"granted_scopes"`
  PleaseFollow []string `json:"please_follow"`
  TTL int64 `json:"ttl"`
}

// FIXME: Måske skal bulk read ligge i invites.go collection mappen og client
// OG identities/invite kan så kun bruge InviterId = access token subject
// så ligger bulk kaldene på collection og single kalde på token (aka. bulk = client_id token, single = authorization code flow token)

type IdentitiesInviteCreateResponse struct {
  *IdentitiesInviteResponse
}

type IdentitiesInviteUpdateRequest struct {
  Id string `json:"id" binding:"required"`
}

type IdentitiesInviteUpdateResponse struct {
  *IdentitiesInviteResponse
}

type IdentitiesInviteReadRequest struct {
  Id string `json:"id" binding:"required"`
}

type IdentitiesInviteReadResponse struct {
  *IdentitiesInviteResponse
}

func ReadInvite(client *IdpClient, inviteUrl string, request *IdentitiesInviteReadRequest) (*IdentitiesInviteReadResponse, error) {
  var response IdentitiesInviteReadResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "GET", inviteUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func UpdateInvite(client *IdpClient, inviteUrl string, request *IdentitiesInviteUpdateRequest) (*IdentitiesInviteUpdateResponse, error) {
  var response IdentitiesInviteUpdateResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "PUT", inviteUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func CreateInvite(client *IdpClient, inviteUrl string, request *IdentitiesInviteCreateRequest) (*IdentitiesInviteCreateResponse, error) {
  var response IdentitiesInviteCreateResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", inviteUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
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

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "GET", identitiesUrl, bytes.NewBuffer(body))
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
