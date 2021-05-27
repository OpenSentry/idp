package idp

import (
	"errors"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	"strings"
)

func CreateClient(tx neo4j.Transaction, managedBy *Identity, newClient Client) (client Client, err error) {
	var result neo4j.Result
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

	cypManages := ""
	if managedBy != nil {
		params["managed_by"] = managedBy.Id
		cypManages = `MATCH (i:Identity {id:$managed_by}) MERGE (i)-[:MANAGES]->(c)`
	}

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

    %s

    RETURN c
  `, cypClientSecret, cypManages)

	logCypher(cypher, params)

	if result, err = tx.Run(cypher, params); err != nil {
		return Client{}, err
	}

	if result.Next() {
		record := result.Record()
		clientNode := record.GetByIndex(0)

		if clientNode != nil {
			client = marshalNodeToClient(clientNode.(neo4j.Node))
		}
	} else {
		return Client{}, errors.New("Unable to create Client")
	}

	// Check if we encountered any error during record streaming
	if err = result.Err(); err != nil {
		return Client{}, err
	}

	return client, nil
}

func FetchClients(tx neo4j.Transaction, managedBy *Identity, iClients []Client) (clients []Client, err error) {
	var result neo4j.Result
	var cypher string
	var params = make(map[string]interface{})

	var cypManages string
	if managedBy != nil {
		cypManages = `(i:Identity {id:$managed_by})-[:MANAGES]->`
		params["managed_by"] = managedBy.Id
	}

	cypFilterClients := ""
	if len(iClients) > 0 {
		var ids []string
		for _, client := range iClients {
			ids = append(ids, client.Id)
		}
		cypFilterClients = ` AND c.id in split($ids, ",") `
		params["ids"] = strings.Join(ids, ",")
	}

	cypher = fmt.Sprintf(`
    MATCH %s(c:Client:Identity) WHERE 1=1 %s
    RETURN c
  `, cypManages, cypFilterClients)

	if result, err = tx.Run(cypher, params); err != nil {
		return nil, err
	}

	for result.Next() {
		record := result.Record()
		clientNode := record.GetByIndex(0)

		if clientNode != nil {
			client := marshalNodeToClient(clientNode.(neo4j.Node))
			clients = append(clients, client)
		}
	}

	logCypher(cypher, params)

	// Check if we encountered any error during record streaming
	if err = result.Err(); err != nil {
		return nil, err
	}

	return clients, nil
}

func DeleteClient(tx neo4j.Transaction, managedBy *Identity, clientToDelete Client) (client Client, err error) {
	var result neo4j.Result
	var cypher string
	var params = make(map[string]interface{})

	if clientToDelete.Id == "" {
		return Client{}, errors.New("Missing Client.Id")
	}
	params["id"] = clientToDelete.Id

	var cypManages string
	if managedBy != nil {
		cypManages = `(i:Identity {id:$managed_by})-[:MANAGES]->`
		params["managed_by"] = managedBy.Id
	}

	params["id"] = clientToDelete.Id

	// Warning: Do not accidentally delete i!
	cypher = fmt.Sprintf(`
    MATCH %s(c:Client:Identity {id:$id})
    DETACH DELETE c
  `, cypManages)

	if result, err = tx.Run(cypher, params); err != nil {
		return Client{}, err
	}

	result.Next()

	logCypher(cypher, params)

	// Check if we encountered any error during record streaming
	if err = result.Err(); err != nil {
		return Client{}, err
	}

	client.Id = clientToDelete.Id
	return client, nil
}
