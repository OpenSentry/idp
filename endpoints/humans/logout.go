package humans

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  hydra "github.com/charmixer/hydra/client"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/client"
)

func PostLogout(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostLogout",
    })

    var requests []client.CreateIdentitiesLogoutRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    hydraClient := hydra.NewHydraClient(env.HydraConfig)

    var handleRequest = func(iRequests []*utils.Request) {

      for _, request := range iRequests {
        r := request.Request.(client.UpdateIdentitiesDeleteVerifyRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        hydraLogoutAcceptResponse, err := hydra.AcceptLogout(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.logoutAccept"), hydraClient, r.Challenge, hydra.LogoutAcceptRequest{
        })
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        ok := client.IdentityRedirect{
          Id: hydraLogoutAcceptResponse.Subject,
          RedirectTo: r.RedirectTo,
        }

        var response client.CreateIdentitiesLogoutResponse
        response.Index = request.Index
        response.Status = http.StatusOK
        response.Ok = ok
        request.Response = response

        log.WithFields(logrus.Fields{ "redirect_to":ok.RedirectTo }).Debug("Logout successful")
        continue
      }

    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
