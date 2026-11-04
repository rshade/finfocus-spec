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
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// declined reports whether code is a decline: an answer the contract allows a plugin to
// give, which the host treats as normal rather than as a fault.
func declined(code codes.Code) bool {
	switch code { //nolint:exhaustive // Every other code is a fault and keeps the ERROR level.
	case codes.InvalidArgument, codes.Unimplemented, codes.NotFound, codes.FailedPrecondition:
		return true
	default:
		return false
	}
}

// handlerLogLevel chooses the level at which the SDK logs a failed handler call. A decline
// logs at debug, so a plugin that answers normally does not fill the log with errors.
// Every other code, and any error that carries no status, logs at ERROR. The code is read
// through error wrapping.
func handlerLogLevel(err error) zerolog.Level {
	if declined(status.Code(err)) {
		return zerolog.DebugLevel
	}
	return zerolog.ErrorLevel
}
