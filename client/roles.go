package client

import (
  bulky "github.com/charmixer/bulky/client"
)

type Role struct {
  Id           string `json:"id"            validate:"required,uuid"`
  Name         string `json:"name"          validate:"required"`
  Description  string `json:"description"   validate:"required"`
}

type CreateRolesResponse Role
type CreateRolesRequest struct {
  Name         string `json:"name"        validate:"required"`
  Description  string `json:"description" validate:"required"`
}

type ReadRolesResponse []Role
type ReadRolesRequest struct {
  Id string `json:"id,omitempty" validate:"uuid"`
}

type DeleteRolesResponse Identity
type DeleteRolesRequest struct {
  Id string `json:"id" validate:"required,uuid"`
}

func CreateRoles(client *IdpClient, url string, requests []CreateRolesRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "POST", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}

func ReadRoles(client *IdpClient, url string, requests []ReadRolesRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "GET", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}

func DeleteRoles(client *IdpClient, url string, requests []DeleteRolesRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "DELETE", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}
