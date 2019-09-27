package client

import (
  "bytes"
  "encoding/json"
)





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

func ReadInvite(client *IdpClient, inviteUrl string, request *IdentitiesInviteReadRequest) (int, *IdentitiesInviteReadResponse, error) {
  var response IdentitiesInviteReadResponse

  body, err := json.Marshal(request)
  if err != nil {
    return 999, nil, err
  }

  status, result, err := callService(client, "GET", inviteUrl, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return 666, nil, err
  }
  return status, &response, nil
}

func UpdateInvite(client *IdpClient, inviteUrl string, request *IdentitiesInviteUpdateRequest) (int, *IdentitiesInviteUpdateResponse, error) {
  var response IdentitiesInviteUpdateResponse

  body, err := json.Marshal(request)
  if err != nil {
    return 999, nil, err
  }

  status, result, err := callService(client, "PUT", inviteUrl, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return 666, nil, err
  }
  return status, &response, nil
}

func CreateInvite(client *IdpClient, inviteUrl string, request *IdentitiesInviteCreateRequest) (int, *IdentitiesInviteCreateResponse, error) {
  var response IdentitiesInviteCreateResponse

  body, err := json.Marshal(request)
  if err != nil {
    return 999, nil, err
  }

  status, result, err := callService(client, "POST", inviteUrl, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return 666, nil, err
  }
  return status, &response, nil
}







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
  InvitedBy string `json:"invited_by" binding:"required"`
  TTL int64 `json:"ttl" binding:"required"`
  Invited string `json:"invited,omitempty"`
  Email string `json:"email,omitempty"`
  Username string `json:"username,omitempty"` // Hint username
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

func ReadInvites(client *IdpClient, challengeUrl string, request *InviteReadRequest) (int, *InviteReadResponse, error) {
  var response InviteReadResponse

  body, err := json.Marshal(request)
  if err != nil {
    return 999, nil, err
  }

  status, result, err := callService(client, "GET", challengeUrl, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return 666, nil, err
  }
  return status, &response, nil
}

func CreateInvites(client *IdpClient, challengeUrl string, request *InviteCreateRequest) (int, *InviteCreateResponse, error) {
  var response InviteCreateResponse

  body, err := json.Marshal(request)
  if err != nil {
    return 999, nil, err
  }

  status, result, err := callService(client, "POST", challengeUrl, bytes.NewBuffer(body))
  if err != nil {
    return status, nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return 666, nil, err
  }
  return status, &response, nil
}