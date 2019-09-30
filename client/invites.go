package client

import (
  "bytes"
  "encoding/json"
)

type Invite struct {
  Id        string `json:"id" validate:"required,uuid"`
  IssuedAt  int64  `json:"iat" validate:"required"`
  ExpiresAt int64  `json:"exp" validate:"required"`

  Email string `json:"email" validate:"email"`
  Invited string `json:"id" validate:"uuid"`
  HintUsername string `json:"hint_username"`

  InvitedBy string `json:"id" validate:"required,uuid"`
}

type CreateInvitesRequest struct {
  Email          string `json:"email,omitempty" validate:"email"`
  Invited        string `json:"invited_id" validate:"uuid"` // FIXME: Mututal exclusive with email
  HintUsername   string `json:"hint_username,omitempty"`
}

type CreateInvitesResponse struct {
  BulkResponse
  Ok Invite `json:"ok,omitempty" validate:"dive"`
}

type ReadInvitesRequest struct {
  Id       string `json:"id,omitempty"        validate:"uuid"`
  Email    string `json:"email,omitempty"     validate:"email"`
  Username string `json:"username,omitempty"`
}

type ReadInvitesResponse struct {
  BulkResponse
  Ok []Invite `json:"ok,omitempty" validate:"dive"`
}

type UpdateInvitesAcceptRequest struct {
  Id string `json:"id" validate:"required,uuid"`
}

type UpdateInvitesAcceptResponse struct {
  BulkResponse
  Ok []Invite `json:"ok,omitempty" validate:"dive"`
}

type CreateInvitesSendRequest struct {
  Id string `json:"id" validate:"required,uuid"`
}

type CreateInvitesSendResponse struct {
  BulkResponse
  Ok []Invite `json:"ok,omitempty" validate:"dive"`
}


func CreateInvites(client *IdpClient, url string, requests []CreateInvitesRequest) (int, []CreateInvitesResponse, error) {
  var response []CreateInvitesResponse

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

func ReadInvites(client *IdpClient, url string, requests []ReadInvitesRequest) (int, []ReadInvitesResponse, error) {
  var response []ReadInvitesResponse

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

func UpdateInvitesAccept(client *IdpClient, url string, requests []UpdateInvitesAcceptRequest) (int, []UpdateInvitesAcceptResponse, error) {
  var response []UpdateInvitesAcceptResponse

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
