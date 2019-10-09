package client

import (
  bulky "github.com/charmixer/bulky/client"
)

type Follow struct {
  From string `json:"from" validate:"required,uuid"`
  To   string `json:"to"   validate:"required,uuid"`
}

type CreateFollowsResponse Follow
type CreateFollowsRequest struct {
  From string `json:"from" validate:"required,uuid"`
  To   string `json:"to"   validate:"required,uuid"`
}

type ReadFollowsResponse []Follow
type ReadFollowsRequest struct {
  From string `json:"id,omitempty" validate:"required,uuid"`
}


func CreateFollows(client *IdpClient, url string, requests []CreateFollowsRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "POST", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}

func ReadFollows(client *IdpClient, url string, requests []ReadFollowsRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "GET", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}