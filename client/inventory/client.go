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
package inventory

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/mendersoftware/go-lib-micro/identity"

	"github.com/mendersoftware/deviceconfig/model"
)

const (
	uriDevicesInGroup = "/api/internal/v1/inventory/devices/:tenantId/ingroup/:name"
)

// Client is the inventory client
//go:generate ../../x/mockgen.sh
type Client interface {
	AreDevicesInGroup(ctx context.Context, devices []string, group string) (bool, error)
}

type client struct {
	client  *http.Client
	uriBase string
}

func NewClient(uriBase string, timeout int) *client {
	return &client{
		uriBase: uriBase,
		client:  &http.Client{Timeout: time.Duration(timeout) * time.Second},
	}
}

func (c *client) AreDevicesInGroup(
	ctx context.Context, devices []string, group string) (bool, error) {

	// get tenant id from context
	identity := identity.FromContext(ctx)
	if identity == nil {
		return false, errors.New("identity missing from the context")
	}

	repl := strings.NewReplacer(":tenantId", identity.Tenant, ":name", group)
	url := c.uriBase + repl.Replace(uriDevicesInGroup)

	deviceIds := model.DeviceIds{
		Devices: devices,
	}
	payload, _ := json.Marshal(deviceIds)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(payload)))
	if err != nil {
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	rsp, err := c.client.Do(req)
	if err != nil {
		return false, err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return false, nil
	}

	return true, nil
}
