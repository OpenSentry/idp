package idp

import (
	"fmt"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	"strconv"
	"strings"
)

func BeginReadTx(driver neo4j.Driver, configurers ...func(*neo4j.TransactionConfig)) (neo4j.Session, neo4j.Transaction, error) {
	session, err := driver.Session(neo4j.AccessModeRead)

	if err != nil {
		return nil, nil, err
	}

	tx, err := session.BeginTransaction(configurers...)

	if err != nil {
		session.Close()
	}

	return session, tx, err
}

func BeginWriteTx(driver neo4j.Driver, configurers ...func(*neo4j.TransactionConfig)) (neo4j.Session, neo4j.Transaction, error) {
	session, err := driver.Session(neo4j.AccessModeWrite)

	if err != nil {
		return nil, nil, err
	}

	tx, err := session.BeginTransaction(configurers...)

	if err != nil {
		session.Close()
	}

	return session, tx, err
}

func logCypher(query string, params map[string]interface{}) {
	for i, e := range params {

		switch t := e.(type) {
		case bool:
			query = strings.Replace(query, "$"+i, "\""+strconv.FormatBool(e.(bool))+"\"", -1)
		case int:
			query = strings.Replace(query, "$"+i, "\""+strconv.Itoa(e.(int))+"\"", -1)
		case int64:
			query = strings.Replace(query, "$"+i, "\""+strconv.FormatInt(e.(int64), 10)+"\"", -1)
		case string:
			query = strings.Replace(query, "$"+i, "\""+e.(string)+"\"", -1)
		case []string:
			query = strings.Replace(query, "$"+i, "["+strings.Join(e.([]string), ",")+"]", -1)
		default:
			panic(fmt.Sprintf("Unsupported type %T", t))
		}

	}

	fmt.Printf("\n========== NEO4J DEBUGGING ==========\nCypher: %v", query)
}
