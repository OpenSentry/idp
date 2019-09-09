package models

type ChallengeResponse struct {
  OtpChallenge string `json:"otp_challenge" binding:"required"`
  Subject      string `json:"sub" binding:"required"`
  Audience     string `json:"aud" binding:"required"`
  IssuedAt     int64  `json:"iat" binding:"required"`
  ExpiresAt    int64  `json:"exp" binding:"required"`
  TTL          int    `json:"ttl" binding:"required"`
  RedirectTo   string `json:"redirect_to" binding:"required"`
  CodeType     string `json:"code_type" binding:"required"`
  Code         string `json:"code" binding:"required"`
}

// CRUD

type ChallengeCreateRequest struct {
  Subject     string `json:"sub" binding:"required"`
  Audience     string `json:"aud" binding:"required"`
  TTL          int    `json:"ttl" binding:"required"`
  RedirectTo   string `json:"redirect_to" binding:"required"`
  CodeType     string `json:"code_type" binding:"required"`
  Code         string `json:"code" binding:"required"`
}

type ChallengeCreateResponse struct {
  *ChallengeResponse
}

type ChallengesReadRequest struct {
  OtpChallenge string `form:"otp_challenge" json:"otp_challenge" binding:"required"`
}

type ChallengesReadResponse struct {
  *ChallengeResponse
}

// Actions

type ChallengeVerifyRequest struct {
  OtpChallenge string `json:"otp_challenge" binding:"required"`
  Code       string `json:"code" binding:"required"`
}

type ChallengeVerifyResponse struct {
  OtpChallenge string `json:"otp_challenge" binding:"required"`
  Verified     bool   `json:"verified" binding:"required"`
  RedirectTo   string `json:"redirect_to" binding:"required"`
}
