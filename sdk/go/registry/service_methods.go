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

package registry

import (
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const costSourceServiceName = "CostSourceService"

// allServiceMethods holds the CostSourceService RPC names, read once from the service descriptor so
// the list cannot drift from the proto.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation validation
var allServiceMethods = costSourceMethods()

func costSourceMethods() []string {
	methods := pbc.File_finfocus_v1_costsource_proto.Services().ByName(costSourceServiceName).Methods()
	out := make([]string, methods.Len())
	for i := range methods.Len() {
		out[i] = string(methods.Get(i).Name())
	}
	return out
}

// AllServiceMethods returns the RPC names of finfocus.v1.CostSourceService in proto declaration
// order. These are the values a manifest may list in specification.service_definition.methods.
func AllServiceMethods() []string {
	return allServiceMethods
}

// IsValidServiceMethod reports whether name is an RPC of finfocus.v1.CostSourceService.
func IsValidServiceMethod(name string) bool {
	for _, method := range allServiceMethods {
		if name == method {
			return true
		}
	}
	return false
}
