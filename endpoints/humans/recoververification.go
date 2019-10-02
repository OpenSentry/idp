package humans

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"
  "github.com/charmixer/idp/utils"
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

    var handleRequest = func(iRequests []*utils.Request) {

      for _, request := range iRequests {
        r := request.Request.(client.UpdateHumansRecoverVerifyRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        deny := client.HumanVerification{
          Id: r.Id,
          Verified: false,
          RedirectTo: "",
        }

        humans, err := idp.FetchHumansById(env.Driver, []string{r.Id})
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        if humans == nil {
          log.WithFields(logrus.Fields{"id":r.Id}).Debug("Human not found")
          request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
          continue
        }
        human := humans[0]

        valid, err := idp.ValidatePassword(human.OtpRecoverCode, r.Code)
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        if valid == true {

          // Update the password
          hashedPassword, err := idp.CreatePassword(r.Password)
          if err != nil {
            log.Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
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
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          accept := client.HumanVerification{
            Id: updatedHuman.Id,
            Verified: true,
            RedirectTo: r.RedirectTo,
          }

          var response client.UpdateHumansRecoverVerifyResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = accept
          request.Response = response

          log.WithFields(logrus.Fields{ "verified":accept.Verified, "redirect_to":accept.RedirectTo }).Debug("Identity deleted")
          continue
        }

        // Deny by default
        var response client.UpdateHumansRecoverVerifyResponse
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
