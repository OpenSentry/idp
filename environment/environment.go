package environment

import (
  "crypto/rsa"
  "golang.org/x/oauth2/clientcredentials"
  oidc "github.com/coreos/go-oidc"
  "github.com/neo4j/neo4j-go-driver/neo4j"
  nats "github.com/nats-io/nats.go"
  "github.com/charmixer/idp/gateway/idp"
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
  AapConfig *clientcredentials.Config
  Driver   neo4j.Driver
  BannedUsernames map[string]bool
  IssuerSignKey *rsa.PrivateKey
  IssuerVerifyKey *rsa.PublicKey
  Nats *nats.Conn
  TemplateMap *map[idp.ChallengeType]EmailTemplate
}

type EmailTemplate struct {
  Sender idp.SMTPSender
  File string
  Subject string
}