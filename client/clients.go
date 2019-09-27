package client

import (
  "bytes"
  "encoding/json"
)

type Client struct {
  Id           string `json:"id"            validate:"required,uuid"`
  ClientSecret string `json:"client_secret" validate:"required"`
  Name         string `json:"name"          validate:"required"`
  Description  string `json:"description"   validate:"required"`
}

// Endpoints

type ReadClientsRequest struct {
  Id string `json:"id,omitempty" validate:"uuid"`
}

type ReadClientsResponse struct {
  BulkResponse
  Ok []Client `json:"ok,omitempty" validate:"dive"`
}

func ReadClients(client *IdpClient, url string, requests []ReadClientsRequest) (int, []ReadClientsResponse, error) {
  var response []ReadClientsResponse

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
