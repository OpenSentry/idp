package environment

import (
  "golang.org/x/oauth2/clientcredentials"
  oidc "github.com/coreos/go-oidc"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

const (
  RequestIdKey string = "RequestId"
  AccessTokenKey string = "access_token"
  IdTokenKey string = "id_token"
  LogKey string = "log"
)

type State struct {
  AppName string
  Provider *oidc.Provider
  HydraConfig *clientcredentials.Config
  Driver   neo4j.Driver
}

type Route struct {
  URL string
  LogId string
}
