package identities

import (
  "net/http"
  "time"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  "github.com/dgrijalva/jwt-go"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  . "github.com/charmixer/idp/client"
)

type InviteClaims struct {
	InviterId string `json:"inviter_id"`
	jwt.StandardClaims
}

func PostInvite(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostInvite",
    })

    var input IdentitiesInviteRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    inviter, exists, err := idp.FetchIdentityById(env.Driver, input.InviterId)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return;
    }

    if exists == false {
      c.JSON(http.StatusNotFound, gin.H{"error": "Inviter not found"})
      c.Abort()
      return;
    }

    identity, exists, err := idp.FetchIdentityById(env.Driver, input.Id)
    if err != nil {
      log.Debug(err.Error())
      c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
      c.Abort()
      return;
    }

    if exists == true {

      anInvitation := idp.Invitation{
        InviterId: inviter.Id,
        Id: identity.Id,
      }
      invitation, err := idp.CreateInvitation(identity, anInvitation)
      if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      // Create JWT invite token. Beware not to put sensitive data into these as it is publicly visible.
      // Iff data is secret, then persist an annonymous token in db instead and use that as token.
	    expirationTime := time.Now().Add(1 * time.Hour) // FIXME: config invite expire time
      claims := &InviteClaims{
        InviterId: invitation.InviterId,
		    StandardClaims: jwt.StandardClaims{
          Issuer: config.GetString("idp.public.issuer"),
          Audience: config.GetString("idp.public.issuer"),
          Subject: identity.Id,
          ExpiresAt: expirationTime.Unix(),
        },
	    }

      token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
      jtwInvitation, err := token.SignedString(env.IssuerSignKey)
      if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        c.Abort()
        return
      }

      c.JSON(http.StatusOK, IdentitiesInviteResponse{
        Invitation: jtwInvitation,
      })
      return
    }

    // Deny by default
    log.WithFields(logrus.Fields{"id": input.Id}).Info("Identity not found")
    c.JSON(http.StatusNotFound, gin.H{"error": "Identity not found"})
  }
  return gin.HandlerFunc(fn)
}
