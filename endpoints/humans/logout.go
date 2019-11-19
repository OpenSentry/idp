package humans

import (
  "net/http"
  "net/url"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"
  hydra "github.com/charmixer/hydra/client"

  "github.com/charmixer/idp/app"
  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"

  bulky "github.com/charmixer/bulky/server"
)

func PostLogout(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostLogout",
    })

    var requests []client.CreateHumansLogoutRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    urlLogout := config.GetString("hydra.public.url") + config.GetString("hydra.public.endpoints.logout")
    if urlLogout == "" {
      log.Debug("Missing config hydra.public.url + hydra.public.endpoints.logout")
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    logoutUrl, err := url.Parse(urlLogout)
    if err != nil {
      log.Debug(err.Error())
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {
        r := request.Input.(client.CreateHumansLogoutRequest)

        _logoutUrl := logoutUrl
        q := _logoutUrl.Query()
        q.Add("id_token_hint", r.IdToken)
        q.Add("state", r.State)

        if r.RedirectTo != ""  {
          q.Add("post_logout_redirect_uri", r.RedirectTo)
        }

        _logoutUrl.RawQuery = q.Encode()

        request.Output = bulky.NewOkResponse(request.Index, client.CreateHumansLogoutResponse{
          RedirectTo: _logoutUrl.String(),
        })
        continue
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        return
      }

      // Deny by default
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func GetLogout(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetLogout",
    })

    var requests []client.ReadHumansLogoutRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    hydraClient := hydra.NewHydraClient(env.HydraConfig)

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {
        r := request.Input.(client.ReadHumansLogoutRequest)

        log = log.WithFields(logrus.Fields{"challenge": r.Challenge})

        hydraLogoutResponse, err := hydra.GetLogout(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.logout"), hydraClient, r.Challenge)
        if err != nil {
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
          log.Debug(err.Error())
          return
        }

        if hydraLogoutResponse.Subject == "" {

          request.Output = bulky.NewClientErrorResponse(request.Index, E.IDENTITY_NOT_FOUND)

        } else {

          request.Output = bulky.NewOkResponse(request.Index, client.ReadHumansLogoutResponse{
            SessionId: hydraLogoutResponse.Sid,
            InitiatedByRelayingParty: hydraLogoutResponse.RpInitiated,
            Id: hydraLogoutResponse.Subject,
            RequestUrl: hydraLogoutResponse.RequestUrl,
          })

        }
        continue
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        return
      }

      // Deny by default
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func PutLogout(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutLogout",
    })

    var requests []client.UpdateHumansLogoutAcceptRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    hydraClient := hydra.NewHydraClient(env.HydraConfig)

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {
        r := request.Input.(client.UpdateHumansLogoutAcceptRequest)

        log = log.WithFields(logrus.Fields{"challenge": r.Challenge})

        hydraLogoutResponse, err := hydra.GetLogout(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.logout"), hydraClient, r.Challenge)
        if err != nil {
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
          log.Debug(err.Error())
          return
        }

        if hydraLogoutResponse.Subject == "" {

          request.Output = bulky.NewClientErrorResponse(request.Index, E.IDENTITY_NOT_FOUND)

        } else {

          hydraLogoutAcceptResponse, err := hydra.AcceptLogout(config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.logoutAccept"), hydraClient, r.Challenge, hydra.LogoutAcceptRequest{})
          if err != nil {
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
            log.Debug(err.Error())
            return
          }

          request.Output = bulky.NewOkResponse(request.Index, client.UpdateHumansLogoutAcceptResponse{
            Id: hydraLogoutResponse.Subject,
            RedirectTo: hydraLogoutAcceptResponse.RedirectTo,
          })

        }
        continue

      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        return
      }

      // Deny by default
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}