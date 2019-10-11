package idp

import (
  "strings"
  "errors"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

type Client struct {
  Identity

  // ClientId     string // Renamed to Identity.Id
  ClientSecret string
  Name         string
  Description  string
}

func marshalNodeToClient(node neo4j.Node) (Client) {
  p := node.Props()

  return Client{
    Identity: marshalNodeToIdentity(node),

    // ClientId:     p["client_id"].(string),
    ClientSecret: p["client_secret"].(string),
    Name:         p["name"].(string),
    Description:  p["description"].(string),
  }
}

// CRUD

func CreateClient(driver neo4j.Driver, client Client) (Client, error) {
  var err error
  type NeoReturnType struct{
    Client Client
    CreatedBy Identity
  }

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Client{}, err
  }
  defer session.Close()

  neoResult, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      CREATE (i:Client:Identity {
        id:randomUUID(), iat:datetime().epochSeconds, iss:$iss, exp:$exp,

        client_secret:$client_secret,

        name:$name,

        description:$description,
      })
      RETURN i
    `
    params := map[string]interface{}{
      "iss": client.Issuer, "exp": client.ExpiresAt,
      "client_secret":client.ClientSecret,
      "name": client.Name,
      "description": client.Description,
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var client Client
    if result.Next() {
      record := result.Record()
      clientNode := record.GetByIndex(0)
      if clientNode != nil {
        client = marshalNodeToClient(clientNode.(neo4j.Node))
      }
    } else {
      return nil, errors.New("Unable to create Client")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return NeoReturnType{Client: client}, nil
  })

  if err != nil {
    return Client{}, err
  }
  return neoResult.(NeoReturnType).Client, nil
}

func FetchClients(driver neo4j.Driver, clients []Client) ([]Client, error) {
  ids := []string{}
  for _, client := range clients {
    ids = append(ids, client.Id)
  }
  return FetchClientsById(driver, ids)
}

func FetchClientsById(driver neo4j.Driver, ids []string) ([]Client, error) {
  var cypher string
  var params map[string]interface{}

  if ids == nil {
    cypher = `
      MATCH (i:Client:Identity {id: $id})
      RETURN i
    `
    params = map[string]interface{}{
      "id": strings.Join(ids, ","),
    }
  } else {
    cypher = `
      MATCH (i:Client:Identity) WHERE i.id in split($ids, ",")
      RETURN i
    `
    params = map[string]interface{}{
      "ids": strings.Join(ids, ","),
    }
  }
  return fetchClientsByQuery(driver, cypher, params)
}

func fetchClientsByQuery(driver neo4j.Driver, cypher string, params map[string]interface{}) ([]Client, error)  {
  var err error
  var session neo4j.Session
  var neoResult interface{}

  session, err = driver.Session(neo4j.AccessModeRead);
  if err != nil {
    return nil, err
  }
  defer session.Close()

  neoResult, err = session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var err error
    var out []Client
    for result.Next() {
      record := result.Record()

      objNode := record.GetByIndex(0)
      if objNode != nil {
        obj := marshalNodeToClient(objNode.(neo4j.Node))
        out = append(out, obj)
      }

    }
    if err = result.Err(); err != nil {
      return nil, err
    }
    return out, nil
  })

  if err != nil {
    return nil, err
  }
  if neoResult == nil {
    return nil, nil
  }
  return neoResult.([]Client), nil
}
