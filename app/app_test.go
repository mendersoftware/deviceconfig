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

	"github.com/mendersoftware/deviceconfig/model"
	mstore "github.com/mendersoftware/deviceconfig/store/mocks"
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

	app := New(store, Config{})

	ctx := context.Background()
	res := app.HealthCheck(ctx)
	assert.Equal(t, err, res)

	store.AssertExpectations(t)
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

	app := New(ds, Config{})
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

	app := New(ds, Config{})
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

	app := New(ds, Config{})
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
		DesiredAttributes: []model.Attribute{
			{
				Key:   "hostname",
				Value: "some0",
			},
		},
		CurrentAttributes: []model.Attribute{
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
	ds.On("UpsertExpectedConfiguration", ctx, deviceMatcher).Return(nil)
	ds.On("GetDevice", ctx, dev.ID).Return(device, nil)

	app := New(ds, Config{})
	err := app.ProvisionDevice(ctx, dev)
	assert.NoError(t, err)

	err = app.SetConfiguration(ctx, dev.ID, device.DesiredAttributes)
	assert.NoError(t, err)

	d, err := app.GetDevice(ctx, dev.ID)
	assert.NoError(t, err)

	assert.Equal(t, d.DesiredAttributes, device.DesiredAttributes)

	err = app.SetConfiguration(ctx, dev.ID, []model.Attribute{
		{
			Key:   "hostname",
			Value: "other",
		},
	})
	assert.NoError(t, err)

	d, err = app.GetDevice(ctx, dev.ID)
	assert.NoError(t, err)

	assert.NotEqual(t, device.DesiredAttributes, d.DesiredAttributes[0])

	err = app.SetConfiguration(ctx, dev.ID, []model.Attribute{
		{
			Key:   "hostname",
			Value: "",
		},
	})
	assert.NoError(t, err)
}

func TestSetReportedConfiguration(t *testing.T) {
	t.Parallel()
	ctx := context.TODO()
	dev := model.NewDevice{
		ID: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
	}
	device := model.Device{
		ID: uuid.NewSHA1(uuid.NameSpaceDNS, []byte("mender.io")),
		DesiredAttributes: []model.Attribute{
			{
				Key:   "hostname",
				Value: "some0",
			},
		},
		CurrentAttributes: []model.Attribute{
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

	app := New(ds, Config{})
	err := app.ProvisionDevice(ctx, dev)
	assert.NoError(t, err)

	err = app.SetReportedConfiguration(ctx, dev.ID, device.CurrentAttributes)
	assert.NoError(t, err)

	d, err := app.GetDevice(ctx, dev.ID)
	assert.NoError(t, err)

	assert.Equal(t, d.CurrentAttributes, device.CurrentAttributes)

	err = app.SetReportedConfiguration(ctx, dev.ID, []model.Attribute{
		{
			Key:   "hostname",
			Value: "other",
		},
	})
	assert.NoError(t, err)

	d, err = app.GetDevice(ctx, dev.ID)
	assert.NoError(t, err)

	assert.NotEqual(t, device.CurrentAttributes, d.CurrentAttributes[0])

	err = app.SetReportedConfiguration(ctx, dev.ID, []model.Attribute{
		{
			Key:   "hostname",
			Value: "",
		},
	})
	assert.NoError(t, err)
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
