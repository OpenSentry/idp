package client

import (
  bulky "github.com/charmixer/bulky/client"
)

type Client struct {
  Id           string `json:"id"            validate:"required,uuid"`
  ClientSecret string `json:"client_secret" validate:"required"`
  Name         string `json:"name"          validate:"required"`
  Description  string `json:"description"   validate:"required"`
}

type CreateClientsResponse Client
type CreateClientsRequest struct {
  ClientSecret string `json:"password"    validate:"required"`
  Name         string `json:"name"        validate:"required"`
  Description  string `json:"description" validate:"required"`
}

type ReadClientsResponse []Client
type ReadClientsRequest struct {
  Id string `json:"id,omitempty" validate:"uuid"`
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
