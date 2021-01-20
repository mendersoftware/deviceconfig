// Copyright 2020 Northern.tech AS
//
//    All Rights Reserved

package mongo

import (
	"os"
	"strings"
	"testing"

	mtesting "github.com/mendersoftware/go-lib-micro/mongo/testing"
	mstore "github.com/mendersoftware/go-lib-micro/store"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/net/context"
)

// TestDB is a stripped down TestDBRunner interface.
type TestDB interface {
	Client() *mongo.Client
}

var db TestDB

func TestMain(m *testing.M) {
	status := mtesting.WithDB(func(d mtesting.TestDBRunner) int {
		db = d
		defer d.Client().Disconnect(context.Background())
		ret := m.Run()
		return ret
	})
	os.Exit(status)
}

var dbNameReplacer = strings.NewReplacer(
	`/`, ``, `\`, ``, `.`, ``, ` `, ``,
	`"`, ``, `$`, ``, `*`, ``, `<`, ``,
	`>`, ``, `:`, ``, `|`, ``, `?`, ``,
)

// legalizeDbName ensures the database name does not contain illegal characters
// and that the length does not exceed the maximum 64 characters.
func legalizeDbName(testName string) string {
	dbName := dbNameReplacer.Replace(testName)
	if len(dbName) >= 64 {
		dbName = dbName[len(dbName)-64:]
	}
	return dbName
}

// GetTestDataStore creates a new DataStoreMongo with the database name
// set to the test name (is safe to call inside subtests, but be aware that
// t.Name() is different from inside and outside of t.Run scope).
// Make sure you always defer DataStore.DropDatabase inside tests to free
// up storage.
func GetTestDataStore(t *testing.T) *MongoStore {
	client := db.Client()
	dbName := legalizeDbName(t.Name())
	return &MongoStore{
		client: client,
		config: MongoStoreConfig{
			DbName: dbName,
		},
	}
}

// GetTestDatabase as function above returns the test-local database.
func GetTestDatabase(ctx context.Context, t *testing.T) *mongo.Database {
	dbName := legalizeDbName(t.Name())
	return db.Client().Database(mstore.DbFromContext(ctx, dbName))
}
