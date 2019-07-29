package identities

import (
  "net/http"

  "github.com/gin-gonic/gin"

  "golang-idp-be/config"
  "golang-idp-be/environment"
  "golang-idp-be/gateway/idpbe"
  "golang-idp-be/gateway/hydra"
)

type AuthenticateRequest struct {
  Id              string            `json:"id"`
  Password        string            `json:"password"`
  Challenge       string            `json:"challenge" binding:"required"`
}

type AuthenticateResponse struct {
  Id              string            `json:"id"`
  Authenticated   bool              `json:"authenticated"`
}

const HydraSessionTimeout = 120 // 2m

func PostAuthenticate(env *environment.State, route environment.Route) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    requestId := c.MustGet("RequestId").(string)
    environment.DebugLog(route.LogId, "PostAuthenticate", "", requestId)

    var input AuthenticateRequest
    err := c.BindJSON(&input)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    // Create a new HTTP client to perform the request, to prevent serialization
    hydraClient := hydra.NewHydraClient(env.HydraConfig)

    hydraLoginResponse, err := hydra.GetLogin(config.Discovery.Hydra.Private.Url + config.Discovery.Hydra.Private.Endpoints.Login, hydraClient, input.Challenge)
    if err != nil {
      c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      c.Abort()
      return
    }

    if hydraLoginResponse.Skip {
      hydraLoginAcceptRequest := hydra.HydraLoginAcceptRequest{
        Subject: hydraLoginResponse.Subject,
        Remember: true,
        RememberFor: HydraSessionTimeout, // This means auto logout in hydra after 30 seconds!
      }

      hydraLoginAcceptResponse := hydra.AcceptLogin(config.Discovery.Hydra.Private.Url + config.Discovery.Hydra.Private.Endpoints.LoginAccept, hydraClient, input.Challenge, hydraLoginAcceptRequest)

      environment.DebugLog(route.LogId, "PostAuthenticate", "id:"+input.Id+" authenticated:true redirect_to:"+hydraLoginAcceptResponse.RedirectTo, requestId)
      c.JSON(http.StatusOK, gin.H{
        "id": input.Id,
        "authenticated": true,
        "redirect_to": hydraLoginAcceptResponse.RedirectTo,
      })
      c.Abort()
      return
    }

    // Only challenge is required in the request, but no need to ask DB for empty id.
    if input.Id == "" {
      environment.DebugLog(route.LogId, "PostAuthenticate", "id:"+input.Id+" authenticated:false redirect_to:", requestId)
      c.JSON(http.StatusOK, gin.H{
        "id": input.Id,
        "authenticated": false,
      })
      c.Abort()
      return
    }

    identities, err := idpbe.FetchIdentitiesForSub(env.Driver, input.Id)
    if err != nil {
      environment.DebugLog(route.LogId, "PostAuthenticate", "id:"+input.Id+" authenticated:false redirect_to:", requestId)
      c.JSON(http.StatusOK, gin.H{
        "id": input.Id,
        "authenticated": false,
      })
      c.Abort()
      return;
    }

    if identities != nil {

      // FIXME: Fail of identities contains more than one. Hint: Missing a unique constraint in the db schema?
      identity := identities[0];

      valid, _ := idpbe.ValidatePassword(identity.Password, input.Password)
      if valid == true {
        hydraLoginAcceptRequest := hydra.HydraLoginAcceptRequest{
          Subject: identity.Id,
          Remember: true,
          RememberFor: HydraSessionTimeout, // This means auto logout in hydra after 30 seconds!
        }

        hydraLoginAcceptResponse := hydra.AcceptLogin(config.Discovery.Hydra.Private.Url + config.Discovery.Hydra.Private.Endpoints.LoginAccept, hydraClient, input.Challenge, hydraLoginAcceptRequest)

        environment.DebugLog(route.LogId, "PostAuthenticate", "id:"+identity.Id+" authenticated:true redirect_to:"+hydraLoginAcceptResponse.RedirectTo, requestId)
        c.JSON(http.StatusOK, gin.H{
          "id": identity.Id,
          "authenticated": true,
          "redirect_to": hydraLoginAcceptResponse.RedirectTo,
        })
        c.Abort()
        return
      }

    } else {
      environment.DebugLog(route.LogId, "PostAuthenticate", "No identities found", requestId)
    }

    // Deny by default
    environment.DebugLog(route.LogId, "PostAuthenticate", "id:"+input.Id+" authenticated:false redirect_to:", requestId)
    c.JSON(http.StatusOK, gin.H{
      "id": input.Id,
      "authenticated": false,
    })
  }
  return gin.HandlerFunc(fn)
}
