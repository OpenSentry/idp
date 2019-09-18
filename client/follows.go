package client

import (
  "bytes"
  "encoding/json"
)

type FollowResponse struct {
  Id string `json:"id" binding:"required"`
  Follow string `json:"follow" binding:"required"`
}

// CRUD

type FollowCreateRequest struct {
  Id string `json:"id" binding:"required"`
  Follow string `json:"follow" binding:"required"`
}

type FollowCreateResponse struct {
  *FollowResponse
}

// Actions

func CreateFollow(client *IdpClient, challengeUrl string, request *FollowCreateRequest) (*FollowCreateResponse, error) {
  var response FollowCreateResponse

  body, err := json.Marshal(request)
  if err != nil {
    return nil, err
  }

  result, err := callService(client, "POST", challengeUrl, bytes.NewBuffer(body))
  if err != nil {
    return nil, err
  }

  err = json.Unmarshal(result, &response)
  if err != nil {
    return nil, err
  }
  return &response, nil
}