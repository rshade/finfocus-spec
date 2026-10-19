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

package testing_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	pt "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func BenchmarkCostParseRequest(b *testing.B) {
	for _, name := range []string{"valid", "rejected", "near_limit"} {
		b.Run(name, func(b *testing.B) {
			req := parseRequest()
			limits := parseLimits()
			switch name {
			case "rejected":
				req.Format = "Bad"
			case "near_limit":
				req.GetFiles().GetFiles()[0].Content = []byte(strings.Repeat("x", 65536))
				limits.MaxInputBytes = 65536
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = pt.ValidateParseRequest(req, limits)
			}
		})
	}
}
func BenchmarkCostParseMetadata(b *testing.B) {
	for _, name := range []string{"valid", "rejected", "max_int64"} {
		b.Run(name, func(b *testing.B) {
			metadata := pt.FormatParseLimits([]string{"fixture"}, []string{"content", "path"}, 65536)
			switch name {
			case "rejected":
				metadata[pt.ParseFormatsKey] = "Bad"
			case "max_int64":
				metadata[pt.ParseMaxInputBytesKey] = "9223372036854775807"
			}
			if name != "rejected" {
				_, err := pt.ParseParseLimits(metadata)
				require.NoError(b, err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, _ = pt.ParseParseLimits(metadata)
			}
		})
	}
}
