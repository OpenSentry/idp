package idp

import (
  "errors"
  "fmt"
  "strings"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

func UpdateInviteSentAt(tx neo4j.Transaction, updatedBy *Identity, inviteToUpdate Invite) (invite Invite, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  if inviteToUpdate.Id == "" {
    return Invite{}, errors.New("Missing Invite.Id")
  }
  params["id"] = inviteToUpdate.Id

  cypher = fmt.Sprintf(`
    MATCH (inv:Invite:Identity {id:$id}) WHERE inv.exp > datetime().epochSeconds
    SET inv.sent_at = datetime().epochSeconds

    WITH inv

    OPTIONAL MATCH (d:Invite:Identity) WHERE id(inv) <> id(d) AND d.exp < datetime().epochSeconds DETACH DELETE d

    RETURN inv
  `)

  if result, err = tx.Run(cypher, params); err != nil {
    return Invite{}, err
  }

  if result.Next() {
    record        := result.Record()
    inviteNode    := record.GetByIndex(0)

    if inviteNode != nil {
      invite = marshalNodeToInvite(inviteNode.(neo4j.Node))
    }
  } else {
    return Invite{}, errors.New("Unable to update Invite")
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Invite{}, err
  }

  return invite, nil
}

func CreateInvite(tx neo4j.Transaction, invitedBy *Identity, newInvite Invite) (invite Invite, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  if newInvite.Email == "" {
    return Invite{}, errors.New("Missing Invite.Email")
  }
  params["email"] = newInvite.Email

  if newInvite.Issuer == "" {
    return Invite{}, errors.New("Missing Invite.Issuer")
  }
  params["iss"] = newInvite.Issuer

  params["exp"] = newInvite.ExpiresAt

  cypUsername := ""
  if newInvite.Username != "" {
    params["username"] = invite.Username
    cypUsername = ", username:$username"
  }

  cypInvites := ""
  if invitedBy != nil {
    params["invited_by"] = invitedBy.Id
    cypInvites = `MATCH (i:Identity {id:$invited_by}) MERGE (i)-[:INVITES]->(inv)`
  }

  cypher = fmt.Sprintf(`
    CREATE (inv:Invite:Identity {id:randomUUID(), email:$email, iat:datetime().epochSeconds, iss:$iss, exp:$exp, sent_at:0, email_confirmed_at:0 %s})

    WITH inv

    %s

    WITH inv

    OPTIONAL MATCH (d:Invite:Identity) WHERE id(inv) <> id(d) AND d.exp < datetime().epochSeconds DETACH DELETE d

    RETURN inv
  `, cypUsername, cypInvites)

  if result, err = tx.Run(cypher, params); err != nil {
    return Invite{}, err
  }

  if result.Next() {
    record        := result.Record()
    inviteNode    := record.GetByIndex(0)

    if inviteNode != nil {
      invite = marshalNodeToInvite(inviteNode.(neo4j.Node))
    }
  } else {
    return Invite{}, errors.New("Unable to create Invite")
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Invite{}, err
  }

  return invite, nil
}

func FetchInvites(tx neo4j.Transaction, invitedBy *Identity, iInvites []Invite) (invites []Invite, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  var cypInvites string
  if invitedBy != nil {
    cypInvites = `(i:Identity {id:$invited_by})-[:INVITES]->`
    params["invited_by"] = invitedBy.Id
  }

  cypfilterInvites := ""
  if len(iInvites) > 0 {
    var ids []string
    for _, invite := range iInvites {
      ids = append(ids, invite.Id)
    }
    cypfilterInvites = ` WHERE inv.id in split($ids, ",") `
    params["ids"] = strings.Join(ids, ",")
  }

  cypher = fmt.Sprintf(`
    MATCH %s(inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds %s
    RETURN inv
  `, cypInvites, cypfilterInvites)

  if result, err = tx.Run(cypher, params); err != nil {
    return nil, err
  }

  for result.Next() {
    record        := result.Record()
    inviteNode    := record.GetByIndex(0)

    if inviteNode != nil {
      i := marshalNodeToInvite(inviteNode.(neo4j.Node))
      invites = append(invites, i)
    }
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return nil, err
  }

  return invites, nil
}

func FetchInvitesByEmail(tx neo4j.Transaction, invitedBy *Identity, iInvites []Invite) (invites []Invite, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  var cypInvites string
  if invitedBy != nil {
    cypInvites = `(i:Identity {id:$invited_by})-[:INVITES]->`
    params["invited_by"] = invitedBy.Id
  }

  cypfilterInvites := ""
  if len(iInvites) > 0 {
    var emails []string
    for _, invite := range iInvites {
      emails = append(emails, invite.Email)
    }
    cypfilterInvites = ` WHERE inv.email in split($emails, ",") `
    params["emails"] = emails
  }

  cypher = fmt.Sprintf(`
    MATCH %s(inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds %s
    RETURN inv
  `, cypInvites, cypfilterInvites)

  if result, err = tx.Run(cypher, params); err != nil {
    return nil, err
  }

  for result.Next() {
    record        := result.Record()
    inviteNode    := record.GetByIndex(0)

    if inviteNode != nil {
      i := marshalNodeToInvite(inviteNode.(neo4j.Node))
      invites = append(invites, i)
    }
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return nil, err
  }

  return invites, nil
}

func FetchInvitesByUsername(tx neo4j.Transaction, invitedBy *Identity, iInvites []Invite) (invites []Invite, err error) {
  var result neo4j.Result
  var cypher string
  var params = make(map[string]interface{})

  var cypInvites string
  if invitedBy != nil {
    cypInvites = `(i:Identity {id:$invited_by})-[:INVITES]->`
    params["invited_by"] = invitedBy.Id
  }

  cypfilterInvites := ""
  if len(iInvites) > 0 {
    var usernames []string
    for _, invite := range iInvites {
      usernames = append(usernames, invite.Username)
    }
    cypfilterInvites = ` WHERE inv.username in split($usernames, ",") `
    params["usernames"] = usernames
  }

  cypher = fmt.Sprintf(`
    MATCH %s(inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds %s
    RETURN inv
  `, cypInvites, cypfilterInvites)

  if result, err = tx.Run(cypher, params); err != nil {
    return nil, err
  }

  for result.Next() {
    record        := result.Record()
    inviteNode    := record.GetByIndex(0)

    if inviteNode != nil {
      i := marshalNodeToInvite(inviteNode.(neo4j.Node))
      invites = append(invites, i)
    }
  }

  logCypher(cypher, params)

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return nil, err
  }

  return invites, nil
}
