package idp

import (
  "errors"
  "strings"
  "fmt"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

func CreateHumanFromInvite(tx neo4j.Transaction, newHuman Human) (human Human, err error) {
  var result neo4j.Result
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

  cypher = fmt.Sprintf(`
    MATCH (i:Invite:Identity {id:$id})
      SET i.email_confirmed_at=$email_confirmed_at,
          i.username=$username,
          i.name=$name,
          i.allow_login=$allow_login,
          i.password=$password,
          i.totp_required=false,
          i.totp_secret="",
          i.otp_recover_code="",
          i.otp_recover_code_expire=0,
          i.otp_delete_code="",
          i.otp_delete_code_expire=0,
          i.exp=0,
          i:Human

    WITH i

    REMOVE i:Invite

    RETURN i
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return Human{}, err
  }

  if result.Next() {
    record          := result.Record()
    humanNode       := record.GetByIndex(0)

    if humanNode != nil {
      human = marshalNodeToHuman(humanNode.(neo4j.Node))
    }
  } else {
    return Human{}, errors.New("Unable to create Human")
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Human{}, err
  }

  return human, nil
}

func CreateHuman(tx neo4j.Transaction, newHuman Human) (human Human, err error) {
  var result neo4j.Result
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

  params["iss"] = newHuman.Issuer
  params["exp"] = newHuman.ExpiresAt
  params["email"] = newHuman.Email
  params["username"] = newHuman.Username
  params["name"] = newHuman.Name
  params["allow_login"] = newHuman.AllowLogin
  params["password"] = newHuman.Password

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
      totp_secret: "",

      otp_recover_code: "",
      otp_recover_code_expire: 0,
      otp_delete_code: "",
      otp_delete_code_expire: 0
    })
    RETURN i
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return Human{}, err
  }

  if result.Next() {
    record          := result.Record()
    humanNode       := record.GetByIndex(0)

    if humanNode != nil {
      human = marshalNodeToHuman(humanNode.(neo4j.Node))
    }
  } else {
    return Human{}, errors.New("Unable to create Human")
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Human{}, err
  }

  return human, nil
}

func FetchHumans(tx neo4j.Transaction, iHumans []Human) (humans []Human, err error) {
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

  cypher = fmt.Sprintf(`
    MATCH (h:Human:Identity) %s
    RETURN h
  `, cypfilterIds)

  humans, err = fetchHumansByQuery(tx, cypher, params)
  return humans, err
}

func FetchHumansByEmail(tx neo4j.Transaction, iHumans []Human) (humans []Human, err error) {
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

  cypher = fmt.Sprintf(`
    MATCH (h:Human:Identity) %s
    RETURN h
  `, cypfilterEmails)

  humans, err = fetchHumansByQuery(tx, cypher, params)
  return humans, err
}

func FetchHumansByUsername(tx neo4j.Transaction, iHumans []Human) (humans []Human, err error) {
  var cypher string
  var params = make(map[string]interface{})

  cypfilterUsernames := ""
  if len(iHumans) > 0 {
    var usernames []string
    for _, human := range iHumans {
      usernames = append(usernames, human.Username)
    }
    cypfilterUsernames = ` WHERE h.email in split($usernames, ",") `
    params["usernames"] = strings.Join(usernames, ",")
  }

  cypher = fmt.Sprintf(`
    MATCH (h:Human:Identity) %s
    RETURN h
  `, cypfilterUsernames)

  humans, err = fetchHumansByQuery(tx, cypher, params)
  return humans, err
}

func fetchHumansByQuery(tx neo4j.Transaction, cypher string, params map[string]interface{}) (humans []Human, err error) {
  var result neo4j.Result

  if result, err = tx.Run(cypher, params); err != nil {
    return nil, err
  }

  for result.Next() {
    record          := result.Record()
    humanNode       := record.GetByIndex(0)

    if humanNode != nil {
      human := marshalNodeToHuman(humanNode.(neo4j.Node))
      humans = append(humans, human)
    }
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return nil, err
  }

  return humans, nil
}

// NOTE: This can update everything that is _NOT_ sensitive to the authentication process like Identity.Password
//       To change the password see recover for that or iff identified UpdatePassword
func UpdateHuman(tx neo4j.Transaction, newHuman Human) (human Human, err error) {
  var result neo4j.Result
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

  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id})
    SET i.name=$name
    RETURN i
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return Human{}, err
  }

  if result.Next() {
    record          := result.Record()
    humanNode       := record.GetByIndex(0)

    if humanNode != nil {
      human = marshalNodeToHuman(humanNode.(neo4j.Node))
    }
  } else {
    return Human{}, errors.New("Unable to update Human")
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Human{}, err
  }

  return human, nil
}

func ConfirmEmail(tx neo4j.Transaction, newHuman Human) (human Human, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  if newHuman.Id == "" {
    return Human{}, errors.New("Missing Human.Id")
  }

  params["id"] = newHuman.Id

  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id, email_confirmed_at:0})
    SET i.email_confirmed_at=datetime().epochSeconds
    RETURN i
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return Human{}, err
  }

  if result.Next() {
    record          := result.Record()
    humanNode       := record.GetByIndex(0)

    if humanNode != nil {
      human = marshalNodeToHuman(humanNode.(neo4j.Node))
    }
  } else {
    return Human{}, errors.New("Unable to confirm email for human")
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Human{}, err
  }

  return human, nil
}

func UpdatePassword(tx neo4j.Transaction, newHuman Human) (human Human, err error) {
  var result neo4j.Result
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

  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id})
    SET i.password=$password
    RETURN i
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return Human{}, err
  }

  if result.Next() {
    record          := result.Record()
    humanNode       := record.GetByIndex(0)

    if humanNode != nil {
      human = marshalNodeToHuman(humanNode.(neo4j.Node))
    }
  } else {
    return Human{}, errors.New("Unable to update password for human")
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Human{}, err
  }

  return human, nil
}

func UpdateAllowLogin(tx neo4j.Transaction, newHuman Human) (human Human, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  if newHuman.Id == "" {
    return Human{}, errors.New("Missing Human.Id")
  }

  params["id"] = newHuman.Id
  params["allow_login"] = newHuman.AllowLogin

  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id})
    SET i.allow_login=$allow_login
    RETURN i
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return Human{}, err
  }

  if result.Next() {
    record          := result.Record()
    humanNode       := record.GetByIndex(0)

    if humanNode != nil {
      human = marshalNodeToHuman(humanNode.(neo4j.Node))
    }
  } else {
    return Human{}, errors.New("Unable to update allow login for human")
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Human{}, err
  }

  return human, nil
}

func UpdateTotp(tx neo4j.Transaction, newHuman Human) (human Human, err error) {
  var result neo4j.Result
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

  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id})
    SET i.totp_required=$totp_required,
        i.totp_secret=$totp_secret
    RETURN i
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return Human{}, err
  }

  if result.Next() {
    record          := result.Record()
    humanNode       := record.GetByIndex(0)

    if humanNode != nil {
      human = marshalNodeToHuman(humanNode.(neo4j.Node))
    }
  } else {
    return Human{}, errors.New("Unable to update TOTP for human")
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Human{}, err
  }

  return human, nil
}

func DeleteHuman(tx neo4j.Transaction, newHuman Human) (human Human, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  if newHuman.Id == "" {
    return Human{}, errors.New("Missing Human.Id")
  }

  params["id"] = newHuman.Id

  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id})
    DETACH DELETE i
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return Human{}, err
  }

  result.Next()

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Human{}, err
  }

  human.Id = newHuman.Id
  return human, nil
}


// @TODO: These should probably be refactored to challenge system and get deleted from here

func UpdateOtpDeleteCode(tx neo4j.Transaction, newHuman Human) (human Human, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  if newHuman.Id == "" {
    return Human{}, errors.New("Missing Human.Id")
  }

  if newHuman.OtpDeleteCode == "" {
    return Human{}, errors.New("Missing Human.OtpDeleteCode")
  }

  params["id"] = newHuman.Id
  params["otp_delete_code"] = newHuman.OtpDeleteCode
  params["otp_delete_code_expire"] = newHuman.OtpDeleteCodeExpire

  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id})
    SET i.otp_delete_code = $otp_delete_code,
        i.otp_delete_code_expire = $otp_delete_code_expire
    RETURN i
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return Human{}, err
  }

  if result.Next() {
    record          := result.Record()
    humanNode       := record.GetByIndex(0)

    if humanNode != nil {
      human = marshalNodeToHuman(humanNode.(neo4j.Node))
    }
  } else {
    return Human{}, errors.New("Unable to update OTP delete code for human")
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Human{}, err
  }

  return human, nil
}

func UpdateOtpRecoverCode(tx neo4j.Transaction, newHuman Human) (human Human, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  if newHuman.Id == "" {
    return Human{}, errors.New("Missing Human.Id")
  }

  if newHuman.OtpRecoverCode == "" {
    return Human{}, errors.New("Missing Human.OtpRecoverCode")
  }

  params["id"] = newHuman.Id
  params["otp_recover_code"] = newHuman.OtpRecoverCode
  params["otp_recover_code_expire"] = newHuman.OtpRecoverCodeExpire

  cypher = fmt.Sprintf(`
    MATCH (i:Human:Identity {id:$id})
    SET i.otp_recover_code = $otp_recover_code,
        i.otp_recover_code_expire = $otp_recover_code_expire
    RETURN i
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return Human{}, err
  }

  if result.Next() {
    record          := result.Record()
    humanNode       := record.GetByIndex(0)

    if humanNode != nil {
      human = marshalNodeToHuman(humanNode.(neo4j.Node))
    }
  } else {
    return Human{}, errors.New("Unable to update OTP recover code for human")
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Human{}, err
  }

  return human, nil
}