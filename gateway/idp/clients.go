package idp

import (
  "errors"
  "strings"
  "fmt"
  "context"
	"database/sql"
	"github.com/google/uuid"
)

func CreateClient(ctx context.Context, tx *sql.Tx, newClient Client) (client Client, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if newClient.Issuer == "" {
    return Client{}, errors.New("Missing Client.Issuer")
  }
  params["iss"] = newClient.Issuer

  params["exp"] = newClient.ExpiresAt

  if newClient.Name == "" {
    return Client{}, errors.New("Missing Client.Name")
  }
  params["name"] = newClient.Name

  if newClient.Description == "" {
    return Client{}, errors.New("Missing Client.Description")
  }
  params["description"] = newClient.Description

  params["grantTypes"] = []string{}
  params["responseTypes"] = []string{}
  params["redirectUris"] = []string{}
  params["postLogoutRedirectUris"] = []string{}
  params["audiences"] = []string{}
  params["tokenEndpointAuthMethod"] = ""

	uuid, err := uuid.NewRandom()
	if err != nil {
		return Client{}, err
	}

  if len(newClient.GrantTypes) > 0 {
    params["grantTypes"] = newClient.GrantTypes
  }
  if len(newClient.ResponseTypes) > 0 {
    params["responseTypes"] = newClient.ResponseTypes
  }
  if len(newClient.RedirectUris) > 0 {
    params["redirectUris"] = newClient.RedirectUris
  }
  if len(newClient.PostLogoutRedirectUris) > 0 {
    params["postLogoutRedirectUris"] = newClient.PostLogoutRedirectUris
  }
  if len(newClient.Audiences) > 0 {
    params["audiences"] = newClient.Audiences
  }
  if newClient.TokenEndpointAuthMethod != "" {
    params["tokenEndpointAuthMethod"] = newClient.TokenEndpointAuthMethod
  }

  cypClientSecret := ""
  if newClient.Secret != "" {
    params["client_secret"] = newClient.Secret
    cypClientSecret = `secret:$client_secret,`
  }

	// TODO SQL
  cypher = fmt.Sprintf(`
    CREATE (c:Client:Identity {
      id:randomUUID(),
      iat:datetime().epochSeconds,
      iss:$iss,
      exp:0,
      %s
      name:$name,
      description:$description,
      grant_types:$grantTypes,
      response_types:$responseTypes,
      redirect_uris:$redirectUris,
      post_logout_redirect_uris:$postLogoutRedirectUris,
      token_endpoint_auth_method:$tokenEndpointAuthMethod,
      audiences:$audiences
    })

    WITH c

    RETURN c
  `, cypClientSecret)

	_, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
    return Client{}, err
  }

	clients, err := FetchClients(ctx, tx, []Client{ { Identity: Identity{Id: uuid.String()} } })
  if err != nil {
    return Client{}, err
  }

  return clients[0], nil
}

func FetchClients(ctx context.Context, tx *sql.Tx, iClients []Client) (clients []Client, err error) {
  var rows *sql.Rows
  var cypher string
  var params = make(map[string]interface{})

  cypFilterClients := ""
  if len(iClients) > 0 {
    var ids []string
    for _, client := range iClients {
      ids = append(ids, client.Id)
    }
    cypFilterClients = ` AND c.id in split($ids, ",") `
    params["ids"] = strings.Join(ids, ",")
  }

	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (c:Client:Identity) WHERE 1=1 %s
    RETURN c
  `, cypFilterClients)

	rows, err = tx.QueryContext(ctx, cypher, params)
  if err != nil {
    return nil, err
  }

  for rows.Next() {
		client := marshalRowToClient(rows)
		clients = append(clients, client)
  }

  return clients, nil
}

func DeleteClient(ctx context.Context, tx *sql.Tx, clientToDelete Client) (client Client, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if clientToDelete.Id == "" {
    return Client{}, errors.New("Missing Client.Id")
  }
  params["id"] = clientToDelete.Id

  // Warning: Do not accidentally delete i!
	// TODO SQL
  cypher = fmt.Sprintf(`
    MATCH (c:Client:Identity {id:$id})
    DETACH DELETE c
  `)

  _, err = tx.ExecContext(ctx, cypher, params)
  if err != nil {
    return Client{}, err
  }

  client.Id = clientToDelete.Id
  return client, nil
}
