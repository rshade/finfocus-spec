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
	"google.golang.org/protobuf/types/known/anypb"
)

// anyTypeURLPrefix is the prefix protobuf puts before a message name in an Any type URL.
const anyTypeURLPrefix = "type.googleapis.com/"

// rpcStatusError carries the gRPC status of a failed Client call. The connect-go client reports
// every failure, over any wire protocol, as a *connect.Error, which has no GRPCStatus method.
// A plugin that returns such an error from a gRPC handler (a routing plugin forwarding to a
// child, for example) would otherwise reach the host as codes.Unknown.
//
// The wrapped error is unchanged: Error, Unwrap, errors.Is, and errors.As behave as before.
type rpcStatusError struct {
	error

	status *status.Status
}

func (e *rpcStatusError) Unwrap() error { return e.error }

// GRPCStatus reports the code, message, and details the child plugin sent.
func (e *rpcStatusError) GRPCStatus() *status.Status { return e.status }

// withGRPCStatus returns err with a GRPCStatus method when err wraps a *connect.Error and no
// status is reachable yet. Any other error is returned as it is. The numeric values of
// codes.Code and connect.Code match for codes 1-16.
func withGRPCStatus(err error) error {
	var withStatus interface{ GRPCStatus() *status.Status }
	if err == nil || errors.As(err, &withStatus) {
		return err
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return err
	}
	st := status.New(codes.Code(connectErr.Code()), connectErr.Message())
	statusProto := st.Proto()
	for _, detail := range connectErr.Details() {
		statusProto.Details = append(statusProto.Details, forwardedDetail(detail))
	}
	st = status.FromProto(statusProto)
	return &rpcStatusError{error: err, status: st}
}

// forwardedDetail returns detail as an Any. A message type the router has not linked cannot be
// decoded, so it is sent on as the raw type name and bytes the child produced.
func forwardedDetail(detail *connect.ErrorDetail) *anypb.Any {
	if msg, err := detail.Value(); err == nil {
		if packed, packErr := anypb.New(msg); packErr == nil {
			return packed
		}
	}
	return &anypb.Any{TypeUrl: anyTypeURLPrefix + detail.Type(), Value: detail.Bytes()}
}
