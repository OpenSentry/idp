package client

import (
  "bytes"
  "encoding/json"
)

type InviteResponse struct {
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

type InviteCreateRequest struct {
  Id string `json:"id" binding:"required"`
  Email string `json:"email" binding:"required"`
  Username string `json:"username"`
  GrantedScopes []string `json:"granted_scopes"`
  PleaseFollow []string `json:"please_follow"`
  TTL int64 `json:"ttl"`
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

func ReadInvites(client *IdpClient, challengeUrl string, request *InviteReadRequest) (*InviteReadResponse, error) {
  var response InviteReadResponse

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