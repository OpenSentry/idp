package idp

import (
  "strings"
  "errors"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

type Invite struct {
  Identity

  Email string
  Username string

  SentAt int64
}
func marshalNodeToInvite(node neo4j.Node) (Invite) {
  p := node.Props()

  var username string
  usr := p["username"]
  if usr != nil {
    username = p["username"].(string)
  }

  return Invite{
    Identity: marshalNodeToIdentity(node),

    Email: p["email"].(string),
    Username: username,
    SentAt: p["sent_at"].(int64),
  }
}

func UpdateInviteSentAt(driver neo4j.Driver, invite Invite) (Invite, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Invite{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (inv:Invite:Identity {id:$id}) SET inv.sent_at=datetime().epochSeconds
      RETURN inv
    `
    params := map[string]interface{}{"id": invite.Id}
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var invite Invite
    if result.Next() {
      record := result.Record()
      inviteNode := record.GetByIndex(0)
      if inviteNode != nil {
        invite = marshalNodeToInvite(inviteNode.(neo4j.Node))
      }
    } else {
      return nil, errors.New("Unable to update Invite")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return invite, nil
  })

  if err != nil {
    return Invite{}, err
  }
  return obj.(Invite), nil
}

func CreateInvite(driver neo4j.Driver, invite Invite) (Invite, error) {
  var err error
  type NeoReturnType struct{
    Invite Invite
  }

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Invite{}, err
  }
  defer session.Close()

  neoResult, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result

    var cypher string
    var params map[string]interface{}

    if invite.Username == "" {
      cypher = `
        CREATE (inv:Invite:Identity {id:randomUUID(), email:$email, iat:datetime().epochSeconds, iss:$iss, exp:$exp, sent_at:0, email_confirmed_at:0})

        WITH inv

        OPTIONAL MATCH (d:Invite:Identity) WHERE id(inv) <> id(d) AND d.exp < datetime().epochSeconds DETACH DELETE d

        RETURN inv
      `
      params = map[string]interface{}{
        "email": invite.Email,
        "iss": invite.Issuer,
        "exp": invite.ExpiresAt,
      }
    } else {
      cypher = `
        CREATE (inv:Invite:Identity {id:randomUUID(), email:$email, iat:datetime().epochSeconds, iss:$iss, exp:$exp, sent_at:0, email_confirmed_at:0, username:$username})

        WITH inv

        OPTIONAL MATCH (d:Invite:Identity) WHERE id(inv) <> id(d) AND d.exp < datetime().epochSeconds DETACH DELETE d

        RETURN inv
      `
      params = map[string]interface{}{
        "email": invite.Email,
        "iss": invite.Issuer,
        "exp": invite.ExpiresAt,
        "username": invite.Username,
      }
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var invite Invite
    if result.Next() {
      record := result.Record()

      inviteNode := record.GetByIndex(0)
      if inviteNode != nil {
        invite = marshalNodeToInvite(inviteNode.(neo4j.Node))
      }

    } else {
      return nil, errors.New("Unable to create Invite")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return NeoReturnType{ Invite:invite }, nil
  })

  if err != nil {
    return Invite{}, err
  }
  return neoResult.(NeoReturnType).Invite, nil
}

func FetchInvites(driver neo4j.Driver, invites []Invite) ([]Invite, error) {
  ids := []string{}
  for _, invite := range invites {
    ids = append(ids, invite.Id)
  }
  return FetchInvitesById(driver, ids)
}

func FetchInvitesAll(driver neo4j.Driver) ([]Invite, error) {
  var cypher string
  var params map[string]interface{}

  cypher = `
    MATCH (inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds
    RETURN inv
  `
  params = map[string]interface{}{}
  return fetchInvitesByQuery(driver, cypher, params)
}

func FetchInvitesById(driver neo4j.Driver, ids []string) ([]Invite, error) {
  var cypher string
  var params map[string]interface{}

  if ids == nil {
    cypher = `
      MATCH (inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds
      RETURN inv
    `
    params = map[string]interface{}{
      "id": strings.Join(ids, ","),
    }
  } else {
    cypher = `
      MATCH (inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds AND inv.id in split($ids, ",")
      RETURN inv
    `
    params = map[string]interface{}{
      "ids": strings.Join(ids, ","),
    }
  }
  return fetchInvitesByQuery(driver, cypher, params)
}

func FetchInvitesByEmail(driver neo4j.Driver, emails []string) ([]Invite, error) {
  var cypher string
  var params map[string]interface{}

  if emails == nil {
    cypher = `
      MATCH (inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds
      RETURN inv
    `
    params = map[string]interface{}{}
  } else {
    cypher = `
      MATCH (inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds AND inv.email in split($emails, ",")
      RETURN inv
    `
    params = map[string]interface{}{
      "emails": strings.Join(emails, ","),
    }
  }
  return fetchInvitesByQuery(driver, cypher, params)
}

/*func FetchInvitesByUsername(driver neo4j.Driver, usernames []string) ([]Invite, error) {
  var cypher string
  var params map[string]interface{}

  if usernames == nil {
    cypher = `
      MATCH (ibi:Human:Identity)-[:INVITES]->(inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds
      RETURN inv, ibi
    `
    params = map[string]interface{}{}
  } else {
    cypher = `
      MATCH (ibi:Human:Identity)-[:INVITES]->(inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds AND inv.username in split($usernames, ",")
      RETURN inv, ibi
    `
    params = map[string]interface{}{
      "usernames": strings.Join(usernames, ","),
    }
  }
  return fetchInvitesByQuery(driver, cypher, params)
}*/

func fetchInvitesByQuery(driver neo4j.Driver, cypher string, params map[string]interface{}) ([]Invite, error)  {
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

    var invites []Invite
    for result.Next() {
      record := result.Record()

      inviteNode := record.GetByIndex(0)
      if inviteNode != nil {
        invite := marshalNodeToInvite(inviteNode.(neo4j.Node))
        invites = append(invites, invite)
      }
    }
    if err = result.Err(); err != nil {
      return nil, err
    }
    return invites, nil
  })

  if err != nil {
    return nil, err
  }
  if neoResult == nil {
    return nil, nil
  }
  return neoResult.([]Invite), nil
}

