package config

import (
  "os"
  "strings"
)

/*
RedirectURL:  redirect url,
ClientID:     "GOOGLE_CLIENT_ID",
ClientSecret: "CLIENT_SECRET",
Scopes:       []string{"scope1", "scope2"},
Endpoint:     oauth2 endpoint,
*/

type HydraConfig struct {
  Url                       string
  AdminUrl                  string
  LoginRequestUrl           string
  LoginRequestAcceptUrl     string
  LoginRequestRejectUrl     string
}

type OAuth2ClientConfig struct {
  ClientId        string
  ClientSecret    string
  Scopes          []string
  RedirectURL     string
  Endpoint        string
}

var Hydra HydraConfig
var OAuth2Client OAuth2ClientConfig

func InitConfigurations() {
  Hydra.Url                   = getEnvStrict("HYDRA_URL")
  Hydra.AdminUrl              = getEnvStrict("HYDRA_ADMIN_URL")
  Hydra.LoginRequestUrl       = Hydra.AdminUrl + "/oauth2/auth/requests/login"
  Hydra.LoginRequestAcceptUrl = Hydra.LoginRequestUrl + "/accept"
  Hydra.LoginRequestRejectUrl = Hydra.LoginRequestUrl + "/reject"

  OAuth2Client.ClientId       = getEnv("OAUTH2_CLIENT_CLIENT_ID")
  OAuth2Client.ClientSecret   = getEnv("OAUTH2_CLIENT_ClIENT_SECRET")
  OAuth2Client.Scopes         = strings.Split(getEnv("OAUTH2_CLIENT_SCOPES"), ",")
  OAuth2Client.RedirectURL    = getEnv("OAUTH2_CLIENT_REDIRECT_URL")
  OAuth2Client.Endpoint       = getEnv("OAUTH2_CLIENT_ENDPOINT")
}

func getEnv(name string) string {
  return os.Getenv(name)
}

func getEnvStrict(name string) string {
  r := getEnv(name)

  if r == "" {
    panic("Missing environment variable: " + name)
  }

  return r
}
