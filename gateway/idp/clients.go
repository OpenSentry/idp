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

  if newClient.ClientSecret == "" {
    return Client{}, errors.New("Missing Client.ClientSecret")
  }
  params["client_secret"] = newClient.ClientSecret

  if newClient.Name == "" {
    return Client{}, errors.New("Missing Client.Name")
  }
  params["name"] = newClient.Name

  if newClient.Description == "" {
    return Client{}, errors.New("Missing Client.Description")
  }
  params["description"] = newClient.Description

  cypher = fmt.Sprintf(`
    CREATE (i:Client:Identity {
      id:randomUUID(),
      iat:datetime().epochSeconds,
      iss:$iss,
      exp:$exp,
      client_secret:$client_secret,
      name:$name,
      description:$description,
    })
    RETURN i
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return Client{}, err
  }

  if result.Next() {
    record          := result.Record()
    clientNode      := record.GetByIndex(0)

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

func FetchClients(tx neo4j.Transaction, iClients []Client) (clients []Client, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  cypfilterClients := ""
  if len(iClients) > 0 {
    var ids []string
    for _, client := range iClients {
      ids = append(ids, client.Id)
    }
    cypfilterClients = ` AND c.id in split($ids, ",") `
    params["ids"] = strings.Join(ids, ",")
  }

  cypher = fmt.Sprintf(`
    MATCH (c:Client:Identity) WHERE c.exp > datetime().epochSeconds %s
    RETURN c
  `, cypfilterClients)

  if result, err = tx.Run(cypher, params); err != nil {
    return nil, err
  }

  for result.Next() {
    record          := result.Record()
    clientNode      := record.GetByIndex(0)

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