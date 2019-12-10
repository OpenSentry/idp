package roles

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/opensentry/idp/app"
  "github.com/opensentry/idp/config"
  "github.com/opensentry/idp/gateway/idp"
  "github.com/opensentry/idp/client"
  _ "github.com/opensentry/idp/client/errors"

  aap "github.com/opensentry/aap/client"

  bulky "github.com/charmixer/bulky/server"
)

func GetRoles(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetRoles",
    })

    var requests []client.ReadRolesRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      session, tx, err := idp.BeginReadTx(env.Driver)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }
      defer tx.Close() // rolls back if not already committed/rolled back
      defer session.Close()

      requestor := c.MustGet("sub").(string)

      for _, request := range iRequests {
        var dbRoles []idp.Role
        var err error
        var ok client.ReadRolesResponse

        if request.Input == nil {
          dbRoles, err = idp.FetchRoles(tx, nil, idp.Identity{Id:requestor})
        } else {
          r := request.Input.(client.ReadRolesRequest)
          dbRoles, err = idp.FetchRoles(tx, []idp.Role{ {Identity: idp.Identity{Id: r.Id}} }, idp.Identity{Id:requestor})
        }
        if err != nil {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
          log.Debug(err.Error())
          return
        }

        if len(dbRoles) > 0 {
          for _, d := range dbRoles {
            ok = append(ok, client.Role{
              Id: d.Id,
              Name: d.Name,
              Description: d.Description,
            })
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        // Deny by default
        request.Output = bulky.NewOkResponse(request.Index, []client.Role{})
        continue
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()
        return
      }

      // Deny by default
      tx.Rollback()
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{EnableEmptyRequest: true})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func PostRoles(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostRoles",
    })

    var requests []client.CreateRolesRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      session, tx, err := idp.BeginWriteTx(env.Driver)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }
      defer tx.Close() // rolls back if not already committed/rolled back
      defer session.Close()

      requestor := c.MustGet("sub").(string)

      var ids []string

      for _, request := range iRequests {
        r := request.Input.(client.CreateRolesRequest)

        newRole := idp.Role{
          Identity: idp.Identity{
            Issuer: config.GetString("idp.public.issuer"),
          },
          Name: r.Name,
          Description: r.Description,
        }

        dbRole, err := idp.CreateRole(tx, newRole, idp.Identity{Id:requestor})
        if err != nil {
          e := tx.Rollback()
          if e != nil {
            log.Debug(e.Error())
          }
          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          log.Debug(err.Error())
          return
        }

        if dbRole != (idp.Role{}) {
          ids = append(ids, dbRole.Id)

          ok := client.CreateRolesResponse{
            Id: dbRole.Id,
            Name: dbRole.Name,
            Description: dbRole.Description,
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          //idp.EmitEventResourceServerCreated(env.Nats, resourceServer)
          continue
        }

        // Deny by default
        e := tx.Rollback()
        if e != nil {
          log.Debug(e.Error())
        }
        bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
        request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
        log.WithFields(logrus.Fields{ "name": r.Name}).Debug(err.Error())
        return
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()

        var createEntitiesRequests []aap.CreateEntitiesRequest
        for _,id := range ids {
          createEntitiesRequests = append(createEntitiesRequests, aap.CreateEntitiesRequest{
            Reference: id,
            Creator: requestor,
            Scopes: []string{
              "aap:read:grants",
              "aap:create:grants",
              "aap:delete:grants",
              "aap:read:publishes",
              "aap:create:publishes",
              "aap:delete:publishes",
              "aap:read:subscriptions",
              "aap:create:subscriptions",
              "aap:delete:subscriptions",
              "aap:read:consents",
              "aap:create:consents",
              "aap:delete:consents",
              "aap:read:shadows",
              "aap:create:shadows",
              "aap:delete:shadows",

              "mg:aap:read:grants",
              "mg:aap:create:grants",
              "mg:aap:delete:grants",
              "mg:aap:read:publishes",
              "mg:aap:create:publishes",
              "mg:aap:delete:publishes",
              "mg:aap:read:subscriptions",
              "mg:aap:create:subscriptions",
              "mg:aap:delete:subscriptions",
              "mg:aap:read:consents",
              "mg:aap:create:consents",
              "mg:aap:delete:consents",
              "mg:aap:read:shadows",
              "mg:aap:create:shadows",
              "mg:aap:delete:shadows",

              "0:mg:aap:read:grants",
              "0:mg:aap:create:grants",
              "0:mg:aap:delete:grants",
              "0:mg:aap:read:publishes",
              "0:mg:aap:create:publishes",
              "0:mg:aap:delete:publishes",
              "0:mg:aap:read:subscriptions",
              "0:mg:aap:create:subscriptions",
              "0:mg:aap:delete:subscriptions",
              "0:mg:aap:read:consents",
              "0:mg:aap:create:consents",
              "0:mg:aap:delete:consents",
              "0:mg:aap:read:shadows",
              "0:mg:aap:create:shadows",
              "0:mg:aap:delete:shadows",
            },
          })
        }

        // Initialize in AAP model
        aapClient := aap.NewAapClient(env.AapConfig)
        url := config.GetString("aap.public.url") + config.GetString("aap.public.endpoints.entities.collection")
        status, response, err := aap.CreateEntities(aapClient, url, createEntitiesRequests)

        if err != nil {
          log.WithFields(logrus.Fields{ "error": err.Error(), "ids": ids }).Debug("Failed to initialize entity in AAP model")
        }

        log.WithFields(logrus.Fields{ "status": status, "response": response }).Debug("Initialize request for role in AAP model")

        return
      }

      // Deny by default
      tx.Rollback()
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}

func DeleteRoles(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "DeleteRoles",
    })

    var requests []client.DeleteRolesRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      session, tx, err := idp.BeginWriteTx(env.Driver)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }
      defer tx.Close() // rolls back if not already committed/rolled back
      defer session.Close()

      requestor := c.MustGet("sub").(string)

      for _, request := range iRequests {
        r := request.Input.(client.DeleteRolesRequest)

        log = log.WithFields(logrus.Fields{"id": requestor})

        dbRoles, err := idp.FetchRoles(tx, []idp.Role{ {Identity: idp.Identity{Id:r.Id}} }, idp.Identity{Id:requestor})
        if err != nil {
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          log.Debug(err.Error())
          return
        }

        if len(dbRoles) <= 0  {
          // not found translate into already deleted
          ok := client.DeleteRolesResponse{ Id: r.Id }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue;
        }
        roleToDelete := dbRoles[0]

        if roleToDelete != (idp.Role{}) {

          dbDeletedRole, err := idp.DeleteRole(tx, roleToDelete, idp.Identity{Id:requestor})
          if err != nil {
            e := tx.Rollback()
            if e != nil {
              log.Debug(e.Error())
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            log.Debug(err.Error())
            return
          }

          ok := client.DeleteRolesResponse{ Id: dbDeletedRole.Id }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        // Deny by default
        e := tx.Rollback()
        if e != nil {
          log.Debug(e.Error())
        }
        bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
        request.Output = bulky.NewInternalErrorResponse(request.Index)
        log.Debug("Delete role failed. Hint: Maybe input validation needs to be improved.")
        return
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()
        return
      }

      // Deny by default
      tx.Rollback()
    }

    responses := bulky.HandleRequest(requests, handleRequests, bulky.HandleRequestParams{MaxRequests: 1})
    c.JSON(http.StatusOK, responses)
  }
  return gin.HandlerFunc(fn)
}
