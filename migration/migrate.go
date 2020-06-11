package migration

import (
  "io/ioutil"
  "strings"
  "context"
  "fmt"
  "database/sql"

  "github.com/opensentry/idp/config"
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

func applyMigrations(ctx context.Context, migrations []string, tx *sql.Tx) (error) {

	for _, query := range migrations {
		if len(strings.TrimSpace(query)) == 0 {
			// nothing to run, caused by split on last ;
			continue
		}

		fmt.Println("Applying query: " + query)

		if _, err := tx.ExecContext(ctx, query, nil); err != nil {
			fmt.Println(err)
			return err
		}
	}

  return nil
}

func Migrate(driver *sql.DB) {
  var err error

	ctx := context.TODO()

	tx, err := driver.BeginTx(ctx, nil);
  if err != nil {
    fmt.Println(err)
    panic("Unable to obtain transaction")
  }

  schemaMigrations := loadMigrationsFromFile(config.GetString("migration.schema.path"))

  err = applyMigrations(ctx, schemaMigrations, tx)
  if err != nil {
    fmt.Println(err)
    panic("Errors occured while applying schema migrations")
  }

  fmt.Println("Schema migrations applied and commited")

  dataMigrations := loadMigrationsFromFile(config.GetString("migration.data.path"))

  err = applyMigrations(ctx, dataMigrations, tx)
  if err != nil {
    fmt.Println(err)
    panic("Errors occured while applying data migrations")
  }

  fmt.Println("Data migrations applied and commited")
}
