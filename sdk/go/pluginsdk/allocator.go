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
	"context"

	"connectrpc.com/connect"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// DefaultConservationEpsilon is the relative tolerance for CheckConservation.
// It is identical to the sdk/go/testing constant of the same name.
const DefaultConservationEpsilon = plugintesting.DefaultConservationEpsilon

// ValidateAllocateRequest is identical to the sdk/go/testing function of the
// same name. Every failure wraps plugintesting.ErrInvalidAllocateRequest (mixed
// currencies also wrap plugintesting.ErrMixedCurrency) and carries
// codes.InvalidArgument, so allocators may return it directly.
func ValidateAllocateRequest(req *pbc.AllocateRequest) error {
	return plugintesting.ValidateAllocateRequest(req)
}

// ResolveCurrency is identical to the sdk/go/testing function of the same
// name: the single distinct non-empty currency across priced=true entries, or
// "USD" when all are empty. More than one distinct value returns an error with
// codes.InvalidArgument.
func ResolveCurrency(priced []*pbc.PricedResource) (string, error) {
	return plugintesting.ResolveCurrency(priced)
}

// CheckConservation is identical to the sdk/go/testing function of the same
// name. Hosts call it on every AllocateResponse before rendering: it verifies
// that the rows sum to the cost of the priced=true resources within
// max(relEpsilon*|expected|, 1e-9) and fails closed on NaN or infinite values.
// A mismatch is a *plugintesting.ConservationError stating expected, actual,
// and difference.
func CheckConservation(req *pbc.AllocateRequest, resp *pbc.AllocateResponse, relEpsilon float64) error {
	return plugintesting.CheckConservation(req, resp, relEpsilon)
}

// ValidateAllocateResponse is identical to the sdk/go/testing function of the
// same name. Hosts call it with CheckConservation on every AllocateResponse: it
// checks every response invariant other than conservation (effective policy and
// digest present, row kinds, idle-row node keys, finite non-negative costs,
// portion totals, currency, and exactly one idle row per priced node). Every
// failure wraps plugintesting.ErrInvalidAllocateResponse.
func ValidateAllocateResponse(req *pbc.AllocateRequest, resp *pbc.AllocateResponse) error {
	return plugintesting.ValidateAllocateResponse(req, resp)
}

// allocatorGRPCServer adapts an AllocatorProvider to the generated gRPC server
// interface, so plugins need not embed the Unimplemented server.
type allocatorGRPCServer struct {
	pbc.UnimplementedAllocatorServiceServer

	provider AllocatorProvider
}

func (s *allocatorGRPCServer) Allocate(
	ctx context.Context, req *pbc.AllocateRequest,
) (*pbc.AllocateResponse, error) {
	return s.provider.Allocate(ctx, req)
}

// allocatorConnectHandler adapts an AllocatorProvider to the generated Connect
// handler interface, preserving gRPC status codes via toConnectError.
type allocatorConnectHandler struct {
	provider AllocatorProvider
}

func (h *allocatorConnectHandler) Allocate(
	ctx context.Context, req *connect.Request[pbc.AllocateRequest],
) (*connect.Response[pbc.AllocateResponse], error) {
	resp, err := h.provider.Allocate(ctx, req.Msg)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(resp), nil
}
