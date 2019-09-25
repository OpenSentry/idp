package idp

import (
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