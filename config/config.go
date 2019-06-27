package config

import (
  "os"
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
  LogoutRequestUrl           string
  LogoutRequestAcceptUrl    string
  LogoutRequestRejectUrl    string
  UserInfoUrl               string
}

type OAuth2ClientConfig struct {
  ClientId        string
  ClientSecret    string
  Scopes          []string
  RedirectURL     string
  Endpoint        string
}

var Hydra HydraConfig

func InitConfigurations() {
  Hydra.Url                   = getEnvStrict("HYDRA_URL")
  Hydra.AdminUrl              = getEnvStrict("HYDRA_ADMIN_URL")
  Hydra.LoginRequestUrl       = Hydra.AdminUrl + "/oauth2/auth/requests/login"
  Hydra.LoginRequestAcceptUrl = Hydra.LoginRequestUrl + "/accept"
  Hydra.LoginRequestRejectUrl = Hydra.LoginRequestUrl + "/reject"
  Hydra.LogoutRequestUrl       = Hydra.AdminUrl + "/oauth2/auth/requests/logout"
  Hydra.LogoutRequestAcceptUrl = Hydra.LogoutRequestUrl + "/accept"
  Hydra.LogoutRequestRejectUrl = Hydra.LogoutRequestUrl + "/reject"
  Hydra.UserInfoUrl            = Hydra.Url + "/userinfo"
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
