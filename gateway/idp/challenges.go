package idp

import (
  "errors"
  "strings"
  "fmt"
  "context"
  "database/sql"
	"github.com/google/uuid"
)

func CreateChallengeUsingTotp(ctx context.Context, tx *sql.Tx, challengeType ChallengeType, newChallenge Challenge) (challenge Challenge, err error) {
  newChallenge.Code = "" // Do not set this on TOTP requests
  challenge, err = createChallenge(ctx, tx, newChallenge, ChallengeAuthenticate)
  if err != nil {
    return Challenge{}, err
  }
  return challenge, nil
}

func CreateChallengeUsingOtp(ctx context.Context, tx *sql.Tx, challengeType ChallengeType, newChallenge Challenge) (challenge Challenge, otpCode ChallengeCode, err error) {
  otpCode, err = CreateChallengeCode()
  if err != nil {
    return Challenge{}, ChallengeCode{}, err
  }

  hashedCode, err := CreatePassword(otpCode.Code)
  if err != nil {
    return Challenge{}, ChallengeCode{}, err
  }
  newChallenge.Code = hashedCode

  challenge, err = createChallenge(ctx, tx, newChallenge, challengeType)
  if err != nil {
    return Challenge{}, ChallengeCode{}, err
  }
  return challenge, otpCode, nil
}

func createChallenge(ctx context.Context, tx *sql.Tx, newChallenge Challenge, challengeType ChallengeType) (challenge Challenge, err error) {
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

  cypData := ""
  if newChallenge.Data != "" {
    cypData = ", data:$data "
    params["data"] = newChallenge.Data
  }

	uuid, err := uuid.NewRandom()
	if err != nil {
		return Challenge{}, err
	}

  cypChallengeType := ""
  switch (challengeType) {
  case ChallengeAuthenticate:
    cypChallengeType = ":Authenticate"
  case ChallengeRecover:
    cypChallengeType = ":Recover"
  case ChallengeDelete:
    cypChallengeType = ":Delete"
  case ChallengeEmailConfirm:
    cypChallengeType = ":EmailConfirm"
  case ChallengeEmailChange:
    cypChallengeType = ":EmailChange"
  default:
    return Challenge{}, errors.New("Unsupported challenge type")
  }

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (i:Identity {id:$sub})
    MERGE (c:Challenge%s {
      id:randomUUID(), iat:datetime().epochSeconds, iss:$iss, exp:$exp, aud:$aud, sub:$sub,
      redirect_to:$redirect_to,
      code_type:$code_type, code:$code,
      verified_at:0
      %s
    })-[:CHALLENGES]->(i)

    WITH c

    OPTIONAL MATCH (d:Challenge) WHERE id(c) <> id(d) AND d.exp < datetime().epochSeconds DETACH DELETE d

    RETURN c
  `, cypChallengeType, cypData)

	_, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
    return Challenge{}, err
  }

	challenges, err := FetchChallenges(ctx, tx, []Challenge{{ Id: uuid.String() }} )
  if err != nil {
    return Challenge{}, err
  }

  return challenges[0], nil
}

func FetchChallenges(ctx context.Context, tx *sql.Tx, iChallenges []Challenge) (challenges []Challenge, err error) {
  var rows *sql.Rows
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

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (c:Challenge) WHERE c.exp > datetime().epochSeconds %s
    RETURN c
  `, cypfilterChallenges)

  rows, err = tx.QueryContext(ctx, cypher, params);
  if err != nil {
    return nil, err
  }

  for rows.Next() {
		i := marshalRowToChallenge(rows)
		challenges = append(challenges, i)
  }

  return challenges, nil
}

func VerifyChallenge(ctx context.Context, tx *sql.Tx, challengeToUpdate Challenge) (updatedChallenge Challenge, err error) {
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

  _, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
    return Challenge{}, err
  }

	challenges, err := FetchChallenges(ctx, tx, []Challenge{ challengeToUpdate } )
  if err != nil {
    return Challenge{}, err
  }

  return challenges[0], nil
}
