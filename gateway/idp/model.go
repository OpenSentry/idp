package idp

import (
  "github.com/neo4j/neo4j-go-driver/neo4j"
	"database/sql"
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
func marshalRowToIdentity(row *sql.Rows) (Identity) {
	var (
		id string
		labels string
		issuer string
		expiresAt int64
		issuedAt int64
	)


  return Identity{
    Id:        id,
    Labels:    labels,
    Issuer:    issuer,
    ExpiresAt: expiresAt,
    IssuedAt:  issuedAt,
  }
}

type Challenge struct {
  Id string
  ChallengeType ChallengeType

  JwtRegisteredClaims

  RedirectTo   string
  CodeType     int64

  Code         string

  VerifiedAt   int64

  Data string
}

type ChallengeType int

const (
    ChallengeNotSupported ChallengeType = iota + 0 // Start a 0
    ChallengeAuthenticate
    ChallengeRecover
    ChallengeDelete
    ChallengeEmailConfirm
    ChallengeEmailChange
)

func (d ChallengeType) String() string {
    return [...]string{"ChallengeNotSupported", "ChallengeAuthenticate", "ChallengeRecover", "ChallengeDelete", "ChallengeEmailConfirm", "ChallengeEmailChange"}[d]
}

func marshalNodeToChallenge(node neo4j.Node) (Challenge) {
  p := node.Props()

  var verifiedAt int64
  if (p["verified_at"] != nil) { verifiedAt = p["verified_at"].(int64) }

  var ct ChallengeType = ChallengeNotSupported
  for _, label := range node.Labels() {

    if label == "Authenticate" {
      ct = ChallengeAuthenticate
      break;
    }

    if label == "Recover" {
      ct = ChallengeRecover
      break;
    }

    if label == "Delete" {
      ct = ChallengeDelete
      break;
    }

    if label == "EmailConfirm" {
      ct = ChallengeEmailConfirm
      break;
    }

    if label == "EmailChange" {
      ct = ChallengeEmailChange
      break;
    }
  }

  var data string
  if p["data"] != nil {
    data = p["data"].(string)
  }

  return Challenge{
    Id:         p["id"].(string),
    ChallengeType: ct,

    JwtRegisteredClaims: marshalNodeToJwtRegisteredClaims(node),

    RedirectTo: p["redirect_to"].(string),

    CodeType:   p["code_type"].(int64),
    Code:       p["code"].(string),

    VerifiedAt:   verifiedAt,

    Data: data,
  }
}

type Invite struct {
  Identity

  Email string
  Username string

  SentAt int64
}
func marshalRowToInvite(row *sql.Rows) (Invite) {
	var (
		id string
		email string
		username string
		sentAt int64
	)
  return Invite{
		Identity: Identity{Id: id},
    Email: email,
    Username: username,
    SentAt: sentAt,
  }
}

type ResourceServer struct {
  Identity
  Name         string
  Description  string
  Audience     string
}
func marshalRowToResourceServer(row *sql.Rows) (ResourceServer) {
	var (
		id string
		name string
		description string
		audience string
	)

	row.Scan(&id, &name, &description, &audience)

  return ResourceServer{
		Identity: Identity{Id: id},
    Name:         name,
    Description:  description,
    Audience:     audience,
  }
}

type Role struct {
  Identity
  Name         string
  Description  string
}
func marshalRowToRole(row *sql.Rows) (Role) {
	var (
		id string
		name string
		description string
	)

	row.Scan(&id, &name, &description)

  return Role{
		Identity: Identity{Id: id},
    Name:         name,
    Description:  description,
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
  aud := p["audiences"]
  if aud != nil {
    for _,e := range aud.([]interface{}) {
      audiences = append(audiences, e.(string))
    }
  }

  var responseTypes []string
  rt := p["response_types"]
  if rt != nil {
    for _,e := range rt.([]interface{}) {
      responseTypes = append(responseTypes, e.(string))
    }
  }

  var redirectUris []string
  ru := p["redirect_uris"]
  if ru != nil {
    for _,e := range ru.([]interface{}) {
      redirectUris = append(redirectUris, e.(string))
    }
  }

  var postLogoutRedirectUris []string
  plru := p["post_logout_redirect_uris"]
  if plru != nil {
    for _,e := range plru.([]interface{}) {
      postLogoutRedirectUris = append(postLogoutRedirectUris, e.(string))
    }
  }

  return Client{
    // Identity:                marshalRowToIdentity(node), // This is client_id // TODO
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
}
func marshalRowToHuman(row *sql.Rows) (Human) {
	var (
    email string
    email_confirmed_at int64
    username string
    name string
    allow_login bool
    password string
    totp_required bool
    totp_secret string
	)

	row.Scan(&email, &email_confirmed_at, &username, &allow_login, &password, &totp_required, &totp_secret)

  return Human{
    // Identity: marshalNodeToIdentity(node), TODO
    Email:                email,
    EmailConfirmedAt:     email_confirmed_at,
    Username:             username,
    Name:                 name,
    AllowLogin:           allow_login,
    Password:             password,
    TotpRequired:         totp_required,
    TotpSecret:           totp_secret,
  }
}
