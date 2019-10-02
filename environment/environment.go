package environment

import (
  "crypto/rsa"
  "golang.org/x/oauth2/clientcredentials"
  oidc "github.com/coreos/go-oidc"
  "github.com/neo4j/neo4j-go-driver/neo4j"
  nats "github.com/nats-io/nats.go"
)

const (
  RequestIdKey string = "RequestId"
  AccessTokenKey string = "access_token"
  IdTokenKey string = "id_token"
  LogKey string = "log"
)

type State struct {
  Provider *oidc.Provider
  HydraConfig *clientcredentials.Config
  Driver   neo4j.Driver
  BannedUsernames map[string]bool
  IssuerSignKey *rsa.PrivateKey
  IssuerVerifyKey *rsa.PublicKey
  Nats *nats.Conn
}

type Route struct {
  URL string
  LogId string
}
