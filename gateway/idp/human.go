package idp

import (
  "errors"
  "strings"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

type Human struct {
  Identity

  // Identity.Id aliasses
  Email                string
  EmailConfirmedAt     int64
  Username             string

  Name                 string

  AllowLogin           bool

  Password             string

  TotpRequired         bool
  TotpSecret           string

  OtpRecoverCode       string
  OtpRecoverCodeExpire int64
}

func marshalNodeToHuman(node neo4j.Node) (Human) {
  p := node.Props()

  return Human{
    Identity: marshalNodeToIdentity(node),

    Email:                p["email"].(string),
    EmailConfirmedAt:     p["email_confirmed_at"].(int64),
    Username:             p["username"].(string),

    Name:                 p["name"].(string),

    AllowLogin:           p["allow_login"].(bool),

    Password:             p["password"].(string),

    TotpRequired:         p["totp_required"].(bool),
    TotpSecret:           p["totp_secret"].(string),

    OtpRecoverCode:       p["otp_recover_code"].(string),
    OtpRecoverCodeExpire: p["otp_recover_code_expire"].(int64),

  }
}

func CreateHuman(driver neo4j.Driver, human Human) (Human, error) {
  var err error
  type NeoReturnType struct{
    Human Human
  }

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Human{}, err
  }
  defer session.Close()

  neoResult, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      CREATE (i:Human:Identity {
        id:randomUUID(), iat:datetime().epochSeconds, iss:$iss, exp:$exp,

        email:$email, email_confirmed_at:0, username:$username,

        name:$name,

        allow_login:$allow_login,

        password:$password,

        totp_required:false, totp_secret:"",

        otp_recover_code:"", otp_recover_code_expire:0,
        otp_delete_code:"", otp_delete_code_expire:0
      })
      RETURN i
    `
    params := map[string]interface{}{
      "iss": human.Issuer, "exp": human.ExpiresAt,
      "email": human.Email, "username": human.Username,
      "name": human.Name,
      "allow_login": human.AllowLogin,
      "password": human.Password,
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var human Human
    if result.Next() {
      record := result.Record()
      humanNode := record.GetByIndex(0)
      if humanNode != nil {
        human = marshalNodeToHuman(humanNode.(neo4j.Node))
      }
    } else {
      return nil, errors.New("Unable to create Human")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return NeoReturnType{Human: human}, nil
  })

  if err != nil {
    return Human{}, err
  }
  return neoResult.(NeoReturnType).Human, nil
}

func FetchHumansAll(driver neo4j.Driver) ([]Human, error) {
  var cypher string
  var params map[string]interface{}

  cypher = `
    MATCH (i:Human:Identity)
    RETURN i
  `
  params = map[string]interface{}{}
  return fetchHumansByQuery(driver, cypher, params)
}

func FetchHumansById(driver neo4j.Driver, ids []string) ([]Human, error) {
  var cypher string
  var params map[string]interface{}

  if ids == nil {
    cypher = `
      MATCH (i:Human:Identity)
      RETURN i
    `
    params = map[string]interface{}{}
  } else {
    cypher = `
      MATCH (i:Human:Identity) WHERE i.id in split($ids, ",")
      RETURN i
    `
    params = map[string]interface{}{
      "ids": strings.Join(ids, ","),
    }
  }
  return fetchHumansByQuery(driver, cypher, params)
}

func FetchHumansByEmail(driver neo4j.Driver, emails []string) ([]Human, error) {
  var cypher string
  var params map[string]interface{}

  if emails == nil {
    cypher = `
      MATCH (i:Human:Identity)
      RETURN i
    `
    params = map[string]interface{}{}
  } else {
    cypher = `
      MATCH (i:Human:Identity) WHERE i.email in split($emails, ",")
      RETURN i
    `
    params = map[string]interface{}{
      "emails": strings.Join(emails, ","),
    }
  }
  return fetchHumansByQuery(driver, cypher, params)
}

func FetchHumansByUsername(driver neo4j.Driver, usernames []string) ([]Human, error) {
  var cypher string
  var params map[string]interface{}

  if usernames == nil {
    cypher = `
      MATCH (i:Human:Identity)
      RETURN i
    `
    params = map[string]interface{}{}
  } else {
    cypher = `
      MATCH (i:Human:Identity) WHERE i.username in split($usernames, ",")
      RETURN i
    `
    params = map[string]interface{}{
      "usernames": strings.Join(usernames, ","),
    }
  }
  return fetchHumansByQuery(driver, cypher, params)
}

func fetchHumansByQuery(driver neo4j.Driver, cypher string, params map[string]interface{}) ([]Human, error)  {
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

    var humans []Human
    for result.Next() {
      record := result.Record()

      humanNode := record.GetByIndex(0)
      if humanNode != nil {
        human := marshalNodeToHuman(humanNode.(neo4j.Node))
        humans = append(humans, human)
      }
    }
    if err = result.Err(); err != nil {
      return nil, err
    }
    return humans, nil
  })

  if err != nil {
    return nil, err
  }
  if neoResult == nil {
    return nil, nil
  }
  return neoResult.([]Human), nil
}

// NOTE: This can update everything that is _NOT_ sensitive to the authentication process like Identity.Password
//       To change the password see recover for that.
func UpdateHuman(driver neo4j.Driver, human Human) (Human, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Human{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id}) WITH i SET i.name=$name, i.email=$email
      RETURN i
    `
    params := map[string]interface{}{"id": human.Id, "name": human.Name, "email": human.Email}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var human Human
    if result.Next() {
      record := result.Record()
      humanNode := record.GetByIndex(0)
      if humanNode != nil {
        human = marshalNodeToHuman(humanNode.(neo4j.Node))
      }
    } else {
      return nil, errors.New("Unable to update Human")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return human, nil
  })

  if err != nil {
    return Human{}, err
  }
  return obj.(Human), nil
}

func UpdateOtpRecoverCode(driver neo4j.Driver, human Human) (Human, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Human{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id}) SET i.otp_recover_code=$code, i.otp_recover_code_expire=$expire
      RETURN i
    `
    params := map[string]interface{}{ "id":human.Id, "code":human.OtpRecoverCode, "expire":human.OtpRecoverCodeExpire }
    if result, err = tx.Run(cypher, params); err != nil {
      return Human{}, err
    }

    var human Human
    if result.Next() {
      record := result.Record()
      humanNode := record.GetByIndex(0)
      if humanNode != nil {
        human = marshalNodeToHuman(humanNode.(neo4j.Node))
      }
    } else {
      return nil, errors.New("Unable to update otp recover code")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return human, nil
  })

  if err != nil {
    return Human{}, err
  }
  return obj.(Human), nil
}

func ConfirmEmail(driver neo4j.Driver, human Human) (Human, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Human{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id, email:$email}) SET i.email_confirmed_at=datetime().epochSeconds
      RETURN i
    `
    params := map[string]interface{}{ "id":human.Id, "email":human.Email }
    if result, err = tx.Run(cypher, params); err != nil {
      return Human{}, err
    }

    var human Human
    if result.Next() {
      record := result.Record()
      humanNode := record.GetByIndex(0)
      if humanNode != nil {
        human = marshalNodeToHuman(humanNode.(neo4j.Node))
      }
    } else {
      return nil, errors.New("Unable to confirm email")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return human, nil
  })

  if err != nil {
    return Human{}, err
  }
  return obj.(Human), nil
}

func UpdatePassword(driver neo4j.Driver, human Human) (Human, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Human{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id}) SET i.password=$password
      RETURN i
    `
    params := map[string]interface{}{ "id":human.Id, "password":human.Password }
    if result, err = tx.Run(cypher, params); err != nil {
      return Human{}, err
    }

    var human Human
    if result.Next() {
      record := result.Record()
      humanNode := record.GetByIndex(0)
      if humanNode != nil {
        human = marshalNodeToHuman(humanNode.(neo4j.Node))
      }
    } else {
      return nil, errors.New("Unable to update password")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return human, nil
  })

  if err != nil {
    return Human{}, err
  }
  return obj.(Human), nil
}

func UpdateAllowLogin(driver neo4j.Driver, human Human) (Human, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Human{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id}) SET i.allow_login=$allow_login
      RETURN i
    `
    params := map[string]interface{}{ "id":human.Id, "allow_login":human.AllowLogin }
    if result, err = tx.Run(cypher, params); err != nil {
      return Human{}, err
    }

    var human Human
    if result.Next() {
      record := result.Record()
      humanNode := record.GetByIndex(0)
      if humanNode != nil {
        human = marshalNodeToHuman(humanNode.(neo4j.Node))
      }
    } else {
      return nil, errors.New("Unable to update allow login")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return human, nil
  })

  if err != nil {
    return Human{}, err
  }
  return obj.(Human), nil
}

func UpdateTotp(driver neo4j.Driver, human Human) (Human, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Human{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id}) SET i.totp_required=$required, i.totp_secret=$secret
      RETURN i
    `
    params := map[string]interface{}{ "id":human.Id, "required":human.TotpRequired, "secret":human.TotpSecret }
    if result, err = tx.Run(cypher, params); err != nil {
      return Human{}, err
    }

    var human Human
    if result.Next() {
      record := result.Record()
      humanNode := record.GetByIndex(0)
      if humanNode != nil {
        human = marshalNodeToHuman(humanNode.(neo4j.Node))
      }
    } else {
      return nil, errors.New("Unable to update TOTP")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return human, nil
  })

  if err != nil {
    return Human{}, err
  }
  return obj.(Human), nil
}

func DeleteHuman(driver neo4j.Driver, human Human) (Human, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Human{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id}) DETACH DELETE i
    `
    params := map[string]interface{}{"id": human.Id}
    if result, err = tx.Run(cypher, params); err != nil {
      return Human{}, err
    }

    result.Next()

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return Human{}, err
    }
    return Human{ Identity:Identity{Id: human.Id} }, nil
  })

  if err != nil {
    return Human{}, err
  }
  return obj.(Human), nil
}

func UpdateOtpDeleteCode(driver neo4j.Driver, human Human) (Human, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Human{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (i:Human:Identity {id:$id}) SET i.otp_delete_code=$code, i.otp_delete_code_expire=$expire
      RETURN i
    `
    params := map[string]interface{}{ "id":human.Id, "code":human.OtpDeleteCode, "expire":human.OtpDeleteCodeExpire }
    if result, err = tx.Run(cypher, params); err != nil {
      return Human{}, err
    }

    var human Human
    if result.Next() {
      record := result.Record()
      identityNode := record.GetByIndex(0)
      if identityNode != nil {
        human = marshalNodeToHuman(identityNode.(neo4j.Node))
      }
    } else {
      return nil, errors.New("Unable to update otp delete code")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return human, nil
  })

  if err != nil {
    return Human{}, err
  }
  return obj.(Human), nil
}