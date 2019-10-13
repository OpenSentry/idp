package client

import (
  bulky "github.com/charmixer/bulky/client"
)

type Invite struct {
  Id        string `json:"id"                    validate:"required,uuid"`
  IssuedAt  int64  `json:"iat"                   validate:"required,numeric"`
  ExpiresAt int64  `json:"exp"                   validate:"required,numeric"`

  Email        string `json:"email"              validate:"required,email"`
  Username     string `json:"username,omitempty" validate:"omitempty"`

  SentAt   int64 `json:"sent_at,omitempty" validate:"omitempty,numeric"`
}

type InviteClaimChallenge struct {
  RedirectTo string `json:"redirect_to"   validate:"required,url"`
}

type CreateInvitesResponse Invite
type CreateInvitesRequest struct {
  Email    string `json:"email"              validate:"required,email"`
  Username string `json:"username,omitempty" validate:"omitempty"`
  ExpiresAt int64 `json:"exp,omitempty"      validate:"omitempty,numeric"`
}

type ReadInvitesResponse []Invite
type ReadInvitesRequest struct {
  Id       string `json:"id,omitempty"        validate:"omitempty,uuid"`
  Email    string `json:"email,omitempty"     validate:"omitempty,email"`
}

type CreateInvitesSendResponse Invite
type CreateInvitesSendRequest struct {
  Id string `json:"id" validate:"required,uuid"`
}

type CreateInvitesClaimResponse InviteClaimChallenge
type CreateInvitesClaimRequest struct {
  Id         string `json:"id" validate:"required,uuid"`
  RedirectTo string `json:"redirect_to" validate:"required,url"`
  TTL        int64  `json:"ttl" validate:"required,numeric"`
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

func CreateInvitesClaim(client *IdpClient, url string, requests []CreateInvitesClaimRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "POST", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}