package idp

import (
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

type Follow struct {
  From Identity
  To Identity
}

func CreateFollow(driver neo4j.Driver, from Identity, to Identity) (Follow, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Follow{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (from:Identity {id:$from})
      MATCH (to:Identity {id:$to})
      MERGE (from)-[:FOLLOW]->(to)
      RETURN from, to
    `
    params := map[string]interface{}{
      "from": from.Id,
      "to": to.Id,
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var out Follow
    if result.Next() {
      record := result.Record()

      fromNode := record.GetByIndex(0)
      if fromNode != nil {
        from := marshalNodeToIdentity(fromNode.(neo4j.Node))
        out.From = from

        toNode := record.GetByIndex(1)
        if toNode != nil {
          to := marshalNodeToIdentity(toNode.(neo4j.Node))
          out.To = to
        }
      }
    }
    if err = result.Err(); err != nil {
      return nil, err
    }
    return out, nil
  })

  if err != nil {
    return Follow{}, err
  }
  return obj.(Follow), nil
}