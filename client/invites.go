package client

import (
  bulky "github.com/charmixer/bulky/client"
)

type Invite struct {
  Id        string `json:"id"                    validate:"required,uuid"`
  IssuedAt  int64  `json:"iat"                   validate:"required"`
  ExpiresAt int64  `json:"exp"                   validate:"required"`

  Email        string `json:"email"              validate:"required,email"`
  Username     string `json:"username,omitempty"`

  SentAt   int64 `json:"sent_at,omitempty" validate:"omitempty,numeric"`

  InvitedBy string `json:"invited_by"            validate:"required,uuid"`
}

type CreateInvitesResponse Invite
type CreateInvitesRequest struct {
  Email    string `json:"email,omitempty"          validate:"omitempty,email"`
  Username string `json:"username,omitempty"`
}

type ReadInvitesResponse []Invite
type ReadInvitesRequest struct {
  Id       string `json:"id,omitempty"        validate:"omitempty,uuid"`
  Email    string `json:"email,omitempty"     validate:"omitempty,email"`
  Username string `json:"username,omitempty"`
}

type UpdateInvitesAcceptResponse Invite
type UpdateInvitesAcceptRequest struct {
  Id string `json:"id" validate:"required,uuid"`
}

type CreateInvitesSendResponse Invite
type CreateInvitesSendRequest struct {
  Id string `json:"id" validate:"required,uuid"`
}


func CreateInvites(client *IdpClient, url string, requests []CreateInvitesRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "POST", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}


func ReadInvites(client *IdpClient, url string, requests []ReadInvitesRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "GET", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}

func CreateInvitesSend(client *IdpClient, url string, requests []CreateInvitesSendRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "POST", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}