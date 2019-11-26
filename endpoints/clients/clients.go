package clients

import (
  "net/http"
  "github.com/sirupsen/logrus"
  "github.com/gin-gonic/gin"

  "github.com/charmixer/idp/app"
  "github.com/charmixer/idp/config"
  "github.com/charmixer/idp/gateway/idp"
  "github.com/charmixer/idp/utils"
  "github.com/charmixer/idp/client"
  _ "github.com/charmixer/idp/client/errors"

  aap "github.com/charmixer/aap/client"
  hydra "github.com/charmixer/hydra/client"
  bulky "github.com/charmixer/bulky/server"
)

func GetClients(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {
    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "GetClients",
    })

    var requests []client.ReadClientsRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    keys := config.GetStringSlice("crypto.keys.clients")
    if len(keys) <= 0 {
      log.WithFields(logrus.Fields{"key":"crypto.keys.clients"}).Debug("Missing config")
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    cryptoKey := keys[0]

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

        var dbClients []idp.Client
        var err error
        var ok client.ReadClientsResponse

        if request.Input == nil {
          dbClients, err = idp.FetchClients(tx, requestedBy, nil)
        } else {
          r := request.Input.(client.ReadClientsRequest)
          dbClients, err = idp.FetchClients(tx, requestedBy, []idp.Client{ {Identity:idp.Identity{Id: r.Id}} })
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

        if len(dbClients) > 0 {
          for _, d := range dbClients {

            var descryptedClientSecret string = ""
            if d.Secret != "" {
              descryptedClientSecret, err = idp.Decrypt(d.Secret, cryptoKey)
              if err != nil {
                log.Debug(err.Error())
                descryptedClientSecret = ""
              }
            }

            ok = append(ok, client.Client{
              Id: d.Id,
              Secret: descryptedClientSecret,
              Name: d.Name,
              Description: d.Description,
              GrantTypes: d.GrantTypes,
              ResponseTypes: d.ResponseTypes,
              RedirectUris: d.RedirectUris,
              TokenEndpointAuthMethod: d.TokenEndpointAuthMethod,
              PostLogoutRedirectUris: d.PostLogoutRedirectUris,
            })
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          continue
        }

        // Deny by default
        request.Output = bulky.NewOkResponse(request.Index, []client.Client{})
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

func PostClients(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "PostClients",
    })

    var requests []client.CreateClientsRequest
    err := c.BindJSON(&requests)
    if err != nil {
      c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

    keys := config.GetStringSlice("crypto.keys.clients")
    if len(keys) <= 0 {
      log.WithFields(logrus.Fields{"key":"crypto.keys.clients"}).Debug("Missing config")
      c.AbortWithStatus(http.StatusInternalServerError)
      return
    }
    cryptoKey := keys[0]

    var handleRequests = func(iRequests []*bulky.Request) {

      session, tx, err := idp.BeginWriteTx(env.Driver)
      if err != nil {
        bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
        log.WithFields(logrus.Fields{ "error": err.Error() }).Debug("Failed to begin transaction")
        return
      }
      defer tx.Close() // rolls back if not already committed/rolled back
      defer session.Close()

      requestor := c.MustGet("sub").(string)
      var requestedBy *idp.Identity
      if requestor != "" {
        identities, err := idp.FetchIdentities(tx, []idp.Identity{ {Id:requestor} })
        if err != nil {
          bulky.FailAllRequestsWithInternalErrorResponse(iRequests)
          log.WithFields(logrus.Fields{ "error": err.Error(), "id": requestor }).Debug("Failed to fetch identity")
          return
        }
        if len(identities) > 0 {
          requestedBy = &identities[0]
        }
      }

      var newClients []idp.Client

      for _, request := range iRequests {
        r := request.Input.(client.CreateClientsRequest)

        newClient := idp.Client{
          Identity: idp.Identity{
            Issuer: config.GetString("idp.public.issuer"),
          },
          Name: r.Name,
          Description: r.Description,
          Secret: r.Secret,
          GrantTypes: r.GrantTypes,
          ResponseTypes: r.ResponseTypes,
          RedirectUris: r.RedirectUris,
          TokenEndpointAuthMethod: r.TokenEndpointAuthMethod,
          PostLogoutRedirectUris: r.PostLogoutRedirectUris,
        }

        var secret string
        if r.IsPublic == false {

          if r.Secret == "" {

            // BCrypt used by hydra to store passwords securely limits password to 55 chars not counting the terminating zero
            secret, err = utils.GenerateRandomHex(55)

            if err != nil {
              log.WithFields(logrus.Fields{ "error": err.Error() }).Debug("Failed to generate random secret")

              e := tx.Rollback()
              if e != nil {
                log.WithFields(logrus.Fields{ "error": e.Error() }).Debug("Failed to rollback transaction")
              }
              bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
              request.Output = bulky.NewInternalErrorResponse(request.Index)
              return
            }
          } else {
            secret = r.Secret
          }

          encryptedClientSecret, err := idp.Encrypt(secret, cryptoKey) // Encrypt the secret before storage
          if err != nil {
            log.WithFields(logrus.Fields{ "error": err.Error() }).Debug("Failed to encrypt secret")

            e := tx.Rollback()
            if e != nil {
              log.WithFields(logrus.Fields{ "error": e.Error() }).Debug("Failed to rollback transaction")
            }
            bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
            request.Output = bulky.NewInternalErrorResponse(request.Index)
            return
          }

          newClient.Secret = encryptedClientSecret
        }

        objClient, err := idp.CreateClient(tx, requestedBy, newClient)
        if err != nil {
          log.WithFields(logrus.Fields{ "error": err.Error() }).Debug("Failed to create client")

          e := tx.Rollback()
          if e != nil {
            log.WithFields(logrus.Fields{ "error": e.Error() }).Debug("Failed to rollback transaction")
          }

          bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
          request.Output = bulky.NewInternalErrorResponse(request.Index)
          return
        }

        if objClient.Id != "" {

          // The client in the db is encrypted, we need the clean password to return to user and use in hydra.
          objClient.Secret = secret

          newClients = append(newClients, objClient)

          ok := client.CreateClientsResponse{
            Id: objClient.Id,
            Secret: objClient.Secret,
            Name: objClient.Name,
            Description: objClient.Description,
            GrantTypes: objClient.GrantTypes,
            ResponseTypes: objClient.ResponseTypes,
            RedirectUris: objClient.RedirectUris,
            TokenEndpointAuthMethod: objClient.TokenEndpointAuthMethod,
            PostLogoutRedirectUris: objClient.PostLogoutRedirectUris,
          }
          request.Output = bulky.NewOkResponse(request.Index, ok)
          idp.EmitEventClientCreated(env.Nats, objClient)
          continue
        }

        // Deny by default
        e := tx.Rollback()
        if e != nil {
          log.WithFields(logrus.Fields{ "error": e.Error() }).Debug("Failed to rollback transaction")
        }
        bulky.FailAllRequestsWithServerOperationAbortedResponse(iRequests) // Fail all with abort
        request.Output = bulky.NewInternalErrorResponse(request.Index) // Specify error on failed one
        // @SecurityRisk: Please _NEVER_ log the hashed client_secret
        log.WithFields(logrus.Fields{ "name": newClient.Name, /* "client_secret": newClient.ClientSecret, */ }).Debug(err.Error())
        return
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()

        // proxy to hydra
        var hydraClients []hydra.CreateClientRequest

        var createEntitiesRequests []aap.CreateEntitiesRequest
        for _,c := range newClients {
          hydraClients = append(hydraClients, hydra.CreateClientRequest{
            Id:                      c.Id,
            Name:                    c.Name,
            Secret:                  c.Secret,
            Scope:                   "", // nothing yet, subscribe does this
            GrantTypes:              c.GrantTypes,
            Audience:                c.Audiences,
            ResponseTypes:           c.ResponseTypes,
            RedirectUris:            c.RedirectUris,
            PostLogoutRedirectUris:  c.PostLogoutRedirectUris,
            TokenEndpointAuthMethod: c.TokenEndpointAuthMethod,
          })

          createEntitiesRequests = append(createEntitiesRequests, aap.CreateEntitiesRequest{
            Reference: c.Id,
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

        url := config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.clients")
        for _, h := range hydraClients {
          c, err :=hydra.CreateClient(url, h)
          if err != nil {
            log.Debug(err.Error())
          } else {
            log.Debug(c)
          }
        }

        // Initialize in AAP model
        aapClient := aap.NewAapClient(env.AapConfig)
        url = config.GetString("aap.public.url") + config.GetString("aap.public.endpoints.entities.collection")
        status, response, err := aap.CreateEntities(aapClient, url, createEntitiesRequests)

        if err != nil {
          log.WithFields(logrus.Fields{ "error": err.Error(), "newClients": newClients }).Debug("Failed to initialize entity in AAP model")
        }

        log.WithFields(logrus.Fields{ "status": status, "response": response }).Debug("Initialize request for clients in AAP model")

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

func DeleteClients(env *app.Environment) gin.HandlerFunc {
  fn := func(c *gin.Context) {

    log := c.MustGet(env.Constants.LogKey).(*logrus.Entry)
    log = log.WithFields(logrus.Fields{
      "func": "DeleteClients",
    })

    var requests []client.DeleteClientsRequest
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

      var deleteHydraClients []string

      for _, request := range iRequests {
        r := request.Input.(client.DeleteClientsRequest)

        log = log.WithFields(logrus.Fields{"id": r.Id})

        dbClients, err := idp.FetchClients(tx, requestedBy, []idp.Client{ {Identity:idp.Identity{Id:r.Id}} })
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

        if len(dbClients) <= 0  {
          // not found translates to already deleted
          request.Output = bulky.NewOkResponse(request.Index, client.DeleteClientsResponse{Id: r.Id})
          continue
        }
        clientToDelete := dbClients[0]

        if clientToDelete.Id != "" {

          deleteHydraClients = append(deleteHydraClients, clientToDelete.Id)

          deletedClient, err := idp.DeleteClient(tx, requestedBy, clientToDelete)
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

          ok := client.DeleteClientsResponse{ Id: deletedClient.Id }
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
        log.Debug("Delete client failed. Hint: Maybe input validation needs to be improved.")
        return
      }

      err = bulky.OutputValidateRequests(iRequests)
      if err == nil {
        tx.Commit()

        // proxy to hydra
        url := config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.clients")
        for _,c := range deleteHydraClients {
          hydra.DeleteClient(url, c)
        }

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
