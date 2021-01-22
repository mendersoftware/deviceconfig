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
