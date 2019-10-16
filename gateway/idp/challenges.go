package idp

import (
  "errors"
  "strings"
  "fmt"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

func CreateChallengeForTOTP(tx neo4j.Transaction, newChallenge Challenge) (challenge Challenge, err error) {
  newChallenge.Code = "" // Do not set this on TOTP requests
  challenge, err = createChallenge(tx, newChallenge)
  if err != nil {
    return Challenge{}, err
  }
  return challenge, nil
}

func CreateChallengeForOTP(tx neo4j.Transaction, newChallenge Challenge) (challenge Challenge, otpCode ChallengeCode, err error) {
  otpCode, err = CreateChallengeCode()
  if err != nil {
    return Challenge{}, ChallengeCode{}, err
  }

  hashedCode, err := CreatePassword(otpCode.Code)
  if err != nil {
    return Challenge{}, ChallengeCode{}, err
  }
  newChallenge.Code = hashedCode

  challenge, err = createChallenge(tx, newChallenge)
  if err != nil {
    return Challenge{}, ChallengeCode{}, err
  }
  return challenge, otpCode, nil
}

func createChallenge(tx neo4j.Transaction, newChallenge Challenge) (challenge Challenge, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  if newChallenge.Subject == "" {
    return Challenge{}, errors.New("Missing Challenge.Subject")
  }
  params["sub"] = newChallenge.Subject

  if newChallenge.Issuer == "" {
    return Challenge{}, errors.New("Missing Challenge.Issuer")
  }
  params["iss"] = newChallenge.Issuer

  params["exp"] = newChallenge.ExpiresAt
  params["aud"] = newChallenge.Audience

  if newChallenge.RedirectTo == "" {
    return Challenge{}, errors.New("Missing Challenge.RedirectTo")
  }
  params["redirect_to"] = newChallenge.RedirectTo

  params["code_type"] = newChallenge.CodeType
  params["code"] = newChallenge.Code

  cypher = fmt.Sprintf(`
    MATCH (i:Identity {id:$sub})
    MERGE (c:Challenge {
      id:randomUUID(), iat:datetime().epochSeconds, iss:$iss, exp:$exp, aud:$aud, sub:$sub,
      redirect_to:$redirect_to,
      code_type:$code_type, code:$code,
      verified_at:0
    })-[:CHALLENGES]->(i)

    WITH c

    OPTIONAL MATCH (d:Challenge) WHERE id(c) <> id(d) AND d.exp < datetime().epochSeconds DETACH DELETE d

    RETURN c
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return Challenge{}, err
  }

  if result.Next() {
    record          := result.Record()
    challengeNode   := record.GetByIndex(0)

    if challengeNode != nil {
      challenge = marshalNodeToChallenge(challengeNode.(neo4j.Node))
    }
  } else {
    return Challenge{}, errors.New("Unable to create Challenge")
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Challenge{}, err
  }

  return challenge, nil
}

func FetchChallenges(tx neo4j.Transaction, iChallenges []Challenge) (challenges []Challenge, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  cypfilterChallenges := ""
  if len(iChallenges) > 0 {
    var ids []string
    for _, challenge := range iChallenges {
      ids = append(ids, challenge.Id)
    }
    cypfilterChallenges = ` AND c.id in split($ids, ",") `
    params["ids"] = strings.Join(ids, ",")
  }

  cypher = fmt.Sprintf(`
    MATCH (c:Challenge) WHERE c.exp > datetime().epochSeconds %s
    RETURN c
  `, cypfilterChallenges)

  if result, err = tx.Run(cypher, params); err != nil {
    return nil, err
  }

  for result.Next() {
    record          := result.Record()
    challengeNode    := record.GetByIndex(0)

    if challengeNode != nil {
      i := marshalNodeToChallenge(challengeNode.(neo4j.Node))

      challenges = append(challenges, i)
    }
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return nil, err
  }

  return challenges, nil
}

func VerifyChallenge(tx neo4j.Transaction, challengeToUpdate Challenge) (updatedChallenge Challenge, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  if challengeToUpdate.Id == "" {
    return Challenge{}, errors.New("Missing Challenge.Id")
  }
  params["id"] = challengeToUpdate.Id

  cypher = fmt.Sprintf(`
    MATCH (c:Challenge {id:$id}) WHERE c.exp > datetime().epochSeconds
    SET c.verified_at = datetime().epochSeconds
    RETURN c
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return Challenge{}, err
  }

  if result.Next() {
    record          := result.Record()
    challengeNode   := record.GetByIndex(0)

    if challengeNode != nil {
      updatedChallenge = marshalNodeToChallenge(challengeNode.(neo4j.Node))
    }
  } else {
    return Challenge{}, errors.New("Unable to set Challenge verified. Hint: Challenge might be expired or non existant.")
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Challenge{}, err
  }

  return updatedChallenge, nil
}
