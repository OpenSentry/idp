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

func PutRecoverVerification(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutRecoverVerification",
    })

    var requests []client.UpdateHumansRecoverVerifyRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {
        r := request.Input.(client.UpdateHumansRecoverVerifyRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        deny := client.UpdateHumansRecoverVerifyResponse{
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

        valid, err := idp.ValidatePassword(human.OtpRecoverCode, r.Code)
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        if valid == true {

          // Update the password
          hashedPassword, err := idp.CreatePassword(r.Password)
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          n := idp.Human{
            Identity: idp.Identity{
              Id: human.Id,
            },
            Password: hashedPassword,
          }
          updatedHuman, err := idp.UpdatePassword(env.Driver, n)
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          accept := client.UpdateHumansRecoverVerifyResponse{
            Id: updatedHuman.Id,
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
