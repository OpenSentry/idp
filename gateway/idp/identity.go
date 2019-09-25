package idp

import (
  //"errors"
  "strings"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

type Identity struct {
  Id        string

  // JWT
  // Subject string // Renamed it to Identity.Id
  Issuer    string
  ExpiresAt int64
  IssuedAt  int64

  OtpDeleteCode        string
  OtpDeleteCodeExpire  int64

  CreatedBy *Identity
}
func marshalNodeToIdentity(node neo4j.Node) (Identity) {
  p := node.Props()

  return Identity{
    Id:        p["id"].(string),
    Issuer:    p["iss"].(string),
    ExpiresAt: p["exp"].(int64),
    IssuedAt:  p["iat"].(int64),
    OtpDeleteCode:        p["otp_delete_code"].(string),
    OtpDeleteCodeExpire:  p["otp_delete_code_expire"].(int64),
  }
}

type Follow struct {
  From Identity
  To Identity
}


// CRUD

// You should never make these, please specialize with another label, see client.go or human.go
// func CreateIdentity() (Identity, error)

func FetchIdentities(driver neo4j.Driver, identities []Identity) ([]Identity, error) {
  ids := []string{}
  for _, identity := range identities {
    ids = append(ids, identity.Id)
  }
  return FetchIdentitiesById(driver, ids)
}

func FetchIdentitiesById(driver neo4j.Driver, ids []string) ([]Identity, error) {
  var cypher string
  var params map[string]interface{}

  if ids == nil {
    cypher = `
      MATCH (i:Identity {id: $id})
      RETURN i
    `
    params = map[string]interface{}{
      "id": strings.Join(ids, ","),
    }
  } else {
    cypher = `
      MATCH (i:Identity) WHERE i.Id in split($ids, ",")
      RETURN i
    `
    params = map[string]interface{}{
      "ids": strings.Join(ids, ","),
    }
  }
  return fetchIdentitiesByQuery(driver, cypher, params)
}

func fetchIdentitiesByQuery(driver neo4j.Driver, cypher string, params map[string]interface{}) ([]Identity, error)  {
  var err error
  var session neo4j.Session
  var neoResult interface{}

  session, err = driver.Session(neo4j.AccessModeRead);
  if err != nil {
    return nil, err
  }
  defer session.Close()

  neoResult, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var err error
    var out []Identity
    for result.Next() {
      record := result.Record()

      objNode := record.GetByIndex(0)
      if objNode != nil {
        obj := marshalNodeToIdentity(objNode.(neo4j.Node))
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
  if neoResult != nil {
    return nil, nil
  }
  return neoResult.([]Identity), nil
}

// Actions

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
