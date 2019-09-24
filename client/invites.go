package client

import (
  "bytes"
  "encoding/json"
)

type InviteResponse struct {
  Id string `json:"id" binding:"required"`
  InvitedBy string `json:"invited_by" binding:"required"`
  TTL int64 `json:"ttl" binding:"required"`
  IssuedAt int64 `json:"iat" binding:"required"`
  ExpiresAt int64 `json:"exp" binding:"required"`
  Email string `json:"email,omitempty"`
  Username string `json:"username,omitempty"`
  Invited string `json:"invited,omitempty"`
}

// CRUD

type InviteCreateRequest struct {
  InvitedByIdentity string `json:"ibi" binding:"required"`
  TTL int64 `json:"ttl" binding:"required"`
  InvitedIdentity string `json:"ii,omitempty"`
  Email string `json:"email,omitempty"`
  HintUsername string `json:"hint_username,omitempty"`
}

type InviteCreateResponse struct {
  *InviteResponse
}

type InviteReadRequest struct {
  Id string `json:"id,omitempty"`
}

type InviteReadResponse struct {
  *InviteResponse
}

// Actions

func ReadInvites(client *IdpClient, challengeUrl string, request *[]InviteReadRequest) (*[]InviteReadResponse, error) {
  var response []InviteReadResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "GET", challengeUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}

func CreateInvites(client *IdpClient, challengeUrl string, request *InviteCreateRequest) (*InviteCreateResponse, error) {
  var response InviteCreateResponse

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