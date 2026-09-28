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
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

type testRule struct {
	Match string `json:"match"`
}

type testNodeSplit struct {
	CPU   float64        `json:"cpu_weight"`
	Extra map[string]int `json:"extra"`
}

type testPolicy struct {
	Version   int                      `json:"version"`
	NodeSplit testNodeSplit            `json:"node_split"`
	Tags      []string                 `json:"tags"`
	Rules     []testRule               `json:"rules"`
	Pools     map[string]testNodeSplit `json:"pools"`
	Raw       json.RawMessage          `json:"raw"`
	Skip      string                   `json:"-"`
}

func defaultTestPolicy() testPolicy {
	return testPolicy{
		Version:   1,
		NodeSplit: testNodeSplit{CPU: 0.5, Extra: map[string]int{"a": 1}},
		Tags:      []string{"x", "y", "z"},
		Rules:     []testRule{{Match: "d"}},
		Pools:     map[string]testNodeSplit{"gpu": {CPU: 0.3, Extra: map[string]int{"k": 1}}},
	}
}

func TestDecodePolicyDefaultsUnchanged(t *testing.T) {
	for _, doc := range []string{"", "  \n", "null", "{}"} {
		t.Run(doc, func(t *testing.T) {
			got := defaultTestPolicy()
			require.NoError(t, pluginsdk.DecodePolicy([]byte(doc), &got))
			assert.Equal(t, defaultTestPolicy(), got)
		})
	}
	t.Run("nil", func(t *testing.T) {
		got := defaultTestPolicy()
		require.NoError(t, pluginsdk.DecodePolicy(nil, &got))
		assert.Equal(t, defaultTestPolicy(), got)
	})
}

func TestDecodePolicyRejects(t *testing.T) {
	tests := []struct {
		name     string
		doc      string
		wantPath string
	}{
		{name: "malformed", doc: `{`},
		{name: "trailing object", doc: `{"version":1}{}`},
		{name: "trailing garbage", doc: `{"version":1} x`},
		{name: "nested unknown", doc: `{"node_split":{"cpu":1}}`, wantPath: "node_split.cpu"},
		{name: "unknown in map value", doc: `{"pools":{"gpu":{"nope":1}}}`, wantPath: "pools.gpu.nope"},
		{name: "unknown in array element", doc: `{"rules":[{"match":"a"},{"nope":1}]}`, wantPath: "rules[1].nope"},
		{name: "case-sensitive", doc: `{"Version":2}`, wantPath: "Version"},
		{name: "ignored field by Go name", doc: `{"Skip":"x"}`, wantPath: "Skip"},
		{name: "ignored field by dash", doc: `{"-":"x"}`, wantPath: "-"},
		{name: "type mismatch", doc: `{"node_split":{"cpu_weight":"high"}}`, wantPath: "node_split.cpu_weight"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultTestPolicy()
			err := pluginsdk.DecodePolicy([]byte(tt.doc), &got)
			require.Error(t, err)
			require.ErrorIs(t, err, pluginsdk.ErrInvalidPolicy)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.False(t, strings.HasPrefix(err.Error(), "rpc error:"), err.Error())
			if tt.wantPath != "" {
				assert.Contains(t, err.Error(), tt.wantPath)
			}
		})
	}
}

func TestDecodePolicyMerge(t *testing.T) {
	tests := []struct {
		name  string
		doc   string
		check func(t *testing.T, got testPolicy)
	}{
		{
			name: "nested field keeps siblings",
			doc:  `{"node_split":{"cpu_weight":0.7}}`,
			check: func(t *testing.T, got testPolicy) {
				assert.InDelta(t, 0.7, got.NodeSplit.CPU, 0)
				assert.Equal(t, map[string]int{"a": 1}, got.NodeSplit.Extra)
				assert.Equal(t, 1, got.Version)
			},
		},
		{
			name: "maps merge by key",
			doc:  `{"node_split":{"extra":{"b":2}}}`,
			check: func(t *testing.T, got testPolicy) {
				assert.Equal(t, map[string]int{"a": 1, "b": 2}, got.NodeSplit.Extra)
			},
		},
		{
			name: "struct map values merge field by field",
			doc:  `{"pools":{"gpu":{"cpu_weight":0.9},"new":{"cpu_weight":0.1}}}`,
			check: func(t *testing.T, got testPolicy) {
				assert.Equal(t, map[string]testNodeSplit{
					"gpu": {CPU: 0.9, Extra: map[string]int{"k": 1}},
					"new": {CPU: 0.1},
				}, got.Pools)
			},
		},
		{
			name: "string array replaced",
			doc:  `{"tags":["q"]}`,
			check: func(t *testing.T, got testPolicy) {
				assert.Equal(t, []string{"q"}, got.Tags)
			},
		},
		{
			name: "struct array replaced without carry-over",
			doc:  `{"rules":[{}]}`,
			check: func(t *testing.T, got testPolicy) {
				assert.Equal(t, []testRule{{Match: ""}}, got.Rules)
			},
		},
		{
			name: "raw message is opaque",
			doc:  `{"raw":{"anything":[1]}}`,
			check: func(t *testing.T, got testPolicy) {
				assert.JSONEq(t, `{"anything":[1]}`, string(got.Raw))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultTestPolicy()
			require.NoError(t, pluginsdk.DecodePolicy([]byte(tt.doc), &got))
			tt.check(t, got)
		})
	}
}

func TestDecodePolicyBadTarget(t *testing.T) {
	var nilPolicy *testPolicy
	for name, target := range map[string]any{
		"non-pointer": defaultTestPolicy(),
		"nil pointer": nilPolicy,
		"nil":         nil,
	} {
		t.Run(name, func(t *testing.T) {
			err := pluginsdk.DecodePolicy([]byte(`{"version":1}`), target)
			require.Error(t, err)
			require.NotErrorIs(t, err, pluginsdk.ErrInvalidPolicy)
			assert.Equal(t, codes.Unknown, status.Code(err))
		})
	}
}
