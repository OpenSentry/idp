package idp

import (
  hydra "github.com/charmixer/hydra/client"
  "github.com/neo4j/neo4j-go-driver/neo4j"
  "github.com/opensentry/idp/config"
)

func SyncClientsToHydra(tx neo4j.Transaction, iClients []Client) (err error) {
  // aap is not master with client data, only client scopes

  dbClients, err := FetchClients(tx, iClients)

  if err != nil {
    return err
  }

  url := config.GetString("hydra.private.url") + config.GetString("hydra.private.endpoints.clients")

  for _,dbClient := range dbClients {

    client, err := hydra.ReadClient(url, dbClient.Id)

    if err != nil {
      return err
    }

    newClient := hydra.UpdateClientRequest(client)
    newClient.Name                    = dbClient.Name
    newClient.Secret                  = dbClient.Secret
    newClient.GrantTypes              = dbClient.GrantTypes
    newClient.ResponseTypes           = dbClient.ResponseTypes
    newClient.RedirectUris            = dbClient.RedirectUris
    newClient.TokenEndpointAuthMethod = dbClient.TokenEndpointAuthMethod
    newClient.PostLogoutRedirectUris  = dbClient.PostLogoutRedirectUris

    _, err = hydra.UpdateClient(url, newClient.Id, newClient)

    if err != nil {
      return err
    }

  }

  return nil
}
