package humans

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  hydra "github.com/charmixer/hydra/client"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/client"

  bulky "github.com/charmixer/bulky/server"  
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

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {
        r := request.Input.(client.CreateHumansLogoutRequest)

        hydraLogoutResponse, err := hydra.GetLogout(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.logout"), hydraClient, r.Challenge)
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        log.Debug(hydraLogoutResponse)

        hydraLogoutAcceptResponse, err := hydra.AcceptLogout(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.logoutAccept"), hydraClient, r.Challenge, hydra.LogoutAcceptRequest{})
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }

        ok := client.CreateHumansLogoutResponse{
          Id: hydraLogoutResponse.Subject,
          RedirectTo: hydraLogoutAcceptResponse.RedirectTo,
        }

        log.WithFields(logrus.Fields{ "id": ok.Id, "redirect_to":ok.RedirectTo }).Debug("Logout successful")
        request.Output = bulky.NewOkResponse(request.Index, ok)
        continue
      }

    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
