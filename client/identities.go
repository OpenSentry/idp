package client

import (
  bulky "github.com/charmixer/bulky/client"
)

type Identity struct {
  Id string `json:"id" validate:"required,uuid"`
  Labels []string `json:"labels"`
}

type ReadIdentitiesResponse []Identity
type ReadIdentitiesRequest struct {
  Id string `json:"id,omitempty" validate:"uuid"`
}

func ReadIdentities(client *IdpClient, url string, requests []ReadIdentitiesRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "GET", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}
