// Copyright 2021 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package mongo

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	mopts "go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// DevicesCollectionName refers to the collection of stored devices
	DevicesCollectionName = "devices"
)

const defaultTimeout = time.Second * 10

type MongoStoreConfig struct {
	// MongoURL holds the URL to the MongoDB server.
	MongoURL *url.URL
	// TLSConfig holds optional tls configuration options for connecting
	// to the MongoDB server.
	TLSConfig *tls.Config
	// Username holds the user id credential for authenticating with the
	// MongoDB server.
	Username string
	// Password holds the password credential for authenticating with the
	// MongoDB server.
	Password string

	// DbName contains the name of the deviceconfig database.
	DbName string
}

// newClient returns a mongo client
func newClient(ctx context.Context, config MongoStoreConfig) (*mongo.Client, error) {

	clientOptions := mopts.Client()
	if config.MongoURL == nil || config.MongoURL.Scheme == "" {
		return nil, errors.Errorf(
			"Invalid mongoURL %s: missing scheme",
			config.MongoURL.String(),
		)
	}
	clientOptions.ApplyURI(config.MongoURL.String())

	if config.Username != "" {
		credentials := mopts.Credential{
			Username: config.Username,
		}
		if config.Password != "" {
			credentials.Password = config.Password
			credentials.PasswordSet = true
		}
		clientOptions.SetAuth(credentials)
	}

	if config.TLSConfig != nil {
		clientOptions.SetTLSConfig(config.TLSConfig)
	}

	if _, ok := ctx.Deadline(); !ok {
		// Set 10s timeout
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to connect to mongo server")
	}

	// Validate connection
	if err = client.Ping(ctx, nil); err != nil {
		return nil, errors.Wrap(err, "Error reaching mongo server")
	}

	return client, nil
}

// MongoStore is the data storage service
type MongoStore struct {
	// client holds the reference to the client used to communicate with the
	// mongodb server.
	client *mongo.Client

	config MongoStoreConfig
}

// SetupDataStore returns the mongo data store and optionally runs migrations
func NewMongoStore(config MongoStoreConfig) (*MongoStore, error) {
	ctx := context.Background()
	dbClient, err := newClient(ctx, config)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to connect to db: %v", err))
	}
	return &MongoStore{
		client: dbClient,
		config: config,
	}, nil
}

// Ping verifies the connection to the database
func (db *MongoStore) Ping(ctx context.Context) error {
	res := db.client.
		Database(db.config.DbName).
		RunCommand(ctx, bson.M{"ping": 1})
	return res.Err()
}

// Close disconnects the client
func (db *MongoStore) Close(ctx context.Context) error {
	err := db.client.Disconnect(ctx)
	return err
}

//nolint:unused
func (db *MongoStore) DropDatabase(ctx context.Context) error {
	err := db.client.
		Database(db.config.DbName).
		Drop(ctx)
	return err
}
