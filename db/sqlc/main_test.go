package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/techschool/simplebank/util"
)

var testQueries *Queries
var testDB *sql.DB

// go test -race ./...
func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../..", "test")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	testContainer, err := NewTestDatabase()
	if err != nil {
		log.Fatal("cannot instantiate test database:", err)
	}
	defer testContainer.Close()
	dbURL, err := testContainer.ConnectionString()
	if err != nil {
		log.Fatal("cannot get test database URL:", dbURL)
	}
	println("Test database URL: ", dbURL)

	testDB, err = sql.Open(config.DBDriver, dbURL)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	RunDBMigration(config.MigrationURL, dbURL)
	testQueries = New(testDB)

	os.Exit(m.Run())
}
