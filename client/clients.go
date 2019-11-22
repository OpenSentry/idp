package client

import (
  bulky "github.com/charmixer/bulky/client"
)

type Client struct {
  Id                      string   `json:"id" validate:"required,uuid"`
  Name                    string   `json:"name"                       validate:"required"`
  Description             string   `json:"description"                validate:"required"`
  Secret                  string   `json:"secret,omitempty"           validate:"omitempty"`
  GrantTypes              []string `json:"grant_types"                validate:"omitempty,dive,eq=authorization_code|eq=implicit|eq=password|eq=client_credentials|eq=device_code|eq=refresh_token"`
  ResponseTypes           []string `json:"response_types"             validate:"omitempty,dive,eq=code|eq=token"`
  RedirectUris            []string `json:"redirect_uris"              validate:"omitempty,dive,url"`
  TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method" validate:"omitempty,eq=none|eq=client_secret_post|eq=client_secret_basic|eq=private_key_jwt"`
  PostLogoutRedirectUris  []string `json:"post_logout_redirect_uris"  validate:"omitempty,dive,url"`
}

type CreateClientsResponse Client
type CreateClientsRequest struct {
  Name                    string   `json:"name"                       validate:"required"`
  Description             string   `json:"description"                validate:"required"`
  IsPublic                bool     `json:"is_public"                  `
  Secret                  string   `json:"secret,omitempty"           validate:"omitempty"`
  GrantTypes              []string `json:"grant_types"                validate:"omitempty,dive,eq=authorization_code|eq=implicit|eq=password|eq=client_credentials|eq=device_code|eq=refresh_token"`
  ResponseTypes           []string `json:"response_types"             validate:"omitempty,dive,eq=code|eq=token"`
  RedirectUris            []string `json:"redirect_uris"              validate:"omitempty,dive,url"`
  TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method" validate:"omitempty,eq=none|eq=client_secret_post|eq=client_secret_basic|eq=private_key_jwt"`
  PostLogoutRedirectUris  []string `json:"post_logout_redirect_uris"  validate:"omitempty,dive,url"`
}

type ReadClientsResponse []Client
type ReadClientsRequest struct {
  Id string `json:"id,omitempty" validate:"uuid"`
}

type DeleteClientsResponse Identity
type DeleteClientsRequest struct {
  Id string `json:"id" validate:"required,uuid"`
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

func DeleteClients(client *IdpClient, url string, requests []DeleteClientsRequest) (status int, responses bulky.Responses, err error) {
  status, err = handleRequest(client, requests, "DELETE", url, &responses)

  if err != nil {
    return status, nil, err
  }

  return status, responses, nil
}
