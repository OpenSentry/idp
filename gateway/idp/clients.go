package idp

import (
  "errors"
  "strings"
  "fmt"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

func CreateClient(tx neo4j.Transaction, newClient Client) (client Client, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  if newClient.Issuer == "" {
    return Client{}, errors.New("Missing Client.Issuer")
  }
  params["iss"] = newClient.Issuer

  params["exp"] = newClient.ExpiresAt

  if newClient.Name == "" {
    return Client{}, errors.New("Missing Client.Name")
  }
  params["name"] = newClient.Name

  if newClient.Description == "" {
    return Client{}, errors.New("Missing Client.Description")
  }
  params["description"] = newClient.Description

  params["grantTypes"] = []string{}
  params["responseTypes"] = []string{}
  params["redirectUris"] = []string{}
  params["postLogoutRedirectUris"] = []string{}
  params["audiences"] = []string{}
  params["tokenEndpointAuthMethod"] = ""

  if len(newClient.GrantTypes) > 0 {
    params["grantTypes"] = newClient.GrantTypes
  }
  if len(newClient.ResponseTypes) > 0 {
    params["responseTypes"] = newClient.ResponseTypes
  }
  if len(newClient.RedirectUris) > 0 {
    params["redirectUris"] = newClient.RedirectUris
  }
  if len(newClient.PostLogoutRedirectUris) > 0 {
    params["postLogoutRedirectUris"] = newClient.PostLogoutRedirectUris
  }
  if len(newClient.Audiences) > 0 {
    params["audiences"] = newClient.Audiences
  }
  if newClient.TokenEndpointAuthMethod != "" {
    params["tokenEndpointAuthMethod"] = newClient.TokenEndpointAuthMethod
  }

  cypClientSecret := ""
  if newClient.Secret != "" {
    params["client_secret"] = newClient.Secret
    cypClientSecret = `secret:$client_secret,`
  }

  cypher = fmt.Sprintf(`
    CREATE (c:Client:Identity {
      id:randomUUID(),
      iat:datetime().epochSeconds,
      iss:$iss,
      exp:0,
      %s
      name:$name,
      description:$description,
      grant_types:$grantTypes,
      response_types:$responseTypes,
      redirect_uris:$redirectUris,
      post_logout_redirect_uris:$postLogoutRedirectUris,
      token_endpoint_auth_method:$tokenEndpointAuthMethod,
      audiences:$audiences
    })

    RETURN c
  `, cypClientSecret)

  logCypher(cypher, params)

  if result, err = tx.Run(cypher, params); err != nil {
    return Client{}, err
  }

  if result.Next() {
    record        := result.Record()
    clientNode    := record.GetByIndex(0)

    if clientNode != nil {
      client = marshalNodeToClient(clientNode.(neo4j.Node))
    }
  } else {
    return Client{}, errors.New("Unable to create Client")
  }


  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Client{}, err
  }

  return client, nil
}

func UpdateClient(tx neo4j.Transaction, iUpdateClient Client) (rClient Client, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  if iUpdateClient.Id == "" {
    return Client{}, errors.New("Missing Client.Id")
  }
  params["id"] = iUpdateClient.Id

  params["exp"] = iUpdateClient.ExpiresAt

  if iUpdateClient.Name == "" {
    return Client{}, errors.New("Missing Client.Name")
  }
  params["name"] = iUpdateClient.Name

  if iUpdateClient.Description == "" {
    return Client{}, errors.New("Missing Client.Description")
  }
  params["description"] = iUpdateClient.Description

  params["grantTypes"] = []string{}
  params["responseTypes"] = []string{}
  params["redirectUris"] = []string{}
  params["postLogoutRedirectUris"] = []string{}
  params["audiences"] = []string{}
  params["tokenEndpointAuthMethod"] = ""

  if len(iUpdateClient.GrantTypes) > 0 {
    params["grantTypes"] = iUpdateClient.GrantTypes
  }
  if len(iUpdateClient.ResponseTypes) > 0 {
    params["responseTypes"] = iUpdateClient.ResponseTypes
  }
  if len(iUpdateClient.RedirectUris) > 0 {
    params["redirectUris"] = iUpdateClient.RedirectUris
  }
  if len(iUpdateClient.PostLogoutRedirectUris) > 0 {
    params["postLogoutRedirectUris"] = iUpdateClient.PostLogoutRedirectUris
  }
  if len(iUpdateClient.Audiences) > 0 {
    params["audiences"] = iUpdateClient.Audiences
  }
  if iUpdateClient.TokenEndpointAuthMethod != "" {
    params["tokenEndpointAuthMethod"] = iUpdateClient.TokenEndpointAuthMethod
  }

  //cypClientSecret := ""
  //if iUpdateClient.Secret != "" {
    //params["client_secret"] = iUpdateClient.Secret
    //cypClientSecret = `secret:$client_secret,`
  //}

  cypher = fmt.Sprintf(`
    MERGE (c:Client:Identity {id: $id})
       ON MATCH SET
          c.name = $name,
          c.description = $description,
          c.grant_types = $grantTypes,
          c.response_types = $responseTypes,
          c.redirect_uris = $redirectUris,
          c.post_logout_redirect_uris = $postLogoutRedirectUris,
          c.token_endpoint_auth_method = $tokenEndpointAuthMethod,
          c.audiences = $audiences

    RETURN c
  `)

  logCypher(cypher, params)

  if result, err = tx.Run(cypher, params); err != nil {
    return Client{}, err
  }

  if result.Next() {
    record        := result.Record()
    clientNode    := record.GetByIndex(0)

    if clientNode != nil {
      rClient = marshalNodeToClient(clientNode.(neo4j.Node))
    }
  } else {
    return Client{}, errors.New("Unable to update Client")
  }

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Client{}, err
  }

  return rClient, nil
}

func FetchClients(tx neo4j.Transaction, iClients []Client) (clients []Client, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  cypFilterClients := ""
  if len(iClients) > 0 {
    var ids []string
    for _, client := range iClients {
      ids = append(ids, client.Id)
    }
    cypFilterClients = ` AND c.id in split($ids, ",") `
    params["ids"] = strings.Join(ids, ",")
  }

  cypher = fmt.Sprintf(`
    MATCH (c:Client:Identity) WHERE 1=1 %s
    RETURN c
  `, cypFilterClients)

  logCypher(cypher, params)

  if result, err = tx.Run(cypher, params); err != nil {
    return nil, err
  }

  for result.Next() {
    record        := result.Record()
    clientNode    := record.GetByIndex(0)

    if clientNode != nil {
      client := marshalNodeToClient(clientNode.(neo4j.Node))
      clients = append(clients, client)
    }
  }

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return nil, err
  }

  return clients, nil
}

func DeleteClient(tx neo4j.Transaction, clientToDelete Client) (client Client, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  if clientToDelete.Id == "" {
    return Client{}, errors.New("Missing Client.Id")
  }
  params["id"] = clientToDelete.Id

  // Warning: Do not accidentally delete i!
  cypher = fmt.Sprintf(`
    MATCH (c:Client:Identity {id:$id})
    DETACH DELETE c
  `)

  logCypher(cypher, params)

  if result, err = tx.Run(cypher, params); err != nil {
    return Client{}, err
  }

  result.Next()

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Client{}, err
  }

  client.Id = clientToDelete.Id
  return client, nil
}
