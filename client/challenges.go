package client

import (
  "bytes"
  "encoding/json"
)

type Challenge struct {
  OtpChallenge string `json:"otp_challenge" validate:"required"`
  Subject      string `json:"sub"           validate:"required,uuid"`
  Audience     string `json:"aud"           validate:"required"`
  IssuedAt     int64  `json:"iat"           validate:"required"`
  ExpiresAt    int64  `json:"exp"           validate:"required"`
  TTL          int64  `json:"ttl"           validate:"required"`
  RedirectTo   string `json:"redirect_to"   validate:"required,url"`
  CodeType     string `json:"code_type"     validate:"required"`
  Code         string `json:"code"          validate:"required"`
}

type CreateChallengesRequest struct {
  Subject     string `json:"sub"         validate:"required,uuid"`
  Audience    string `json:"aud"         validate:"required"`
  TTL         int64  `json:"ttl"         validate:"required"`
  RedirectTo  string `json:"redirect_to" validate:"required,url"`
  CodeType    string `json:"code_type"   validate:"required"`
  Code        string `json:"code"        validate:"required"`
}

type CreateChallengesResponse struct {
  BulkResponse
  Ok Challenge `json:"ok,omitempty" validate:"dive"`
}

type ReadChallengesRequest struct {
  OtpChallenge  string `json:"otp_challenge" validate:"required"`
}

type ReadChallengesResponse struct {
  BulkResponse
  Ok []Challenge `json:"ok,omitempty" validate:"dive"`
}

type ChallengeVerification struct {
  OtpChallenge string `json:"otp_challenge" validate:"required"`
  Verified     bool   `json:"verified"      `
  RedirectTo   string `json:"redirect_to"   validate:"required,url"`
}

type UpdateChallengesVerifyRequest struct {
  OtpChallenge string `json:"otp_challenge" validate:"required"`
  Code         string `json:"code"          validate:"required"`
}

type UpdateChallengesVerifyResponse struct {
  BulkResponse
  Ok ChallengeVerification `json:"ok,omitempty" validate:"dive"`
}

func ReadChallenges(client *IdpClient, url string, requests []ReadChallengesRequest) (int, []ReadChallengesResponse, error) {
  var response []ReadChallengesResponse

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

func CreateChallenges(client *IdpClient, url string, requests []CreateChallengesRequest) (int, []CreateChallengesResponse, error) {
  var response []CreateChallengesResponse

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

func VerifyChallenges(client *IdpClient, url string, requests []UpdateChallengesVerifyRequest) (int, []UpdateChallengesVerifyResponse, error) {
  var response []UpdateChallengesVerifyResponse

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

