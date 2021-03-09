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
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	mapp "github.com/mendersoftware/deviceconfig/app/mocks"
	"github.com/mendersoftware/deviceconfig/model"
	"github.com/mendersoftware/deviceconfig/store"
	"github.com/mendersoftware/go-lib-micro/rest.utils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var contextMatcher = mock.MatchedBy(func(v context.Context) bool {
	return true
})

func TestAlive(t *testing.T) {
	t.Parallel()

	router := NewRouter(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", URIInternal+URIAlive, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Nil(t, w.Body.Bytes())
}

func TestHealth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		Name string

		Error      error
		StatusCode int
	}{{
		Name: "ok",

		StatusCode: http.StatusNoContent,
	}, {
		Name: "error, from application layer",

		Error:      errors.New("mongo: Connection refused"),
		StatusCode: http.StatusInternalServerError,
	}}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			app := new(mapp.App)
			app.On("HealthCheck",
				mock.MatchedBy(func(_ context.Context) bool {
					return true
				}),
			).Return(tc.Error)
			defer app.AssertExpectations(t)
			router := NewRouter(app)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", URIInternal+URIHealth, nil)
			req.Header.Set("X-Men-Requestid", "test")

			router.ServeHTTP(w, req)
			assert.Equal(t, tc.StatusCode, w.Code)
			if tc.Error == nil {
				assert.Nil(t, w.Body.Bytes())
			} else {
				err := rest.Error{
					Err:       tc.Error.Error(),
					RequestID: "test",
				}
				b, _ := json.Marshal(err)
				assert.Equal(t,
					string(b),
					w.Body.String(),
				)
			}
		})
	}
}

func TestProvisionTenant(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name string

		TenantID string
		Request  *http.Request

		App    *mapp.App
		Error  *rest.Error
		Status int
	}{{
		Name: "ok",

		Request: func() *http.Request {
			body, _ := json.Marshal(map[string]interface{}{
				"tenant_id": "0123456789abcdef01234567",
			})

			req, _ := http.NewRequest("POST",
				"http://localhost"+URIInternal+URITenants,
				bytes.NewReader(body),
			)
			req.Header.Set("Content-Type", "application/json")
			return req
		}(),

		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("ProvisionTenant",
				contextMatcher,
				model.NewTenant{
					TenantID: "0123456789abcdef01234567",
				},
			).Return(nil)
			return app
		}(),
		Status: http.StatusCreated,
	}, {
		Name: "error bad request body",

		Request: func() *http.Request {
			req, _ := http.NewRequest("POST",
				"http://localhost"+URIInternal+URITenants,
				bytes.NewReader([]byte("tenant_id=foobar")),
			)
			req.Header.Set("X-Men-Requestid", "test")
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			return req
		}(),

		App: new(mapp.App),
		Error: &rest.Error{
			Err: "malformed request body: invalid character " +
				"'e' in literal true (expecting 'r')",
			RequestID: "test",
		},
		Status: http.StatusBadRequest,
	}, {
		Name: "error invalid request body",

		Request: func() *http.Request {
			body, _ := json.Marshal(map[string]interface{}{
				"user_id": uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				),
			})

			req, _ := http.NewRequest("POST",
				"http://localhost"+URIInternal+URITenants,
				bytes.NewReader(body),
			)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Men-Requestid", "test")
			return req
		}(),

		App: new(mapp.App),
		Error: &rest.Error{
			Err:       "invalid request body: tenant_id: cannot be blank.",
			RequestID: "test",
		},
		Status: http.StatusBadRequest,
	}, {
		Name: "error, internal (app) error",

		Request: func() *http.Request {
			body, _ := json.Marshal(map[string]interface{}{
				"tenant_id": uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				),
			})

			req, _ := http.NewRequest("POST",
				"http://localhost"+URIInternal+URITenants,
				bytes.NewReader(body),
			)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Men-Requestid", "test")
			return req
		}(),

		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("ProvisionTenant",
				contextMatcher,
				model.NewTenant{
					TenantID: uuid.NewSHA1(
						uuid.NameSpaceDNS, []byte("mender.io"),
					).String(),
				},
			).Return(errors.New("something went wrong!"))
			return app
		}(),
		Error: &rest.Error{
			Err:       http.StatusText(http.StatusInternalServerError),
			RequestID: "test",
		},
		Status: http.StatusInternalServerError,
	}}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			defer tc.App.AssertExpectations(t)
			router := NewRouter(tc.App)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, tc.Request)
			assert.Equal(t, tc.Status, w.Code)
			if tc.Error != nil {
				b, _ := json.Marshal(tc.Error)
				assert.JSONEq(t, string(b), string(w.Body.Bytes()))
			}
		})
	}
}

func TestProvisionDevice(t *testing.T) {
	t.Parallel()
	newDeviceMatcher := func(expected string) interface{} {
		return mock.MatchedBy(func(dev model.NewDevice) bool {
			return assert.Equal(t, expected, dev.ID)
		})
	}
	testCases := []struct {
		Name string

		Request *http.Request

		App    *mapp.App
		Error  *rest.Error
		Status int
	}{{
		Name: "ok",

		Request: func() *http.Request {
			body, _ := json.Marshal(map[string]interface{}{
				"device_id": uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				),
			})

			req, _ := http.NewRequest("POST",
				"http://localhost"+URIInternal+URITenantDevices,
				bytes.NewReader(body),
			)
			return req
		}(),

		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("ProvisionDevice",
				contextMatcher,
				newDeviceMatcher(uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				).String()),
			).Return(nil)
			return app
		}(),
		Status: http.StatusCreated,
	}, {
		Name: "error bad request body",

		Request: func() *http.Request {
			req, _ := http.NewRequest("POST",
				"http://localhost"+URIInternal+URITenantDevices,
				bytes.NewReader([]byte("device_id=foobar")),
			)
			req.Header.Set("X-Men-Requestid", "test")
			return req
		}(),

		App: new(mapp.App),
		Error: &rest.Error{
			Err: "malformed request body: invalid character " +
				"'d' looking for beginning of value",
			RequestID: "test",
		},
		Status: http.StatusBadRequest,
	}, {
		Name: "error invalid request body",

		Request: func() *http.Request {
			body, _ := json.Marshal(map[string]interface{}{
				"user_id": uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				),
			})

			req, _ := http.NewRequest("POST",
				"http://localhost"+URIInternal+URITenantDevices,
				bytes.NewReader(body),
			)
			req.Header.Set("X-Men-Requestid", "test")
			return req
		}(),

		App: new(mapp.App),
		Error: &rest.Error{
			Err:       "invalid request body: device_id: cannot be blank.",
			RequestID: "test",
		},
		Status: http.StatusBadRequest,
	}, {
		Name: "error, internal (app) error",

		Request: func() *http.Request {
			body, _ := json.Marshal(map[string]interface{}{
				"device_id": uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				),
			})

			req, _ := http.NewRequest("POST",
				"http://localhost"+URIInternal+URITenantDevices,
				bytes.NewReader(body),
			)
			req.Header.Set("X-Men-Requestid", "test")
			return req
		}(),

		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("ProvisionDevice",
				contextMatcher,
				newDeviceMatcher(uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				).String()),
			).Return(errors.New("something went wrong!"))
			return app
		}(),
		Error: &rest.Error{
			Err:       http.StatusText(http.StatusInternalServerError),
			RequestID: "test",
		},
		Status: http.StatusInternalServerError,
	}, {
		Name: "error, device already exists",

		Request: func() *http.Request {
			body, _ := json.Marshal(map[string]interface{}{
				"device_id": uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				),
			})

			req, _ := http.NewRequest("POST",
				"http://localhost"+URIInternal+URITenantDevices,
				bytes.NewReader(body),
			)
			req.Header.Set("X-Men-Requestid", "test")
			return req
		}(),

		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("ProvisionDevice",
				contextMatcher,
				newDeviceMatcher(uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				).String()),
			).Return(store.ErrDeviceAlreadyExists)
			return app
		}(),
		Error: &rest.Error{
			Err:       store.ErrDeviceAlreadyExists.Error(),
			RequestID: "test",
		},
		Status: http.StatusConflict,
	}}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			defer tc.App.AssertExpectations(t)
			router := NewRouter(tc.App)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, tc.Request)
			assert.Equal(t, tc.Status, w.Code)
			if tc.Error != nil {
				b, _ := json.Marshal(tc.Error)
				assert.JSONEq(t, string(b), string(w.Body.Bytes()))
			}
		})
	}
}

func TestDecommissionDevice(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name string

		Request *http.Request

		App    *mapp.App
		Error  *rest.Error
		Status int
	}{{
		Name: "ok",

		Request: func() *http.Request {
			repl := strings.NewReplacer(
				":tenant_id", "123456789012345678901234",
				":device_id", uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				).String(),
			)
			req, _ := http.NewRequest("DELETE",
				"http://localhost"+URIInternal+
					repl.Replace(URITenantDevice),
				nil,
			)
			return req
		}(),

		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("DecommissionDevice",
				contextMatcher,
				uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				).String(),
			).Return(nil)
			return app
		}(),
		Status: http.StatusNoContent,
	}, {
		Name: "error device not found",

		Request: func() *http.Request {
			repl := strings.NewReplacer(
				":tenant_id", "123456789012345678901234",
				":device_id", uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				).String(),
			)
			req, _ := http.NewRequest("DELETE",
				"http://localhost"+URIInternal+
					repl.Replace(URITenantDevice),
				bytes.NewReader([]byte("device_id=foobar")),
			)
			req.Header.Set("X-Men-Requestid", "test")
			return req
		}(),

		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("DecommissionDevice",
				contextMatcher,
				uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				).String(),
			).Return(errors.Wrap(store.ErrDeviceNoExist, "mongo"))
			return app
		}(),
		Error: &rest.Error{
			Err:       store.ErrDeviceNoExist.Error(),
			RequestID: "test",
		},
		Status: http.StatusNotFound,
	}, {
		Name: "error, internal server error",

		Request: func() *http.Request {
			repl := strings.NewReplacer(
				":tenant_id", "123456789012345678901234",
				":device_id", uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				).String(),
			)
			req, _ := http.NewRequest("DELETE",
				"http://localhost"+URIInternal+
					repl.Replace(URITenantDevice),
				bytes.NewReader([]byte("device_id=foobar")),
			)
			req.Header.Set("X-Men-Requestid", "test")
			return req
		}(),

		App: func() *mapp.App {
			app := new(mapp.App)
			app.On("DecommissionDevice",
				contextMatcher,
				uuid.NewSHA1(
					uuid.NameSpaceDNS, []byte("mender.io"),
				).String(),
			).Return(errors.New("Oh noez!"))
			return app
		}(),
		Error: &rest.Error{
			Err:       http.StatusText(http.StatusInternalServerError),
			RequestID: "test",
		},
		Status: http.StatusInternalServerError,
	}}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			defer tc.App.AssertExpectations(t)
			router := NewRouter(tc.App)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, tc.Request)
			assert.Equal(t, tc.Status, w.Code)
			if tc.Error != nil {
				b, _ := json.Marshal(tc.Error)
				assert.JSONEq(t, string(b), string(w.Body.Bytes()))
			}
		})
	}
}
