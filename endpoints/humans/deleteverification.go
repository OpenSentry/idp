package humans

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"

  bulky "github.com/charmixer/bulky/server"
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

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {
        r := request.Input.(client.UpdateHumansDeleteVerifyRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        deny := client.UpdateHumansDeleteVerifyResponse{
          Id: r.Id,
          Verified: false,
          RedirectTo: "",
        }

        humans, err := idp.FetchHumansById(env.Driver, []string{r.Id})
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        if humans == nil {
          log.WithFields(logrus.Fields{"id":r.Id}).Debug("Human not found")
          request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
          continue
        }
        human := humans[0]

        valid, err := idp.ValidatePassword(human.OtpDeleteCode, r.Code)
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        if valid == true {

          log.WithFields(logrus.Fields{"fixme":1}).Debug("Revoke all access tokens for identity - put them on revoked list or rely on expire")
          log.WithFields(logrus.Fields{"fixme":1}).Debug("Revoke all consents in hydra for identity - this is probably aap?")

          n := idp.Human{
            Identity: idp.Identity{
              Id: human.Id,
            },
          }
          deletedHuman, err := idp.DeleteHuman(env.Driver, n)
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          accept := client.UpdateHumansDeleteVerifyResponse{
            Id: deletedHuman.Id,
            Verified: true,
            RedirectTo: r.RedirectTo,
          }

          log.WithFields(logrus.Fields{ "verified":accept.Verified, "redirect_to":accept.RedirectTo }).Debug("Identity deleted")
          request.Output = bulky.NewOkResponse(request.Index, accept)
          continue
        }

        // Deny by default
        log.Debug("Verification denied")
        request.Output = bulky.NewOkResponse(request.Index, deny)
      }

    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
