package idp

import (
  "errors"
  "strings"
  "fmt"
	"context"
	"database/sql"
	"github.com/google/uuid"
)

func CreateResourceServer(ctx context.Context, tx *sql.Tx, newResourceServer ResourceServer) (resourceServer ResourceServer, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if newResourceServer.Issuer == "" {
    return ResourceServer{}, errors.New("Missing ResourceServer.Issuer")
  }
  params["iss"] = newResourceServer.Issuer

  if newResourceServer.Name == "" {
    return ResourceServer{}, errors.New("Missing ResourceServer.Name")
  }
  params["name"] = newResourceServer.Name

  if newResourceServer.Description == "" {
    return ResourceServer{}, errors.New("Missing ResourceServer.Description")
  }
  params["description"] = newResourceServer.Description

  params["exp"] = newResourceServer.ExpiresAt

  if newResourceServer.Audience == "" {
    return ResourceServer{}, errors.New("Missing ResourceServer.Audience")
  }
  params["aud"] = newResourceServer.Audience

	uuid, err := uuid.NewRandom()
	if err != nil {
		return ResourceServer{}, err
	}

	// TODO SQL
  cypher = fmt.Sprintf(`
    CREATE (rs:ResourceServer:Identity {
      id:randomUUID(),
      iat:datetime().epochSeconds,
      iss:$iss,
      exp:0,
      name:$name,
      description:$description,
      aud:$aud
    })

    WITH rs

    %s

    RETURN rs
  `)

  _, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
    return ResourceServer{}, err
  }

	resourceServers, err := FetchResourceServers(ctx, tx, []ResourceServer{ {Identity: Identity{Id: uuid.String()}} })
  if err != nil {
    return ResourceServer{}, err
  }

  return resourceServers[0], nil
}

func FetchResourceServers(ctx context.Context, tx *sql.Tx, iResourceServers []ResourceServer) (resourceServers []ResourceServer, err error) {
  var rows *sql.Rows
  var cypher string
  var params = make(map[string]interface{})

  cypFilterResourceServers := ""
  if len(iResourceServers) > 0 {
    var ids []string
    for _, rs := range iResourceServers {
      ids = append(ids, rs.Id)
    }
    cypFilterResourceServers = ` AND rs.id in split($ids, ",") `
    params["ids"] = strings.Join(ids, ",")
  }

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (rs:ResourceServer:Identity) WHERE 1=1 %s
    RETURN rs
  `, cypFilterResourceServers)

	rows, err = tx.QueryContext(ctx, cypher, params)
  if err != nil {
    return nil, err
  }

  for rows.Next() {
		rs := marshalRowToResourceServer(rows)
		resourceServers = append(resourceServers, rs)
  }

  return resourceServers, nil
}

func DeleteResourceServer(ctx context.Context, tx *sql.Tx, resourceServerToDelete ResourceServer) (resourceServer ResourceServer, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if resourceServerToDelete.Id == "" {
    return ResourceServer{}, errors.New("Missing ResourceServer.Id")
  }
  params["id"] = resourceServerToDelete.Id

  // Warning: Do not accidentally delete i!
	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH %s(c:ResourceServer:Identity {id:$id})
    DETACH DELETE c
  `)

  if _, err = tx.ExecContext(ctx, cypher, params); err != nil {
    return ResourceServer{}, err
  }

  resourceServer.Id = resourceServerToDelete.Id
  return resourceServer, nil
}
