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
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	// DefaultConservationEpsilon is the relative tolerance hosts and the
	// conformance suite use when checking allocation conservation.
	DefaultConservationEpsilon = 1e-6
	// ConservationAbsoluteFloor is the absolute tolerance floor, so totals
	// near zero compare equal.
	ConservationAbsoluteFloor = 1e-9

	defaultAllocationCurrency = "USD"
)

var (
	// ErrConservation is wrapped by every *ConservationError.
	ErrConservation = errors.New("allocation does not conserve cost")
	// ErrInvalidAllocateRequest is wrapped by every ValidateAllocateRequest failure.
	ErrInvalidAllocateRequest = errors.New("invalid allocate request")
	// ErrInvalidAllocateResponse is wrapped by every ValidateAllocateResponse failure.
	ErrInvalidAllocateResponse = errors.New("invalid allocate response")
	// ErrMixedCurrency is wrapped by ResolveCurrency failures.
	ErrMixedCurrency = errors.New("mixed currencies")
)

// invalidArgumentError is a plain error that also carries codes.InvalidArgument,
// so allocators can return it directly and gRPC and Connect clients both see
// InvalidArgument, while Error() stays free of the "rpc error:" prefix.
type invalidArgumentError struct {
	msg     string
	wrapped error
}

func (e *invalidArgumentError) Error() string { return e.msg }

func (e *invalidArgumentError) Unwrap() error { return e.wrapped }

// GRPCStatus reports the error as codes.InvalidArgument with the same message.
func (e *invalidArgumentError) GRPCStatus() *status.Status {
	return status.New(codes.InvalidArgument, e.msg)
}

func newInvalidArgument(sentinel error, format string, args ...any) error {
	return &invalidArgumentError{
		msg:     sentinel.Error() + ": " + fmt.Sprintf(format, args...),
		wrapped: sentinel,
	}
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// ResolveCurrency returns the single distinct non-empty currency across
// entries with priced=true, or "USD" when all of them are empty. Currencies
// compare exactly (no case folding). More than one distinct value returns an
// error that wraps ErrMixedCurrency, lists the currencies sorted, and carries
// codes.InvalidArgument. Nil entries are skipped.
func ResolveCurrency(priced []*pbc.PricedResource) (string, error) {
	var currencies []string
	for _, entry := range priced {
		if !entry.GetPriced() {
			continue
		}
		c := entry.GetCurrency()
		if c == "" || containsString(currencies, c) {
			continue
		}
		currencies = append(currencies, c)
	}
	switch len(currencies) {
	case 0:
		return defaultAllocationCurrency, nil
	case 1:
		return currencies[0], nil
	}
	sort.Strings(currencies)
	return "", newInvalidArgument(ErrMixedCurrency, "priced resources use %s", strings.Join(currencies, ", "))
}

// ValidateAllocateRequest returns nil if req is a consistent AllocateRequest
// (rules Q1–Q6 in data-model.md): the request and every priced entry are
// non-nil; unpriced entries cost 0; costs are finite and non-negative; priced
// entries resolve to one currency; no two entries share
// (resource.tags["kind"], resource.id); priced nodes have a non-empty
// resource.id; and start and end are set together, with start not after end
// (the GetStatsRequest window rule). Usage rows and the selector are not
// validated.
//
// Every failure wraps ErrInvalidAllocateRequest (mixed currencies also wrap
// ErrMixedCurrency), names the offending priced[i] entry, and carries
// codes.InvalidArgument, so allocators may return it directly.
func ValidateAllocateRequest(req *pbc.AllocateRequest) error {
	if req == nil {
		return newInvalidArgument(ErrInvalidAllocateRequest, "request is nil")
	}
	if err := validateAllocateWindow(req.GetStart(), req.GetEnd()); err != nil {
		return err
	}
	priced := req.GetPriced()
	for i, entry := range priced {
		if err := validatePricedEntry(i, entry); err != nil {
			return err
		}
	}
	if _, err := ResolveCurrency(priced); err != nil {
		return &invalidArgumentError{
			msg:     ErrInvalidAllocateRequest.Error() + ": " + err.Error(),
			wrapped: errors.Join(ErrInvalidAllocateRequest, err),
		}
	}

	type identity struct{ kind, id string }
	seen := make(map[identity]int, len(priced))
	for i, entry := range priced {
		key := identity{kind: entry.GetResource().GetTags()[subjectKind], id: entry.GetResource().GetId()}
		if first, dup := seen[key]; dup {
			return newInvalidArgument(ErrInvalidAllocateRequest,
				"priced[%d]: duplicates priced[%d] (kind %q, id %q)", i, first, key.kind, key.id)
		}
		seen[key] = i
	}
	return nil
}

func validateAllocateWindow(start, end *timestamppb.Timestamp) error {
	if (start == nil) != (end == nil) {
		return newInvalidArgument(ErrInvalidAllocateRequest, "start and end must be set together")
	}
	if start != nil && timestampAfter(start, end) {
		return newInvalidArgument(ErrInvalidAllocateRequest,
			"start is after end (start %s, end %s)", start.AsTime(), end.AsTime())
	}
	return nil
}

func validatePricedEntry(i int, entry *pbc.PricedResource) error {
	if entry == nil {
		return newInvalidArgument(ErrInvalidAllocateRequest, "priced[%d]: entry is nil", i)
	}
	cost := entry.GetCost()
	if !entry.GetPriced() && cost != 0 {
		return newInvalidArgument(ErrInvalidAllocateRequest,
			"priced[%d]: cost %v must be 0 when priced is false", i, cost)
	}
	if !isFinite(cost) || cost < 0 {
		return newInvalidArgument(ErrInvalidAllocateRequest,
			"priced[%d]: cost %v must be finite and non-negative", i, cost)
	}
	resource := entry.GetResource()
	if entry.GetPriced() && resource.GetTags()[subjectKind] == kindNode && resource.GetId() == "" {
		return newInvalidArgument(ErrInvalidAllocateRequest,
			"priced[%d]: a priced node needs a non-empty resource.id", i)
	}
	return nil
}

// ConservationError reports that allocation rows do not sum to the priced
// total. It wraps ErrConservation.
type ConservationError struct {
	// Expected is the sum of cost over priced=true entries.
	Expected float64
	// Actual is the sum of total_cost over all rows.
	Actual float64
	// Difference is Actual - Expected; positive means the rows overshoot.
	Difference float64
	// Currency is the resolved currency.
	Currency string
}

// Error states the expected and actual totals and their difference.
func (e *ConservationError) Error() string {
	return fmt.Sprintf("allocation rows total %g %s, expected %g %s (difference %+g)",
		e.Actual, e.Currency, e.Expected, e.Currency, e.Difference)
}

// Unwrap returns ErrConservation.
func (e *ConservationError) Unwrap() error { return ErrConservation }

// allocationTolerance is max(relEpsilon*|x|, ConservationAbsoluteFloor).
func allocationTolerance(x, relEpsilon float64) float64 {
	return math.Max(relEpsilon*math.Abs(x), ConservationAbsoluteFloor)
}

// CheckConservation verifies that the rows of resp sum to the cost of the
// priced=true entries of req, within max(relEpsilon*|expected|,
// ConservationAbsoluteFloor). It never passes on a NaN or infinite cost or row
// total, and rejects a negative or non-finite relEpsilon. Currency errors from
// ResolveCurrency are returned unchanged; a mismatch is a *ConservationError.
func CheckConservation(req *pbc.AllocateRequest, resp *pbc.AllocateResponse, relEpsilon float64) error {
	if !isFinite(relEpsilon) || relEpsilon < 0 {
		return fmt.Errorf("relative epsilon %v must be finite and non-negative", relEpsilon)
	}
	if req == nil || resp == nil {
		return errors.New("allocate request and response must be non-nil")
	}
	currency, err := ResolveCurrency(req.GetPriced())
	if err != nil {
		return err
	}

	expected := 0.0
	for i, entry := range req.GetPriced() {
		if !entry.GetPriced() {
			continue
		}
		if !isFinite(entry.GetCost()) {
			return fmt.Errorf("%w: priced[%d]: cost %v is not finite", ErrConservation, i, entry.GetCost())
		}
		expected += entry.GetCost()
	}
	actual := 0.0
	for i, row := range resp.GetRows() {
		if !isFinite(row.GetTotalCost()) {
			return fmt.Errorf("%w: rows[%d]: total_cost %v is not finite", ErrConservation, i, row.GetTotalCost())
		}
		actual += row.GetTotalCost()
	}

	if math.Abs(actual-expected) <= allocationTolerance(expected, relEpsilon) {
		return nil
	}
	return &ConservationError{Expected: expected, Actual: actual, Difference: actual - expected, Currency: currency}
}

// ValidateAllocateResponse returns nil if resp satisfies the AllocatorService
// invariants for req, other than conservation (rules P1–P7 in data-model.md):
// effective policy and digest are present; every row's kind is "workload",
// "__idle__", or "__cluster__"; idle rows name their node; costs are finite
// and non-negative; non-cluster rows have total_cost = cpu_cost + mem_cost
// within tolerance; every row carries the resolved currency; and every priced
// node has exactly one idle row; and an echoed start and end, when present, equal the
// request's (a response without them is accepted). Every error wraps ErrInvalidAllocateResponse
// and names the offending rows[i] entry. Use CheckConservation for totals.
func ValidateAllocateResponse(req *pbc.AllocateRequest, resp *pbc.AllocateResponse) error {
	if resp == nil {
		return fmt.Errorf("%w: response is nil", ErrInvalidAllocateResponse)
	}
	if resp.GetPolicyDigest() == "" {
		return fmt.Errorf("%w: policy_digest is empty", ErrInvalidAllocateResponse)
	}
	if len(resp.GetEffectivePolicyJson()) == 0 {
		return fmt.Errorf("%w: effective_policy_json is empty", ErrInvalidAllocateResponse)
	}
	if err := validateWindowEcho(req, resp); err != nil {
		return err
	}
	currency, err := ResolveCurrency(req.GetPriced())
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidAllocateResponse, err)
	}

	idleRows := make(map[string]int)
	for i, row := range resp.GetRows() {
		if rowErr := validateAllocationRow(i, row, currency); rowErr != nil {
			return rowErr
		}
		if row.GetSubject()[subjectKind] == kindIdle {
			idleRows[row.GetSubject()[subjectNode]]++
		}
	}

	for i, entry := range req.GetPriced() {
		if !entry.GetPriced() || entry.GetResource().GetTags()[subjectKind] != kindNode {
			continue
		}
		id := entry.GetResource().GetId()
		if n := idleRows[id]; n != 1 {
			return fmt.Errorf("%w: priced[%d]: node %q has %d %q rows, want exactly 1",
				ErrInvalidAllocateResponse, i, id, n, kindIdle)
		}
	}
	return nil
}

// validateWindowEcho accepts a response without a period, so allocators built before
// the period fields stay valid. An echoed period must equal the request's in seconds and
// nanoseconds; an unset request bound must stay unset.
func validateWindowEcho(req *pbc.AllocateRequest, resp *pbc.AllocateResponse) error {
	if resp.GetStart() == nil && resp.GetEnd() == nil {
		return nil
	}
	if !sameTimestamp(req.GetStart(), resp.GetStart()) {
		return fmt.Errorf("%w: start %v does not echo the request start %v",
			ErrInvalidAllocateResponse, resp.GetStart(), req.GetStart())
	}
	if !sameTimestamp(req.GetEnd(), resp.GetEnd()) {
		return fmt.Errorf("%w: end %v does not echo the request end %v",
			ErrInvalidAllocateResponse, resp.GetEnd(), req.GetEnd())
	}
	return nil
}

func sameTimestamp(a, b *timestamppb.Timestamp) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.GetSeconds() == b.GetSeconds() && a.GetNanos() == b.GetNanos()
}

func validateAllocationRow(i int, row *pbc.AllocationRow, currency string) error {
	subject := row.GetSubject()
	kind := subject[subjectKind]
	if !containsString(validAllocationKinds, kind) {
		return fmt.Errorf("%w: rows[%d]: subject kind %q is not one of %s",
			ErrInvalidAllocateResponse, i, kind, strings.Join(validAllocationKinds, ", "))
	}
	if kind == kindIdle && subject[subjectNode] == "" {
		return fmt.Errorf("%w: rows[%d]: idle row needs a non-empty %q subject key",
			ErrInvalidAllocateResponse, i, subjectNode)
	}
	cpu, mem, total := row.GetCpuCost(), row.GetMemCost(), row.GetTotalCost()
	for _, v := range []float64{cpu, mem, total} {
		if !isFinite(v) || v < 0 {
			return fmt.Errorf("%w: rows[%d]: costs (cpu %v, mem %v, total %v) must be finite and non-negative",
				ErrInvalidAllocateResponse, i, cpu, mem, total)
		}
	}
	if kind != kindCluster && math.Abs(total-(cpu+mem)) > allocationTolerance(total, DefaultConservationEpsilon) {
		return fmt.Errorf("%w: rows[%d]: total_cost %v differs from cpu_cost + mem_cost %v",
			ErrInvalidAllocateResponse, i, total, cpu+mem)
	}
	if row.GetCurrency() != currency {
		return fmt.Errorf(
			"%w: rows[%d]: currency %q, want %q",
			ErrInvalidAllocateResponse,
			i,
			row.GetCurrency(),
			currency,
		)
	}
	if row.GetAllocatedMethodId() != "" && row.GetAllocatedResourceId() == "" {
		return fmt.Errorf("%w: rows[%d]: allocated_method_id %q requires allocated_resource_id",
			ErrInvalidAllocateResponse, i, row.GetAllocatedMethodId())
	}
	return nil
}
