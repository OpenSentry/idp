package environment

import (
  //"fmt"

  "golang.org/x/oauth2/clientcredentials"

  oidc "github.com/coreos/go-oidc"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

const (
  RequestIdKey string = "RequestId"
  AccessTokenKey = "access_token"
  IdTokenKey = "id_token"
  LogKey = "log"
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
