package idpapi

import (
  "golang.org/x/crypto/bcrypt"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

type Identity struct {
  Id            string          `json:"id" binding:"required"`
  Name          string          `json:"name"`
  Email         string          `json:"email"`
  Password      string          `json:"password"`
}

func ValidatePassword(storedPassword string, password string) (bool, error) {
  err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
  if err != nil {
		return false, err
	}
  return true, nil
}

func CreatePassword(password string) (string, error) {
  hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
  if err != nil {
    return "", err
  }
  return string(hash), nil
}

func UpdatePassword(driver neo4j.Driver, identity Identity) (Identity, error) {
  var err error
  var session neo4j.Session
  var id interface{}

  session, err = driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Identity{}, err
  }
  defer session.Close()

  id, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := "MATCH (i:Identity {sub:$sub}) SET i.password=$password RETURN i.sub, i.password, i.name, i.email"
    params := map[string]interface{}{"sub": identity.Id, "password": identity.Password}
    if result, err = tx.Run(cypher, params); err != nil {
      return Identity{}, err
    }

    var ret Identity
    if result.Next() {
      record := result.Record()

      // NOTE: This means the statment sequence of the RETURN (possible order by)
      // https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
      // If results are consumed in the same order as they are produced, records merely pass through the buffer; if they are consumed out of order, the buffer will be utilized to retain records until
      // they are consumed by the application. For large results, this may require a significant amount of memory and impact performance. For this reason, it is recommended to consume results in order wherever possible.
      sub := record.GetByIndex(0).(string)
      password := record.GetByIndex(1).(string)
      name := record.GetByIndex(2).(string)
      email := record.GetByIndex(3).(string)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        Password: password,
      }
      ret = identity
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return ret, nil
  })

  if err != nil {
    return Identity{}, err
  }
  return id.(Identity), nil
}

func CreateIdentities(driver neo4j.Driver, identity Identity) ([]Identity, error) {
  var err error
  var session neo4j.Session
  var ids interface{}

  session, err = driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return nil, err
  }
  defer session.Close()

  ids, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := "CREATE (i:Identity {sub:$sub, password:$password, name:$name, email:$email}) RETURN i.sub, i.password, i.name, i.email"
    params := map[string]interface{}{"sub": identity.Id, "password": identity.Password, "name": identity.Name, "email": identity.Email}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var identities []Identity
    if result.Next() {
      record := result.Record()

      // NOTE: This means the statment sequence of the RETURN (possible order by)
      // https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
      // If results are consumed in the same order as they are produced, records merely pass through the buffer; if they are consumed out of order, the buffer will be utilized to retain records until
      // they are consumed by the application. For large results, this may require a significant amount of memory and impact performance. For this reason, it is recommended to consume results in order wherever possible.
      sub := record.GetByIndex(0).(string)
      password := record.GetByIndex(1).(string)
      name := record.GetByIndex(2).(string)
      email := record.GetByIndex(3).(string)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        Password: password,
      }
      identities = append(identities, identity)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return identities, nil
  })

  if err != nil {
    return nil, err
  }
  return ids.([]Identity), nil
}

// NOTE: This can update eveything but the Identity.sub and Identity.password
//       To change the password see recover for that.
func UpdateIdentities(driver neo4j.Driver, identity Identity) ([]Identity, error) {
  var err error
  var session neo4j.Session
  var ids interface{}

  session, err = driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return nil, err
  }
  defer session.Close()

  ids, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := "MATCH (i:Identity {sub:$sub}) WITH i SET i.name=$name, i.email=$email) RETURN i.sub, i.password, i.name, i.email"
    params := map[string]interface{}{"sub": identity.Id, "name": identity.Name, "email": identity.Email}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var identities []Identity
    if result.Next() {
      record := result.Record()

      // NOTE: This means the statment sequence of the RETURN (possible order by)
      // https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
      // If results are consumed in the same order as they are produced, records merely pass through the buffer; if they are consumed out of order, the buffer will be utilized to retain records until
      // they are consumed by the application. For large results, this may require a significant amount of memory and impact performance. For this reason, it is recommended to consume results in order wherever possible.
      sub := record.GetByIndex(0).(string)
      password := record.GetByIndex(1).(string)
      name := record.GetByIndex(2).(string)
      email := record.GetByIndex(3).(string)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        Password: password,
      }
      identities = append(identities, identity)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return identities, nil
  })

  if err != nil {
    return nil, err
  }
  return ids.([]Identity), nil
}

// https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
func FetchIdentitiesForSub(driver neo4j.Driver, sub string) ([]Identity, error) {
  var err error
  var session neo4j.Session
  var ids interface{}

  session, err = driver.Session(neo4j.AccessModeRead);
  if err != nil {
    return nil, err
  }
  defer session.Close()

  ids, err = session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result

    cypher := "MATCH (i:Identity {sub: $sub}) RETURN i.sub, i.password, i.name, i.email ORDER BY i.sub"
    params := map[string]interface{}{"sub": sub}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var identities []Identity
    if result.Next() {
      record := result.Record()

      // NOTE: This means the statment sequence of the RETURN (possible order by)
      // https://neo4j.com/docs/driver-manual/current/cypher-values/index.html
      // If results are consumed in the same order as they are produced, records merely pass through the buffer; if they are consumed out of order, the buffer will be utilized to retain records until
      // they are consumed by the application. For large results, this may require a significant amount of memory and impact performance. For this reason, it is recommended to consume results in order wherever possible.
      sub := record.GetByIndex(0).(string)
      password := record.GetByIndex(1).(string)
      name := record.GetByIndex(2).(string)
      email := record.GetByIndex(3).(string)

      identity := Identity{
        Id: sub,
        Name: name,
        Email: email,
        Password: password,
      }
      identities = append(identities, identity)
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return identities, nil
  })
  if err != nil {
    return nil, err
  }
  return ids.([]Identity), nil
}
