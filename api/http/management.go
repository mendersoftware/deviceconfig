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

	"github.com/mendersoftware/deviceconfig/model"
	"github.com/mendersoftware/deviceconfig/store"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/mendersoftware/go-lib-micro/plan"
	"github.com/mendersoftware/go-lib-micro/rest.utils"

	"github.com/mendersoftware/deviceconfig/app"
)

// API errors
var (
	errUpdateContrloMapForbidden = errors.New(
		"forbidden: update control map is available only for Enterprise customers")
)

type ManagementAPI struct {
	App app.App
}

func NewManagementAPI(app app.App) *ManagementAPI {
	return &ManagementAPI{
		App: app,
	}
}

func (api *ManagementAPI) SetConfiguration(c *gin.Context) {
	var configuration model.Attributes

	ctx := c.Request.Context()
	devID := c.Param("device_id")

	err := c.ShouldBindJSON(&configuration)
	if err != nil {
		rest.RenderError(c,
			http.StatusBadRequest,
			errors.Wrap(err, "malformed request body"),
		)
		return
	}

	for _, a := range configuration {
		if err := a.Validate(); err != nil {
			rest.RenderError(c,
				http.StatusBadRequest,
				errors.Wrap(err, "invalid request body"),
			)
			return
		}
	}

	err = api.App.SetConfiguration(ctx, devID, configuration)
	if err != nil {
		c.Error(err) //nolint:errcheck
		rest.RenderError(c,
			http.StatusInternalServerError,
			errors.New(http.StatusText(http.StatusInternalServerError)),
		)
		return
	}
	c.Status(http.StatusNoContent)
}

func (api *ManagementAPI) GetConfiguration(c *gin.Context) {
	ctx := c.Request.Context()

	devID := c.Param("device_id")

	device, err := api.App.GetDevice(ctx, devID)
	if err != nil {
		switch cause := errors.Cause(err); cause {
		case store.ErrDeviceNoExist:
			c.Error(err) //nolint:errcheck
			rest.RenderError(c,
				http.StatusNotFound,
				cause,
			)
			return
		default:
			c.Error(err) //nolint:errcheck
			rest.RenderError(c,
				http.StatusInternalServerError,
				errors.New(http.StatusText(http.StatusInternalServerError)),
			)
			return
		}
	}

	c.JSON(http.StatusOK, device)
}

func (api *ManagementAPI) DeployConfiguration(c *gin.Context) {
	ctx := c.Request.Context()
	devID := c.Param("device_id")

	device, err := api.App.GetDevice(ctx, devID)
	if err != nil {
		switch cause := errors.Cause(err); cause {
		case store.ErrDeviceNoExist:
			c.Error(err) //nolint:errcheck
			rest.RenderError(c,
				http.StatusNotFound,
				cause,
			)
			return
		default:
			c.Error(err) //nolint:errcheck
			rest.RenderError(c,
				http.StatusInternalServerError,
				errors.New(http.StatusText(http.StatusInternalServerError)),
			)
			return
		}
	}

	request := model.DeployConfigurationRequest{}
	err = c.ShouldBindJSON(&request)
	if err != nil {
		rest.RenderError(c,
			http.StatusBadRequest,
			errors.Wrap(err, "malformed request body"),
		)
		return
	}

	identity := identity.FromContext(ctx)
	if identity == nil {
		rest.RenderError(c, http.StatusForbidden, errInvalidIdentity)
		return
	}
	// udpate control map is available only for Enterprise customers
	if len(request.UpdateControlMap) > 0 &&
		!plan.IsHigherOrEqual(identity.Plan, plan.PlanEnterprise) {
		rest.RenderError(c, http.StatusForbidden, errUpdateContrloMapForbidden)
		return
	}

	response, err := api.App.DeployConfiguration(ctx, device, request)
	if err != nil {
		rest.RenderError(c,
			http.StatusInternalServerError,
			errors.Wrap(err, "configuration deployment failed"),
		)
		return
	}

	c.JSON(http.StatusOK, response)
}
