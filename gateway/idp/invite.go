package idp

import (
  "strings"
  "errors"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

type Invite struct {
  Human

  HintUsername string

  Invited *Human
  InvitedBy *Human

  SentTo *Email
}
func marshalNodeToInvite(node neo4j.Node) (Invite) {
  p := node.Props()

  return Invite{
    Human: marshalNodeToHuman(node),

    HintUsername: p["hint_username"].(string),
  }
}

type Email struct {
  Email string
}
func marshalNodeToEmail(node neo4j.Node) (Email) {
  p := node.Props()

  return Email{
    Email: p["email"].(string),
  }
}

// CRUD

func CreateInvite(driver neo4j.Driver, invite Invite, invitedBy Human, email Email) (Invite, Human, Email, error) {
  var err error
  type NeoReturnType struct{
    Invite Invite
    InvitedBy Human
    Email Email
  }

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Invite{}, Human{}, Email{}, err
  }
  defer session.Close()

  neoResult, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (ibi:Identity {id:$ibi})
      OPTIONAL MATCH (i:Human:Identity {email:$email})

      CREATE (inv:Invite:Identity {
        id:randomUUID(), iat:datetime().epochSeconds, iss:$iss, exp:$exp,
        hint_username:$hint_username,
      })-[:INVITED_BY]->(ibi)

      WITH inv, e, ibi, i, collect(i) as h

      FOREACH( n in h | MERGE (n)-[:IS_INVITED]->(inv) )

      WITH inv, e, ibi, i

      MERGE (inv)-[:SENT_TO]->(e:Email{email:$email})

      WITH inv, e, ibi, i

      OPTIONAL MATCH (d:Invite:Identity)-[:INVITED_BY]->(ibi) WHERE id(inv) <> id(d) AND d.exp < datetime().epochSeconds DETACH DELETE d

      RETURN inv, e, ibi, i
    `
    params := map[string]interface{}{
      "ibi": invitedBy.Id,
      "email": email.Email,
      "iss": invite.Issuer, "exp": invite.ExpiresAt,
      "hint_username": invite.HintUsername,
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var invite Invite
    var invited Human
    var invitedBy Human
    var email Email
    if result.Next() {
      record := result.Record()

      inviteNode := record.GetByIndex(0)
      if inviteNode != nil {
        invite := marshalNodeToInvite(inviteNode.(neo4j.Node))

        emailNode := record.GetByIndex(1)
        if emailNode != nil {
          email = marshalNodeToEmail(emailNode.(neo4j.Node))
          invite.SentTo = &email
        }

        ibiNode := record.GetByIndex(2)
        if ibiNode != nil {
          invitedBy = marshalNodeToHuman(ibiNode.(neo4j.Node))
          invite.InvitedBy = &invitedBy
        }

        invitedNode := record.GetByIndex(3)
        if invitedNode != nil {
          invited = marshalNodeToHuman(invitedNode.(neo4j.Node))
          invite.Invited = &invited
        }
      }

    } else {
      return nil, errors.New("Unable to create Invite")
    }

    // Check if we encountered any error during record streaming
    if err = result.Err(); err != nil {
      return nil, err
    }
    return NeoReturnType{ Invite:invite, InvitedBy:invitedBy, Email:email }, nil
  })

  if err != nil {
    return Invite{}, Human{}, Email{}, err
  }
  return neoResult.(NeoReturnType).Invite, neoResult.(NeoReturnType).InvitedBy, neoResult.(NeoReturnType).Email, nil
}

func FetchInvites(driver neo4j.Driver, invites []Invite) ([]Invite, error) {
  ids := []string{}
  for _, invite := range invites {
    ids = append(ids, invite.Id)
  }
  return FetchInvitesById(driver, ids)
}

func FetchInvitesById(driver neo4j.Driver, ids []string) ([]Invite, error) {
  var cypher string
  var params map[string]interface{}

  if ids == nil {
    cypher = `
      MATCH (inv:Invite:Identity {id: $id}) WHERE inv.exp > datetime().epochSeconds
      MATCH (inv)-[:SENT_TO]->(e:Email)

      OPTIONAL MATCH (i:Identity)-[:IS_INVITED]->(inv)-[:INVITED_BY]->(ibi:Identity)

      RETURN inv, e, i, ibi
    `
    params = map[string]interface{}{
      "id": strings.Join(ids, ","),
    }
  } else {
    cypher = `
      MATCH (inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds AND inv.Id in split($ids, ",")
      MATCH (inv)-[:SENT_TO]->(e:Email)

      OPTIONAL MATCH (i:Identity)-[:IS_INVITED]->(inv)-[:INVITED_BY]->(ibi:Identity)

      RETURN inv, e, ibi, i
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

  neoResult, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var err error
    var out []Invite
    for result.Next() {
      record := result.Record()

      inviteNode := record.GetByIndex(0)
      if inviteNode != nil {
        invite := marshalNodeToInvite(inviteNode.(neo4j.Node))

        emailNode := record.GetByIndex(1)
        if emailNode != nil {
          email := marshalNodeToEmail(emailNode.(neo4j.Node))
          invite.SentTo = &email
        }

        ibiNode := record.GetByIndex(2)
        if ibiNode != nil {
          ibi := marshalNodeToHuman(ibiNode.(neo4j.Node))
          invite.InvitedBy = &ibi
        }

        invitedNode := record.GetByIndex(3)
        if invitedNode != nil {
          invited := marshalNodeToHuman(invitedNode.(neo4j.Node))
          invite.Invited = &invited
        }

        out = append(out, invite)
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
  if neoResult != nil {
    return nil, nil
  }
  return neoResult.([]Invite), nil
}

// Actions

func AcceptInvite(driver neo4j.Driver, invite Invite) (Invite, error) {
  var err error

  session, err := driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    return Invite{}, err
  }
  defer session.Close()

  obj, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
    var result neo4j.Result
    cypher := `
      MATCH (inv:Invite:Identity {id:$id})-[:INVITED_BY]->(ibi:Identity)
      MATCH (inv)-[:SENT_TO]->(e:Email)
      MATCH (i:Human:Identity {email:e.email)

      SET inv.verified_at = datetime().epochSeconds

      WITH inv, e, ibi, i, collect(i) as h

      FOREACH( n in h | MERGE (n)-[:IS_INVITED]->(inv) )

      WITH inv, e, ibi, i

      OPTIONAL MATCH (d:Invite:Identity)-[:INVITED_BY]->(ibi) WHERE id(inv) <> id(d) AND d.exp < datetime().epochSeconds DETACH DELETE d

      RETURN inv, e, ibi, i
    `
    params := map[string]interface{}{
      "id": invite.Id,
    }
    if result, err = tx.Run(cypher, params); err != nil {
      return nil, err
    }

    var invite Invite
    if result.Next() {
      record := result.Record()

      inviteNode := record.GetByIndex(0)
      if inviteNode != nil {
        invite := marshalNodeToInvite(inviteNode.(neo4j.Node))

        emailNode := record.GetByIndex(1)
        if emailNode != nil {
          email := marshalNodeToEmail(emailNode.(neo4j.Node))
          invite.SentTo = &email
        }

        ibiNode := record.GetByIndex(2)
        if ibiNode != nil {
          invitedBy := marshalNodeToHuman(ibiNode.(neo4j.Node))
          invite.InvitedBy = &invitedBy
        }

        invitedNode := record.GetByIndex(3)
        if invitedNode != nil {
          invited := marshalNodeToHuman(invitedNode.(neo4j.Node))
          invite.Invited = &invited
        }
      }

    } else {
      return nil, errors.New("Unable to accept Invite")
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
