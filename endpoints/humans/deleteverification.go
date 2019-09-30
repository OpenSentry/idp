package humans

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  "github.com/charmixer/idp/utils"
  E "github.com/charmixer/idp/client/errors"
)

func PutDeleteVerification(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutDeleteVerification",
    })

    var requests []client.UpdateHumansDeleteVerifyRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequest = func(iRequests []*utils.Request) {

      for _, request := range iRequests {
        r := request.Request.(client.UpdateHumansDeleteVerifyRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        deny := client.HumanVerification{
          Id: r.Id,
          Verified: false,
          RedirectTo: "",
        }

        identities, err := idp.FetchHumansById(env.Driver, []string{r.Id})
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        if identities == nil {
          log.WithFields(logrus.Fields{"id":r.Id}).Debug("Human not found")
          request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
          continue
        }
        identity := identities[0]

        valid, err := idp.ValidatePassword(identity.OtpDeleteCode, r.Code)
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        if valid == true {

          log.WithFields(logrus.Fields{"fixme":1}).Debug("Revoke all access tokens for identity - put them on revoked list or rely on expire")
          log.WithFields(logrus.Fields{"fixme":1}).Debug("Revoke all consents in hydra for identity - this is probably aap?")

          n := idp.Human{
            Identity: idp.Identity{
              Id: identity.Id,
            },
          }
          updatedIdentity, err := idp.DeleteHuman(env.Driver, n)
          if err != nil {
            log.Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          accept := client.HumanVerification{
            Id: updatedIdentity.Id,
            Verified: true,
            RedirectTo: r.RedirectTo,
          }

          var response client.UpdateHumansDeleteVerifyResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = accept
          request.Response = response

          log.WithFields(logrus.Fields{ "verified":accept.Verified, "redirect_to":accept.RedirectTo }).Debug("Identity deleted")
          continue
        }

        // Deny by default
        var response client.UpdateHumansDeleteVerifyResponse
        response.Index = request.Index
        response.Status = http.StatusOK
        response.Ok = deny
        request.Response = response
        log.Debug("Verification denied")
      }

    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
