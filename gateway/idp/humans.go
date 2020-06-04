package idp

import (
  "errors"
  "context"
  "strings"
  "fmt"
	"database/sql"
	"github.com/google/uuid"
)

func CreateHumanFromInvite(ctx context.Context, tx *sql.Tx, newHuman Human) (human Human, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if newHuman.Id == "" {
    return Human{}, errors.New("Missing Human.Id. Hint this should be the Invite.Id")
  }

  if newHuman.Username == "" {
    return Human{}, errors.New("Missing Human.Username")
  }

  if newHuman.Name == "" {
    return Human{}, errors.New("Missing Human.Name")
  }

  if newHuman.Password == "" {
    return Human{}, errors.New("Missing Human.Password")
  }

  if newHuman.EmailConfirmedAt == 0 {
    return Human{}, errors.New("Missing Human.EmailConfirmedAt. Hint must be larger than 0")
  }

  params["id"] = newHuman.Id
  params["username"] = newHuman.Username
  params["name"] = newHuman.Name
  params["allow_login"] = newHuman.AllowLogin
  params["password"] = newHuman.Password
  params["email_confirmed_at"] = newHuman.EmailConfirmedAt

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (i:Invite:Identity {id:$id})
      SET i.email_confirmed_at=$email_confirmed_at,
          i.username=$username,
          i.name=$name,
          i.allow_login=$allow_login,
          i.password=$password,
          i.totp_required=false,
          i.totp_secret="",
          i.exp=0,
          i:Human

    WITH i

    REMOVE i:Invite

    RETURN i
  `)

  _, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
    return Human{}, err
  }

	humans, err := FetchHumans(ctx, tx, []Human{ newHuman } )
  if err != nil {
    return Human{}, err
  }

  return humans[0], nil
}

func CreateHuman(ctx context.Context, tx *sql.Tx, newHuman Human) (human Human, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if newHuman.Issuer == "" {
    return Human{}, errors.New("Missing Human.Issuer")
  }

  if newHuman.Email == "" {
    return Human{}, errors.New("Missing Human.Email")
  }

  if newHuman.Username == "" {
    return Human{}, errors.New("Missing Human.Username")
  }

  if newHuman.Name == "" {
    return Human{}, errors.New("Missing Human.Name")
  }

  if newHuman.Password == "" {
    return Human{}, errors.New("Missing Human.Password")
  }

	uuid, err := uuid.NewRandom()
	if err != nil {
		return Human{}, err
	}

  params["iss"] = newHuman.Issuer
  params["exp"] = newHuman.ExpiresAt
  params["email"] = newHuman.Email
  params["username"] = newHuman.Username
  params["name"] = newHuman.Name
  params["allow_login"] = newHuman.AllowLogin
  params["password"] = newHuman.Password

	// TODO SQL
  cypher = fmt.Sprintf(`
    CREATE (i:Human:Identity {
      id: randomUUID(),
      iat: datetime().epochSeconds,
      iss: $iss,
      exp: $exp,

      email: $email,
      email_confirmed_at: 0,

      username: $username,

      name: $name,

      allow_login: $allow_login,

      password: $password,

      totp_required: false,
      totp_secret: ""
    })
    RETURN i
  `)

	_, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
    return Human{}, err
  }

	humans, err := FetchHumans(ctx, tx, []Human{{ Identity: Identity{Id:uuid.String()} }} )
  if err != nil {
    return Human{}, err
  }

  return humans[0], nil
}

func FetchHumans(ctx context.Context, tx *sql.Tx, iHumans []Human) (humans []Human, err error) {
  var cypher string
  var params = make(map[string]interface{})

  cypfilterIds := ""
  if len(iHumans) > 0 {
    var ids []string
    for _, human := range iHumans {
      ids = append(ids, human.Id)
    }
    cypfilterIds = ` WHERE h.id in split($ids, ",") `
    params["ids"] = strings.Join(ids, ",")
  }

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (h:Human:Identity) %s
    RETURN h
  `, cypfilterIds)

  humans, err = fetchHumansByQuery(ctx, tx, cypher, params)
  return humans, err
}

func FetchHumansByEmail(ctx context.Context, tx *sql.Tx, iHumans []Human) (humans []Human, err error) {
  var cypher string
  var params = make(map[string]interface{})

  cypfilterEmails := ""
  if len(iHumans) > 0 {
    var emails []string
    for _, human := range iHumans {
      emails = append(emails, human.Email)
    }
    cypfilterEmails = ` WHERE h.email in split($emails, ",") `
    params["emails"] = strings.Join(emails, ",")
  }

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (h:Human:Identity) %s
    RETURN h
  `, cypfilterEmails)

  humans, err = fetchHumansByQuery(ctx, tx, cypher, params)
  return humans, err
}

func FetchHumansByUsername(ctx context.Context, tx *sql.Tx, iHumans []Human) (humans []Human, err error) {
  var cypher string
  var params = make(map[string]interface{})

  cypfilterUsernames := ""
  if len(iHumans) > 0 {
    var usernames []string
    for _, human := range iHumans {
      usernames = append(usernames, human.Username)
    }
    cypfilterUsernames = ` WHERE h.username in split($usernames, ",") `
    params["usernames"] = strings.Join(usernames, ",")
  }

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (h:Human:Identity) %s
    RETURN h
  `, cypfilterUsernames)

  humans, err = fetchHumansByQuery(ctx, tx, cypher, params)
  return humans, err
}

func fetchHumansByQuery(ctx context.Context, tx *sql.Tx, cypher string, params map[string]interface{}) (humans []Human, err error) {
  var rows *sql.Rows

	rows, err = tx.QueryContext(ctx, cypher, params)
  if err != nil {
    return nil, err
  }

  for rows.Next() {
		human := marshalRowToHuman(rows)
		humans = append(humans, human)
  }

  return humans, nil
}

// NOTE: This can update everything that is _NOT_ sensitive to the authentication process like Identity.Password
//       To change the password see recover for that or iff identified UpdatePassword
func UpdateHuman(ctx context.Context, tx sql.Tx, newHuman Human) (human Human, err error) {
  var rows *sql.Rows
  var cypher string
  var params = make(map[string]interface{})

  if newHuman.Id == "" {
    return Human{}, errors.New("Missing Human.Id")
  }

  if newHuman.Name == "" {
    return Human{}, errors.New("Missing Human.Name")
  }

  params["id"] = newHuman.Id
  params["name"] = newHuman.Name

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id})
    SET i.name=$name
    RETURN i
  `)

	rows, err = tx.QueryContext(ctx, cypher, params)
  if err != nil {
    return Human{}, err
  }

  if rows.Next() {
		human = marshalRowToHuman(rows)
  } else {
    return Human{}, errors.New("Unable to update Human")
  }

  return human, nil
}

func ConfirmEmail(ctx context.Context, tx *sql.Tx, newHuman Human) (human Human, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if newHuman.Id == "" {
    return Human{}, errors.New("Missing Human.Id")
  }

  params["id"] = newHuman.Id

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id, email_confirmed_at:0})
    SET i.email_confirmed_at=datetime().epochSeconds
    RETURN i
  `)

	_, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
    return Human{}, err
  }

	humans, err := FetchHumans(ctx, tx, []Human{ newHuman })
  if err != nil {
    return Human{}, err
  }

  return humans[0], nil
}

func UpdatePassword(ctx context.Context, tx *sql.Tx, newHuman Human) (human Human, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if newHuman.Id == "" {
    return Human{}, errors.New("Missing Human.Id")
  }

  if newHuman.Password == "" {
    return Human{}, errors.New("Missing Human.Password")
  }

  params["id"] = newHuman.Id
  params["password"] = newHuman.Password

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id})
    SET i.password=$password
    RETURN i
  `)

	_, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
    return Human{}, err
  }

	humans, err := FetchHumans(ctx, tx, []Human{ newHuman })
  if err != nil {
    return Human{}, err
  }

  return humans[0], nil
}

func UpdateEmail(ctx context.Context, tx *sql.Tx, newHuman Human) (human Human, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if newHuman.Id == "" {
    return Human{}, errors.New("Missing Human.Id")
  }

  if newHuman.Email == "" {
    return Human{}, errors.New("Missing Human.Email")
  }

  params["id"] = newHuman.Id
  params["email"] = newHuman.Email

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id})
    SET i.email=$email
    RETURN i
  `)

	_, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
    return Human{}, err
  }

	humans, err := FetchHumans(ctx, tx, []Human{ newHuman })
  if err != nil {
    return Human{}, err
  }

  return humans[0], nil
}

func UpdateAllowLogin(ctx context.Context, tx *sql.Tx, newHuman Human) (human Human, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if newHuman.Id == "" {
    return Human{}, errors.New("Missing Human.Id")
  }

  params["id"] = newHuman.Id
  params["allow_login"] = newHuman.AllowLogin

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id})
    SET i.allow_login=$allow_login
    RETURN i
  `)

	_, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
	  return Human{}, err
  }

	humans, err := FetchHumans(ctx, tx, []Human{ newHuman })
  if err != nil {
    return Human{}, err
  }

  return humans[0], nil
}

func UpdateTotp(ctx context.Context, tx *sql.Tx, newHuman Human) (human Human, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if newHuman.Id == "" {
    return Human{}, errors.New("Missing Human.Id")
  }

  if newHuman.TotpRequired == true && newHuman.TotpSecret == "" {
    return Human{}, errors.New("Missing Human.TotpSecret")
  }

  params["id"] = newHuman.Id
  params["totp_required"] = newHuman.TotpRequired
  params["totp_secret"] = newHuman.TotpSecret

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id})
    SET i.totp_required=$totp_required,
        i.totp_secret=$totp_secret
    RETURN i
  `)

	_, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
    return Human{}, err
  }

	humans, err := FetchHumans(ctx, tx, []Human{ newHuman })
  if err != nil {
    return Human{}, err
  }

  return humans[0], nil
}

func DeleteHuman(ctx context.Context, tx *sql.Tx, newHuman Human) (human Human, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if newHuman.Id == "" {
    return Human{}, errors.New("Missing Human.Id")
  }

  params["id"] = newHuman.Id

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id})
    DETACH DELETE i
  `)

	_, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
    return Human{}, err
  }

  human.Id = newHuman.Id
  return human, nil
}
