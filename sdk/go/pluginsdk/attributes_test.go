// Copyright 2026 The FinFocus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pluginsdk_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

func attributeFixture(t testing.TB) *structpb.Struct {
	t.Helper()
	attrs, err := structpb.NewStruct(map[string]any{
		"replicas": 3,
		"labels":   map[string]any{"app": "web"},
		"nothing":  nil,
		"a.b":      "dotted key",
		"spec": map[string]any{
			"containers": []any{
				map[string]any{"name": "web", "resources": map[string]any{
					"requests": map[string]any{"cpu": "250m"},
				}},
				map[string]any{"name": "sidecar"},
			},
			"jobTemplate": map[string]any{"spec": map[string]any{"template": map[string]any{
				"spec": map[string]any{"containers": []any{
					map[string]any{"resources": map[string]any{"requests": map[string]any{"cpu": "500m"}}},
				}},
			}}},
		},
	})
	require.NoError(t, err)
	return attrs
}

func TestAttributeValue(t *testing.T) {
	attrs := attributeFixture(t)

	tests := []struct {
		name  string
		attrs *structpb.Struct
		path  string
		want  *structpb.Value
	}{
		{name: "top-level number", attrs: attrs, path: "replicas", want: structpb.NewNumberValue(3)},
		{name: "nested string", attrs: attrs, path: "labels.app", want: structpb.NewStringValue("web")},
		{name: "list index", attrs: attrs, path: "spec.containers.1.name", want: structpb.NewStringValue("sidecar")},
		{
			name:  "list index then nested",
			attrs: attrs,
			path:  "spec.containers.0.resources.requests.cpu",
			want:  structpb.NewStringValue("250m"),
		},
		{
			name:  "ten-segment CronJob path",
			attrs: attrs,
			path:  "spec.jobTemplate.spec.template.spec.containers.0.resources.requests.cpu",
			want:  structpb.NewStringValue("500m"),
		},
		{name: "explicit null is found", attrs: attrs, path: "nothing", want: structpb.NewNullValue()},
		{name: "struct value", attrs: attrs, path: "labels", want: attrs.GetFields()["labels"]},
		{name: "missing key", attrs: attrs, path: "spec.volumes"},
		{name: "missing intermediate", attrs: attrs, path: "status.phase"},
		{name: "index out of range", attrs: attrs, path: "spec.containers.2.name"},
		{name: "negative index", attrs: attrs, path: "spec.containers.-1.name"},
		{name: "non-numeric index", attrs: attrs, path: "spec.containers.first.name"},
		{name: "index with sign", attrs: attrs, path: "spec.containers.+0.name"},
		{name: "huge index", attrs: attrs, path: "spec.containers.99999999999999999999999.name"},
		{name: "segment through scalar", attrs: attrs, path: "replicas.value"},
		{name: "key segment applied to list", attrs: attrs, path: "spec.containers.name"},
		{name: "dotted key not addressable", attrs: attrs, path: "a.b"},
		{name: "empty path", attrs: attrs, path: ""},
		{name: "empty middle segment", attrs: attrs, path: "labels..app"},
		{name: "leading dot", attrs: attrs, path: ".labels"},
		{name: "trailing dot", attrs: attrs, path: "labels."},
		{name: "nil struct", attrs: nil, path: "labels.app"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, found := pluginsdk.AttributeValue(tc.attrs, tc.path)
			if tc.want == nil {
				assert.False(t, found)
				assert.Nil(t, got)
				return
			}
			require.True(t, found)
			assert.True(t, proto.Equal(tc.want, got), "got %v, want %v", got, tc.want)
		})
	}
}

func TestAttributeValueAllocationFree(t *testing.T) {
	attrs := attributeFixture(t)
	const path = "spec.jobTemplate.spec.template.spec.containers.0.resources.requests.cpu"
	allocs := testing.AllocsPerRun(100, func() {
		_, _ = pluginsdk.AttributeValue(attrs, path)
	})
	assert.Zero(t, allocs)
}

func BenchmarkAttributeValue(b *testing.B) {
	attrs := attributeFixture(b)
	const path = "spec.jobTemplate.spec.template.spec.containers.0.resources.requests.cpu"
	b.ReportAllocs()
	for b.Loop() {
		if _, ok := pluginsdk.AttributeValue(attrs, path); !ok {
			b.Fatal("path not found")
		}
	}
}
