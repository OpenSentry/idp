package idp

import (
  "strings"
  "errors"
  "fmt"
  "context"
	"database/sql"
	"github.com/google/uuid"
)

func CreateRole(ctx context.Context, tx *sql.Tx, iRole Role) (rRole Role, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if iRole.Issuer == "" {
    return Role{}, errors.New("Missing Role.Issuer")
  }
  params["iss"] = iRole.Issuer

  if iRole.Name == "" {
    return Role{}, errors.New("Missing Role.Name")
  }
  params["name"] = iRole.Name

  if iRole.Description == "" {
    return Role{}, errors.New("Missing Role.Description")
  }
  params["description"] = iRole.Description

	uuid, err := uuid.NewRandom()
	if err != nil {
		return Role{}, err
	}

	// TODO SQL
  cypher = fmt.Sprintf(`
    // Create Role

    CREATE (role:Role:Identity {
      id:randomUUID(),
      iat:datetime().epochSeconds,
      exp:0,
      iss:$iss,
      name:$name,
      description:$description
    })

    RETURN role
  `)

	_, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
    return Role{}, err
  }

	rRoles, err := FetchRoles(ctx, tx, []Role{ {Identity: Identity{Id: uuid.String()}} })

  if err != nil {
    return Role{}, err
  }

  return rRoles[0], nil
}

func FetchRoles(ctx context.Context, tx *sql.Tx, iFilterRoles []Role) (rRoles []Role, err error) {
  var rows *sql.Rows
  var cypher string
  var params = make(map[string]interface{})

  var where1 string
  if len(iFilterRoles) > 0 {
    var filterRoles []string
    for _,e := range iFilterRoles {
      filterRoles = append(filterRoles, e.Id)
    }

    where1 = "and role.id in split($filterRoles, \",\")"
    params["filterRoles"] = strings.Join(filterRoles, ",")
  }

	// TODO SQL
  cypher = fmt.Sprintf(`
    // Fetch roles

    MATCH (role:Role:Identity)
    WHERE 1=1 %s
    RETURN role
  `, where1)

  logCypher(cypher, params)
	rows, err = tx.QueryContext(ctx, cypher, params)
  if err != nil {
    return nil, err
  }
	defer rows.Close()

  for rows.Next() {
		role := marshalRowToRole(rows)
		rRoles = append(rRoles, role)
  }

  return rRoles, nil
}

func DeleteRole(ctx context.Context, tx *sql.Tx, iRole Role) (rRole Role, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if iRole.Id == "" {
    return Role{}, errors.New("Missing Role.Id")
  }
  params["id"] = iRole.Id

  // Warning: Do not accidentally delete i!
	// TODO SQL
  cypher = fmt.Sprintf(`
    // Delete role

    MATCH (role:Role:Identity {id:$id})
    DETACH DELETE role
  `)

  logCypher(cypher, params)
  if _, err = tx.ExecContext(ctx, cypher, params); err != nil {
    return Role{}, err
  }

  rRole.Id = iRole.Id
  return rRole, nil
}
