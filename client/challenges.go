package client

import (
  "net/http"
  "bytes"
  "encoding/json"
)

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

func ReadChallenge(client *IdpClient, challengeUrl string, request *ChallengesReadRequest) (*ChallengesReadResponse, error) {
  var response ChallengesReadResponse

  req, err := http.NewRequest("GET", challengeUrl, nil)
  if err != nil {
    return nil, err
  }

  // TODO: Can we marshal this somehow?
  query := req.URL.Query()
  query.Add("otp_challenge", request.OtpChallenge)
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

func CreateChallenge(client *IdpClient, challengeUrl string, request *ChallengeCreateRequest) (*ChallengeCreateResponse, error) {
  var response ChallengeCreateResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", challengeUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func VerifyChallenge(client *IdpClient, verifyUrl string, request *ChallengeVerifyRequest) (*ChallengeVerifyResponse, error) {
  var response ChallengeVerifyResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", verifyUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}
