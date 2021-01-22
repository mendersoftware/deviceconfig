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
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/mendersoftware/deviceconfig/app"
	"github.com/mendersoftware/go-lib-micro/accesslog"
	"github.com/mendersoftware/go-lib-micro/identity"
	"github.com/mendersoftware/go-lib-micro/requestid"
)

// API URL used by the HTTP router
const (
	URIDevices    = "/api/devices/v1/deviceconfig"
	URIInternal   = "/api/internal/v1/deviceconfig"
	URIManagement = "/api/management/v1/deviceconfig"

	URITenantDevices = "/tenants/:tenant_id/devices"
	URITenantDevice  = "/tenants/:tenant_id/devices/:device_id"

	URIAlive  = "/alive"
	URIHealth = "/health"
)

func init() {
	if mode := os.Getenv(gin.EnvGinMode); mode != "" {
		gin.SetMode(mode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	gin.DisableConsoleColor()
}

// NewRouter initializes a new gin.Engine as a http.Handler
func NewRouter(app app.App) http.Handler {
	router := gin.New()
	// accesslog provides logging of http responses and recovery on panic.
	router.Use(accesslog.Middleware())
	// requestid attaches X-Men-Requestid header to context
	router.Use(requestid.Middleware())

	intrnlAPI := NewInternalAPI(app)
	intrnlGrp := router.Group(URIInternal)

	intrnlGrp.GET(URIAlive, intrnlAPI.Alive)
	intrnlGrp.GET(URIHealth, intrnlAPI.Health)

	intrnlGrp.POST(URITenantDevices, intrnlAPI.ProvisionDevice)
	intrnlGrp.DELETE(URITenantDevice, intrnlAPI.DecommissionDevice)

	// mgmtAPI := NewManagementAPI(app)
	mgmtGrp := router.Group(URIManagement)
	// identity middleware for collecting JWT claims into request Context.
	mgmtGrp.Use(identity.Middleware())
	// cors middleware for checking origin headers.
	mgmtGrp.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowCredentials: true,
		AllowHeaders: []string{
			"Accept",
			"Allow",
			"Content-Type",
			"Origin",
			"Authorization",
			"Accept-Encoding",
			"Access-Control-Request-Headers",
			"Header-Access-Control-Request",
		},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowWebSockets: true,
		ExposeHeaders: []string{
			"Location",
			"Link",
		},
		MaxAge: time.Hour * 12,
	}))

	// mgmtAPI := NewManagementAPI(app)
	devGrp := router.Group(URIDevices)
	devGrp.Use(identity.Middleware())

	return router
}
