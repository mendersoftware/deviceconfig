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

	"github.com/mendersoftware/deviceconfig/store"
	"github.com/pkg/errors"
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
