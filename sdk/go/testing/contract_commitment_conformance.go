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
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	commitmentScenarioTimeout = 30 * time.Second
	// maxCommitmentWalkPages bounds a page walk so a source that never ends
	// its token chain fails instead of hanging.
	maxCommitmentWalkPages = 10000
	// maxConsecutiveEmptyPages matches the TypeScript iterator's guard.
	maxConsecutiveEmptyPages = 10
	// stableOrderPages is how many page-size-1 pages stable_order compares.
	stableOrderPages   = 25
	malformedPageToken = "!not-a-token!"

	nameFullWalk         = "full_walk"
	nameStableOrder      = "stable_order"
	nameWindowFilter     = "window_filter"
	nameWindowOneBound   = "window_one_bound"
	nameWindowInverted   = "window_inverted"
	nameNegativePageSize = "negative_page_size"
	nameMalformedPage    = "malformed_page_token"
)

// ContractCommitmentServer is satisfied by any type with a
// GetContractCommitments method, including pbc.SupplementalDatasetServiceServer
// implementations, pluginsdk.ContractCommitmentProvider plugins, and
// MockContractCommitmentSource.
type ContractCommitmentServer interface {
	GetContractCommitments(ctx context.Context, req *pbc.GetContractCommitmentsRequest) (
		*pbc.GetContractCommitmentsResponse, error)
}

type contractCommitmentAdapter struct {
	pbc.UnimplementedSupplementalDatasetServiceServer

	impl ContractCommitmentServer
}

func (a *contractCommitmentAdapter) GetContractCommitments(
	ctx context.Context, req *pbc.GetContractCommitmentsRequest,
) (*pbc.GetContractCommitmentsResponse, error) {
	return a.impl.GetContractCommitments(ctx, req)
}

// ContractCommitmentHarness serves a ContractCommitmentServer as
// SupplementalDatasetService over an in-memory bufconn, so calls exercise
// proto serialization and the status codes clients really see.
type ContractCommitmentHarness struct {
	bufconnHarness[pbc.SupplementalDatasetServiceClient]
}

// NewContractCommitmentHarness creates a harness serving impl as
// SupplementalDatasetService.
func NewContractCommitmentHarness(impl ContractCommitmentServer) *ContractCommitmentHarness {
	return &ContractCommitmentHarness{newBufconnHarness(func(s *grpc.Server) {
		pbc.RegisterSupplementalDatasetServiceServer(s, &contractCommitmentAdapter{impl: impl})
	}, pbc.NewSupplementalDatasetServiceClient)}
}

// Client returns the SupplementalDatasetService client; call Start first.
func (h *ContractCommitmentHarness) Client() pbc.SupplementalDatasetServiceClient {
	return h.client
}

// commitmentWalk is the result of following next_page_token to the end.
type commitmentWalk struct {
	ids   []string
	total int32
	seen  map[string]int
	empty int
}

// addPage checks one page against req and the pages before it: the response
// passes ValidateGetContractCommitmentsResponse, reports the first page's
// total, repeats no earlier ID, and is not one of too many consecutive empty
// pages that still carry a token.
func (w *commitmentWalk) addPage(
	page int, req *pbc.GetContractCommitmentsRequest, resp *pbc.GetContractCommitmentsResponse,
) error {
	if err := ValidateGetContractCommitmentsResponse(req, resp); err != nil {
		return fmt.Errorf("page %d: %w", page, err)
	}
	if page == 0 {
		w.total = resp.GetTotalCount()
	} else if resp.GetTotalCount() != w.total {
		return fmt.Errorf("page %d: total_count %d differs from the first page's %d",
			page, resp.GetTotalCount(), w.total)
	}
	for _, c := range resp.GetCommitments() {
		id := c.GetContractCommitmentId()
		if first, dup := w.seen[id]; dup {
			return fmt.Errorf("page %d: contract_commitment_id %q was already returned on page %d", page, id, first)
		}
		w.seen[id] = page
		w.ids = append(w.ids, id)
	}
	if len(resp.GetCommitments()) > 0 || resp.GetNextPageToken() == "" {
		w.empty = 0
		return nil
	}
	w.empty++
	if w.empty >= maxConsecutiveEmptyPages {
		return fmt.Errorf("%d consecutive empty pages still carried a next_page_token", w.empty)
	}
	return nil
}

// walkCommitments follows the token chain from base, checking every page with
// commitmentWalk.addPage. It stops after maxPages pages when maxPages > 0 and
// fails after maxCommitmentWalkPages pages.
func walkCommitments(
	ctx context.Context, client pbc.SupplementalDatasetServiceClient,
	base *pbc.GetContractCommitmentsRequest, maxPages int,
) (commitmentWalk, error) {
	walk := commitmentWalk{seen: make(map[string]int)}
	req := &pbc.GetContractCommitmentsRequest{Start: base.GetStart(), End: base.GetEnd(), PageSize: base.GetPageSize()}
	for page := range maxCommitmentWalkPages {
		resp, err := client.GetContractCommitments(ctx, req)
		if err != nil {
			return walk, fmt.Errorf("page %d: %w", page, err)
		}
		if err = walk.addPage(page, req, resp); err != nil {
			return walk, err
		}
		if resp.GetNextPageToken() == "" || (maxPages > 0 && page+1 >= maxPages) {
			return walk, nil
		}
		req.PageToken = resp.GetNextPageToken()
	}
	return walk, fmt.Errorf("page walk did not end after %d pages", maxCommitmentWalkPages)
}

func fullWalk(ctx context.Context, client pbc.SupplementalDatasetServiceClient) (commitmentWalk, error) {
	walk, err := walkCommitments(ctx, client, &pbc.GetContractCommitmentsRequest{PageSize: MaxPageSize}, 0)
	if err != nil {
		return walk, err
	}
	if int(walk.total) != len(walk.ids) {
		return walk, fmt.Errorf("total_count %d does not match the %d commitments returned across all pages",
			walk.total, len(walk.ids))
	}
	return walk, nil
}

func scenarioFullWalk(ctx context.Context, client pbc.SupplementalDatasetServiceClient) error {
	_, err := fullWalk(ctx, client)
	return err
}

func scenarioStableOrder(ctx context.Context, client pbc.SupplementalDatasetServiceClient) error {
	full, err := fullWalk(ctx, client)
	if err != nil {
		return fmt.Errorf("full walk: %w", err)
	}
	small, err := walkCommitments(ctx, client, &pbc.GetContractCommitmentsRequest{PageSize: 1}, stableOrderPages)
	if err != nil {
		return fmt.Errorf("page_size 1: %w", err)
	}
	want := min(len(full.ids), stableOrderPages)
	if len(small.ids) != want {
		return fmt.Errorf("page_size 1 returned %d commitments in %d pages, want %d", len(small.ids),
			stableOrderPages, want)
	}
	for i, id := range small.ids {
		if id != full.ids[i] {
			return fmt.Errorf("page_size 1 position %d is %q, but the full walk has %q there", i, id, full.ids[i])
		}
	}
	return nil
}

func scenarioWindowFilter(ctx context.Context, client pbc.SupplementalDatasetServiceClient) error {
	full, err := fullWalk(ctx, client)
	if err != nil {
		return fmt.Errorf("full walk: %w", err)
	}
	all := make(map[string]struct{}, len(full.ids))
	for _, id := range full.ids {
		all[id] = struct{}{}
	}
	windows := [][2]time.Time{
		{time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(1900, 2, 1, 0, 0, 0, 0, time.UTC)},
		{time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, w := range windows {
		req := &pbc.GetContractCommitmentsRequest{
			Start: timestamppb.New(w[0]), End: timestamppb.New(w[1]), PageSize: MaxPageSize,
		}
		walk, walkErr := walkCommitments(ctx, client, req, 0)
		if walkErr != nil {
			return fmt.Errorf("window %s to %s: %w", w[0].Format(time.DateOnly), w[1].Format(time.DateOnly), walkErr)
		}
		for _, id := range walk.ids {
			if _, ok := all[id]; !ok {
				return fmt.Errorf("window %s to %s returned %q, which the unfiltered walk did not",
					w[0].Format(time.DateOnly), w[1].Format(time.DateOnly), id)
			}
		}
	}
	return nil
}

func expectInvalidArgument(
	ctx context.Context, client pbc.SupplementalDatasetServiceClient, req *pbc.GetContractCommitmentsRequest,
	what string,
) error {
	_, err := client.GetContractCommitments(ctx, req)
	if err == nil {
		return fmt.Errorf("%s was accepted; want InvalidArgument", what)
	}
	if code := status.Code(err); code != codes.InvalidArgument {
		return fmt.Errorf("%s returned %s; want InvalidArgument: %w", what, code, err)
	}
	return nil
}

func conformanceWindowStart() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }

func scenarioWindowOneBound(ctx context.Context, client pbc.SupplementalDatasetServiceClient) error {
	req := &pbc.GetContractCommitmentsRequest{Start: timestamppb.New(conformanceWindowStart())}
	return expectInvalidArgument(ctx, client, req, "a window with only start set")
}

func scenarioWindowInverted(ctx context.Context, client pbc.SupplementalDatasetServiceClient) error {
	start := conformanceWindowStart()
	req := &pbc.GetContractCommitmentsRequest{
		Start: timestamppb.New(start), End: timestamppb.New(start.Add(-24 * time.Hour)),
	}
	return expectInvalidArgument(ctx, client, req, "a window whose end is before its start")
}

func scenarioNegativePageSize(ctx context.Context, client pbc.SupplementalDatasetServiceClient) error {
	return expectInvalidArgument(ctx, client, &pbc.GetContractCommitmentsRequest{PageSize: -1}, "page_size -1")
}

func scenarioMalformedPageToken(ctx context.Context, client pbc.SupplementalDatasetServiceClient) error {
	req := &pbc.GetContractCommitmentsRequest{PageToken: malformedPageToken}
	return expectInvalidArgument(ctx, client, req, fmt.Sprintf("page_token %q", malformedPageToken))
}

type commitmentScenario struct {
	name string
	run  func(ctx context.Context, client pbc.SupplementalDatasetServiceClient) error
}

// commitmentScenarios lists the conformance scenarios in suite order.
func commitmentScenarios() []commitmentScenario {
	return []commitmentScenario{
		{name: nameFullWalk, run: scenarioFullWalk},
		{name: nameStableOrder, run: scenarioStableOrder},
		{name: nameWindowFilter, run: scenarioWindowFilter},
		{name: nameWindowOneBound, run: scenarioWindowOneBound},
		{name: nameWindowInverted, run: scenarioWindowInverted},
		{name: nameNegativePageSize, run: scenarioNegativePageSize},
		{name: nameMalformedPage, run: scenarioMalformedPageToken},
	}
}

func runCommitmentScenario(
	ctx context.Context, client pbc.SupplementalDatasetServiceClient, s commitmentScenario,
) error {
	ctx, cancel := context.WithTimeout(ctx, commitmentScenarioTimeout)
	defer cancel()
	err := s.run(ctx, client)
	if errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded {
		return fmt.Errorf("scenario timed out after %s: %w", commitmentScenarioTimeout, err)
	}
	return err
}

// runContractCommitmentScenarios runs every scenario against client and returns
// each scenario's error (nil on success) keyed by subtest name.
func runContractCommitmentScenarios(
	ctx context.Context, client pbc.SupplementalDatasetServiceClient,
) map[string]error {
	scenarios := commitmentScenarios()
	results := make(map[string]error, len(scenarios))
	for _, s := range scenarios {
		results[s.name] = runCommitmentScenario(ctx, client, s)
	}
	return results
}

// RunContractCommitmentConformance serves impl over a ContractCommitmentHarness
// and runs the contract commitment scenarios as subtests. The scenarios are
// source-agnostic: they check the source's own data for consistency, never
// specific records, so they pass for an empty source too. The subtests are:
//
//   - full_walk: no window, page size MaxPageSize; every page passes
//     ValidateGetContractCommitmentsResponse, no contract_commitment_id repeats
//     across pages, and total_count equals the number walked on every page
//   - stable_order: page size 1 for up to 25 pages returns the same IDs in the
//     same order as full_walk
//   - window_filter: windows [1900-01-01, 1900-02-01) and
//     [2000-01-01, 2100-01-01) return only records matching the window
//     (ContractCommitmentMatchesWindow) and only records full_walk returned
//   - window_one_bound: only start set is rejected with InvalidArgument
//   - window_inverted: end before start is rejected with InvalidArgument
//   - negative_page_size: page_size -1 is rejected with InvalidArgument
//   - malformed_page_token: page_token "!not-a-token!" is rejected with
//     InvalidArgument
func RunContractCommitmentConformance(t *testing.T, impl ContractCommitmentServer) {
	t.Helper()
	harness := NewContractCommitmentHarness(impl)
	harness.Start(t)
	defer harness.Stop()

	for _, s := range commitmentScenarios() {
		t.Run(s.name, func(t *testing.T) {
			if err := runCommitmentScenario(context.Background(), harness.Client(), s); err != nil {
				t.Error(err)
			}
		})
	}
}
