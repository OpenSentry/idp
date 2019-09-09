package models

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
