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

package pluginsdk

import (
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

// AttributeValue returns the value at a dot-separated path within attrs, such as
// ResourceDescriptor.attributes or EstimateCostRequest.attributes.
//
// Each segment names a key of a struct value. A segment applied to a list value
// is a non-negative decimal index: "spec.containers.0.resources.requests.cpu".
// The second result reports whether every segment was found; a missing key, a
// non-numeric, negative, or out-of-range index, a segment applied to a scalar,
// an empty segment, an empty path, and nil attrs all return (nil, false), never
// an error. An explicit JSON null at the path is found and returned as a
// NullValue. A key that itself contains "." cannot be addressed by path; read it
// from attrs.GetFields() directly.
//
// The walk does not allocate. Read the result with the Value getters, for
// example GetStringValue or GetNumberValue.
func AttributeValue(attrs *structpb.Struct, path string) (*structpb.Value, bool) {
	if attrs == nil || path == "" {
		return nil, false
	}
	obj, list := attrs, (*structpb.ListValue)(nil)
	for {
		seg, rest, more := strings.Cut(path, ".")
		if seg == "" {
			return nil, false
		}
		var val *structpb.Value
		if obj != nil {
			val = obj.GetFields()[seg]
		} else {
			idx, ok := listIndex(seg, len(list.GetValues()))
			if !ok {
				return nil, false
			}
			val = list.GetValues()[idx]
		}
		if val == nil {
			return nil, false
		}
		if !more {
			return val, true
		}
		obj, list, path = val.GetStructValue(), val.GetListValue(), rest
		if obj == nil && list == nil {
			return nil, false
		}
	}
}

const decimalBase = 10

// listIndex parses seg as a decimal index below n. It accepts digits only, so
// signs, empty segments, and overflow are rejected without strconv.
func listIndex(seg string, n int) (int, bool) {
	idx := 0
	for i := range len(seg) {
		c := seg[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		idx = idx*decimalBase + int(c-'0')
		if idx >= n {
			return 0, false
		}
	}
	return idx, true
}
