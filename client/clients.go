package client

import (
  bulky "github.com/charmixer/bulky/client"
)

type Client struct {
  Id           string `json:"id"            validate:"required,uuid"`
  ClientSecret string `json:"client_secret,omitempty" validate:"omitempty"`
  Name         string `json:"name"          validate:"required"`
  Description  string `json:"description"   validate:"required"`
}

type CreateClientsResponse Client
type CreateClientsRequest struct {
  Name         string `json:"name"                    validate:"required"`
  Description  string `json:"description"             validate:"required"`
  IsPublic     bool   `json:"is_public"               `
  ClientSecret string `json:"client_secret,omitempty" validate:"omitempty"`
}

type ReadClientsResponse []Client
type ReadClientsRequest struct {
  Id string `json:"id,omitempty" validate:"uuid"`
}

type DeleteClientsResponse Identity
type DeleteClientsRequest struct {
  Id string `json:"id" validate:"required,uuid"`
}

func CreateClients(client *IdpClient, url string, requests []CreateClientsRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "POST", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}

func ReadClients(client *IdpClient, url string, requests []ReadClientsRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "GET", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}
