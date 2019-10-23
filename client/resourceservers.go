package client

import (
  bulky "github.com/charmixer/bulky/client"
)

type ResourceServer struct {
  Id           string `json:"id"            validate:"required,uuid"`
  Name         string `json:"name"          validate:"required"`
  Description  string `json:"description"   validate:"required"`
  Audience     string `json:"aud"           validate:"required"`
}

type CreateResourceServersResponse ResourceServer
type CreateResourceServersRequest struct {
  Name         string `json:"name"        validate:"required"`
  Description  string `json:"description" validate:"required"`
  Audience     string `json:"aud"         validate:"required"`
}

type ReadResourceServersResponse []ResourceServer
type ReadResourceServersRequest struct {
  Id string `json:"id,omitempty" validate:"uuid"`
}

type DeleteResourceServersResponse Identity
type DeleteResourceServersRequest struct {
  Id string `json:"id" validate:"required,uuid"`
}

func CreateResourceServers(client *IdpClient, url string, requests []CreateResourceServersRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "POST", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}

func ReadResourceServers(client *IdpClient, url string, requests []ReadResourceServersRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "GET", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}

func DeleteResourceServers(client *IdpClient, url string, requests []DeleteResourceServersRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "DELETE", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}