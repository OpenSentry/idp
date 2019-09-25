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

  return JwtRegisteredClaims{
    Issuer:    p["id"].(string),
    Subject:   p["sub"].(string),
    Audience:  p["name"].(string),
    ExpiresAt: p["exp"].(int64),
    NotBefore: p["nbf"].(int64),
    IssuedAt:  p["iat"].(int64),
    JwtId:     p["jti"].(string),
  }
}