package app

import (
	"crypto/rsa"
	oidc "github.com/coreos/go-oidc/v3/oidc"
	nats "github.com/nats-io/nats.go"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/opensentry/idp/gateway/idp"
)

type EnvironmentConstants struct {
	RequestIdKey   string
	LogKey         string
	AccessTokenKey string
	IdTokenKey     string

	ContextAccessTokenKey        string
	ContextIdTokenKey            string
	ContextIdTokenRawKey         string
	ContextIdTokenHintKey        string
	ContextIdentityKey           string
	ContextOAuth2ConfigKey       string
	ContextRequiredScopesKey     string
	ContextPrecalculatedStateKey string
}

type Environment struct {
	Constants *EnvironmentConstants

	Logger *logrus.Logger

	Provider *oidc.Provider

	HydraConfig *clientcredentials.Config
	AapConfig   *clientcredentials.Config

	Driver          neo4j.Driver
	BannedUsernames map[string]bool
	IssuerSignKey   *rsa.PrivateKey
	IssuerVerifyKey *rsa.PublicKey
	Nats            *nats.Conn
	TemplateMap     *map[idp.ChallengeType]EmailTemplate
}

type EmailTemplate struct {
	Sender  idp.SMTPSender
	File    string
	Subject string
}
