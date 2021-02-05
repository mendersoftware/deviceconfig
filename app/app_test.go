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

package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/mendersoftware/deviceconfig/client/workflows"
	mworkflows "github.com/mendersoftware/deviceconfig/client/workflows/mocks"
	"github.com/mendersoftware/deviceconfig/model"
	mstore "github.com/mendersoftware/deviceconfig/store/mocks"
	"github.com/mendersoftware/go-lib-micro/identity"
)

func TestHealthCheck(t *testing.T) {
	t.Parallel()
	err := errors.New("error")

	store := &mstore.DataStore{}
	store.On("Ping",
		mock.MatchedBy(func(ctx context.Context) bool {
			return true
		}),
	).Return(err)

	app := New(store, nil, Config{})

	ctx := context.Background()
	res := app.HealthCheck(ctx)
	assert.Equal(t, err, res)

	store.AssertExpectations(t)
}

func TestProvisionTenant(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()

	const tenantID = "dummy"
	tenant := model.NewTenant{
		TenantID: tenantID,
	}

	ds := new(mstore.DataStore)
	ds.On("MigrateLatest",
		mock.MatchedBy(func(ctx context.Context) bool {
			id := identity.FromContext(ctx)
			assert.NotNil(t, id)
			assert.Equal(t, id.Tenant, tenantID)
			return true
		}),
	).Return(nil)

	defer ds.AssertExpectations(t)

	app := New(ds, nil, Config{})
	err := app.ProvisionTenant(ctx, tenant)
	assert.NoError(t, err)
}

func TestProvisionDevice(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	dev := model.NewDevice{
		ID: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
	}
	deviceMatcher := mock.MatchedBy(func(d model.Device) bool {
		if !assert.Equal(t, dev.ID, d.ID) {
			return false
		}
		return assert.WithinDuration(t, time.Now(), d.UpdatedTS, time.Minute)
	})

	ds := new(mstore.DataStore)
	defer ds.AssertExpectations(t)
	ds.On("InsertDevice", ctx, deviceMatcher).Return(nil)

	app := New(ds, nil, Config{})
	err := app.ProvisionDevice(ctx, dev)
	assert.NoError(t, err)
}

func TestGetDevice(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	dev := model.NewDevice{
		ID: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
	}
	device := model.Device{
		ID: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
	}
	deviceMatcher := mock.MatchedBy(func(d model.Device) bool {
		if !assert.Equal(t, dev.ID, d.ID) {
			return false
		}
		return assert.WithinDuration(t, time.Now(), d.UpdatedTS, time.Minute)
	})

	ds := new(mstore.DataStore)
	defer ds.AssertExpectations(t)
	ds.On("InsertDevice", ctx, deviceMatcher).Return(nil)
	ds.On("GetDevice", ctx, dev.ID).Return(device, nil)

	app := New(ds, nil, Config{})
	err := app.ProvisionDevice(ctx, dev)
	assert.NoError(t, err)

	d, err := app.GetDevice(ctx, dev.ID)
	assert.NoError(t, err)
	assert.Equal(t, dev.ID, d.ID)
}

func TestDecommissionDevice(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	devID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io"))

	ds := new(mstore.DataStore)
	defer ds.AssertExpectations(t)
	ds.On("DeleteDevice", ctx, devID).Return(nil)

	app := New(ds, nil, Config{})
	err := app.DecommissionDevice(ctx, devID)
	assert.NoError(t, err)
}

func TestSetConfiguration(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	dev := model.NewDevice{
		ID: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
	}
	device := model.Device{
		ID: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
		ConfiguredAttributes: []model.Attribute{
			{
				Key:   "hostname",
				Value: "some0",
			},
		},
		ReportedAttributes: []model.Attribute{
			{
				Key:   "hostname",
				Value: "some0other",
			},
		},
	}
	deviceMatcher := mock.MatchedBy(func(d model.Device) bool {
		if !assert.Equal(t, dev.ID, d.ID) {
			return false
		}
		return assert.WithinDuration(t, time.Now(), d.UpdatedTS, time.Minute)
	})

	ds := new(mstore.DataStore)
	defer ds.AssertExpectations(t)
	ds.On("InsertDevice", ctx, deviceMatcher).Return(nil)
	ds.On("UpsertConfiguration", ctx, deviceMatcher).Return(nil)
	ds.On("GetDevice", ctx, dev.ID).Return(device, nil)

	app := New(ds, nil, Config{})
	err := app.ProvisionDevice(ctx, dev)
	assert.NoError(t, err)

	err = app.SetConfiguration(ctx, dev.ID, device.ConfiguredAttributes)
	assert.NoError(t, err)

	d, err := app.GetDevice(ctx, dev.ID)
	assert.NoError(t, err)

	assert.Equal(t, d.ConfiguredAttributes, device.ConfiguredAttributes)

	err = app.SetConfiguration(ctx, dev.ID, []model.Attribute{
		{
			Key:   "hostname",
			Value: "other",
		},
	})
	assert.NoError(t, err)

	d, err = app.GetDevice(ctx, dev.ID)
	assert.NoError(t, err)

	assert.NotEqual(t, device.ConfiguredAttributes, d.ConfiguredAttributes[0])

	err = app.SetConfiguration(ctx, dev.ID, []model.Attribute{
		{
			Key:   "hostname",
			Value: "",
		},
	})
	assert.NoError(t, err)
}

func TestSetConfigurationWithAuditLogs(t *testing.T) {
	const userID = "user-id"

	testCases := map[string]struct {
		err error
	}{
		"ok": {
			err: nil,
		},
		"error": {
			err: errors.New("workflows error"),
		},
	}

	t.Parallel()

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.TODO()
			ctx = identity.WithContext(ctx, &identity.Identity{
				Subject: userID,
				IsUser:  true,
			})

			dev := model.NewDevice{
				ID: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
			}
			configuration := []model.Attribute{
				{
					Key:   "hostname",
					Value: "some0",
				},
			}

			deviceMatcher := mock.MatchedBy(func(d model.Device) bool {
				if !assert.Equal(t, dev.ID, d.ID) {
					return false
				}
				return assert.WithinDuration(t, time.Now(), d.UpdatedTS, time.Minute)
			})

			ds := new(mstore.DataStore)
			defer ds.AssertExpectations(t)
			ds.On("InsertDevice", ctx, deviceMatcher).Return(nil)
			ds.On("UpsertConfiguration", ctx, deviceMatcher).Return(nil)

			wflows := &mworkflows.Client{}
			defer wflows.AssertExpectations(t)
			wflows.On("SubmitAuditLog",
				mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}),
				mock.MatchedBy(func(log workflows.AuditLog) bool {
					assert.Equal(t, workflows.ActionSetConfiguration, log.Action)
					assert.Equal(t, workflows.Actor{
						ID:   userID,
						Type: workflows.ActorUser,
					}, log.Actor)
					assert.Equal(t, workflows.Object{
						ID:   dev.ID.String(),
						Type: workflows.ObjectDevice,
					}, log.Object)
					assert.Equal(t, "{\"hostname\":\"some0\"}", log.Change)
					assert.WithinDuration(t, time.Now(), log.EventTS, time.Minute)

					return true
				}),
			).Return(tc.err)

			app := New(ds, wflows, Config{HaveAuditLogs: true})
			err := app.ProvisionDevice(ctx, dev)
			assert.NoError(t, err)

			err = app.SetConfiguration(ctx, dev.ID, configuration)
			if tc.err == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.err.Error())
			}
		})
	}
}

func TestSetReportedConfiguration(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	dev := model.NewDevice{
		ID: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
	}
	device := model.Device{
		ID: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
		ConfiguredAttributes: []model.Attribute{
			{
				Key:   "hostname",
				Value: "some0",
			},
		},
		ReportedAttributes: []model.Attribute{
			{
				Key:   "hostname",
				Value: "some0other",
			},
		},
	}
	deviceMatcher := mock.MatchedBy(func(d model.Device) bool {
		if !assert.Equal(t, dev.ID, d.ID) {
			return false
		}
		return assert.WithinDuration(t, time.Now(), d.UpdatedTS, time.Minute)
	})
	deviceMatcherReport := mock.MatchedBy(func(d model.Device) bool {
		if !assert.Equal(t, dev.ID, d.ID) {
			return false
		}
		return assert.WithinDuration(t, time.Now(), d.ReportTS, time.Minute)
	})

	ds := new(mstore.DataStore)
	defer ds.AssertExpectations(t)
	ds.On("InsertDevice", ctx, deviceMatcher).Return(nil)
	ds.On("UpsertReportedConfiguration", ctx, deviceMatcherReport).Return(nil)
	ds.On("GetDevice", ctx, dev.ID).Return(device, nil)

	app := New(ds, nil, Config{})
	err := app.ProvisionDevice(ctx, dev)
	assert.NoError(t, err)

	err = app.SetReportedConfiguration(ctx, dev.ID, device.ReportedAttributes)
	assert.NoError(t, err)

	d, err := app.GetDevice(ctx, dev.ID)
	assert.NoError(t, err)

	assert.Equal(t, d.ReportedAttributes, device.ReportedAttributes)

	err = app.SetReportedConfiguration(ctx, dev.ID, []model.Attribute{
		{
			Key:   "hostname",
			Value: "other",
		},
	})
	assert.NoError(t, err)

	d, err = app.GetDevice(ctx, dev.ID)
	assert.NoError(t, err)

	assert.NotEqual(t, device.ReportedAttributes, d.ReportedAttributes[0])

	err = app.SetReportedConfiguration(ctx, dev.ID, []model.Attribute{
		{
			Key:   "hostname",
			Value: "",
		},
	})
	assert.NoError(t, err)
}

func TestDeployConfiguration(t *testing.T) {
	t.Parallel()

	const userID = "user-id"

	testCases := map[string]struct {
		device  model.Device
		request model.DeployConfigurationRequest
		err     error
		wfErr   error
	}{
		"ok": {},
		"ko, deploy error": {
			err: errors.New("error"),
		},
		"ko, wfErr": {
			wfErr: errors.New("workflow error"),
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			ctx = identity.WithContext(ctx, &identity.Identity{
				Tenant:  "tenantID",
				IsUser:  true,
				Subject: userID,
			})

			ds := new(mstore.DataStore)
			defer ds.AssertExpectations(t)

			wflows := &mworkflows.Client{}
			defer wflows.AssertExpectations(t)

			configuration, _ := tc.device.ConfiguredAttributes.MarshalJSON()
			wflows.On("DeployConfiguration",
				mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}),
				"tenantID",
				tc.device.ID,
				mock.AnythingOfType("uuid.UUID"),
				configuration,
				tc.request.Retries,
			).Return(tc.err)

			if tc.err == nil || tc.wfErr != nil {
				wflows.On("SubmitAuditLog",
					mock.MatchedBy(func(ctx context.Context) bool {
						return true
					}),
					mock.MatchedBy(func(log workflows.AuditLog) bool {
						assert.Equal(t, workflows.ActionDeployConfiguration, log.Action)
						assert.Equal(t, workflows.Actor{
							ID:   userID,
							Type: workflows.ActorUser,
						}, log.Actor)
						assert.Equal(t, workflows.Object{
							ID:   tc.device.ID.String(),
							Type: workflows.ObjectDevice,
						}, log.Object)
						assert.Equal(t, string(configuration), log.Change)
						assert.WithinDuration(t, time.Now(), log.EventTS, time.Minute)

						return true
					}),
				).Return(tc.wfErr)
			}

			app := New(ds, wflows, Config{HaveAuditLogs: true})
			_, err := app.DeployConfiguration(ctx, tc.device, tc.request)
			if tc.err != nil {
				assert.Error(t, err, tc.err)
			} else if tc.wfErr != nil {
				assert.Error(t, err, tc.wfErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func map2Attributes(configurationMap map[string]interface{}) model.Attributes {
	attributes := make(model.Attributes, len(configurationMap))
	i := 0
	for k, v := range configurationMap {
		attributes[i] = model.Attribute{
			Key:   k,
			Value: v,
		}
		i++
	}

	return attributes
}
