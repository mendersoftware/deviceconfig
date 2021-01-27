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
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mendersoftware/deviceconfig/model"
	"github.com/mendersoftware/deviceconfig/store"
	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPing(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping TestPing in short mode.")
	}
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
	defer cancel()

	ds := GetTestDataStore(t)
	err := ds.Ping(ctx)
	assert.NoError(t, err)
}

func TestNewMongoStore(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name string
		CTX  context.Context

		Config MongoStoreConfig

		Error error
	}{{
		Name: "ok",

		Config: MongoStoreConfig{
			DbName: t.Name(),
			MongoURL: func() *url.URL {
				uri, _ := url.Parse(db.URL())
				return uri
			}(),
		},
	}, {
		Name: "error, bad uri scheme",

		Config: MongoStoreConfig{
			DbName: t.Name(),
			MongoURL: func() *url.URL {
				uri, _ := url.Parse(db.URL())
				uri.Scheme = "notMongo"
				return uri
			}(),
		},
		Error: errors.New("mongo: failed to connect with server"),
	}, {
		Name: "error, wrong username/password",

		Config: MongoStoreConfig{
			DbName: t.Name(),
			MongoURL: func() *url.URL {
				uri, _ := url.Parse(db.URL())
				return uri
			}(),
			Username: "user",
			Password: "password",
		},
		Error: errors.New("mongo: error reaching mongo server"),
	}, {
		Name: "error, wrong username/password",

		Config: MongoStoreConfig{
			DbName: t.Name(),
			MongoURL: func() *url.URL {
				uri, _ := url.Parse(db.URL())
				return uri
			}(),
			Username: "user",
			Password: "password",
		},
		Error: errors.New("^mongo: error reaching mongo server: "),
	}, {
		Name: "error, missing url",

		Config: MongoStoreConfig{
			DbName: t.Name(),
		},
		Error: errors.New("^mongo: missing URL"),
	}, {
		Name: "error, context canceled",
		CTX: func() context.Context {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx
		}(),

		Config: MongoStoreConfig{
			DbName: t.Name(),
			MongoURL: func() *url.URL {
				uri, _ := url.Parse(db.URL())
				return uri
			}(),
			TLSConfig: &tls.Config{},
		},
		Error: errors.New("^mongo: error reaching mongo server: " +
			context.Canceled.Error(),
		),
	}}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			if tc.CTX == nil {
				ctx, cancel := context.WithTimeout(
					context.Background(),
					time.Second*5,
				)
				tc.CTX = ctx
				defer cancel()
			}
			ds, err := NewMongoStore(tc.CTX, tc.Config)
			if tc.Error != nil {
				if assert.Error(t, err) {
					assert.Regexp(t, tc.Error.Error(), err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ds)
			}
			if ds != nil {
				ds.Close(tc.CTX)
			}
		})
	}
}

func TestInsertDevice(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name string

		CTX     context.Context
		Devices []model.Device

		Error error
	}{{
		Name: "ok",

		Devices: []model.Device{{
			ID:        uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
			UpdatedTS: time.Now(),
		}},
	}, {
		Name: "error, invalid document",

		Devices: []model.Device{{
			UpdatedTS: time.Now(),
		}},
		Error: errors.New(`^invalid device object: id: cannot be blank.$`),
	}, {
		Name: "error, context canceled",
		CTX: func() context.Context {
			ctx, cancel := context.WithCancel(context.TODO())
			cancel()
			return ctx
		}(),

		Devices: []model.Device{{
			ID:        uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
			UpdatedTS: time.Now(),
		}},
		Error: errors.New(
			`mongo: failed to store device configuration: .*` +
				context.Canceled.Error() + `$`,
		),
	}, {
		Name: "error, duplicate key",

		Devices: []model.Device{{
			ID:        uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
			UpdatedTS: time.Now(),
		}, {
			ID:        uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
			UpdatedTS: time.Now(),
		}},
		Error: store.ErrDeviceAlreadyExists,
	}}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			var err error

			ds := GetTestDataStore(t)
			if tc.CTX == nil {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
				defer cancel()
				tc.CTX = ctx
				defer ds.DropDatabase(tc.CTX)
			}
			for _, dev := range tc.Devices {
				err = ds.InsertDevice(tc.CTX, dev)
				if err != nil {
					break
				}
			}
			if tc.Error != nil {
				if assert.Error(t, err) {
					t.Logf("%T", errors.Cause(err))
					assert.Regexp(t, tc.Error.Error(), err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeleteDevice(t *testing.T) {
	t.Parallel()

	var testDevice = model.Device{
		ID:        uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
		UpdatedTS: time.Now(),
	}

	testCases := []struct {
		Name string

		CTX context.Context
		ID  uuid.UUID

		Error error
	}{{
		Name: "ok",

		ID: testDevice.ID,
	}, {
		Name: "ok, tenant",

		ID: testDevice.ID,

		CTX: identity.WithContext(context.Background(),
			&identity.Identity{
				Tenant: "123456789012345678901234",
			},
		),
	}, {

		Name: "error, device does not exist",

		Error: errors.New(`^mongo: ` + store.ErrDeviceNoExist.Error()),
	}, {
		Name: "error, context canceled",
		ID:   testDevice.ID,
		CTX: func() context.Context {
			ctx, cancel := context.WithCancel(context.TODO())
			cancel()
			return ctx
		}(),

		Error: errors.New(
			`mongo: failed to delete device configuration: .*` +
				context.Canceled.Error() + `$`,
		),
	}}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			ds := GetTestDataStore(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			defer ds.DropDatabase(ctx)
			if tc.CTX == nil {
				tc.CTX = ctx
			} else if id := identity.FromContext(tc.CTX); id != nil {
				ctx = identity.WithContext(ctx, id)
			}
			err := ds.InsertDevice(ctx, testDevice)
			require.NoError(t, err)

			err = ds.DeleteDevice(tc.CTX, tc.ID)
			if tc.Error != nil {
				if assert.Error(t, err) {
					assert.Regexp(t, tc.Error.Error(), err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
