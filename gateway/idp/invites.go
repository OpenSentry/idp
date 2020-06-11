package idp

import (
  "errors"
  "fmt"
  "strings"
	"context"
	"database/sql"
	"github.com/google/uuid"
)

func UpdateInviteSentAt(ctx context.Context, tx *sql.Tx, inviteToUpdate Invite) (invite Invite, err error) {
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

	_, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
    return Invite{}, err
  }

	invites, err := FetchInvites(ctx, tx, []Invite{ inviteToUpdate })
	if err != nil {
		return Invite{}, nil
	}

  return invites[0], nil
}

func CreateInvite(ctx context.Context, tx *sql.Tx, newInvite Invite) (invite Invite, err error) {
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

  cypUsername := ""
  if newInvite.Username != "" {
    params["username"] = newInvite.Username
    cypUsername = `, username:$username`
  }

  params["exp"] = newInvite.ExpiresAt

	uuid, err := uuid.NewRandom()
	if err != nil {
		return Invite{}, err
	}

	// TODO SQL
  cypher = fmt.Sprintf(`
    CREATE (inv:Invite:Identity {id:randomUUID(), email:$email, iat:datetime().epochSeconds, iss:$iss, exp:$exp, sent_at:0, email_confirmed_at:0 %s})

    WITH inv

    OPTIONAL MATCH (d:Invite:Identity) WHERE id(inv) <> id(d) AND d.exp < datetime().epochSeconds DETACH DELETE d

    RETURN inv
  `, cypUsername)

  if _, err = tx.ExecContext(ctx, cypher, params); err != nil {
    return Invite{}, err
  }

	invites, err := FetchInvites(ctx, tx, []Invite{ {Identity: Identity{Id: uuid.String()}} })
	if err != nil {
		return Invite{}, nil
	}

  return invites[0], nil
}

func FetchInvites(ctx context.Context, tx *sql.Tx, iInvites []Invite) (invites []Invite, err error) {
  var rows *sql.Rows
  var cypher string
  var params = make(map[string]interface{})

  cypfilterInvites := ""
  if len(iInvites) > 0 {
    var ids []string
    for _, invite := range iInvites {
      ids = append(ids, invite.Id)
    }
    cypfilterInvites = ` AND inv.id in split($ids, ",") `
    params["ids"] = strings.Join(ids, ",")
  }

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds %s
    RETURN inv
  `, cypfilterInvites)

  if rows, err = tx.QueryContext(ctx, cypher, params); err != nil {
    return nil, err
  }

  for rows.Next() {
		i := marshalRowToInvite(rows)
		invites = append(invites, i)
  }

  return invites, nil
}

func FetchInvitesByEmail(ctx context.Context, tx *sql.Tx, iInvites []Invite) (invites []Invite, err error) {
  var rows *sql.Rows
  var cypher string
  var params = make(map[string]interface{})

  cypfilterInvites := ""
  if len(iInvites) > 0 {
    var emails []string
    for _, invite := range iInvites {
      emails = append(emails, invite.Email)
    }
    cypfilterInvites = ` AND inv.email in split($emails, ",") `
    params["emails"] = strings.Join(emails, ",")
  }

  cypher = fmt.Sprintf(`
    MATCH (inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds %s
    RETURN inv
  `, cypfilterInvites)

  if rows, err = tx.QueryContext(ctx, cypher, params); err != nil {
    return nil, err
  }

  for rows.Next() {
		i := marshalRowToInvite(rows)
		invites = append(invites, i)
  }

  return invites, nil
}

func FetchInvitesByUsername(ctx context.Context, tx *sql.Tx, iInvites []Invite) (invites []Invite, err error) {
  var rows *sql.Rows
  var cypher string
  var params = make(map[string]interface{})

  cypfilterInvites := ""
  if len(iInvites) > 0 {
    var usernames []string
    for _, invite := range iInvites {
      usernames = append(usernames, invite.Username)
    }
    cypfilterInvites = ` AND inv.username in split($usernames, ",") `
    params["usernames"] = strings.Join(usernames, ",")
  }

  cypher = fmt.Sprintf(`
    MATCH (inv:Invite:Identity) WHERE inv.exp > datetime().epochSeconds %s
    RETURN inv
  `, cypfilterInvites)

  if rows, err = tx.QueryContext(ctx, cypher, params); err != nil {
    return nil, err
  }

  for rows.Next() {
		i := marshalRowToInvite(rows)
		invites = append(invites, i)
  }

  return invites, nil
}
