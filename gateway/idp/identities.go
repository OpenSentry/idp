package idp

import (
  "fmt"
  "strings"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

// You should never make these, please specialize with another label, see client.go or human.go
// func CreateIdentities(driver neo4j.Driver, identities []Identity) ([]Identity, error)

func FetchIdentities(tx neo4j.Transaction, iIdentities []Identity) (identities []Identity, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  cypfilterIdentites := ""
  if len(iIdentities) > 0 {
    var ids []string
    for _, identity := range iIdentities {
      ids = append(ids, identity.Id)
    }
    cypfilterIdentites = ` WHERE i.id in split($ids, ",") `
    params["ids"] = strings.Join(ids, ",")
  }

  cypher = fmt.Sprintf(`
    MATCH (i:Identity) %s RETURN i
  `, cypfilterIdentites)

  if result, err = tx.Run(cypher, params); err != nil {
    return nil, err
  }

  for result.Next() {
    record          := result.Record()
    identityNode    := record.GetByIndex(0)

    if identityNode != nil {
      i := marshalNodeToIdentity(identityNode.(neo4j.Node))

      identities = append(identities, i)
    }
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return nil, err
  }

  return identities, nil
}
