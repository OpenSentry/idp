package idp

import (
  "fmt"
  "strings"
  "context"
	"database/sql"
)

// You should never make these, please specialize with another label, see client.go or human.go
// func CreateIdentities(driver neo4j.Driver, identities []Identity) ([]Identity, error)

func FetchIdentities(ctx context.Context, tx *sql.Tx, iIdentities []Identity) (identities []Identity, err error) {
  var rows *sql.Rows
  var cypher string
  var params = make(map[string]interface{})

  cypFilterIdentities := ""
  if len(iIdentities) > 0 {
    var ids []string
    for _, identity := range iIdentities {
      ids = append(ids, identity.Id)
    }
    cypFilterIdentities = ` AND i.id in split($ids, ",") `
    params["ids"] = strings.Join(ids, ",")
  }

  cypher = fmt.Sprintf(`
    MATCH (i:Identity) WHERE 1=1 %s RETURN i
  `, cypFilterIdentities)

  if rows, err = tx.QueryContext(ctx, cypher, params); err != nil {
    return nil, err
  }

  for rows.Next() {
		i := marshalRowToIdentity(rows)
		identities = append(identities, i)
  }

  return identities, nil
}

func SearchIdentities(ctx context.Context, tx *sql.Tx, iSearch string) (identities []Identity, err error) {
  var rows *sql.Rows
  var cypher string
  var params = make(map[string]interface{})

  cypFilterIdentities := ""
  if iSearch != "" {
    cypFilterIdentities = ` AND ( i.name =~ $search or i.email =~ $search)`
    params["search"] = fmt.Sprintf(`(?i).*%s.*`, iSearch)
  }

  cypher = fmt.Sprintf(`
    MATCH (i:Identity) WHERE 1=1 %s RETURN i
  `, cypFilterIdentities)

  if rows, err = tx.QueryContext(ctx, cypher, params); err != nil {
    return nil, err
  }

  for rows.Next() {
		i := marshalRowToIdentity(rows)
		identities = append(identities, i)
  }

  return identities, nil
}
