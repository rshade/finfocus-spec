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
	"slices"
	"strconv"
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	// ScorerMaxBatchSizeKey is the GetPluginInfo metadata key holding a scorer's
	// max_batch_size as a decimal integer of at least 1.
	ScorerMaxBatchSizeKey = "scorer_max_batch_size"
	// ScorerSupportedSignalsKey is the GetPluginInfo metadata key holding a
	// scorer's supported signals as comma-separated lowercase names without the
	// SCORE_SIGNAL_ prefix, for example "risk,priority".
	ScorerSupportedSignalsKey = "scorer_supported_signals"

	// BatchTooLargeReason is the ErrorInfo reason on the error a scorer returns
	// for a batch above its max_batch_size.
	BatchTooLargeReason = "BATCH_TOO_LARGE"
	// ScorerErrorDomain is the ErrorInfo domain of scorer errors.
	ScorerErrorDomain = "finfocus.v1.RecommendationScorerService"

	scoreSignalPrefix = "SCORE_SIGNAL_"
)

// ErrInvalidScorerLimits is wrapped by every ParseScorerLimits failure.
var ErrInvalidScorerLimits = errors.New("invalid scorer limits metadata")

// FormatScorerLimits returns the GetPluginInfo metadata entries that advertise
// a scorer's batch limit and supported signals. It does not validate: a limit
// below 1 or an unspecified signal is written as given, and ParseScorerLimits
// (run by PluginInfo.Validate) rejects it.
func FormatScorerLimits(maxBatchSize int32, signals []pbc.ScoreSignal) map[string]string {
	names := make([]string, len(signals))
	for i, signal := range signals {
		names[i] = strings.ToLower(strings.TrimPrefix(signal.String(), scoreSignalPrefix))
	}
	return map[string]string{
		ScorerMaxBatchSizeKey:     strconv.FormatInt(int64(maxBatchSize), 10),
		ScorerSupportedSignalsKey: strings.Join(names, ","),
	}
}

// ScorerLimits are the batch limit and signals a scorer advertised.
type ScorerLimits struct {
	MaxBatchSize     int32
	SupportedSignals []pbc.ScoreSignal
}

// ParseScorerLimits reads the advertised batch limit and signals from
// GetPluginInfo metadata. It returns nil with a nil error when neither key is
// present. One key without the other, a limit below 1, an empty, unknown,
// unspecified or repeated signal all return an error wrapping
// ErrInvalidScorerLimits.
func ParseScorerLimits(metadata map[string]string) (*ScorerLimits, error) {
	rawLimit, hasLimit := metadata[ScorerMaxBatchSizeKey]
	rawSignals, hasSignals := metadata[ScorerSupportedSignalsKey]
	if !hasLimit && !hasSignals {
		return nil, nil //nolint:nilnil // Nil limits with no error means the scorer advertised nothing.
	}
	if hasLimit != hasSignals {
		return nil, fmt.Errorf("%w: %s and %s must be set together",
			ErrInvalidScorerLimits, ScorerMaxBatchSizeKey, ScorerSupportedSignalsKey)
	}
	limit, parseErr := strconv.ParseInt(rawLimit, 10, 32)
	if parseErr != nil || limit < 1 {
		return nil, fmt.Errorf("%w: %s is %q, want an integer of at least 1",
			ErrInvalidScorerLimits, ScorerMaxBatchSizeKey, rawLimit)
	}
	var signals []pbc.ScoreSignal
	for _, name := range strings.Split(rawSignals, ",") {
		value, ok := pbc.ScoreSignal_value[scoreSignalPrefix+strings.ToUpper(name)]
		signal := pbc.ScoreSignal(value)
		if !ok || name != strings.ToLower(name) || !isDefinedScoreSignal(signal) {
			return nil, fmt.Errorf("%w: %s holds unknown signal %q",
				ErrInvalidScorerLimits, ScorerSupportedSignalsKey, name)
		}
		if slices.Contains(signals, signal) {
			return nil, fmt.Errorf("%w: %s repeats %q",
				ErrInvalidScorerLimits, ScorerSupportedSignalsKey, name)
		}
		signals = append(signals, signal)
	}
	return &ScorerLimits{MaxBatchSize: int32(limit), SupportedSignals: signals}, nil
}

// IsBatchTooLarge reports whether err is the error a scorer returns for a batch
// above its max_batch_size: a gRPC status of InvalidArgument carrying an
// ErrorInfo detail with reason BATCH_TOO_LARGE. Other InvalidArgument errors
// (empty request, duplicate ids, unsupported signals) return false.
func IsBatchTooLarge(err error) bool {
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		return false
	}
	for _, detail := range st.Details() {
		if info, isInfo := detail.(*errdetails.ErrorInfo); isInfo &&
			info.GetReason() == BatchTooLargeReason && info.GetDomain() == ScorerErrorDomain {
			return true
		}
	}
	return false
}

type batchTooLargeError struct {
	invalidArgumentError
}

// GRPCStatus reports InvalidArgument with a BATCH_TOO_LARGE ErrorInfo detail.
func (e *batchTooLargeError) GRPCStatus() *status.Status {
	st, err := status.New(codes.InvalidArgument, e.msg).WithDetails(&errdetails.ErrorInfo{
		Reason: BatchTooLargeReason,
		Domain: ScorerErrorDomain,
	})
	if err != nil {
		return status.New(codes.InvalidArgument, e.msg)
	}
	return st
}

func newBatchTooLarge(got int, limit int32) error {
	return &batchTooLargeError{invalidArgumentError{
		msg:     fmt.Sprintf("%s: %d recommendations exceed max_batch_size %d", ErrInvalidScoreRequest, got, limit),
		wrapped: ErrInvalidScoreRequest,
	}}
}
