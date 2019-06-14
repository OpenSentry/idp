package db

import (
  "github.com/neo4j/neo4j-go-driver/neo4j"
)

func HelloWorld(uri, username, password string) (string, error) {
    var (
        err      error
        driver   neo4j.Driver
        session  neo4j.Session
        result   neo4j.Result
        greeting interface{}
    )

    driver, err = neo4j.NewDriver(uri, neo4j.BasicAuth(username, password, ""))
    if err != nil {
        return "test 1", err
    }
    defer driver.Close()

    session, err = driver.Session(neo4j.AccessModeWrite)
    if err != nil {
        return "test 2", err
    }
    defer session.Close()

    greeting, err = session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
        result, err = transaction.Run(
            "CREATE (a:Greeting) SET a.message = $message RETURN a.message + ', from node ' + id(a)",
            map[string]interface{}{"message": "hello, world"})
        if err != nil {
            return "test 3", err
        }

        if result.Next() {
            return result.Record().GetByIndex(0), nil
        }

        return "test 4", result.Err()
    })

    if err != nil {
        return "test 5 ", err
    }

    return greeting.(string), nil
}
