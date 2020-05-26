package resourceservers

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

func GetResourceServers(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetResourceServers",
    })

    var requests []client.ReadResourceServersRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      tx, err := env.Driver.BeginTx(c, nil)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }

      requestor := c.MustGet("sub").(string)
      var requestedBy *idp.Identity
      if requestor != "" {
        identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
        if err != nil {
          bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
          log.Debug(err.Error())
          return
        }
        if len(identities) > 0 {
          requestedBy = &identities[0]
        }
      }

      for _, request := range iRequests {

        var dbResourceServers []idp.ResourceServer
        var err error
        var ok client.ReadResourceServersResponse

        if request.Input == nil {
          dbResourceServers, err = idp.FetchResourceServers(tx, requestedBy, nil)
        } else {
          r := request.Input.(client.ReadResourceServersRequest)
          dbResourceServers, err = idp.FetchResourceServers(tx, requestedBy, []idp.ResourceServer{ {Identity:idp.Identity{Id: r.Id}} })
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

        if len(dbResourceServers) > 0 {
          for _, d := range dbResourceServers {
            ok = append(ok, client.ResourceServer{
              Id: d.Id,
              Name: d.Name,
              Description: d.Description,
              Audience: d.Audience,
            })
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        // Deny by default
        request.Output = bulky.NewOkResponse(request.Index, []client.ResourceServer{})
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

func PostResourceServers(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostResourceServers",
    })

    var requests []client.CreateResourceServersRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      tx, err := env.Driver.BeginTx(c, nil)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }

      requestor := c.MustGet("sub").(string)
      var requestedBy *idp.Identity
      if requestor != "" {
        identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
        if err != nil {
          bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
          log.Debug(err.Error())
          return
        }
        if len(identities) > 0 {
          requestedBy = &identities[0]
        }
      }

      var ids []string

      for _, request := range iRequests {
        r := request.Input.(client.CreateResourceServersRequest)

        newResourceServer := idp.ResourceServer{
          Identity: idp.Identity{
            Issuer: config.GetString("idp.public.issuer"),
          },
          Name: r.Name,
          Description: r.Description,
          Audience: r.Audience,
        }

        resourceServer, err := idp.CreateResourceServer(tx, requestedBy, newResourceServer)
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

        if resourceServer != (idp.ResourceServer{}) {
          ids = append(ids, resourceServer.Id)

          ok := client.CreateResourceServersResponse{
            Id: resourceServer.Id,
            Name: resourceServer.Name,
            Description: resourceServer.Description,
            Audience: r.Audience,
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          idp.EmitEventResourceServerCreated(env.Nats, resourceServer)
          continue
        }

        // Deny by default
        e := tx.Rollback()
        if e != nil {
          log.Debug(e.Error())
        }
        bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
        request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
        log.WithFields(logrus.Fields{ "name": newResourceServer.Name}).Debug(err.Error())
        return
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()

        var createEntitiesRequests []aap.CreateEntitiesRequest
        for _,id := range ids {
          createEntitiesRequests = append(createEntitiesRequests, aap.CreateEntitiesRequest{
            Reference: id,
            Creator: requestedBy.Id,
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

        log.WithFields(logrus.Fields{ "status": status, "response": response }).Debug("Initialize request for resourceserver in AAP model")

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

func DeleteResourceServers(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "DeleteResourceServers",
    })

    var requests []client.DeleteResourceServersRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    var handleRequests = func(iRequests []*bulky.Request) {

      tx, err := env.Driver.BeginTx(c, nil)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.Debug(err.Error())
        return
      }

      requestor := c.MustGet("sub").(string)
      var requestedBy *idp.Identity
        if requestor != "" {
        identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
        if err != nil {
          bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
          log.Debug(err.Error())
          return
        }
        if len(identities) > 0 {
          requestedBy = &identities[0]
        }
      }

      for _, request := range iRequests {
        r := request.Input.(client.DeleteResourceServersRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        dbResourceServers, err := idp.FetchResourceServers(tx, requestedBy, []idp.ResourceServer{ {Identity:idp.Identity{Id:r.Id}} })
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

        if len(dbResourceServers) <= 0  {
          // not found translate into already deleted
          ok := client.DeleteResourceServersResponse{ Id: r.Id }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue;
        }
        resourceServerToDelete := dbResourceServers[0]

        if resourceServerToDelete != (idp.ResourceServer{}) {

          deletedResourceServer, err := idp.DeleteResourceServer(tx, requestedBy, resourceServerToDelete)
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

          ok := client.DeleteResourceServersResponse{ Id: deletedResourceServer.Id }
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
        log.Debug("Delete resource server failed. Hint: Maybe input validation needs to be improved.")
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
