package invites

import (
  "net/http"
  "net/url"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/environment"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/endpoints/challenges"
  "github.com/charmixer/idp/client"
  E "github.com/charmixer/idp/client/errors"

  bulky "github.com/charmixer/bulky/server"
)

type EmailConfirmTemplateData struct {
  Name string
  Code string
  Sender string
}

func PostInvitesClaim(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostInvitesClaim",
    })

    var requests []client.CreateInvitesClaimRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    requestor := c.MustGet("sub").(string)
    var requestedBy *idp.Identity
    if requestor != "" {
      identities, err := idp.FetchIdentitiesById(env.Driver, []string{ requestor })
      if err != nil {
        log.Debug(err.Error())
        c.AbortWithStatus(http.StatusInternalServerError)
        return
      }
      if len(identities) > 0 {
        requestedBy = &identities[0]
      }
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      for _, request := range iRequests {
        r := request.Input.(client.CreateInvitesClaimRequest)

        // Does indentity on email already exists?
        dbInvites, err := idp.FetchInvitesById(env.Driver, requestedBy, []string{r.Id})
        if err != nil {
          log.Debug(err.Error())
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          continue
        }
        if len(dbInvites) > 0 {
          invite := dbInvites[0]

          // Create email claim challenge based of the invite
          redirectToUrlWhenVerified, err := url.Parse( r.RedirectTo )
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          epVerifyController := config.GetString("idpui.public.url") + config.GetString("idpui.public.endpoints.emailconfirm")
          redirectToConfirm, err := url.Parse(epVerifyController)
          if err != nil {
            log.WithFields(logrus.Fields{ "url":epVerifyController }).Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          r := client.CreateChallengesRequest{
            Subject: invite.Id,
            TTL: r.TTL,
            RedirectTo: redirectToUrlWhenVerified.String(),
            CodeType: int64(client.OTP),
            Email: invite.Email,
            Template: client.ConfirmEmail,
          }
          challenge, err := challenges.CreateChallengeForOTP(env, r)
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          q := redirectToConfirm.Query()
          q.Add("email_challenge", challenge.Id)
          redirectToConfirm.RawQuery = q.Encode()

          ok := client.CreateInvitesClaimResponse{ RedirectTo: redirectToConfirm.String() }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        // Deny by default
        request.Output = bulky.NewClientErrorResponse(request.Index, E.INVITE_NOT_FOUND)
        continue
      }
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
