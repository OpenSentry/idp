package server

import (
  "github.com/gin-gonic/gin"
  v1 "golang-idp-be/controller/v1"
)

func V1Routes(r *gin.RouterGroup) {
  r.GET( "/identities", v1.GetIdentities)
  r.POST("/identities", v1.PostIdentities)
  r.PUT( "/identities", v1.PutIdentities)
  r.POST( "/identities/authenticate", v1.PostIdentitiesAuthenticate)
  r.POST( "/identities/revoke", v1.PostIdentitiesRevoke)
  r.POST( "/identities/recover", v1.PostIdentitiesRecover)
}
