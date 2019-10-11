package idp

import (
  "strings"
  "errors"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

type Invite struct {
  Identity

  Username string
  Email string

  InvitedBy Human
}
func marshalNodeToInvite(node neo4j.Node) (Invite) {
  p := node.Props()

  return Invite{
    Identity: marshalNodeToIdentity(node),

    Username: p["username"].(string),
    Email: p["email"].(string),
  }
}

func CreateInvite(driver neo4j.Driver, invite Invite, invitedBy Human) (Invite, Human, error) {
  var err error
  type NeoReturnType struct{
    Invite Invite
    InvitedBy Human
  }

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Invite{}, Human{}, err
  }
  defer session.Close()

  neoResult, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (ibi:Identity {id:$ibi})
      MERGE (ibi)-[:INVITES]->(inv:Invite:Identity {id:randomUUID(), iat:datetime().epochSeconds, iss:$iss, exp:$exp, email:$email, username:$username})

      WITH inv, ibi

      OPTIONAL MATCH (ibi)-[:INVITES]->(d:Invite:Identity) WHERE id(inv) <> id(d) AND d.exp < datetime().epochSeconds DETACH DELETE d

      RETURN inv, ibi
    `
    params := map[string]interface{}{
      "ibi": invitedBy.Id,
      "email": invite.Email,
      "iss": invite.Issuer,
      "exp": invite.ExpiresAt,
      "username": invite.Username,
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var invite Invite
    var invitedBy Human
    if result.Next() {
      record := result.Record()

      inviteNode := record.GetByIndex(0)
      if inviteNode != nil {
        invite = marshalNodeToInvite(inviteNode.(neo4j.Node))

        ibiNode := record.GetByIndex(1)
        if ibiNode != nil {
          invitedBy = marshalNodeToHuman(ibiNode.(neo4j.Node))
          invite.InvitedBy = invitedBy
        }

      }

    } else {
      return nil, errors.New("Unable to create Invite")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return NeoReturnType{ Invite:invite, InvitedBy:invitedBy }, nil
  })

  if err != nil {
    return Invite{}, Human{}, err
  }
  return neoResult.(NeoReturnType).Invite, neoResult.(NeoReturnType).InvitedBy, nil
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
    MATCH (ibi:Human:Identity)-[:INVITES]->(inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds
    RETURN inv, ibi
  `
  params = map[string]interface{}{}
  return fetchInvitesByQuery(driver, cypher, params)
}

func FetchInvitesById(driver neo4j.Driver, ids []string) ([]Invite, error) {
  var cypher string
  var params map[string]interface{}

  if ids == nil {
    cypher = `
      MATCH (ibi:Human:Identity)-[:INVITES]->(inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds
      RETURN inv, ibi
    `
    params = map[string]interface{}{
      "id": strings.Join(ids, ","),
    }
  } else {
    cypher = `
      MATCH (ibi:Human:Identity)-[:INVITES]->(inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds AND inv.Id in split($ids, ",")
      RETURN inv, ibi
    `
    params = map[string]interface{}{
      "ids": strings.Join(ids, ","),
    }
  }
  return fetchInvitesByQuery(driver, cypher, params)
}

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

        ibiNode := record.GetByIndex(1)
        if ibiNode != nil {
          ibi := marshalNodeToHuman(ibiNode.(neo4j.Node))
          invite.InvitedBy = ibi
        }

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

