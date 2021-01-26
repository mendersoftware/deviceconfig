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

package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/mendersoftware/go-lib-micro/log"
	"github.com/mendersoftware/go-lib-micro/rest.utils"
	"github.com/pkg/errors"

	"github.com/mendersoftware/deviceconfig/app"
	"github.com/mendersoftware/deviceconfig/model"
	"github.com/mendersoftware/deviceconfig/store"
)

type InternalAPI struct {
	App app.App
}

func NewInternalAPI(app app.App) *InternalAPI {
	return &InternalAPI{
		App: app,
	}
}

func (api *InternalAPI) Alive(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func (api *InternalAPI) Health(c *gin.Context) {
	err := api.App.HealthCheck(c.Request.Context())
	if err != nil {
		rest.RenderError(c, http.StatusInternalServerError, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (api *InternalAPI) ProvisionDevice(c *gin.Context) {
	var dev model.NewDevice
	ctx := c.Request.Context()
	id := &identity.Identity{
		Tenant: c.Param("tenant_id"),
	}
	ctx = identity.WithContext(ctx, id)
	c.Request = c.Request.WithContext(ctx)
	err := c.ShouldBindJSON(&dev)
	if err != nil {
		rest.RenderError(c,
			http.StatusBadRequest,
			errors.Wrap(err, "invalid request body"),
		)
		return
	}
	if err = dev.Validate(); err != nil {
		rest.RenderError(c,
			http.StatusBadRequest,
			errors.Wrap(err, "invalid request body"),
		)
		return
	}
	err = api.App.ProvisionDevice(c.Request.Context(), dev)
	if err != nil {
		switch cause := errors.Cause(err); cause {
		case store.ErrDeviceAlreadyExists:
			rest.RenderError(c, http.StatusConflict, cause)
		default:
			c.Error(err) //nolint:errcheck
			rest.RenderError(c,
				http.StatusInternalServerError,
				errors.New(http.StatusText(http.StatusInternalServerError)),
			)
		}
		return
	}
	c.Status(http.StatusCreated)
}

func (api *InternalAPI) DecommissionDevice(c *gin.Context) {
	deviceID := c.Param("device_id")
	ctx := identity.WithContext(c.Request.Context(),
		&identity.Identity{
			Tenant:  c.Param("tenant_id"),
			Subject: deviceID,
		},
	)
	l := log.FromContext(ctx)

	deviceUUID, err := uuid.Parse(deviceID)
	if err != nil {
		rest.RenderError(c, http.StatusBadRequest,
			errors.New("device_id is not a valid UUID."),
		)
		return
	}

	err = api.App.DecommissionDevice(ctx, deviceUUID)
	if err != nil {
		switch errors.Cause(err) {
		case store.ErrDeviceNoExist:
			rest.RenderError(c, http.StatusNotFound, store.ErrDeviceNoExist)
		default:
			l.Error(err.Error())
			rest.RenderError(c,
				http.StatusInternalServerError,
				errors.New("internal error"),
			)
		}
		return
	}
	c.Status(http.StatusNoContent)
}
