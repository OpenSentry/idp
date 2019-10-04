package client

import (
  "bytes"
  "encoding/json"
)

type Identity struct {
  Id string `json:"id" validate:"required,uuid"`
  Labels []string `json:"labels"`
}

type ReadIdentitiesRequest struct {
  Id string `json:"id,omitempty" validate:"uuid"`
}

type ReadIdentitiesResponse struct {
  BulkResponse
  Ok []Identity `json:"ok,omitempty" validate:"dive"`
}

func ReadIdentities(client *IdpClient, url string, requests []ReadIdentitiesRequest) (int, []ReadIdentitiesResponse, error) {
  var response []ReadIdentitiesResponse

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
