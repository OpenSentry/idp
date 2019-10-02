package humans

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  hydra "github.com/charmixer/hydra/client"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/client"
  "github.com/charmixer/idp/utils"
)

func PostLogout(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostLogout",
    })

    var requests []client.CreateHumansLogoutRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    hydraClient := hydra.NewHydraClient(env.HydraConfig)

    var handleRequest = func(iRequests []*utils.Request) {

      for _, request := range iRequests {
        r := request.Request.(client.CreateHumansLogoutRequest)

        hydraLogoutResponse, err := hydra.GetLogout(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.logout"), hydraClient, r.Challenge)
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        log.Debug(hydraLogoutResponse)

        hydraLogoutAcceptResponse, err := hydra.AcceptLogout(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.logoutAccept"), hydraClient, r.Challenge, hydra.LogoutAcceptRequest{})
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        ok := client.HumanRedirect{
          Id: hydraLogoutResponse.Subject,
          RedirectTo: hydraLogoutAcceptResponse.RedirectTo,
        }

        var response client.CreateHumansLogoutResponse
        response.Index = request.Index
        response.Status = http.StatusOK
        response.Ok = ok
        request.Response = response

        log.WithFields(logrus.Fields{ "id": ok.Id, "redirect_to":ok.RedirectTo }).Debug("Logout successful")
        continue
      }

    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
