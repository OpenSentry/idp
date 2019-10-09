package invites

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

func PutInvitesAccept(env *environment.State) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(environment.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PutInvitesAccept",
    })

    var requests []client.UpdateInvitesAcceptRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      acceptedBy := c.MustGet("sub").(string)

      for _, request := range iRequests {
        r := request.Input.(client.UpdateInvitesAcceptRequest)

        dbInvite, err := idp.FetchInvitesById(env.Driver, []string{r.Id})
        if err != nil {
          log.Debug(err.Error())
          bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
          return
        }

        if len(dbInvite) > 0 {
          invite := dbInvite[0]

          accept, err := idp.AcceptInvite(env.Driver, invite, idp.Identity{ Id:acceptedBy})
          if err != nil {
            log.Debug(err.Error())
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            continue
          }

          ok := client.UpdateInvitesAcceptResponse{
            Id: accept.Id,
            IssuedAt: accept.IssuedAt,
            ExpiresAt: accept.ExpiresAt,
            Email: accept.SentTo.Email,
            Invited: accept.Invited.Id,
            HintUsername: accept.HintUsername,
            InvitedBy: accept.InvitedBy.Id,
          }

          request.Output = bulky.NewOkResponse(request.Index, ok)
          log.WithFields(logrus.Fields{ "id": accept.Id }).Debug("Invite accepted")
          continue
        }

        // Deny by default
        request.Output = bulky.NewClientErrorResponse(request.Index, E.IDENTITY_NOT_FOUND)
        continue
      }
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
