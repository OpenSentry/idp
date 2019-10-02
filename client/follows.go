package client

import (
  "bytes"
  "encoding/json"
)

type Follow struct {
  From string `json:"from" validate:"required,uuid"`
  To   string `json:"to"   validate:"required,uuid"`
}

type CreateFollowsRequest struct {
  From string `json:"from" validate:"required,uuid"`
  To   string `json:"to"   validate:"required,uuid"`
}

type CreateFollowsResponse struct {
  BulkResponse
  Ok Follow `json:"ok,omitempty" validate:"dive"`
}

type ReadFollowsRequest struct {
  From string `json:"id,omitempty" validate:"required,uuid"`
}

type ReadFollowsResponse struct {
  BulkResponse
  Ok []Follow `json:"ok,omitempty" validate:"dive"`
}


func CreateFollows(client *IdpClient, url string, requests []CreateFollowsRequest) (status int, response []CreateFollowsResponse, err error) {
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

func ReadFollows(client *IdpClient, url string, requests []ReadFollowsRequest) (status int, response []ReadFollowsResponse, err error) {
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