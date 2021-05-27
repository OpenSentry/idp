package idp

import (
	"fmt"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	"strings"
)

// You should never make these, please specialize with another label, see client.go or human.go
// func CreateIdentities(driver neo4j.Driver, identities []Identity) ([]Identity, error)

func FetchIdentities(tx neo4j.Transaction, iIdentities []Identity) (identities []Identity, err error) {
	var result neo4j.Result
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

	logCypher(cypher, params)
	if result, err = tx.Run(cypher, params); err != nil {
		return nil, err
	}

	for result.Next() {
		record := result.Record()
		identityNode := record.GetByIndex(0)

		if identityNode != nil {
			i := marshalNodeToIdentity(identityNode.(neo4j.Node))

			identities = append(identities, i)
		}
	}

	// Check if we encountered any error during record streaming
	if err = result.Err(); err != nil {
		return nil, err
	}

	return identities, nil
}

func SearchIdentities(tx neo4j.Transaction, iSearch string) (identities []Identity, err error) {
	var result neo4j.Result
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

	if result, err = tx.Run(cypher, params); err != nil {
		return nil, err
	}

	logCypher(cypher, params)
	for result.Next() {
		record := result.Record()
		identityNode := record.GetByIndex(0)

		if identityNode != nil {
			i := marshalNodeToIdentity(identityNode.(neo4j.Node))

			identities = append(identities, i)
		}
	}

	// Check if we encountered any error during record streaming
	if err = result.Err(); err != nil {
		return nil, err
	}

	return identities, nil
}
