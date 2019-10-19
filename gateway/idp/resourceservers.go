package idp

import (
  "errors"
  "strings"
  "fmt"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

func CreateResourceServer(tx neo4j.Transaction, managedBy *Identity, newResourceServer ResourceServer) (resourceServer ResourceServer, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  if newResourceServer.Name == "" {
    return ResourceServer{}, errors.New("Missing ResourceServer.Name")
  }
  params["name"] = newResourceServer.Name

  if newResourceServer.Audience == "" {
    return ResourceServer{}, errors.New("Missing ResourceServer.Audience")
  }
  params["aud"] = newResourceServer.Audience

  if newResourceServer.Description == "" {
    return ResourceServer{}, errors.New("Missing ResourceServer.Description")
  }
  params["description"] = newResourceServer.Description

  cypher = fmt.Sprintf(`
    CREATE (i:ResourceServer:Identity {
      id:randomUUID(),
      iat:datetime().epochSeconds,
      name:$name,
      description:$description,
      aud:$aud
    })
    RETURN i
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return ResourceServer{}, err
  }

  if result.Next() {
    record          := result.Record()
    resourceServerNode   := record.GetByIndex(0)

    if resourceServerNode != nil {
      resourceServer = marshalNodeToResourceServer(resourceServerNode.(neo4j.Node))
    }
  } else {
    return ResourceServer{}, errors.New("Unable to create ResourceServer")
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return ResourceServer{}, err
  }

  return resourceServer, nil
}

func FetchResourceServers(tx neo4j.Transaction, managedBy *Identity, iResourceServers []ResourceServer) (resourceServers []ResourceServer, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  cypfilterResourceServers := ""
  if len(iResourceServers) > 0 {
    var ids []string
    for _, rs := range iResourceServers {
      ids = append(ids, rs.Id)
    }
    cypfilterResourceServers = ` WHERE i.id in split($ids, ",") `
    params["ids"] = strings.Join(ids, ",")
  }

  cypher = fmt.Sprintf(`
    MATCH (i:ResourceServer:Identity) %s
    RETURN i
  `, cypfilterResourceServers)

  if result, err = tx.Run(cypher, params); err != nil {
    return nil, err
  }

  for result.Next() {
    record             := result.Record()
    resourceServerNode := record.GetByIndex(0)

    if resourceServerNode != nil {
      i := marshalNodeToResourceServer(resourceServerNode.(neo4j.Node))
      resourceServers = append(resourceServers, i)
    }
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return nil, err
  }

  return resourceServers, nil
}