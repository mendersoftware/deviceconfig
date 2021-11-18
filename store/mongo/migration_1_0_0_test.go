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

package mongo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	mstore "github.com/mendersoftware/go-lib-micro/store/v2"
)

type index struct {
	Keys        map[string]int `bson:"key"`
	Name        string         `bson:"name"`
	ExpireAfter *int           `bson:"expireAfterSeconds"`
}

func TestMigration_1_0_0(t *testing.T) {
	m := &migration_1_0_0{
		client: client,
		db:     DbName,
	}
	from := migrate.MakeVersion(0, 0, 0)

	err := m.Up(from)
	require.NoError(t, err)

	iv := client.Database(DbName).
		Collection(CollDevices).
		Indexes()
	ctx := context.Background()
	cur, err := iv.List(ctx)
	require.NoError(t, err)

	var idxes []index
	err = cur.All(ctx, &idxes)
	require.NoError(t, err)
	require.Len(t, idxes, 2)
	for _, idx := range idxes {
		if _, ok := idx.Keys["_id"]; ok && len(idx.Keys) == 1 {
			// Skip default index
			continue
		}
		switch idx.Name {
		case mstore.FieldTenantID + "_" + fieldID:
			assert.Equal(t, map[string]int{
				fieldID:     1,
				KeyTenantID: 1,
			}, idx.Keys)

		default:
			assert.Failf(t, "index test failed", "Index name \"%s\" not recognized", idx.Name)
		}
	}
	assert.Equal(t, "1.0.0", m.Version().String())
}
