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

func ptrNow() *time.Time {
	now := time.Now()
	return &now
}

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
		Error: errors.New("^mongo: error reaching mongo server:.*" +
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
			ID:        uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")).String(),
			UpdatedTS: ptrNow(),
		}},
	}, {
		Name: "error, invalid document",

		Devices: []model.Device{{
			UpdatedTS: ptrNow(),
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
			ID:        uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")).String(),
			UpdatedTS: ptrNow(),
		}},
		Error: errors.New(
			`mongo: failed to store device configuration: .*` +
				context.Canceled.Error(),
		),
	}, {
		Name: "error, duplicate key",

		Devices: []model.Device{{
			ID:        uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")).String(),
			UpdatedTS: ptrNow(),
		}, {
			ID:        uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")).String(),
			UpdatedTS: ptrNow(),
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

func TestGetDevice(t *testing.T) {
	t.Parallel()
	deviceID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")).String()
	testCases := []struct {
		Name string

		CTX          context.Context
		Devices      []model.Device
		FoundDevices []model.Device

		Error error
	}{
		{
			Name: "ok",

			Devices: []model.Device{
				{
					ID: deviceID,
					ConfiguredAttributes: []model.Attribute{
						{
							Key:   "key0",
							Value: "value0",
						},
					},
					ReportedAttributes: []model.Attribute{
						{
							Key:   "key2",
							Value: "value2",
						},
					},
					UpdatedTS: ptrNow(),
				},
			},
			FoundDevices: []model.Device{
				{
					ID: deviceID,
					ConfiguredAttributes: []model.Attribute{
						{
							Key:   "key0",
							Value: "value0",
						},
					},
					ReportedAttributes: []model.Attribute{
						{
							Key:   "key2",
							Value: "value2",
						},
					},
					UpdatedTS: ptrNow(),
				},
			},
		},

		{
			Name: "error device not found",

			Devices: []model.Device{
				{
					ID: deviceID,
					ConfiguredAttributes: []model.Attribute{
						{
							Key:   "key0",
							Value: "value0",
						},
					},
					ReportedAttributes: []model.Attribute{
						{
							Key:   "key2",
							Value: "value2",
						},
					},
					UpdatedTS: ptrNow(),
				},
			},
			FoundDevices: []model.Device{
				{
					ConfiguredAttributes: []model.Attribute{
						{
							Key:   "key0",
							Value: "value0",
						},
					},
					ReportedAttributes: []model.Attribute{
						{
							Key:   "key2",
							Value: "value2",
						},
					},
					UpdatedTS: ptrNow(),
				},
			},
			Error: errors.New("mongo: device does not exist"),
		},
	}
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

			for _, dev := range tc.Devices {
				if tc.Error != nil {
					_, err := ds.GetDevice(tc.CTX, uuid.New().String())
					assert.EqualError(t, tc.Error, err.Error())
					break
				}
				d, err := ds.GetDevice(tc.CTX, dev.ID)
				assert.NoError(t, err)
				timeVal := time.Unix(1, 0)
				d.UpdatedTS = &timeVal
				dev.UpdatedTS = &timeVal
				assert.Equal(t, d, dev)
			}
		})
	}
}

func TestUpsertConfiguration(t *testing.T) {
	t.Parallel()
	deviceID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")).String()
	testCases := []struct {
		Name string

		CTX            context.Context
		Devices        []model.Device
		UpdatedDevices []model.Device

		Error error
	}{
		{
			Name: "ok new configured set",

			Devices: []model.Device{
				{
					ID:        deviceID,
					UpdatedTS: ptrNow(),
				},
			},

			UpdatedDevices: []model.Device{
				{
					ID: deviceID,
					ConfiguredAttributes: model.Attributes{
						{
							Key:   "key0",
							Value: "value0",
						},
					},
					UpdatedTS: ptrNow(),
				},
			},
		},

		{
			Name: "ok removed configured",

			Devices: []model.Device{
				{
					ID: deviceID,
					ConfiguredAttributes: model.Attributes{
						{
							Key:   "key0",
							Value: "value0",
						},
					},
					UpdatedTS: ptrNow(),
				},
			},

			UpdatedDevices: []model.Device{
				{
					ID:                   deviceID,
					ConfiguredAttributes: model.Attributes{},
					UpdatedTS:            ptrNow(),
				},
			},
		},

		{
			Name: "error not a valid device",

			Devices: []model.Device{
				{
					ConfiguredAttributes: model.Attributes{
						{
							Key:   "key0",
							Value: "value0",
						},
					},
					UpdatedTS: ptrNow(),
				},
			},

			UpdatedDevices: []model.Device{
				{
					ID:                   deviceID,
					ConfiguredAttributes: model.Attributes{},
					UpdatedTS:            ptrNow(),
				},
			},

			Error: errors.New("invalid device object: id: cannot be blank."),
		},
	}
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
				err = ds.UpsertConfiguration(tc.CTX, dev)
				if err != nil {
					if tc.Error != nil {
						assert.EqualError(t, tc.Error, err.Error())
						return
					}
					break
				}
			}
			for _, dev := range tc.Devices {
				d, err := ds.GetDevice(tc.CTX, dev.ID)
				assert.NoError(t, err)
				assert.Equal(t, dev.ID, d.ID)
				assert.Equal(t, dev.ConfiguredAttributes, d.ConfiguredAttributes)
				assert.Equal(t, dev.ReportedAttributes, d.ReportedAttributes)
			}

			for _, dev := range tc.UpdatedDevices {
				err = ds.UpsertConfiguration(tc.CTX, dev)
				if err != nil {
					break
				}
			}
			for _, dev := range tc.UpdatedDevices {
				d, err := ds.GetDevice(tc.CTX, dev.ID)
				assert.NoError(t, err)
				assert.Equal(t, dev.ID, d.ID)
				assert.Equal(t, dev.ConfiguredAttributes, d.ConfiguredAttributes)
				assert.Equal(t, dev.ReportedAttributes, d.ReportedAttributes)
			}

			assert.NoError(t, err)
		})
	}
}

func TestUpsertReportedConfiguration(t *testing.T) {
	t.Parallel()
	deviceID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")).String()
	testCases := []struct {
		Name string

		CTX            context.Context
		Devices        []model.Device
		UpdatedDevices []model.Device

		Error error
	}{
		{
			Name: "ok new reported set",

			Devices: []model.Device{
				{
					ID:        deviceID,
					UpdatedTS: ptrNow(),
				},
			},

			UpdatedDevices: []model.Device{
				{
					ID: deviceID,
					ReportedAttributes: model.Attributes{
						{
							Key:   "key0",
							Value: "value0",
						},
					},
					UpdatedTS: ptrNow(),
				},
			},
		},
		{
			Name: "ok removed reported",

			Devices: []model.Device{
				{
					ID: deviceID,
					ReportedAttributes: model.Attributes{
						{
							Key:   "key0",
							Value: "value0",
						},
					},
					UpdatedTS: ptrNow(),
				},
			},

			UpdatedDevices: []model.Device{
				{
					ID:                 deviceID,
					ReportedAttributes: model.Attributes{},
					UpdatedTS:          ptrNow(),
				},
			},
		},
		{
			Name: "error not a valid device",

			Devices: []model.Device{
				{
					ReportedAttributes: model.Attributes{
						{
							Key:   "key0",
							Value: "value0",
						},
					},
					UpdatedTS: ptrNow(),
				},
			},

			UpdatedDevices: []model.Device{
				{
					ID:                 deviceID,
					ReportedAttributes: model.Attributes{},
					UpdatedTS:          ptrNow(),
				},
			},

			Error: errors.New("invalid device object: id: cannot be blank."),
		},
	}
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
				err = ds.UpsertReportedConfiguration(tc.CTX, dev)
				if err != nil {
					if tc.Error != nil {
						assert.EqualError(t, tc.Error, err.Error())
						return
					}
					break
				}
			}
			for _, dev := range tc.Devices {
				d, err := ds.GetDevice(tc.CTX, dev.ID)
				assert.NoError(t, err)
				assert.Equal(t, dev.ID, d.ID)
				assert.Equal(t, dev.ConfiguredAttributes, d.ConfiguredAttributes)
				assert.Equal(t, dev.ReportedAttributes, d.ReportedAttributes)
			}

			for _, dev := range tc.UpdatedDevices {
				err = ds.UpsertReportedConfiguration(tc.CTX, dev)
				if err != nil {
					break
				}
			}
			for _, dev := range tc.UpdatedDevices {
				d, err := ds.GetDevice(tc.CTX, dev.ID)
				assert.NoError(t, err)
				assert.Equal(t, dev.ID, d.ID)
				assert.Equal(t, dev.ConfiguredAttributes, d.ConfiguredAttributes)
				assert.Equal(t, dev.ReportedAttributes, d.ReportedAttributes)
			}

			assert.NoError(t, err)
		})
	}
}

func TestSetDeploymentID(t *testing.T) {
	t.Parallel()

	var testDevice = model.Device{
		ID:        uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")).String(),
		UpdatedTS: ptrNow(),
	}

	testCases := []struct {
		Name string

		CTX          context.Context
		ID           string
		DeploymentID uuid.UUID

		Error error
	}{{
		Name: "ok",

		ID:           testDevice.ID,
		DeploymentID: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
	}, {
		Name: "ok, tenant",

		ID:           testDevice.ID,
		DeploymentID: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),

		CTX: identity.WithContext(context.Background(),
			&identity.Identity{
				Tenant: "123456789012345678901234",
			},
		),
	}, {

		Name:         "error, device does not exist",
		DeploymentID: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),

		Error: errors.New(`^mongo: ` + store.ErrDeviceNoExist.Error()),
	}, {
		Name:         "error, context canceled",
		ID:           testDevice.ID,
		DeploymentID: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
		CTX: func() context.Context {
			ctx, cancel := context.WithCancel(context.TODO())
			cancel()
			return ctx
		}(),

		Error: errors.New(
			`mongo: failed to set the deployment ID: .*` +
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

			err = ds.SetDeploymentID(tc.CTX, tc.ID, tc.DeploymentID)
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

func TestDeleteDevice(t *testing.T) {
	t.Parallel()

	var testDevice = model.Device{
		ID:        uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")).String(),
		UpdatedTS: ptrNow(),
	}

	testCases := []struct {
		Name string

		CTX context.Context
		ID  string

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
