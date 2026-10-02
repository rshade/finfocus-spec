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

package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// InvoiceDatasetServer is satisfied by any type with GetBillingPeriods and
// GetInvoiceDetails, including pluginsdk.InvoiceDatasetProvider plugins and
// MockInvoiceDatasetSource.
type InvoiceDatasetServer interface {
	GetBillingPeriods(ctx context.Context, req *pbc.GetBillingPeriodsRequest) (*pbc.GetBillingPeriodsResponse, error)
	GetInvoiceDetails(ctx context.Context, req *pbc.GetInvoiceDetailsRequest) (*pbc.GetInvoiceDetailsResponse, error)
}

type invoiceDatasetAdapter struct {
	pbc.UnimplementedSupplementalDatasetServiceServer

	impl InvoiceDatasetServer
}

func (a *invoiceDatasetAdapter) GetBillingPeriods(
	ctx context.Context, req *pbc.GetBillingPeriodsRequest,
) (*pbc.GetBillingPeriodsResponse, error) {
	return a.impl.GetBillingPeriods(ctx, req)
}

func (a *invoiceDatasetAdapter) GetInvoiceDetails(
	ctx context.Context, req *pbc.GetInvoiceDetailsRequest,
) (*pbc.GetInvoiceDetailsResponse, error) {
	return a.impl.GetInvoiceDetails(ctx, req)
}

// InvoiceDatasetHarness serves an InvoiceDatasetServer as
// SupplementalDatasetService over an in-memory bufconn.
type InvoiceDatasetHarness struct {
	bufconnHarness[pbc.SupplementalDatasetServiceClient]
}

// NewInvoiceDatasetHarness creates a harness serving impl as
// SupplementalDatasetService. GetContractCommitments stays unimplemented.
func NewInvoiceDatasetHarness(impl InvoiceDatasetServer) *InvoiceDatasetHarness {
	return &InvoiceDatasetHarness{newBufconnHarness(func(s *grpc.Server) {
		pbc.RegisterSupplementalDatasetServiceServer(s, &invoiceDatasetAdapter{impl: impl})
	}, pbc.NewSupplementalDatasetServiceClient)}
}

// Client returns the SupplementalDatasetService client; call Start first.
func (h *InvoiceDatasetHarness) Client() pbc.SupplementalDatasetServiceClient {
	return h.client
}

// RunInvoiceDatasetConformance serves impl over an InvoiceDatasetHarness and
// runs the billing-period and invoice-detail scenarios as subtests. The
// scenarios check the source's own data: valid pages, unique keys across a
// walk, an exact total_count, stable order, window matching, and
// InvalidArgument for a one-sided window, an inverted window, a negative page
// size, and a malformed token. They pass for an empty source too.
func RunInvoiceDatasetConformance(t *testing.T, impl InvoiceDatasetServer) {
	t.Helper()
	harness := NewInvoiceDatasetHarness(impl)
	harness.Start(t)
	defer harness.Stop()

	client := harness.Client()
	t.Run("billing_periods", func(t *testing.T) {
		runDatasetScenarios(t, func(ctx context.Context, name string) error {
			return runBillingPeriodScenario(ctx, client, name)
		})
	})
	t.Run("invoice_details", func(t *testing.T) {
		runDatasetScenarios(t, func(ctx context.Context, name string) error {
			return runInvoiceDetailScenario(ctx, client, name)
		})
	})
}

func runDatasetScenarios(t *testing.T, run func(context.Context, string) error) {
	t.Helper()
	names := []string{
		nameFullWalk, nameStableOrder, nameWindowFilter,
		nameWindowOneBound, nameWindowInverted, nameNegativePageSize, nameMalformedPage,
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), commitmentScenarioTimeout)
			defer cancel()
			if err := run(ctx, name); err != nil {
				t.Error(err)
			}
		})
	}
}

func runBillingPeriodScenario(
	ctx context.Context, client pbc.SupplementalDatasetServiceClient, name string,
) error {
	switch name {
	case nameFullWalk:
		_, err := fullBillingPeriodWalk(ctx, client)
		return err
	case nameStableOrder:
		return billingPeriodStableOrder(ctx, client)
	case nameWindowFilter:
		return billingPeriodWindowFilter(ctx, client)
	default:
		return rejectBillingPeriodRequest(ctx, client, name)
	}
}

func runInvoiceDetailScenario(
	ctx context.Context, client pbc.SupplementalDatasetServiceClient, name string,
) error {
	switch name {
	case nameFullWalk:
		_, err := fullInvoiceDetailWalk(ctx, client)
		return err
	case nameStableOrder:
		return invoiceDetailStableOrder(ctx, client)
	case nameWindowFilter:
		return invoiceDetailWindowFilter(ctx, client)
	default:
		return rejectInvoiceDetailRequest(ctx, client, name)
	}
}

type keyedWalk struct {
	keys  []string
	total int32
	seen  map[string]int
	empty int
}

func (w *keyedWalk) add(page int, total int32, keys []string, next string) error {
	if page == 0 {
		w.total = total
	} else if total != w.total {
		return fmt.Errorf("page %d: total_count %d differs from the first page's %d", page, total, w.total)
	}
	for _, key := range keys {
		if first, dup := w.seen[key]; dup {
			return fmt.Errorf("page %d: key %q was already returned on page %d", page, key, first)
		}
		w.seen[key] = page
		w.keys = append(w.keys, key)
	}
	if len(keys) > 0 || next == "" {
		w.empty = 0
		return nil
	}
	w.empty++
	if w.empty >= maxConsecutiveEmptyPages {
		return fmt.Errorf("%d consecutive empty pages still carried a next_page_token", w.empty)
	}
	return nil
}

func fullBillingPeriodWalk(
	ctx context.Context, client pbc.SupplementalDatasetServiceClient,
) (keyedWalk, error) {
	walk := keyedWalk{seen: make(map[string]int)}
	req := &pbc.GetBillingPeriodsRequest{PageSize: MaxPageSize}
	for page := range maxCommitmentWalkPages {
		resp, err := client.GetBillingPeriods(ctx, req)
		if err != nil {
			return walk, fmt.Errorf("page %d: %w", page, err)
		}
		if err = ValidateGetBillingPeriodsResponse(req, resp); err != nil {
			return walk, fmt.Errorf("page %d: %w", page, err)
		}
		keys := make([]string, len(resp.GetBillingPeriods()))
		for i, p := range resp.GetBillingPeriods() {
			keys[i] = billingPeriodKeyText(p)
		}
		if err = walk.add(page, resp.GetTotalCount(), keys, resp.GetNextPageToken()); err != nil {
			return walk, err
		}
		if resp.GetNextPageToken() == "" {
			break
		}
		req.PageToken = resp.GetNextPageToken()
		if page == maxCommitmentWalkPages-1 {
			return walk, fmt.Errorf("page walk did not end after %d pages", maxCommitmentWalkPages)
		}
	}
	if int(walk.total) != len(walk.keys) {
		return walk, fmt.Errorf(
			"total_count %d does not match the %d billing periods returned",
			walk.total,
			len(walk.keys),
		)
	}
	return walk, nil
}

func billingPeriodKeyText(p *pbc.BillingPeriod) string {
	start := p.GetBillingPeriodStart()
	return fmt.Sprintf("%s/%d.%d", p.GetInvoiceIssuerName(), start.GetSeconds(), start.GetNanos())
}

func billingPeriodStableOrder(ctx context.Context, client pbc.SupplementalDatasetServiceClient) error {
	full, err := fullBillingPeriodWalk(ctx, client)
	if err != nil {
		return fmt.Errorf("full walk: %w", err)
	}
	req := &pbc.GetBillingPeriodsRequest{PageSize: 1}
	var got []string
	for page := range stableOrderPages {
		resp, callErr := client.GetBillingPeriods(ctx, req)
		if callErr != nil {
			return fmt.Errorf("page %d: %w", page, callErr)
		}
		if err = ValidateGetBillingPeriodsResponse(req, resp); err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		for _, p := range resp.GetBillingPeriods() {
			got = append(got, billingPeriodKeyText(p))
		}
		if resp.GetNextPageToken() == "" {
			break
		}
		req.PageToken = resp.GetNextPageToken()
	}
	want := min(len(full.keys), stableOrderPages)
	if len(got) != want {
		return fmt.Errorf("page_size 1 returned %d billing periods, want %d", len(got), want)
	}
	for i, key := range got {
		if key != full.keys[i] {
			return fmt.Errorf("page_size 1 position %d is %q, but the full walk has %q there", i, key, full.keys[i])
		}
	}
	return nil
}

func billingPeriodWindowFilter(ctx context.Context, client pbc.SupplementalDatasetServiceClient) error {
	full, err := fullBillingPeriodWalk(ctx, client)
	if err != nil {
		return fmt.Errorf("full walk: %w", err)
	}
	known := make(map[string]struct{}, len(full.keys))
	for _, key := range full.keys {
		known[key] = struct{}{}
	}
	for _, window := range conformanceWindows() {
		req := &pbc.GetBillingPeriodsRequest{
			Start:    timestamppb.New(window[0]),
			End:      timestamppb.New(window[1]),
			PageSize: MaxPageSize,
		}
		resp, callErr := client.GetBillingPeriods(ctx, req)
		if callErr != nil {
			return callErr
		}
		if err = ValidateGetBillingPeriodsResponse(req, resp); err != nil {
			return err
		}
		for _, p := range resp.GetBillingPeriods() {
			key := billingPeriodKeyText(p)
			if _, ok := known[key]; !ok {
				return fmt.Errorf("window returned %q, which the unfiltered walk did not", key)
			}
		}
	}
	return nil
}

func rejectBillingPeriodRequest(ctx context.Context, client pbc.SupplementalDatasetServiceClient, name string) error {
	req := badBillingPeriodRequest(name)
	_, err := client.GetBillingPeriods(ctx, req)
	return wantInvalidArgument(err, name)
}

func badBillingPeriodRequest(name string) *pbc.GetBillingPeriodsRequest {
	start := conformanceWindowStart()
	switch name {
	case nameWindowOneBound:
		return &pbc.GetBillingPeriodsRequest{Start: timestamppb.New(start)}
	case nameWindowInverted:
		return &pbc.GetBillingPeriodsRequest{
			Start: timestamppb.New(start),
			End:   timestamppb.New(start.Add(-24 * time.Hour)),
		}
	case nameNegativePageSize:
		return &pbc.GetBillingPeriodsRequest{PageSize: -1}
	default:
		return &pbc.GetBillingPeriodsRequest{PageToken: malformedPageToken}
	}
}

func fullInvoiceDetailWalk(ctx context.Context, client pbc.SupplementalDatasetServiceClient) (keyedWalk, error) {
	walk := keyedWalk{seen: make(map[string]int)}
	req := &pbc.GetInvoiceDetailsRequest{PageSize: MaxPageSize}
	for page := range maxCommitmentWalkPages {
		resp, err := client.GetInvoiceDetails(ctx, req)
		if err != nil {
			return walk, fmt.Errorf("page %d: %w", page, err)
		}
		if err = ValidateGetInvoiceDetailsResponse(req, resp); err != nil {
			return walk, fmt.Errorf("page %d: %w", page, err)
		}
		keys := make([]string, len(resp.GetInvoiceDetails()))
		for i, d := range resp.GetInvoiceDetails() {
			keys[i] = d.GetInvoiceDetailId()
		}
		if err = walk.add(page, resp.GetTotalCount(), keys, resp.GetNextPageToken()); err != nil {
			return walk, err
		}
		if resp.GetNextPageToken() == "" {
			break
		}
		req.PageToken = resp.GetNextPageToken()
		if page == maxCommitmentWalkPages-1 {
			return walk, fmt.Errorf("page walk did not end after %d pages", maxCommitmentWalkPages)
		}
	}
	if int(walk.total) != len(walk.keys) {
		return walk, fmt.Errorf(
			"total_count %d does not match the %d invoice details returned",
			walk.total,
			len(walk.keys),
		)
	}
	return walk, nil
}

func invoiceDetailStableOrder(ctx context.Context, client pbc.SupplementalDatasetServiceClient) error {
	full, err := fullInvoiceDetailWalk(ctx, client)
	if err != nil {
		return fmt.Errorf("full walk: %w", err)
	}
	req := &pbc.GetInvoiceDetailsRequest{PageSize: 1}
	var got []string
	for page := range stableOrderPages {
		resp, callErr := client.GetInvoiceDetails(ctx, req)
		if callErr != nil {
			return fmt.Errorf("page %d: %w", page, callErr)
		}
		if err = ValidateGetInvoiceDetailsResponse(req, resp); err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		for _, d := range resp.GetInvoiceDetails() {
			got = append(got, d.GetInvoiceDetailId())
		}
		if resp.GetNextPageToken() == "" {
			break
		}
		req.PageToken = resp.GetNextPageToken()
	}
	want := min(len(full.keys), stableOrderPages)
	if len(got) != want {
		return fmt.Errorf("page_size 1 returned %d invoice details, want %d", len(got), want)
	}
	for i, key := range got {
		if key != full.keys[i] {
			return fmt.Errorf("page_size 1 position %d is %q, but the full walk has %q there", i, key, full.keys[i])
		}
	}
	return nil
}

func invoiceDetailWindowFilter(ctx context.Context, client pbc.SupplementalDatasetServiceClient) error {
	full, err := fullInvoiceDetailWalk(ctx, client)
	if err != nil {
		return fmt.Errorf("full walk: %w", err)
	}
	known := make(map[string]struct{}, len(full.keys))
	for _, key := range full.keys {
		known[key] = struct{}{}
	}
	for _, window := range conformanceWindows() {
		req := &pbc.GetInvoiceDetailsRequest{
			Start: timestamppb.New(window[0]), End: timestamppb.New(window[1]), PageSize: MaxPageSize,
		}
		resp, callErr := client.GetInvoiceDetails(ctx, req)
		if callErr != nil {
			return callErr
		}
		if err = ValidateGetInvoiceDetailsResponse(req, resp); err != nil {
			return err
		}
		for _, d := range resp.GetInvoiceDetails() {
			if _, ok := known[d.GetInvoiceDetailId()]; !ok {
				return fmt.Errorf("window returned %q, which the unfiltered walk did not", d.GetInvoiceDetailId())
			}
		}
	}
	return nil
}

func rejectInvoiceDetailRequest(ctx context.Context, client pbc.SupplementalDatasetServiceClient, name string) error {
	start := conformanceWindowStart()
	var req *pbc.GetInvoiceDetailsRequest
	switch name {
	case nameWindowOneBound:
		req = &pbc.GetInvoiceDetailsRequest{Start: timestamppb.New(start)}
	case nameWindowInverted:
		req = &pbc.GetInvoiceDetailsRequest{
			Start: timestamppb.New(start),
			End:   timestamppb.New(start.Add(-24 * time.Hour)),
		}
	case nameNegativePageSize:
		req = &pbc.GetInvoiceDetailsRequest{PageSize: -1}
	default:
		req = &pbc.GetInvoiceDetailsRequest{PageToken: malformedPageToken}
	}
	_, err := client.GetInvoiceDetails(ctx, req)
	return wantInvalidArgument(err, name)
}

func conformanceWindows() [][2]time.Time {
	return [][2]time.Time{
		{time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(1900, 2, 1, 0, 0, 0, 0, time.UTC)},
		{time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
}

func wantInvalidArgument(err error, what string) error {
	if err == nil {
		return fmt.Errorf("%s was accepted; want InvalidArgument", what)
	}
	if code := status.Code(err); code != codes.InvalidArgument {
		return fmt.Errorf("%s returned %s; want InvalidArgument: %w", what, code, err)
	}
	return nil
}
