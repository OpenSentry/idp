package idp

import (
  "errors"
  "strings"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

type Challenge struct {
  Id string

  JwtRegisteredClaims

  RedirectTo   string
  CodeType     string

  Code         string

  VerifiedAt   int64

  RequestedBy *Human
}

func marshalNodeToChallenge(node neo4j.Node) (Challenge) {
  p := node.Props()

  var verifiedAt int64
  if (p["verified_at"] != nil) { verifiedAt = p["verified_at"].(int64) }

  return Challenge{
    Id:         p["id"].(string),

    JwtRegisteredClaims: marshalNodeToJwtRegisteredClaims(node),

    RedirectTo: p["redirect_to"].(string),

    CodeType:   p["code_type"].(string),
    Code:       p["code"].(string),

    VerifiedAt:   verifiedAt,
  }
}

// CRUD

func CreateChallenge(driver neo4j.Driver, challenge Challenge, requestedBy Human) (Challenge, Human, error) {
  var err error
  type NeoReturnType struct{
    Challenge Challenge
    RequestedBy Human
  }

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Challenge{}, Human{}, err
  }
  defer session.Close()

  neoResult, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (rbi:Human:Identity {id:$rbi})
      MERGE (c:Challenge {
        id:randomUUID(), iat:datetime().epochSeconds, iss:$iss, exp:$exp, aud:$aud, sub:$sub,
        redirect_to:$redirect_to,
        code_type:$code_type, code:$code,
        verified_at:0
      })-[:REQUESTED_BY]->(rbi)

      WITH c, rbi

      OPTIONAL MATCH (d:Challenge)-[:REQUESTED_BY]->(i:Identity) WHERE id(c) <> id(d) AND d.exp < datetime().epochSeconds DETACH DELETE d

      RETURN c, rbi
    `
    params := map[string]interface{}{
      "rbi": requestedBy.Id,
      "sub": requestedBy.Id,
      "iss": challenge.Issuer,
      "exp": challenge.ExpiresAt,
      "aud": challenge.Audience,
      "redirect_to": challenge.RedirectTo,
      "code_type": challenge.CodeType,
      "code": challenge.Code,
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var challenge Challenge
    var requestedBy Human
    if result.Next() {
      record := result.Record()

      challengeNode := record.GetByIndex(0)
      if challengeNode != nil {
        challenge = marshalNodeToChallenge(challengeNode.(neo4j.Node))

        rbiNode := record.GetByIndex(1)
        if rbiNode != nil {
          requestedBy = marshalNodeToHuman(rbiNode.(neo4j.Node))
          challenge.RequestedBy = &requestedBy
        }

      }
    } else {
      return nil, errors.New("Unable to create Challenge")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return NeoReturnType{Challenge: challenge, RequestedBy: requestedBy}, nil
  })

  if err != nil {
    return Challenge{}, Human{}, err
  }
  r := neoResult.(NeoReturnType)
  return r.Challenge, r.RequestedBy, nil
}

func FetchChallenges(driver neo4j.Driver, challenges []Challenge) ([]Challenge, error) {
  ids := []string{}
  for _, challenge := range challenges {
    ids = append(ids, challenge.Id)
  }
  return FetchChallengesById(driver, ids)
}

func FetchChallengesById(driver neo4j.Driver, ids []string) ([]Challenge, error) {
  var cypher string
  var params map[string]interface{}

  if ids == nil {
    cypher = `
      MATCH (c:Challenge)-[:REQUESTED_BY]->(i:Human:Identity) WHERE c.exp > datetime().epochSeconds
      RETURN c, i
    `
  } else {
    // c.exp > datetime().epochSeconds AND
    cypher = `
      MATCH (c:Challenge)-[:REQUESTED_BY]->(i:Human:Identity) WHERE c.id in split($ids, ",")
      RETURN c, i
    `
    params = map[string]interface{}{
      "ids": strings.Join(ids, ","),
    }
  }
  return fetchChallengesByQuery(driver, cypher, params)
}

func fetchChallengesByQuery(driver neo4j.Driver, cypher string, params map[string]interface{}) ([]Challenge, error)  {
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
    var out []Challenge
    for result.Next() {
      record := result.Record()

      challengeNode := record.GetByIndex(0)
      if challengeNode != nil {
        challenge := marshalNodeToChallenge(challengeNode.(neo4j.Node))

        humanNode := record.GetByIndex(1)
        if humanNode != nil {
          human := marshalNodeToHuman(humanNode.(neo4j.Node))
          challenge.RequestedBy = &human
        }

        out = append(out, challenge)
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
  return neoResult.([]Challenge), nil
}


// Actions

func VerifyChallenge(driver neo4j.Driver, challenge Challenge) (Challenge, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Challenge{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (c:Challenge {id:$id})-[:REQUESTED_BY]->(i:Identity) WHERE c.exp > datetime().epochSeconds
      SET c.verified_at = datetime().epochSeconds
      RETURN c, i
    `
    params := map[string]interface{}{"id": challenge.Id}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var verifiedChallenge Challenge
    if result.Next() {
      record := result.Record()

      challengeNode := record.GetByIndex(0)
      if challengeNode != nil {
        verifiedChallenge = marshalNodeToChallenge(challengeNode.(neo4j.Node))

        humanNode := record.GetByIndex(1)
        if humanNode != nil {
          human := marshalNodeToHuman(humanNode.(neo4j.Node))
          verifiedChallenge.RequestedBy = &human
        }

      }
    } else {
      return nil, errors.New("Challenge not found. Hint: It might be expired.")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return verifiedChallenge, nil
  })

  if err != nil {
    return Challenge{}, err
  }
  return obj.(Challenge), nil
}