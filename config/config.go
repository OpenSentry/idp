package config

import (
  "os"
)

type SelfConfig struct {
  Port          string
}

type HydraConfig struct {
  Url             string
  AdminUrl        string
  AuthenticateUrl string
  TokenUrl        string
  UserInfoUrl     string

  LoginRequestUrl           string
  LoginRequestAcceptUrl     string
  LoginRequestRejectUrl     string
  LogoutRequestUrl          string
  LogoutRequestAcceptUrl    string
  LogoutRequestRejectUrl    string
}

type IdpBeConfig struct {
  ClientId string
  ClientSecret string
  RequiredScopes []string
}

var Hydra HydraConfig
var IdpBe IdpBeConfig
var Self SelfConfig

func InitConfigurations() {
  Self.Port                   = getEnvStrict("PORT")

  Hydra.Url                   = getEnvStrict("HYDRA_URL")
  Hydra.AdminUrl              = getEnvStrict("HYDRA_ADMIN_URL")
  Hydra.AuthenticateUrl       = Hydra.Url + "/oauth2/auth"
  Hydra.TokenUrl              = Hydra.Url + "/oauth2/token"
  Hydra.UserInfoUrl           = Hydra.Url + "/userinfo"

  Hydra.LoginRequestUrl       = Hydra.AdminUrl + "/oauth2/auth/requests/login"
  Hydra.LoginRequestAcceptUrl = Hydra.LoginRequestUrl + "/accept"
  Hydra.LoginRequestRejectUrl = Hydra.LoginRequestUrl + "/reject"

  Hydra.LogoutRequestUrl       = Hydra.AdminUrl + "/oauth2/auth/requests/logout"
  Hydra.LogoutRequestAcceptUrl = Hydra.LogoutRequestUrl + "/accept"
  Hydra.LogoutRequestRejectUrl = Hydra.LogoutRequestUrl + "/reject"

  IdpBe.ClientId              = getEnvStrict("IDP_BACKEND_OAUTH2_CLIENT_ID")
  IdpBe.ClientSecret          = getEnvStrict("IDP_BACKEND_OAUTH2_CLIENT_SECRET")
  IdpBe.RequiredScopes        = []string{"openid"}
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
