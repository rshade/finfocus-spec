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
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// maxUnknownCapabilitiesTracked caps the dedup set below, so a caller sending many
// distinct unknown values cannot grow plugin memory without limit.
const maxUnknownCapabilitiesTracked = 64

// unknownCapabilitiesSeen records unknown capability values already logged, so a
// host that asks on every routing decision produces one warning per value.
// unknownCapabilitiesTracked counts its entries; once it reaches
// maxUnknownCapabilitiesTracked, new values are no longer recorded or logged.
//
//nolint:gochecknoglobals // Process-wide dedup state for the unknown-capability warning.
var (
	unknownCapabilitiesSeen    sync.Map
	unknownCapabilitiesTracked atomic.Int32
)

// RequestedCapability returns the capability a Supports request asks about.
// Known values are returned unchanged. PLUGIN_CAPABILITY_UNSPECIFIED and a nil
// request return PLUGIN_CAPABILITY_UNSPECIFIED, which means the host is asking
// about any capability. Any other value is treated as PLUGIN_CAPABILITY_UNSPECIFIED,
// so a newer host keeps working against this plugin, and is logged as a warning
// the first time each distinct value is seen. After 64 distinct unknown values,
// further new values are not logged.
//
// Plugins call it from SupportsProvider.Supports to answer per capability:
//
//	switch pluginsdk.RequestedCapability(req) {
//	case pbc.PluginCapability_PLUGIN_CAPABILITY_PROJECTED_COSTS:
//	    return &pbc.SupportsResponse{Supported: false, Reason: "cannot price this type"}, nil
//	default:
//	    return &pbc.SupportsResponse{Supported: true}, nil
//	}
func RequestedCapability(req *pbc.SupportsRequest) pbc.PluginCapability {
	capability := req.GetCapability()
	if capability == pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED || IsValidCapability(capability) {
		return capability
	}
	warnUnknownCapability(int32(capability))
	return pbc.PluginCapability_PLUGIN_CAPABILITY_UNSPECIFIED
}

// warnUnknownCapability logs an unknown value the first time it is seen, until
// maxUnknownCapabilitiesTracked values are recorded. Concurrent first sightings can
// overshoot the cap by at most the number of racing goroutines.
func warnUnknownCapability(raw int32) {
	if unknownCapabilitiesTracked.Load() >= maxUnknownCapabilitiesTracked {
		return
	}
	if _, seen := unknownCapabilitiesSeen.LoadOrStore(raw, struct{}{}); seen {
		return
	}
	log.Warn().Int32("capability", raw).Msg("Unknown Supports capability, treating as UNSPECIFIED")
	if unknownCapabilitiesTracked.Add(1) == maxUnknownCapabilitiesTracked {
		log.Warn().
			Int("limit", maxUnknownCapabilitiesTracked).
			Msg("Too many unknown Supports capabilities, no longer logging new values")
	}
}
