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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// ErrInvalidPluginSettings is returned when a settings list breaks a descriptor rule.
var ErrInvalidPluginSettings = plugintesting.ErrInvalidPluginSettings

// clonePluginSettings copies settings in order and keeps nil entries.
// An empty input returns nil so a plugin with no settings allocates nothing.
func clonePluginSettings(in []*pbc.PluginSetting) []*pbc.PluginSetting {
	if len(in) == 0 {
		return nil
	}
	out := make([]*pbc.PluginSetting, len(in))
	for i, setting := range in {
		if setting != nil {
			out[i] = proto.CloneOf(setting)
		}
	}
	return out
}

// rejectBadSettings logs a malformed list and returns a client error that does
// not quote the default. An empty list is valid.
func (s *Server) rejectBadSettings(settings []*pbc.PluginSetting) error {
	if len(settings) == 0 {
		return nil
	}
	err := plugintesting.ValidatePluginSettings(settings)
	if err == nil {
		return nil
	}
	s.logger.Error().Err(err).Msg("plugin settings are invalid")
	return status.Error(codes.Internal, "plugin settings are invalid")
}
