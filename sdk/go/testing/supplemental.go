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
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/currency"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// pairwiseDuplicateLimit is the largest page a supplemental response validator
// checks for duplicate keys by comparing pairs, which allocates nothing. Larger
// pages use a map, which is faster there but allocates.
const pairwiseDuplicateLimit = 64

// findDuplicate reports the first record whose key repeats an earlier one. It
// returns the lowest such index, the earliest index holding the same key, and true.
// Lists of up to pairwiseDuplicateLimit records are compared pairwise and do
// not allocate when key is a method expression or top-level function; longer
// lists use a map. Both paths report the same pair.
func findDuplicate[T any, K comparable](list []T, key func(T) K) (int, int, bool) {
	if len(list) <= pairwiseDuplicateLimit {
		// key is an indirect call in generic code, so each key is computed once
		// into a stack array rather than once per comparison.
		var buf [pairwiseDuplicateLimit]K
		keys := buf[:len(list)]
		for n := range keys {
			keys[n] = key(list[n])
			for j := range n {
				if keys[j] == keys[n] {
					return n, j, true
				}
			}
		}
		return 0, 0, false
	}
	seen := make(map[K]int, len(list))
	for n := range list {
		k := key(list[n])
		if j, dup := seen[k]; dup {
			return n, j, true
		}
		seen[k] = n
	}
	return 0, 0, false
}

var (
	// ErrInvalidContractCommitment is wrapped by every ValidateContractCommitment failure.
	ErrInvalidContractCommitment = errors.New("invalid contract commitment")
	// ErrInvalidContractCommitmentsRequest is wrapped by every
	// ValidateGetContractCommitmentsRequest and PaginateContractCommitments failure.
	ErrInvalidContractCommitmentsRequest = errors.New("invalid contract commitments request")
	// ErrInvalidContractCommitmentsResponse is wrapped by every
	// ValidateGetContractCommitmentsResponse failure.
	ErrInvalidContractCommitmentsResponse = errors.New("invalid contract commitments response")
)

// commitmentError reports a rule violation with the message
// ContractCommitmentBuilder.Build has always returned, unprefixed.
func commitmentError(format string, args ...any) error {
	return &invalidArgumentError{msg: fmt.Sprintf(format, args...), wrapped: ErrInvalidContractCommitment}
}

// ValidateContractCommitment returns nil if c satisfies every FOCUS Contract
// Commitment rule: the ValidateContractCommitmentBase rules, then the FOCUS 1.4
// columns in validateFocus14Commitment. It is what pluginsdk.ContractCommitmentBuilder.BuildFocus14,
// the mock source, and the conformance suite enforce.
//
// Failures use the builder's messages without a prefix (for example
// "contract_commitment_id is required"), wrap ErrInvalidContractCommitment, and
// carry codes.InvalidArgument. It does not allocate on valid input.
func ValidateContractCommitment(c *pbc.ContractCommitment) error {
	if err := ValidateContractCommitmentBase(c); err != nil {
		return err
	}
	return validateFocus14Commitment(c)
}

// ValidateContractCommitmentBase returns nil if c satisfies the rules that
// predate FOCUS 1.4 and that pluginsdk.ContractCommitmentBuilder.Build enforces:
// contract_commitment_id and contract_id are set; the category is SPEND or
// USAGE; billing_currency is set for SPEND (USAGE may leave it empty) and is
// an ISO 4217 code when set; each period's end is not before its start when
// both bounds are set; and cost and quantity are finite and non-negative. Rules
// are checked in that order. Errors and allocation behavior match
// ValidateContractCommitment.
func ValidateContractCommitmentBase(c *pbc.ContractCommitment) error {
	if c == nil {
		return commitmentError("contract commitment is nil")
	}
	if c.GetContractCommitmentId() == "" {
		return commitmentError("contract_commitment_id is required")
	}
	if c.GetContractId() == "" {
		return commitmentError("contract_id is required")
	}
	category := c.GetContractCommitmentCategory()
	if c.GetBillingCurrency() == "" &&
		category != pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE {
		return commitmentError("billing_currency is required")
	}
	if category != pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_SPEND &&
		category != pbc.FocusContractCommitmentCategory_FOCUS_CONTRACT_COMMITMENT_CATEGORY_USAGE {
		return commitmentError("contract_commitment_category must be SPEND or USAGE, got %v", category)
	}
	if c.GetBillingCurrency() != "" && !currency.IsValid(c.GetBillingCurrency()) {
		return commitmentError("billing_currency must be a valid ISO 4217 currency code, got %q",
			c.GetBillingCurrency())
	}
	if err := validatePeriod("contract_commitment_period",
		c.GetContractCommitmentPeriodStart(), c.GetContractCommitmentPeriodEnd()); err != nil {
		return err
	}
	if err := validatePeriod("contract_period", c.GetContractPeriodStart(), c.GetContractPeriodEnd()); err != nil {
		return err
	}
	if err := validateAmount("contract_commitment_cost", c.GetContractCommitmentCost()); err != nil {
		return err
	}
	return validateAmount("contract_commitment_quantity", c.GetContractCommitmentQuantity())
}

func validatePeriod(name string, start, end *timestamppb.Timestamp) error {
	if start == nil || end == nil {
		return nil
	}
	s, e := start.AsTime(), end.AsTime()
	if e.Before(s) {
		return commitmentError("%s_end (%s) must be >= %s_start (%s)",
			name, e.Format(time.RFC3339), name, s.Format(time.RFC3339))
	}
	return nil
}

func validateAmount(name string, v float64) error {
	if !isFinite(v) {
		return commitmentError("%s must be finite", name)
	}
	if v < 0 {
		return commitmentError("%s must be non-negative", name)
	}
	return nil
}

func responseError(format string, args ...any) error {
	return newInvalidArgument(ErrInvalidContractCommitmentsResponse, format, args...)
}

// validateDatasetRequest checks the window and page size shared by every
// supplemental dataset request. missing reports a nil request. A valid call
// does not allocate.
func validateDatasetRequest(
	sentinel error, missing bool, start, end *timestamppb.Timestamp, pageSize int32,
) error {
	if missing {
		return newInvalidArgument(sentinel, "request is nil")
	}
	if (start == nil) != (end == nil) {
		return newInvalidArgument(sentinel, "start and end must both be set or both be unset")
	}
	if start != nil {
		if err := start.CheckValid(); err != nil {
			return newInvalidArgument(sentinel, "start: %v", err)
		}
		if err := end.CheckValid(); err != nil {
			return newInvalidArgument(sentinel, "end: %v", err)
		}
		if !end.AsTime().After(start.AsTime()) {
			return newInvalidArgument(sentinel, "end must be after start")
		}
	}
	if pageSize < 0 {
		return newInvalidArgument(sentinel, "page_size must not be negative, got %d", pageSize)
	}
	return nil
}

// effectiveDatasetPageSize maps a requested page size to the number of records
// a source serves: 0 (or less) means DefaultPageSize, and values above
// MaxPageSize mean MaxPageSize.
func effectiveDatasetPageSize(pageSize int32) int {
	switch {
	case pageSize <= 0:
		return DefaultPageSize
	case pageSize > MaxPageSize:
		return MaxPageSize
	default:
		return int(pageSize)
	}
}

// paginateRecords returns one page of records, the token for the next page
// (empty on the last page), and the total count, len(records) clamped to
// math.MaxInt32. Tokens are base64-encoded offsets. The page does not share
// backing array past its end; only a next-page token allocates.
func paginateRecords[T any](records []T, pageSize int32, pageToken string, sentinel error) (
	[]T, string, int32, error,
) {
	if pageSize < 0 {
		return nil, "", 0, newInvalidArgument(sentinel, "page_size must not be negative, got %d", pageSize)
	}
	offset := 0
	if pageToken != "" {
		decoded, err := decodeMockPageToken(pageToken)
		if err != nil {
			return nil, "", 0, newInvalidArgument(sentinel, "page_token is malformed: %v", err)
		}
		offset = decoded
	}
	total := int32(math.MaxInt32)
	if len(records) < math.MaxInt32 {
		total = int32(len(records)) //nolint:gosec // bounded by the check above
	}
	if offset >= len(records) {
		return nil, "", total, nil
	}
	end := min(offset+effectiveDatasetPageSize(pageSize), len(records))
	next := ""
	if end < len(records) {
		next = encodeMockPageToken(end)
	}
	return records[offset:end:end], next, total, nil
}

func checkPageBound(sentinel error, noun string, n int, pageSize int32) error {
	if limit := effectiveDatasetPageSize(pageSize); n > limit {
		return newInvalidArgument(sentinel, "%d %s exceed page size %d", n, noun, limit)
	}
	return nil
}

func checkTotalCount(sentinel error, noun string, total int32, n int) error {
	if total < 0 {
		return newInvalidArgument(sentinel, "total_count must not be negative, got %d", total)
	}
	if int(total) < n {
		return newInvalidArgument(sentinel, "total_count %d is less than the %d %s returned", total, n, noun)
	}
	return nil
}

func invalidRecord(sentinel error, field string, i int, err error) error {
	return &invalidArgumentError{
		msg:     fmt.Sprintf("%s: %s[%d]: %s", sentinel, field, i, err),
		wrapped: errors.Join(sentinel, err),
	}
}

// validatePagedResponse applies the page-size, per-record, duplicate, and
// total_count checks shared by supplemental dataset responses.
func validatePagedResponse(
	respNil bool, n int, pageSize int32, noun string, sentinel error, total int32,
	each func(i int) error, duplicates func() error,
) error {
	if respNil {
		return newInvalidArgument(sentinel, "response is nil")
	}
	if err := checkPageBound(sentinel, noun, n, pageSize); err != nil {
		return err
	}
	for i := range n {
		if err := each(i); err != nil {
			return err
		}
	}
	if err := duplicates(); err != nil {
		return err
	}
	return checkTotalCount(sentinel, noun, total, n)
}

// ValidateGetContractCommitmentsRequest returns nil if req is a well-formed
// GetContractCommitmentsRequest: start and end are both set or both unset;
// when set, both are valid timestamps and end is strictly after start; and
// page_size is not negative. A page_size above MaxPageSize is valid (sources
// serve MaxPageSize). The page token is opaque and not checked here;
// PaginateContractCommitments rejects tokens it did not issue.
//
// Failures wrap ErrInvalidContractCommitmentsRequest, start with its text, and
// carry codes.InvalidArgument, so providers may return them unchanged. It does
// not allocate on valid input.
func ValidateGetContractCommitmentsRequest(req *pbc.GetContractCommitmentsRequest) error {
	if req == nil {
		return validateDatasetRequest(ErrInvalidContractCommitmentsRequest, true, nil, nil, 0)
	}
	return validateDatasetRequest(
		ErrInvalidContractCommitmentsRequest, false, req.GetStart(), req.GetEnd(), req.GetPageSize())
}

// ContractCommitmentMatchesWindow reports whether c's period overlaps the
// half-open window [start, end). The period is the contract commitment period
// when either of its bounds is set, otherwise the contract period; an unset
// bound is open-ended, so a commitment with no period bounds matches every
// window. When start or end is nil there is no window and every non-nil
// commitment matches. A nil commitment never matches. It does not allocate.
func ContractCommitmentMatchesWindow(c *pbc.ContractCommitment, start, end *timestamppb.Timestamp) bool {
	if c == nil {
		return false
	}
	if start == nil || end == nil {
		return true
	}
	periodStart, periodEnd := c.GetContractCommitmentPeriodStart(), c.GetContractCommitmentPeriodEnd()
	if periodStart == nil && periodEnd == nil {
		periodStart, periodEnd = c.GetContractPeriodStart(), c.GetContractPeriodEnd()
	}
	if periodStart != nil && !periodStart.AsTime().Before(end.AsTime()) {
		return false
	}
	if periodEnd != nil && !periodEnd.AsTime().After(start.AsTime()) {
		return false
	}
	return true
}

// PaginateContractCommitments returns one page of commitments, the token for
// the next page (empty on the last page), and the total count, len(commitments)
// clamped to math.MaxInt32. A page_size of 0 means DefaultPageSize and values
// above MaxPageSize mean MaxPageSize. Tokens are base64-encoded offsets, the
// same format as pluginsdk.EncodePageToken; a token past the end returns an
// empty last page.
//
// Callers filter by window first and keep the list in a stable order across
// calls. A negative page size or a malformed token returns an error that wraps
// ErrInvalidContractCommitmentsRequest and carries codes.InvalidArgument. The
// page shares the input's backing array; only a next-page token allocates.
func PaginateContractCommitments(
	commitments []*pbc.ContractCommitment, pageSize int32, pageToken string,
) ([]*pbc.ContractCommitment, string, int32, error) {
	return paginateRecords(commitments, pageSize, pageToken, ErrInvalidContractCommitmentsRequest)
}

// ValidateGetContractCommitmentsResponse returns nil if resp is a valid answer
// to req: resp is non-nil; it holds no more commitments than the effective page
// size for req.page_size (0 means DefaultPageSize, values above MaxPageSize mean
// MaxPageSize); every commitment is non-nil, passes ValidateContractCommitment,
// and matches req's window (ContractCommitmentMatchesWindow); no two share a
// contract_commitment_id; and total_count is not negative and not less than
// the number of commitments. A nil req is treated as an empty request.
//
// Failures wrap ErrInvalidContractCommitmentsResponse (an invalid record also
// wraps ErrInvalidContractCommitment), name the offending commitments[i], and
// carry codes.InvalidArgument. It does not allocate on valid pages of up to 64
// commitments; larger pages use one map for the duplicate check.
func ValidateGetContractCommitmentsResponse(
	req *pbc.GetContractCommitmentsRequest, resp *pbc.GetContractCommitmentsResponse,
) error {
	var list []*pbc.ContractCommitment
	var total int32
	if resp != nil {
		list = resp.GetCommitments()
		total = resp.GetTotalCount()
	}
	start, end := req.GetStart(), req.GetEnd()
	return validatePagedResponse(resp == nil, len(list), req.GetPageSize(), "commitments",
		ErrInvalidContractCommitmentsResponse, total,
		func(i int) error { return validateResponseCommitment(i, list[i], start, end) },
		func() error { return checkDuplicateCommitmentIDs(list) },
	)
}

func validateResponseCommitment(i int, c *pbc.ContractCommitment, start, end *timestamppb.Timestamp) error {
	if c == nil {
		return responseError("commitments[%d]: record is nil", i)
	}
	if err := ValidateContractCommitment(c); err != nil {
		return invalidRecord(ErrInvalidContractCommitmentsResponse, "commitments", i, err)
	}
	if !ContractCommitmentMatchesWindow(c, start, end) {
		return responseError("commitments[%d]: contract_commitment_id %q does not overlap the requested window",
			i, c.GetContractCommitmentId())
	}
	return nil
}

func checkDuplicateCommitmentIDs(list []*pbc.ContractCommitment) error {
	if i, first, dup := findDuplicate(list, (*pbc.ContractCommitment).GetContractCommitmentId); dup {
		return duplicateError(i, first, list[i].GetContractCommitmentId())
	}
	return nil
}

func duplicateError(i, first int, id string) error {
	return responseError("commitments[%d]: duplicates contract_commitment_id %q of commitments[%d]", i, id, first)
}
