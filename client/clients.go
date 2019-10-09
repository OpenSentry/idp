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

type ReadClientsResponse []Client
type ReadClientsRequest struct {
  Id string `json:"id,omitempty" validate:"uuid"`
}

func ReadClients(client *IdpClient, url string, requests []ReadClientsRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "GET", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}
