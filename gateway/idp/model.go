package idp

import (
  "strings"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

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