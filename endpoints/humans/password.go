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

func PutPassword(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutPassword",
    })

    var requests []client.UpdateHumansPasswordRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {
        r := request.Input.(client.UpdateHumansPasswordRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        humans, err := idp.FetchHumansById( env.Driver, []string{r.Id} )
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        if humans == nil {
          log.WithFields(logrus.Fields{"id": r.Id}).Debug("Human not found")
          request.Output = bulky.NewClientErrorResponse(request.Index, E.HUMAN_NOT_FOUND)
          continue
        }
        human := humans[0]


        valid, _ := idp.ValidatePassword(human.Password, r.Password)
        if valid == true {
          // Nothing to change was the new password is same as current password

          ok := client.UpdateHumansPasswordResponse{
            Id: human.Id,
            Username: human.Username,
            Password: human.Password,
            Name: human.Name,
            Email: human.Email,
            AllowLogin: human.AllowLogin,
            TotpRequired: human.TotpRequired,
            TotpSecret: human.TotpSecret,
            OtpRecoverCode: human.OtpRecoverCode,
            OtpRecoverCodeExpire: human.OtpRecoverCodeExpire,
            OtpDeleteCode: human.OtpDeleteCode,
            OtpDeleteCodeExpire: human.OtpDeleteCodeExpire,
          }

          log.WithFields(logrus.Fields{ "id":human.Id }).Debug("Password updated. Hint: No change")
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        hashedPassword, err := idp.CreatePassword(r.Password)
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        updatedHuman, err := idp.UpdatePassword(env.Driver, idp.Human{
          Identity: idp.Identity{
            Id: human.Id,
          },
          Password: hashedPassword,
        })
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        ok := client.UpdateHumansPasswordResponse{
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

        log.WithFields(logrus.Fields{ "id":updatedHuman.Id }).Debug("Password updated")
        request.Output = bulky.NewOkResponse(request.Index, ok)
        continue
      }

    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
