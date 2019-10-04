package idp

import (
  "errors"
  "strings"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

type Identity struct {
  Id        string
  Labels    string

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

  var otpDeleteCode string
  var OtpDeleteCodeExpire int64

  if p["otp_delete_code"] != nil { otpDeleteCode = p["otp_delete_code"].(string) }
  if p["otp_delete_code_expire"] != nil { OtpDeleteCodeExpire = p["otp_delete_code_expire"].(int64) }

  return Identity{
    Id:        p["id"].(string),
    Labels:    strings.Join(node.Labels(), ":"),
    Issuer:    p["iss"].(string),
    ExpiresAt: p["exp"].(int64),
    IssuedAt:  p["iat"].(int64),
    OtpDeleteCode:        otpDeleteCode,
    OtpDeleteCodeExpire:  OtpDeleteCodeExpire,
  }
}

func fetchByIdentityId(id string, tx neo4j.Transaction) (Identity, error) {
  var err error
  var result neo4j.Result
  var identity Identity

  cypher := `MATCH (i:Identity {id:$id}) return i`
  params := map[string]interface{}{
    "id": id,
  }

  if result, err = tx.Run(cypher, params); err != nil {
    return Identity{}, err
  }

  if result.Next() {
    record := result.Record()

    identityNode := record.GetByIndex(0)

    if identityNode != nil {
      identity = marshalNodeToIdentity(identityNode.(neo4j.Node))
    } else {
      return Identity{}, errors.New("Identity not found")
    }
  }

  if err = result.Err(); err != nil {
    return Identity{}, err
  }

  return identity, nil
}

// CRUD

// You should never make these, please specialize with another label, see client.go or human.go
// func CreateIdentities(driver neo4j.Driver, identities []Identity) ([]Identity, error)

func FetchIdentities(driver neo4j.Driver, identities []Identity) ([]Identity, error) {
  ids := []string{}
  for _, identity := range identities {
    ids = append(ids, identity.Id)
  }
  return FetchIdentitiesById(driver, ids)
}

func FetchIdentitiesAll(driver neo4j.Driver) ([]Identity, error) {  
  return FetchIdentitiesById(driver, nil)
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
      MATCH (i:Identity) WHERE i.id in split($ids, ",")
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

  neoResult, err = session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
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
  if neoResult == nil {
    return nil, nil
  }
  return neoResult.([]Identity), nil
}

