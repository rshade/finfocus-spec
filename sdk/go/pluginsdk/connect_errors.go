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
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// toConnectError converts an error carrying a gRPC status into a *connect.Error
// with the same code and message. nil, *connect.Error values, and errors
// without a gRPC status are returned unchanged.
//
// connect-go reports any error that is not a *connect.Error as CodeUnknown, so
// without this conversion a gRPC status such as PermissionDenied would lose its
// code over Connect. The numeric values of codes.Code and connect.Code match
// for codes 1-16.
func toConnectError(err error) error {
	if err == nil {
		return nil
	}
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return err
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() == codes.OK {
		return err
	}
	return connect.NewError(connect.Code(st.Code()), errors.New(st.Message()))
}
