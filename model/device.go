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
	"time"

	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Device struct {
	// ID is the device id assigned by deviceauth
	ID uuid.UUID `bson:"_id" json:"id"`

	// DesiredAttributes is the configured attributes for the device.
	DesiredAttributes []Attribute `bson:"desired,omitempty" json:"desired"`
	// CurrentAttributes is the configuration reported by the device.
	CurrentAttributes []Attribute `bson:"current,omitempty" json:"current"`

	// UpdatedTS holds the timestamp for when the desired state changed,
	// including when the object was created.
	UpdatedTS time.Time `bson:"updated_ts" json:"updated_ts"`
	// ReportTS holds the timestamp when the device last reported its' state.
	ReportTS time.Time `bson:"report_ts,omitempty" json:"report_ts,omitempty"`
}

func (dev Device) Validate() error {
	err := validation.ValidateStruct(&dev,
		validation.Field(&dev.ID, uuidNotEmpty),
		validation.Field(&dev.DesiredAttributes),
		validation.Field(&dev.CurrentAttributes),
		validation.Field(&dev.UpdatedTS, validation.Required),
	)
	return errors.Wrap(err, "invalid device object")
}
