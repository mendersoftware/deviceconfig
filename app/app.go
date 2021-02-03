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
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/mendersoftware/deviceconfig/model"
	"github.com/mendersoftware/deviceconfig/store"
	"github.com/mendersoftware/go-lib-micro/identity"
)

// App errors
var (
	ErrDeviceNotFound     = errors.New("device not found")
	ErrDeviceNotConnected = errors.New("device not connected")
)

// App interface describes app objects
//nolint:lll
//go:generate ../x/mockgen.sh
type App interface {
	HealthCheck(ctx context.Context) error

	ProvisionTenant(ctx context.Context, tenant model.NewTenant) error

	ProvisionDevice(ctx context.Context, dev model.NewDevice) error
	DecommissionDevice(ctx context.Context, devID uuid.UUID) error

	SetConfiguration(ctx context.Context, devID uuid.UUID, configuration model.Attributes) error
	SetReportedConfiguration(ctx context.Context, devID uuid.UUID, configuration model.Attributes) error
	GetDevice(ctx context.Context, devID uuid.UUID) (model.Device, error)
}

// app is an app object
type app struct {
	store store.DataStore
	Config
}

type Config struct {
	HaveAuditLogs bool
}

// NewApp initialize a new deviceconfig App
func New(ds store.DataStore, config ...Config) App {
	conf := Config{}
	for _, cfgIn := range config {
		if cfgIn.HaveAuditLogs {
			conf.HaveAuditLogs = true
		}
	}
	return &app{
		store:  ds,
		Config: conf,
	}
}

// HealthCheck performs a health check and returns an error if it fails
func (a *app) HealthCheck(ctx context.Context) error {
	return a.store.Ping(ctx)
}

func (a *app) ProvisionTenant(ctx context.Context, tenant model.NewTenant) error {
	ctx = identity.WithContext(ctx, &identity.Identity{
		Tenant: tenant.TenantID,
	})
	return a.store.MigrateLatest(ctx)
}

func (a *app) ProvisionDevice(ctx context.Context, dev model.NewDevice) error {
	return a.store.InsertDevice(ctx, model.Device{
		ID:        dev.ID,
		UpdatedTS: time.Now(),
	})
}

func (a *app) DecommissionDevice(ctx context.Context, devID uuid.UUID) error {
	return a.store.DeleteDevice(ctx, devID)
}

func (a *app) SetConfiguration(ctx context.Context,
	devID uuid.UUID,
	configuration model.Attributes) error {
	return a.store.UpsertConfiguration(ctx, model.Device{
		ID:                   devID,
		ConfiguredAttributes: configuration,
		UpdatedTS:            time.Now(),
	})
}

func (a *app) SetReportedConfiguration(ctx context.Context,
	devID uuid.UUID,
	configuration model.Attributes) error {
	return a.store.UpsertReportedConfiguration(ctx, model.Device{
		ID:                 devID,
		ReportedAttributes: configuration,
		ReportTS:           time.Now(),
	})
}

func (a *app) GetDevice(ctx context.Context, devID uuid.UUID) (model.Device, error) {
	return a.store.GetDevice(ctx, devID)
}
