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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

// ValidateContractCommitment is identical to the sdk/go/testing function of
// the same name, and is the rule set ContractCommitmentBuilder.Build applies:
// required IDs and currency, SPEND or USAGE category, ISO 4217 currency,
// ordered periods, and finite non-negative amounts. Failures wrap
// plugintesting.ErrInvalidContractCommitment and carry codes.InvalidArgument.
func ValidateContractCommitment(c *pbc.ContractCommitment) error {
	return plugintesting.ValidateContractCommitment(c)
}

// ValidateGetContractCommitmentsRequest is identical to the sdk/go/testing
// function of the same name: start and end set together or not at all, valid
// timestamps with end after start, and a non-negative page_size. Providers call
// it first and may return its error unchanged (codes.InvalidArgument).
func ValidateGetContractCommitmentsRequest(req *pbc.GetContractCommitmentsRequest) error {
	return plugintesting.ValidateGetContractCommitmentsRequest(req)
}

// ValidateGetContractCommitmentsResponse is identical to the sdk/go/testing
// function of the same name. Hosts call it on every response: it checks the
// page size bound, each record's validity and window match, unique
// contract_commitment_id values, and total_count. Failures wrap
// plugintesting.ErrInvalidContractCommitmentsResponse.
func ValidateGetContractCommitmentsResponse(
	req *pbc.GetContractCommitmentsRequest, resp *pbc.GetContractCommitmentsResponse,
) error {
	return plugintesting.ValidateGetContractCommitmentsResponse(req, resp)
}

// ContractCommitmentMatchesWindow is identical to the sdk/go/testing function
// of the same name: it reports whether c's period (the commitment period, or
// the contract period when neither commitment bound is set) overlaps the
// half-open window [start, end). Unset bounds are open-ended; a nil start or
// end means no window.
func ContractCommitmentMatchesWindow(c *pbc.ContractCommitment, start, end *timestamppb.Timestamp) bool {
	return plugintesting.ContractCommitmentMatchesWindow(c, start, end)
}

// PaginateContractCommitments is identical to the sdk/go/testing function of
// the same name. It returns one page of an already filtered, stably ordered
// list, the next page token (empty on the last page), and the total count.
// A page_size of 0 means DefaultPageSize and values above MaxPageSize mean
// MaxPageSize. Tokens use the EncodePageToken format; a negative page size or
// malformed token fails with codes.InvalidArgument.
func PaginateContractCommitments(
	commitments []*pbc.ContractCommitment, pageSize int32, pageToken string,
) ([]*pbc.ContractCommitment, string, int32, error) {
	return plugintesting.PaginateContractCommitments(commitments, pageSize, pageToken)
}

// ValidateGetBillingPeriodsRequest is identical to the sdk/go/testing function
// of the same name. Providers call it first and may return its error unchanged.
func ValidateGetBillingPeriodsRequest(req *pbc.GetBillingPeriodsRequest) error {
	return plugintesting.ValidateGetBillingPeriodsRequest(req)
}

// ValidateGetInvoiceDetailsRequest is identical to the sdk/go/testing function
// of the same name.
func ValidateGetInvoiceDetailsRequest(req *pbc.GetInvoiceDetailsRequest) error {
	return plugintesting.ValidateGetInvoiceDetailsRequest(req)
}

// ValidateGetBillingPeriodsResponse is identical to the sdk/go/testing function
// of the same name. Hosts call it on every response.
func ValidateGetBillingPeriodsResponse(
	req *pbc.GetBillingPeriodsRequest, resp *pbc.GetBillingPeriodsResponse,
) error {
	return plugintesting.ValidateGetBillingPeriodsResponse(req, resp)
}

// ValidateGetInvoiceDetailsResponse is identical to the sdk/go/testing function
// of the same name. Hosts call it on every response.
func ValidateGetInvoiceDetailsResponse(
	req *pbc.GetInvoiceDetailsRequest, resp *pbc.GetInvoiceDetailsResponse,
) error {
	return plugintesting.ValidateGetInvoiceDetailsResponse(req, resp)
}

// BillingPeriodMatchesWindow is identical to the sdk/go/testing function of
// the same name.
func BillingPeriodMatchesWindow(p *pbc.BillingPeriod, start, end *timestamppb.Timestamp) bool {
	return plugintesting.BillingPeriodMatchesWindow(p, start, end)
}

// InvoiceDetailMatchesWindow is identical to the sdk/go/testing function of
// the same name.
func InvoiceDetailMatchesWindow(d *pbc.InvoiceDetail, start, end *timestamppb.Timestamp) bool {
	return plugintesting.InvoiceDetailMatchesWindow(d, start, end)
}

// PaginateBillingPeriods is identical to the sdk/go/testing function of the
// same name.
func PaginateBillingPeriods(
	periods []*pbc.BillingPeriod, pageSize int32, pageToken string,
) ([]*pbc.BillingPeriod, string, int32, error) {
	return plugintesting.PaginateBillingPeriods(periods, pageSize, pageToken)
}

// PaginateInvoiceDetails is identical to the sdk/go/testing function of the
// same name.
func PaginateInvoiceDetails(
	details []*pbc.InvoiceDetail, pageSize int32, pageToken string,
) ([]*pbc.InvoiceDetail, string, int32, error) {
	return plugintesting.PaginateInvoiceDetails(details, pageSize, pageToken)
}

func unimplementedDataset(method string) error {
	return status.Errorf(codes.Unimplemented, "method %s not implemented", method)
}

// supplementalGRPCServer adapts the supplemental providers a plugin implements.
// A nil provider returns codes.Unimplemented for that RPC. The other RPCs
// still serve.
type supplementalGRPCServer struct {
	pbc.UnimplementedSupplementalDatasetServiceServer

	commitments ContractCommitmentProvider
	invoices    InvoiceDatasetProvider
}

func (s *supplementalGRPCServer) GetContractCommitments(
	ctx context.Context, req *pbc.GetContractCommitmentsRequest,
) (*pbc.GetContractCommitmentsResponse, error) {
	if s.commitments == nil {
		return nil, unimplementedDataset("GetContractCommitments")
	}
	return s.commitments.GetContractCommitments(ctx, req)
}

func (s *supplementalGRPCServer) GetBillingPeriods(
	ctx context.Context, req *pbc.GetBillingPeriodsRequest,
) (*pbc.GetBillingPeriodsResponse, error) {
	if s.invoices == nil {
		return nil, unimplementedDataset("GetBillingPeriods")
	}
	return s.invoices.GetBillingPeriods(ctx, req)
}

func (s *supplementalGRPCServer) GetInvoiceDetails(
	ctx context.Context, req *pbc.GetInvoiceDetailsRequest,
) (*pbc.GetInvoiceDetailsResponse, error) {
	if s.invoices == nil {
		return nil, unimplementedDataset("GetInvoiceDetails")
	}
	return s.invoices.GetInvoiceDetails(ctx, req)
}

// supplementalConnectHandler adapts the same providers to Connect, preserving
// gRPC status codes via toConnectError.
type supplementalConnectHandler struct {
	commitments ContractCommitmentProvider
	invoices    InvoiceDatasetProvider
}

func (h *supplementalConnectHandler) GetContractCommitments(
	ctx context.Context, req *connect.Request[pbc.GetContractCommitmentsRequest],
) (*connect.Response[pbc.GetContractCommitmentsResponse], error) {
	if h.commitments == nil {
		return nil, toConnectError(unimplementedDataset("GetContractCommitments"))
	}
	resp, err := h.commitments.GetContractCommitments(ctx, req.Msg)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(resp), nil
}

func (h *supplementalConnectHandler) GetBillingPeriods(
	ctx context.Context, req *connect.Request[pbc.GetBillingPeriodsRequest],
) (*connect.Response[pbc.GetBillingPeriodsResponse], error) {
	if h.invoices == nil {
		return nil, toConnectError(unimplementedDataset("GetBillingPeriods"))
	}
	resp, err := h.invoices.GetBillingPeriods(ctx, req.Msg)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(resp), nil
}

func (h *supplementalConnectHandler) GetInvoiceDetails(
	ctx context.Context, req *connect.Request[pbc.GetInvoiceDetailsRequest],
) (*connect.Response[pbc.GetInvoiceDetailsResponse], error) {
	if h.invoices == nil {
		return nil, toConnectError(unimplementedDataset("GetInvoiceDetails"))
	}
	resp, err := h.invoices.GetInvoiceDetails(ctx, req.Msg)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(resp), nil
}
