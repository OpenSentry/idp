package idp

import (
	"errors"
	"fmt"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	"strings"
)

func CreateResourceServer(tx neo4j.Transaction, managedBy *Identity, newResourceServer ResourceServer) (resourceServer ResourceServer, err error) {
	var result neo4j.Result
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

	cypManages := ""
	if managedBy != nil {
		params["managed_by"] = managedBy.Id
		cypManages = `MATCH (i:Identity {id:$managed_by}) MERGE (i)-[:MANAGES]->(rs)`
	}

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
  `, cypManages)

	if result, err = tx.Run(cypher, params); err != nil {
		return ResourceServer{}, err
	}

	if result.Next() {
		record := result.Record()
		resourceServerNode := record.GetByIndex(0)

		if resourceServerNode != nil {
			resourceServer = marshalNodeToResourceServer(resourceServerNode.(neo4j.Node))
		}
	} else {
		return ResourceServer{}, errors.New("Unable to create ResourceServer")
	}

	logCypher(cypher, params)

	// Check if we encountered any error during record streaming
	if err = result.Err(); err != nil {
		return ResourceServer{}, err
	}

	return resourceServer, nil
}

func FetchResourceServers(tx neo4j.Transaction, managedBy *Identity, iResourceServers []ResourceServer) (resourceServers []ResourceServer, err error) {
	var result neo4j.Result
	var cypher string
	var params = make(map[string]interface{})

	var cypManages string
	if managedBy != nil {
		cypManages = `(i:Identity {id:$managed_by})-[:MANAGES]->`
		params["managed_by"] = managedBy.Id
	}

	cypFilterResourceServers := ""
	if len(iResourceServers) > 0 {
		var ids []string
		for _, rs := range iResourceServers {
			ids = append(ids, rs.Id)
		}
		cypFilterResourceServers = ` AND rs.id in split($ids, ",") `
		params["ids"] = strings.Join(ids, ",")
	}

	cypher = fmt.Sprintf(`
    MATCH %s(rs:ResourceServer:Identity) WHERE 1=1 %s
    RETURN rs
  `, cypManages, cypFilterResourceServers)

	if result, err = tx.Run(cypher, params); err != nil {
		return nil, err
	}

	for result.Next() {
		record := result.Record()
		resourceServerNode := record.GetByIndex(0)

		if resourceServerNode != nil {
			rs := marshalNodeToResourceServer(resourceServerNode.(neo4j.Node))
			resourceServers = append(resourceServers, rs)
		}
	}

	logCypher(cypher, params)

	// Check if we encountered any error during record streaming
	if err = result.Err(); err != nil {
		return nil, err
	}

	return resourceServers, nil
}

func DeleteResourceServer(tx neo4j.Transaction, managedBy *Identity, resourceServerToDelete ResourceServer) (resourceServer ResourceServer, err error) {
	var result neo4j.Result
	var cypher string
	var params = make(map[string]interface{})

	if resourceServerToDelete.Id == "" {
		return ResourceServer{}, errors.New("Missing ResourceServer.Id")
	}
	params["id"] = resourceServerToDelete.Id

	var cypManages string
	if managedBy != nil {
		cypManages = `(i:Identity {id:$managed_by})-[:MANAGES]->`
		params["managed_by"] = managedBy.Id
	}

	params["id"] = resourceServerToDelete.Id

	// Warning: Do not accidentally delete i!
	cypher = fmt.Sprintf(`
    MATCH %s(c:ResourceServer:Identity {id:$id})
    DETACH DELETE c
  `, cypManages)

	if result, err = tx.Run(cypher, params); err != nil {
		return ResourceServer{}, err
	}

	result.Next()

	logCypher(cypher, params)

	// Check if we encountered any error during record streaming
	if err = result.Err(); err != nil {
		return ResourceServer{}, err
	}

	resourceServer.Id = resourceServerToDelete.Id
	return resourceServer, nil
}
