package humans

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"
  "github.com/charmixer/idp/utils"
)

type TotpResponse struct {
  Id string `json:"id" binding:"required"`
}

func PutTotp(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutTotp",
    })

    var requests []client.UpdateHumansTotpRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequest = func(iRequests []*utils.Request) {

      for _, request := range iRequests {
        r := request.Request.(client.UpdateHumansTotpRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        humans, err := idp.FetchHumansById( env.Driver, []string{r.Id} )
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        if humans == nil {
          log.WithFields(logrus.Fields{"id": r.Id}).Debug("Human not found")
          request.Response = utils.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
          continue
        }
        human := humans[0]

        encryptedSecret, err := idp.Encrypt(r.TotpSecret, config.GetString("totp.cryptkey"))
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        updatedHuman, err := idp.UpdateTotp(env.Driver, idp.Human{
          Identity: idp.Identity{
            Id: human.Id,
          },
          TotpRequired: r.TotpRequired,
          TotpSecret: encryptedSecret,
        })
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        var response client.UpdateHumansTotpResponse
        response.Index = request.Index
        response.Status = http.StatusOK
        response.Ok = client.Human{
          Id: updatedHuman.Id,
          Username: updatedHuman.Username,
          Password: updatedHuman.Password,
          Name: updatedHuman.Name,
          Email: updatedHuman.Email,
          AllowLogin: updatedHuman.AllowLogin,
          TotpRequired: updatedHuman.TotpRequired,
          TotpSecret: updatedHuman.TotpSecret,
          OtpRecoverCode: updatedHuman.OtpRecoverCode,
          OtpRecoverCodeExpire: updatedHuman.OtpRecoverCodeExpire,
          OtpDeleteCode: updatedHuman.OtpDeleteCode,
          OtpDeleteCodeExpire: updatedHuman.OtpDeleteCodeExpire,
        }
        request.Response = response

        log.WithFields(logrus.Fields{ "id":updatedHuman.Id }).Debug("TOTP updated")
        continue
      }

    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
