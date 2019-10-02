package idp

import (
  "strings"
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

func FetchFollows(driver neo4j.Driver, follows []Follow) ([]Follow, error) {
  ids := []string{}
  for _, follow := range follows {
    ids = append(ids, follow.From.Id)
  }
  return FetchFollowsForFrom(driver, ids)
}

func FetchFollowsForFrom(driver neo4j.Driver, froms []string) ([]Follow, error) {
  var cypher string
  var params map[string]interface{}

  if froms == nil {
    cypher = `
      MATCH (from:Identity)<-[:FOLLOWED_BY]-(to:Identity)
      RETURN from, to
    `
    params = map[string]interface{}{}
  } else {
    cypher = `
      MATCH (from:Identity)<-[:FOLLOWED_BY]-(to:Identity) WHERE from.Id in split($froms, ",")
      RETURN from, to
    `
    params = map[string]interface{}{
      "froms": strings.Join(froms, ","),
    }
  }
  return fetchFollowsByQuery(driver, cypher, params)
}

func fetchFollowsByQuery(driver neo4j.Driver, cypher string, params map[string]interface{}) ([]Follow, error)  {
  var err error
  var session neo4j.Session
  var neoResult interface{}

  session, err = driver.Session(neo4j.AccessModeRead);
  if err != nil {
    return nil, err
  }
  defer session.Close()

  neoResult, err = session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {

    var err error
    var result neo4j.Result

    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var follows []Follow
    for result.Next() {
      record := result.Record()

      fromNode := record.GetByIndex(0)
      if fromNode != nil {
        from := marshalNodeToIdentity(fromNode.(neo4j.Node))

        toNode := record.GetByIndex(1)
        if toNode != nil {
          to := marshalNodeToIdentity(toNode.(neo4j.Node))

          follows = append(follows, Follow{
            From: from,
            To: to,
          })
        }

      }

    }
    if err = result.Err(); err != nil {
      return nil, err
    }
    return follows, nil
  })

  if err != nil {
    return nil, err
  }
  if neoResult == nil {
    return nil, nil
  }
  return neoResult.([]Follow), nil
}