package idp

import (
  "strings"
  "fmt"
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

func BeginReadTx(driver neo4j.Driver, configurers ...func(*neo4j.TransactionConfig)) (neo4j.Session, neo4j.Transaction, error) {
  session, err := driver.Session(neo4j.AccessModeRead)

  if err != nil {
    return nil, nil, err
  }

  tx, err := session.BeginTransaction(configurers...)

  return session, tx, err
}

func BeginWriteTx(driver neo4j.Driver, configurers ...func(*neo4j.TransactionConfig)) (neo4j.Session, neo4j.Transaction, error) {
  session, err := driver.Session(neo4j.AccessModeWrite)

  if err != nil {
    return nil, nil, err
  }

  tx, err := session.BeginTransaction(configurers...)

  return session, tx, err
}

func logCypher(query string, params map[string]interface{}) {
  for i,e := range params {
    query = strings.Replace(query, "$"+i, "\""+e.(string)+"\"", -1)
  }

  fmt.Printf("\n========== NEO4J DEBUGGING ==========\nCypher: %v", query)
}
