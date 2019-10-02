package invites

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

    var handleRequest = func(iRequests []*utils.Request) {

      acceptedBy := c.MustGet("sub").(string)

      for _, request := range iRequests {
        r := request.Request.(client.UpdateInvitesAcceptRequest)

        dbInvite, err := idp.FetchInvitesById(env.Driver, []string{r.Id})
        if err != nil {
          log.Debug(err.Error())
          request.Response = utils.NewInternalErrorResponse(request.Index)
          continue
        }

        if len(dbInvite) > 0 {
          invite := dbInvite[0]

          accept, err := idp.AcceptInvite(env.Driver, invite, idp.Identity{ Id:acceptedBy})
          if err != nil {
            log.Debug(err.Error())
            request.Response = utils.NewInternalErrorResponse(request.Index)
            continue
          }

          ok := []client.Invite{ {
              Id: accept.Id,
              IssuedAt: accept.IssuedAt,
              ExpiresAt: accept.ExpiresAt,
              Email: accept.SentTo.Email,
              Invited: accept.Invited.Id,
              HintUsername: accept.HintUsername,
              InvitedBy: accept.InvitedBy.Id,
            },
          }

          var response client.UpdateInvitesAcceptResponse
          response.Index = request.Index
          response.Status = http.StatusOK
          response.Ok = ok
          request.Response = response
          log.WithFields(logrus.Fields{ "id": accept.Id }).Debug("Invite accepted")
          continue
        }

        // Deny by default
        request.Response = utils.NewClientErrorResponse(request.Index, E.INVITE_NOT_FOUND)
        continue
      }
    }

    responses := utils.HandleBulkRestRequest(requests, handleRequest, utils.HandleBulkRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
