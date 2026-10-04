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

//nolint:testpackage // Benchmarks the configured path through unexported harness state.
package testing

import (
	"testing"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// BenchmarkHarnessSampleResource measures the copy every resource-bearing check takes.
func BenchmarkHarnessSampleResource(b *testing.B) {
	configured := CreateResourceDescriptor("custom", "instance", "standard", "region-1")
	configured.Tags = map[string]string{"team": "platform", "env": "test"}

	cases := []struct {
		name   string
		sample *pbc.ResourceDescriptor
	}{
		{"default", nil},
		{"configured", configured},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			h := &TestHarness{sampleResource: tc.sample}
			b.ReportAllocs()
			for b.Loop() {
				_ = h.SampleResource()
			}
		})
	}
}
