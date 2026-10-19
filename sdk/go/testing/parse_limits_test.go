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
	"errors"
	"math"
	"reflect"
	"testing"

	pt "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestCostParseLimits(t *testing.T) {
	if limits, err := pt.ParseParseLimits(nil); limits != nil || err != nil {
		t.Fatalf("absent: %v %v", limits, err)
	}
	want := &pt.ParseLimits{
		Formats:       []string{"fixture", "cfn"},
		InputModes:    []string{"content", "path"},
		MaxInputBytes: math.MaxInt64,
	}
	m := pt.FormatParseLimits(want.Formats, want.InputModes, want.MaxInputBytes)
	if m["parse_formats"] != "fixture,cfn" || m["parse_input_modes"] != "content,path" ||
		m["parse_max_input_bytes"] != "9223372036854775807" {
		t.Fatal(m)
	}
	got, err := pt.ParseParseLimits(m)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("roundtrip: %v %v", got, err)
	}
	cases := map[string][]string{
		"parse_formats":         {"", "a,", "a,a", "A", "1a", "a b", "a, cfn"},
		"parse_input_modes":     {"", "content,", "content,content", "Content", "path,unknown", " content"},
		"parse_max_input_bytes": {"", "0", "-1", "+1", "01", " 1", "1 ", "1.0", "1e3", "9223372036854775808"},
	}
	for key, values := range cases {
		for _, value := range values {
			t.Run(key+value, func(t *testing.T) {
				bad := pt.FormatParseLimits(want.Formats, want.InputModes, want.MaxInputBytes)
				bad[key] = value
				if _, parseErr := pt.ParseParseLimits(bad); !errors.Is(parseErr, pt.ErrInvalidParseLimits) {
					t.Fatalf("%v: %v", bad, parseErr)
				}
			})
		}
	}
	for key := range m {
		bad := pt.FormatParseLimits(want.Formats, want.InputModes, want.MaxInputBytes)
		delete(bad, key)
		if _, parseErr := pt.ParseParseLimits(bad); !errors.Is(parseErr, pt.ErrInvalidParseLimits) {
			t.Fatal("missing", key, parseErr)
		}
	}
	invalid := pt.FormatParseLimits([]string{"Fixture"}, []string{"other"}, -1)
	if invalid["parse_formats"] != "Fixture" || invalid["parse_input_modes"] != "other" ||
		invalid["parse_max_input_bytes"] != "-1" {
		t.Fatal("formatter repaired invalid values", invalid)
	}
}
