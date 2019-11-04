package idp

import (
  "strings"
  "errors"
  "fmt"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

func CreateRole(tx neo4j.Transaction, iRole Role, requestor Identity) (rRole Role, err error) {
  var result neo4j.Result
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

  cypher = fmt.Sprintf(`
    // Create Role

    CREATE (role:Role:Identity {
      id:randomUUID(),
      iat:datetime().epochSeconds,
      exp:0,
      iss:$iss,
      name:$name,
      description:$description,
    })

    RETURN role
  `)

  logCypher(cypher, params)
  if result, err = tx.Run(cypher, params); err != nil {
    return Role{}, err
  }

  if result.Next() {
    record      := result.Record()
    roleNode    := record.GetByIndex(0)

    if roleNode != nil {
      rRole = marshalNodeToRole(roleNode.(neo4j.Node))
    }
  } else {
    return Role{}, errors.New("Unable to create Role")
  }

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return Role{}, err
  }

  return rRole, nil
}

func FetchRoles(tx neo4j.Transaction, iFilterRoles []Role, iRequest Identity) (rRoles []Role, err error) {
  var result neo4j.Result
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

  cypher = fmt.Sprintf(`
    // Fetch roles

    MATCH (role:ResourceServer:Identity)
    WHERE 1=1 %s
    RETURN rs
  `, where1)

  logCypher(cypher, params)
  if result, err = tx.Run(cypher, params); err != nil {
    return nil, err
  }

  for result.Next() {
    record      := result.Record()
    roleNode    := record.GetByIndex(0)

    if roleNode != nil {
      role := marshalNodeToRole(roleNode.(neo4j.Node))
      rRoles = append(rRoles, role)
    }
  }

  // Check if we encountered any error during record streaming
  if err = result.Err(); err != nil {
    return nil, err
  }

  return rRoles, nil
}

func DeleteRole(tx neo4j.Transaction, iRole Role, requestor Identity) (rRole Role, err error) {
  var cypher string
  var params = make(map[string]interface{})

  if iRole.Id == "" {
    return Role{}, errors.New("Missing Role.Id")
  }
  params["id"] = iRole.Id

  // Warning: Do not accidentally delete i!
  cypher = fmt.Sprintf(`
    // Delete role

    MATCH (role:Role:Identity {id:$id})
    DETACH DELETE role
  `)

  logCypher(cypher, params)
  if _, err = tx.Run(cypher, params); err != nil {
    return Role{}, err
  }

  rRole.Id = iRole.Id
  return rRole, nil
}
