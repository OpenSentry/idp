package migration

import (
  "fmt"
  "github.com/neo4j/neo4j-go-driver/neo4j"
  "io/ioutil"
  "strings"
  "golang-idp-be/config"
)

func loadMigrationsFromFile(path string) []string {
  fmt.Println("Loading migrations from " + path)
  dat, err := ioutil.ReadFile(path)
  if err != nil {
    fmt.Println(err)
    panic("Failed to retrieve migration file from " + path)
  }

  return strings.Split(string(dat), ";")
}

func applyMigrations(migrations []string, session neo4j.Session) (error) {
  _, err := session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {

    for _, query := range migrations {
      if len(strings.TrimSpace(query)) == 0 {
        // nothing to run, caused by split on last ;
        continue
      }

      fmt.Println("Applying query: " + query)

      if _, err := tx.Run(query, nil); err != nil {
        fmt.Println(err)
        return nil, err
      }
    }

    return nil, nil
  })

  return err
}

func Migrate(driver neo4j.Driver) {
  var err error
  var session neo4j.Session

  session, err = driver.Session(neo4j.AccessModeWrite);
  if err != nil {
    fmt.Println(err)
    panic("Unable to obtain neo4j write session")
  }
  defer session.Close()

  schemaMigrations := loadMigrationsFromFile(config.GetString("migration.schema.path"))

  err = applyMigrations(schemaMigrations, session)
  if err != nil {
    fmt.Println(err)
    panic("Errors occured while applying schema migrations")
  }

  fmt.Println("Schema migrations applied and commited")

  dataMigrations := loadMigrationsFromFile(config.GetString("migration.data.path"))

  err = applyMigrations(dataMigrations, session)
  if err != nil {
    fmt.Println(err)
    panic("Errors occured while applying data migrations")
  }

  fmt.Println("Data migrations applied and commited")
}
