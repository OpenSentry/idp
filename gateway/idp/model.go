package idp

import (
  "strings"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

type JwtRegisteredClaims struct {
  Issuer    string
  Subject   string
  Audience  string
  ExpiresAt int64
  NotBefore int64
  IssuedAt  int64
  JwtId     string
}
func marshalNodeToJwtRegisteredClaims(node neo4j.Node) (JwtRegisteredClaims) {
  p := node.Props()

  var iss string
  var sub string
  var aud string
  var exp int64
  var nbf int64
  var iat int64
  var jti string

  if p["iss"] != nil { iss = p["iss"].(string) }
  if p["sub"] != nil { sub = p["sub"].(string) }
  if p["aud"] != nil { aud = p["aud"].(string) }
  if p["exp"] != nil { exp = p["exp"].(int64) }
  if p["nbf"] != nil { nbf = p["nbf"].(int64) }
  if p["iat"] != nil { iat = p["iat"].(int64) }
  if p["jti"] != nil { jti = p["jti"].(string) }

  return JwtRegisteredClaims{
    Issuer:    iss,
    Subject:   sub,
    Audience:  aud,
    ExpiresAt: exp,
    NotBefore: nbf,
    IssuedAt:  iat,
    JwtId:     jti,
  }
}

type Identity struct {
  Id        string
  Labels    string

  // JWT
  // Subject string // Renamed it to Identity.Id
  Issuer    string
  ExpiresAt int64
  IssuedAt  int64

  OtpDeleteCode        string
  OtpDeleteCodeExpire  int64

  CreatedBy *Identity
}
func marshalNodeToIdentity(node neo4j.Node) (Identity) {
  p := node.Props()

  var otpDeleteCode string
  var OtpDeleteCodeExpire int64

  if p["otp_delete_code"] != nil { otpDeleteCode = p["otp_delete_code"].(string) }
  if p["otp_delete_code_expire"] != nil { OtpDeleteCodeExpire = p["otp_delete_code_expire"].(int64) }

  return Identity{
    Id:        p["id"].(string),
    Labels:    strings.Join(node.Labels(), ":"),
    Issuer:    p["iss"].(string),
    ExpiresAt: p["exp"].(int64),
    IssuedAt:  p["iat"].(int64),
    OtpDeleteCode:        otpDeleteCode,
    OtpDeleteCodeExpire:  OtpDeleteCodeExpire,
  }
}

type Challenge struct {
  Id string

  JwtRegisteredClaims

  RedirectTo   string
  CodeType     int64

  Code         string

  VerifiedAt   int64
}
func marshalNodeToChallenge(node neo4j.Node) (Challenge) {
  p := node.Props()

  var verifiedAt int64
  if (p["verified_at"] != nil) { verifiedAt = p["verified_at"].(int64) }

  return Challenge{
    Id:         p["id"].(string),

    JwtRegisteredClaims: marshalNodeToJwtRegisteredClaims(node),

    RedirectTo: p["redirect_to"].(string),

    CodeType:   p["code_type"].(int64),
    Code:       p["code"].(string),

    VerifiedAt:   verifiedAt,
  }
}

type Invite struct {
  Identity

  Email string
  Username string

  SentAt int64
}
func marshalNodeToInvite(node neo4j.Node) (Invite) {
  p := node.Props()

  var username string
  usr := p["username"]
  if usr != nil {
    username = p["username"].(string)
  }

  return Invite{
    Identity: marshalNodeToIdentity(node),

    Email: p["email"].(string),
    Username: username,
    SentAt: p["sent_at"].(int64),
  }
}

type ResourceServer struct {
  Identity
  Name         string
  Description  string
  Audience     string
}
func marshalNodeToResourceServer(node neo4j.Node) (ResourceServer) {
  p := node.Props()

  return ResourceServer{
    Identity: marshalNodeToIdentity(node),
    Name:         p["name"].(string),
    Description:  p["description"].(string),
    Audience:     p["aud"].(string),
  }
}

type Role struct {
  Identity
  Name         string
  Description  string
}
func marshalNodeToRole(node neo4j.Node) (Role) {
  p := node.Props()

  return Role{
    Identity: marshalNodeToIdentity(node),
    Name:         p["name"].(string),
    Description:  p["description"].(string),
  }
}

type Client struct {
  Identity
  Secret                   string
  Name                     string
  Description              string
  GrantTypes               []string
  Audiences                []string
  ResponseTypes            []string
  RedirectUris             []string
  PostLogoutRedirectUris   []string
  TokenEndpointAuthMethod  string
}
func marshalNodeToClient(node neo4j.Node) (Client) {
  p := node.Props()

  var secret string
  cs := p["secret"]
  if cs == nil {
    secret = ""
  } else {
    secret = cs.(string)
  }

  var grantTypes []string
  for _,e := range p["grant_types"].([]interface{}) {
    grantTypes = append(grantTypes, e.(string))
  }

  var audiences []string
  for _,e := range p["audiences"].([]interface{}) {
    audiences = append(audiences, e.(string))
  }

  var responseTypes []string
  for _,e := range p["response_types"].([]interface{}) {
    responseTypes = append(responseTypes, e.(string))
  }

  var redirectUris []string
  for _,e := range p["redirect_uris"].([]interface{}) {
    redirectUris = append(redirectUris, e.(string))
  }

  var postLogoutRedirectUris []string
  for _,e := range p["post_logout_redirect_uris"].([]interface{}) {
    postLogoutRedirectUris = append(postLogoutRedirectUris, e.(string))
  }

  return Client{
    Identity:                marshalNodeToIdentity(node), // This is client_id
    Secret:                  secret,
    Name:                    p["name"].(string),
    Description:             p["description"].(string),
    GrantTypes:              grantTypes,
    Audiences:               audiences,
    ResponseTypes:           responseTypes,
    RedirectUris:            redirectUris,
    PostLogoutRedirectUris:  postLogoutRedirectUris,
    TokenEndpointAuthMethod: p["token_endpoint_auth_method"].(string),
  }
}

type Human struct {
  Identity

  // Identity.Id aliasses
  Email                string
  EmailConfirmedAt     int64
  Username             string

  Name                 string

  AllowLogin           bool

  Password             string

  TotpRequired         bool
  TotpSecret           string

  OtpRecoverCode       string
  OtpRecoverCodeExpire int64
}
func marshalNodeToHuman(node neo4j.Node) (Human) {
  p := node.Props()

  return Human{
    Identity: marshalNodeToIdentity(node),

    Email:                p["email"].(string),
    EmailConfirmedAt:     p["email_confirmed_at"].(int64),
    Username:             p["username"].(string),

    Name:                 p["name"].(string),

    AllowLogin:           p["allow_login"].(bool),

    Password:             p["password"].(string),

    TotpRequired:         p["totp_required"].(bool),
    TotpSecret:           p["totp_secret"].(string),

    OtpRecoverCode:       p["otp_recover_code"].(string),
    OtpRecoverCodeExpire: p["otp_recover_code_expire"].(int64),

  }
}
