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
	"github.com/mendersoftware/go-lib-micro/rest.utils"

	"github.com/mendersoftware/deviceconfig/app"
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
	c.Status(http.StatusOK)
}

func (api *InternalAPI) Health(c *gin.Context) {
	err := api.App.HealthCheck(c.Request.Context())
	if err != nil {
		rest.RenderError(c, http.StatusInternalServerError, err)
		return
	}
	c.Status(http.StatusOK)
}
