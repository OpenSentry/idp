package idpbe

import (
  _ "golang.org/x/net/context"
  _ "golang.org/x/oauth2"
  "golang.org/x/oauth2/clientcredentials"

  oidc "github.com/coreos/go-oidc"
)

type Identity struct {
  Id            string          `json:"id"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
  Password      string          `json:"password"`
}

type IdpBeEnv struct {
  Provider *oidc.Provider
  HydraConfig *clientcredentials.Config
  Database map[string]*Identity
}
