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

package model

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDeviceValidate(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name string

		Device Device
		Error  error
	}{{
		Name: "ok",

		Device: Device{
			ID: uuid.NewSHA1(uuid.NameSpaceOID, []byte("digest")),
			ConfiguredAttributes: []Attribute{{
				Key:   "HOME",
				Value: "/root",
			}},
			UpdatedTS: time.Now(),
		},
	}, {
		Name: "error, bad type",

		Device: Device{
			ID: uuid.NewSHA1(uuid.NameSpaceOID, []byte("digest")),
			ReportedAttributes: []Attribute{{
				Key:   "illegal",
				Value: true,
			}, {
				Key:   "illegal#2",
				Value: func() { return },
			}},
			UpdatedTS: time.Now(),
		},
		Error: errors.New(
			"invalid device object: " +
				"reported: (0: (value: invalid type: bool.); " +
				"1: (value: invalid type: func().).).",
		),
	}, {
		Name: "error, empty device id",

		Device: Device{},
		Error: errors.New(
			"invalid device object: id: cannot be blank.",
		),
	}}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			err := tc.Device.Validate()
			if tc.Error != nil {
				assert.EqualError(t, err, tc.Error.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
