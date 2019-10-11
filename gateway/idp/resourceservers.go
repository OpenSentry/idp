package idp

import (
  "strings"
  "errors"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

type ResourceServer struct {
  Identity
  Name         string
  Description  string
  Audience     string
}

func marshalNodeToResourceServer(node neo4j.Node) (ResourceServer) {
  p := node.Props()

  return ResourceServer{
    Identity: marshalNodeToIdentity(node),
    Name:         p["name"].(string),
    Description:  p["description"].(string),
    Audience:     p["aud"].(string),
  }
}

// CRUD

func CreateResourceServer(driver neo4j.Driver, rs ResourceServer, requestedBy Identity) (ResourceServer, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return ResourceServer{}, err
  }
  defer session.Close()

  neoResult, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      CREATE (i:ResourceServer:Identity {
        id:randomUUID(),
        iat:datetime().epochSeconds,
        name:$name,
        description:$description,
        aud:$aud
      })
      RETURN i
    `
    params := map[string]interface{}{
      "name": rs.Name,
      "description": rs.Description,
      "aud": rs.Audience,
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var dbRs ResourceServer
    if result.Next() {
      record := result.Record()
      rsNode := record.GetByIndex(0)
      if rsNode != nil {
        dbRs = marshalNodeToResourceServer(rsNode.(neo4j.Node))
      }
    } else {
      return nil, errors.New("Unable to create ResourceServer")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return dbRs, nil
  })

  if err != nil {
    return ResourceServer{}, err
  }
  return neoResult.(ResourceServer), nil
}

func FetchResourceServers(driver neo4j.Driver, resourceServers []ResourceServer) ([]ResourceServer, error) {
  var ids []string
  for _, resourceServer := range resourceServers {
    ids = append(ids, resourceServer.Id)
  }
  return FetchResourceServersByIds(driver, ids)
}

func FetchResourceServersByIds(driver neo4j.Driver, ids []string) ([]ResourceServer, error) {
  var cypher string
  var params map[string]interface{}

  if ids == nil {
    cypher = `
      MATCH (i:ResourceServer:Identity)
      RETURN i
    `
  } else {
    cypher = `
      MATCH (i:ResourceServer:Identity) WHERE i.id in split($ids, ",")
      RETURN i
    `
    params = map[string]interface{}{
      "ids": strings.Join(ids, ","),
    }
  }
  logCypher(cypher,params)
  return fetchResourceServersByQuery(driver, cypher, params)
}

func fetchResourceServersByQuery(driver neo4j.Driver, cypher string, params map[string]interface{}) ([]ResourceServer, error)  {
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
    var out []ResourceServer
    for result.Next() {
      record := result.Record()

      objNode := record.GetByIndex(0)
      if objNode != nil {
        obj := marshalNodeToResourceServer(objNode.(neo4j.Node))
        out = append(out, obj)
      }

    }
    if err = result.Err(); err != nil {
      return nil, err
    }
    return out, nil
  })

  logCypher(cypher,params)

  if err != nil {
    return nil, err
  }
  if neoResult == nil {
    return nil, nil
  }
  return neoResult.([]ResourceServer), nil
}
