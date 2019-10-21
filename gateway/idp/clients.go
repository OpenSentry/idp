package idp

import (
  "errors"
  "strings"
  "fmt"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

func CreateClient(tx neo4j.Transaction, managedBy *Identity, newClient Client) (client Client, err error) {
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

  cypClientSecret := ""
  if newClient.ClientSecret != "" {
    params["client_secret"] = newClient.ClientSecret
    cypClientSecret = `client_secret:$client_secret,`
  }

  cypManages := ""
  if managedBy != nil {
    params["managed_by"] = managedBy.Id
    cypManages = `MATCH (i:Identity {id:$managed_by}) MERGE (i)-[:MANAGES]->(c)`
  }

  cypher = fmt.Sprintf(`
    CREATE (c:Client:Identity {
      id:randomUUID(),
      iat:datetime().epochSeconds,
      iss:$iss,
      exp:0,
      %s
      name:$name,
      description:$description
    })

    WITH c

    %s

    RETURN c
  `, cypClientSecret, cypManages)

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

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Client{}, err
  }

  return client, nil
}

func FetchClients(tx neo4j.Transaction, managedBy *Identity, iClients []Client) (clients []Client, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  var cypManages string
  if managedBy != nil {
    cypManages = `(i:Identity {id:$managed_by})-[:MANAGES]->`
    params["managed_by"] = managedBy.Id
  }

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
    MATCH %s(c:Client:Identity) WHERE 1=1 %s
    RETURN c
  `, cypManages, cypFilterClients)

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

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return nil, err
  }

  return clients, nil
}

func DeleteClient(tx neo4j.Transaction, managedBy *Identity, clientToDelete Client) (client Client, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  if clientToDelete.Id == "" {
    return Client{}, errors.New("Missing Client.Id")
  }

  var cypManages string
  if managedBy != nil {
    cypManages = `(i:Identity {id:$managed_by})-[:MANAGES]->`
    params["managed_by"] = managedBy.Id
  }

  params["id"] = clientToDelete.Id

  cypher = fmt.Sprintf(`
    MATCH %s(c:Client:Identity) WHERE 1=1 %s    
    DETACH DELETE i
  `, cypManages)

  if result, err = tx.Run(cypher, params); err != nil {
    return Client{}, err
  }

  result.Next()

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Client{}, err
  }

  client.Id = clientToDelete.Id
  return client, nil
}