package server

import (
	"github.com/gin-gonic/gin"
	v1 "idp/controller/v1"
)

func V1Routes(r *gin.RouterGroup) {
	r.GET( "/identities", v1.GetIdentities)
	r.POST("/identities", v1.PostIdentities)
	r.PUT( "/identities", v1.PutIdentities)
	r.GET( "/identities/authenticate", v1.GetIdentitiesAuthenticate)
	r.GET( "/identities/revoke", v1.GetIdentitiesRevoke)
	r.GET( "/identities/recover", v1.GetIdentitiesRecover)
}