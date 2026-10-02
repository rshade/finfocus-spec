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
	"slices"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	scorerScenarioTimeout = 10 * time.Second
	// maxOversizeProbe is the largest max_batch_size for which the
	// oversize_batch scenario builds a request; larger limits are not probed.
	maxOversizeProbe = 1000
	// scorerFixtureSize is the batch size of the standard fixture.
	scorerFixtureSize = 6
	// scorerFixtureSavingsStep is the monthly saving added per fixture recommendation.
	scorerFixtureSavingsStep = 10.0

	scoreConformanceProvider = "conformance"
)

// ScoreServer is satisfied by any type with a ScoreRecommendations method,
// including pbc.RecommendationScorerServiceServer implementations,
// pluginsdk.RecommendationScorerProvider plugins and MockRecommendationScorer.
type ScoreServer interface {
	ScoreRecommendations(
		ctx context.Context, req *pbc.ScoreRecommendationsRequest,
	) (*pbc.ScoreRecommendationsResponse, error)
}

type scoreAdapter struct {
	pbc.UnimplementedRecommendationScorerServiceServer

	impl ScoreServer
}

func (a *scoreAdapter) ScoreRecommendations(
	ctx context.Context, req *pbc.ScoreRecommendationsRequest,
) (*pbc.ScoreRecommendationsResponse, error) {
	return a.impl.ScoreRecommendations(ctx, req)
}

// ScorerHarness serves a ScoreServer over an in-memory bufconn, so calls
// exercise proto serialization and the status codes clients really see.
type ScorerHarness struct {
	bufconnHarness[pbc.RecommendationScorerServiceClient]
}

// NewScorerHarness creates a harness serving impl as RecommendationScorerService.
func NewScorerHarness(impl ScoreServer) *ScorerHarness {
	return &ScorerHarness{newBufconnHarness(func(s *grpc.Server) {
		pbc.RegisterRecommendationScorerServiceServer(s, &scoreAdapter{impl: impl})
	}, pbc.NewRecommendationScorerServiceClient)}
}

// Client returns the RecommendationScorerService client; call Start first.
func (h *ScorerHarness) Client() pbc.RecommendationScorerServiceClient {
	return h.client
}

// scorerFixture returns n recommendations with distinct ids. The last two
// share a resource and action, so a scorer that groups duplicates has
// something to find; the others target distinct resources.
func scorerFixture(n int) []*pbc.Recommendation {
	recs := make([]*pbc.Recommendation, n)
	for i := range recs {
		resource := fmt.Sprintf("conformance-resource-%d", i)
		if i == scorerFixtureSize-1 {
			resource = fmt.Sprintf("conformance-resource-%d", i-1)
		}
		recs[i] = &pbc.Recommendation{
			Id:          fmt.Sprintf("conformance-rec-%d", i),
			Category:    pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
			ActionType:  pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_RIGHTSIZE,
			Description: "Downsize an underused instance.",
			Reasoning:   []string{"CPU stays below 10 percent."},
			Priority:    pbc.RecommendationPriority(i%3 + 1),
			Source:      scoreConformanceProvider,
			Resource: &pbc.ResourceRecommendationInfo{
				Id:       resource,
				Provider: scoreConformanceProvider,
				Tags:     map[string]string{"environment": "staging"},
			},
			Impact: &pbc.RecommendationImpact{
				EstimatedSavings: float64(i+1) * scorerFixtureSavingsStep,
				Currency:         defaultAllocationCurrency,
			},
		}
	}
	return recs
}

// scoreValid calls the scorer and checks the response against the request.
func scoreValid(
	ctx context.Context, client pbc.RecommendationScorerServiceClient, req *pbc.ScoreRecommendationsRequest,
) (*pbc.ScoreRecommendationsResponse, error) {
	resp, err := client.ScoreRecommendations(ctx, req)
	if err != nil {
		return nil, err
	}
	if err = ValidateScoreRecommendationsResponse(req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// probeScorer sends one recommendation to learn max_batch_size and
// supported_signals.
func probeScorer(
	ctx context.Context, client pbc.RecommendationScorerServiceClient,
) (*pbc.ScoreRecommendationsResponse, error) {
	resp, err := scoreValid(ctx, client, &pbc.ScoreRecommendationsRequest{Recommendations: scorerFixture(1)})
	if err != nil {
		return nil, fmt.Errorf("probe with one recommendation: %w", err)
	}
	return resp, nil
}

type scorerScenario struct {
	name string
	run  func(ctx context.Context, client pbc.RecommendationScorerServiceClient) error
}

func scorerScenarios() []scorerScenario {
	return []scorerScenario{
		{"single_recommendation", scorerCheckSingleRecommendation},
		{"mixed_batch", scorerCheckMixedBatch},
		{"reversed_order", scorerCheckReversedOrder},
		{"signal_subset", scorerCheckSignalSubset},
		{"unsupported_signal", scorerCheckUnsupportedSignal},
		{"unspecified_signal", scorerCheckUnspecifiedSignal},
		{"identifier_modes", scorerCheckIdentifierModes},
		{"empty_request", scorerCheckEmptyRequest},
		{"duplicate_ids", scorerCheckDuplicateIDs},
		{"oversize_batch", scorerCheckOversizeBatch},
		{"session_echo", scorerCheckSessionEcho},
		{"session_across_batches", scorerCheckSessionAcrossBatches},
		{"session_isolation", scorerCheckSessionIsolation},
		{"omitted_fields_accepted", scorerCheckOmittedFieldsAccepted},
		{"omitted_fields_rejected", scorerCheckOmittedFieldsRejected},
		{"unscorable_item", scorerCheckUnscorableItem},
	}
}

// scorerCheckUnscorableItem sends a normal recommendation next to one with no
// resource. A scorer may score both, return a per-item error for the second, or
// reject the request with InvalidArgument. Any per-item error goes through
// ValidateScoreRecommendationsResponse, so a scorer that sets
// resource_type_unsupported fails here.
func scorerCheckUnscorableItem(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	recs := append(scorerFixture(1), &pbc.Recommendation{Id: "conformance-no-resource"})
	_, err := scoreValid(ctx, client, &pbc.ScoreRecommendationsRequest{Recommendations: recs})
	if status.Code(err) == codes.InvalidArgument {
		return nil
	}
	return err
}

func scorerCheckSingleRecommendation(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	_, err := probeScorer(ctx, client)
	return err
}

func fixtureForLimit(ctx context.Context, client pbc.RecommendationScorerServiceClient) ([]*pbc.Recommendation, error) {
	probe, err := probeScorer(ctx, client)
	if err != nil {
		return nil, err
	}
	return scorerFixture(min(scorerFixtureSize, int(probe.GetMaxBatchSize()))), nil
}

func scorerCheckMixedBatch(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	recs, err := fixtureForLimit(ctx, client)
	if err != nil {
		return err
	}
	_, err = scoreValid(ctx, client, &pbc.ScoreRecommendationsRequest{Recommendations: recs})
	return err
}

func scorerCheckReversedOrder(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	recs, err := fixtureForLimit(ctx, client)
	if err != nil {
		return err
	}
	slices.Reverse(recs)
	_, err = scoreValid(ctx, client, &pbc.ScoreRecommendationsRequest{Recommendations: recs})
	return err
}

func scorerCheckSignalSubset(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	probe, err := probeScorer(ctx, client)
	if err != nil {
		return err
	}
	supported := probe.GetSupportedSignals()
	if len(supported) == 0 {
		return nil
	}
	only := supported[len(supported)-1]
	recs, err := fixtureForLimit(ctx, client)
	if err != nil {
		return err
	}
	_, err = scoreValid(ctx, client, &pbc.ScoreRecommendationsRequest{
		Recommendations: recs,
		Signals:         []pbc.ScoreSignal{only},
	})
	if err != nil {
		return fmt.Errorf("requesting only %s: %w", only, err)
	}
	return nil
}

func scorerCheckUnsupportedSignal(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	probe, err := probeScorer(ctx, client)
	if err != nil {
		return err
	}
	for signal := range pbc.ScoreSignal_name {
		candidate := pbc.ScoreSignal(signal)
		if !isDefinedScoreSignal(candidate) || slices.Contains(probe.GetSupportedSignals(), candidate) {
			continue
		}
		_, callErr := client.ScoreRecommendations(ctx, &pbc.ScoreRecommendationsRequest{
			Recommendations: scorerFixture(1),
			Signals:         []pbc.ScoreSignal{candidate},
		})
		return wantInvalidArgument(callErr, "unsupported signal "+candidate.String())
	}
	return nil
}

func scorerCheckUnspecifiedSignal(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	_, err := client.ScoreRecommendations(ctx, &pbc.ScoreRecommendationsRequest{
		Recommendations: scorerFixture(1),
		Signals:         []pbc.ScoreSignal{pbc.ScoreSignal_SCORE_SIGNAL_UNSPECIFIED},
	})
	return wantInvalidArgument(err, "SCORE_SIGNAL_UNSPECIFIED")
}

func scorerCheckIdentifierModes(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	recs, err := fixtureForLimit(ctx, client)
	if err != nil {
		return err
	}
	modes := []pbc.IdentifierMode{
		pbc.IdentifierMode_IDENTIFIER_MODE_UNSPECIFIED,
		pbc.IdentifierMode_IDENTIFIER_MODE_RAW,
		pbc.IdentifierMode_IDENTIFIER_MODE_PSEUDONYMIZED,
		pbc.IdentifierMode_IDENTIFIER_MODE_OMITTED,
	}
	for _, mode := range modes {
		_, err = scoreValid(ctx, client, &pbc.ScoreRecommendationsRequest{Recommendations: recs, IdentifierMode: mode})
		if err != nil {
			return fmt.Errorf("%s: %w", mode, err)
		}
	}
	return nil
}

func scorerCheckEmptyRequest(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	_, err := client.ScoreRecommendations(ctx, &pbc.ScoreRecommendationsRequest{})
	return wantInvalidArgument(err, "empty request")
}

func scorerCheckDuplicateIDs(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	recs := scorerFixture(minDuplicateGroupSize)
	recs[1].Id = recs[0].GetId()
	_, err := client.ScoreRecommendations(ctx, &pbc.ScoreRecommendationsRequest{Recommendations: recs})
	return wantInvalidArgument(err, "duplicate recommendation ids")
}

func scorerCheckOversizeBatch(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	probe, err := probeScorer(ctx, client)
	if err != nil {
		return err
	}
	if probe.GetMaxBatchSize() >= maxOversizeProbe {
		return nil
	}
	recs := scorerFixture(int(probe.GetMaxBatchSize()) + 1)
	_, err = client.ScoreRecommendations(ctx, &pbc.ScoreRecommendationsRequest{Recommendations: recs})
	if checkErr := wantInvalidArgument(err, "a batch above max_batch_size"); checkErr != nil {
		return checkErr
	}
	if !IsBatchTooLarge(err) {
		return fmt.Errorf("a batch above max_batch_size was rejected without a %s ErrorInfo detail: %w",
			BatchTooLargeReason, err)
	}
	return nil
}

// AdvertisedScorerMetadataSource is implemented by scorers that advertise their
// limits through GetPluginInfo metadata. MockRecommendationScorer implements
// it; for a real plugin, return the metadata the plugin's PluginInfo carries.
// RunScorerConformance checks the advertised values against the response.
type AdvertisedScorerMetadataSource interface {
	AdvertisedScorerMetadata() map[string]string
}

// scorerPluginInfoServer is implemented by a scorer that also serves the
// CostSourceService GetPluginInfo RPC, for example by embedding
// pluginsdk.BasePlugin. sdk/go/testing cannot import pluginsdk (import cycle),
// so RunScorerConformance reads the served metadata through this method set.
type scorerPluginInfoServer interface {
	GetPluginInfo(context.Context, *pbc.GetPluginInfoRequest) (*pbc.GetPluginInfoResponse, error)
}

// servedScorerMetadata reads the metadata server serves through GetPluginInfo,
// so the check sees what hosts see.
func servedScorerMetadata(ctx context.Context, server scorerPluginInfoServer) (map[string]string, error) {
	resp, err := server.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
	if err != nil {
		return nil, fmt.Errorf("reading GetPluginInfo metadata: %w", err)
	}
	return resp.GetMetadata(), nil
}

// scorerCheckServedLimits runs scorerCheckAdvertisedLimits on the metadata
// server serves. A GetPluginInfo that advertises neither key passes, because
// advertising is optional.
func scorerCheckServedLimits(
	ctx context.Context, client pbc.RecommendationScorerServiceClient, server scorerPluginInfoServer,
) error {
	metadata, err := servedScorerMetadata(ctx, server)
	if err != nil {
		return err
	}
	_, hasLimit := metadata[ScorerMaxBatchSizeKey]
	_, hasSignals := metadata[ScorerSupportedSignalsKey]
	if !hasLimit && !hasSignals {
		return nil
	}
	return scorerCheckAdvertisedLimits(ctx, client, metadata)
}

// scorerCheckAdvertisedLimits compares the metadata a scorer advertises with
// the max_batch_size and supported_signals of its response.
func scorerCheckAdvertisedLimits(
	ctx context.Context, client pbc.RecommendationScorerServiceClient, metadata map[string]string,
) error {
	advertised, err := ParseScorerLimits(metadata)
	if err != nil {
		return err
	}
	if advertised == nil {
		return fmt.Errorf("%w: neither %s nor %s is advertised",
			ErrInvalidScorerLimits, ScorerMaxBatchSizeKey, ScorerSupportedSignalsKey)
	}
	probe, err := probeScorer(ctx, client)
	if err != nil {
		return err
	}
	if advertised.MaxBatchSize != probe.GetMaxBatchSize() {
		return fmt.Errorf("advertised %s is %d but the response max_batch_size is %d",
			ScorerMaxBatchSizeKey, advertised.MaxBatchSize, probe.GetMaxBatchSize())
	}
	got := slices.Clone(probe.GetSupportedSignals())
	want := slices.Clone(advertised.SupportedSignals)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		return fmt.Errorf("advertised %s is %v but the response supported_signals is %v",
			ScorerSupportedSignalsKey, want, got)
	}
	return nil
}

// scorerOmittedMetadata is a valid omitted_fields path used by the omitted-field scenarios.
const scorerOmittedMetadata = "metadata"

func scorerCheckOmittedFieldsAccepted(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	recs, err := fixtureForLimit(ctx, client)
	if err != nil {
		return err
	}
	req := &pbc.ScoreRecommendationsRequest{
		Recommendations: recs,
		OmittedFields:   []string{"resource.tags", scorerOmittedMetadata, "reasoning", "action_detail"},
	}
	if _, err = scoreValid(ctx, client, req); err != nil {
		return fmt.Errorf("a valid omitted_fields list: %w", err)
	}
	return nil
}

func scorerCheckOmittedFieldsRejected(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
	malformed := map[string][]string{
		"an empty entry":   {scorerOmittedMetadata, ""},
		"an unknown path":  {"not_a_field"},
		"a duplicate path": {scorerOmittedMetadata, scorerOmittedMetadata},
	}
	for _, name := range []string{"an empty entry", "an unknown path", "a duplicate path"} {
		req := &pbc.ScoreRecommendationsRequest{Recommendations: scorerFixture(1), OmittedFields: malformed[name]}
		_, err := client.ScoreRecommendations(ctx, req)
		if err = wantInvalidArgument(err, "omitted_fields with "+name); err != nil {
			return err
		}
	}
	return nil
}

// runScorerScenarios runs every scenario against client and returns each
// scenario's error (nil on success) keyed by subtest name.
func runScorerScenarios(ctx context.Context, client pbc.RecommendationScorerServiceClient) map[string]error {
	scenarios := scorerScenarios()
	results := make(map[string]error, len(scenarios))
	for _, s := range scenarios {
		results[s.name] = s.run(ctx, client)
	}
	return results
}

// RunScorerConformance serves impl over a ScorerHarness and runs the standard
// scorer scenarios as subtests. Assertions are model-agnostic: no score values
// are checked, only structure. Every scenario that expects a result requires
// that the call succeeds and that the response passes
// ValidateScoreRecommendationsResponse. The subtests are:
//
//   - single_recommendation: one recommendation; discovers max_batch_size and
//     supported_signals
//   - mixed_batch: up to six recommendations, two of them on one resource
//   - reversed_order: the same batch reversed; results follow request order
//   - signal_subset: only the last supported signal; no other signal is set
//   - unsupported_signal: a signal outside supported_signals is rejected with
//     InvalidArgument (passes when the scorer supports every signal)
//   - unspecified_signal: SCORE_SIGNAL_UNSPECIFIED is rejected with InvalidArgument
//   - identifier_modes: every identifier_mode is accepted
//   - empty_request: no recommendations is rejected with InvalidArgument
//   - duplicate_ids: two recommendations with one id are rejected with InvalidArgument
//   - oversize_batch: max_batch_size + 1 recommendations are rejected with
//     InvalidArgument and a BATCH_TOO_LARGE ErrorInfo detail (not probed when
//     max_batch_size is 1000 or more)
//   - advertised_limits: only when impl serves GetPluginInfo or implements
//     AdvertisedScorerMetadataSource; the advertised limit and signals equal the
//     response's. The metadata served by GetPluginInfo wins. A GetPluginInfo
//     that advertises neither key passes, because advertising is optional;
//     AdvertisedScorerMetadataSource opts in and must advertise both
//   - session_echo: a request with a session_id is answered with the same
//     session_id or none
//   - session_across_batches: a pair grouped in one call of a session gets the
//     same group id when its members are scored in separate calls (passes when
//     the scorer does not echo sessions or does not group the pair)
//   - session_isolation: the same pair gets different group ids in different
//     sessions (same pass condition)
//   - omitted_fields_accepted: a valid omitted_fields list is accepted and the
//     response validates; no score value is compared
//   - omitted_fields_rejected: an empty entry, an unknown path, and a duplicate
//     path are each rejected with InvalidArgument
//   - unscorable_item: a recommendation with no resource next to a normal one;
//     scores, a per-item error, or InvalidArgument all pass, but a per-item
//     error must follow the rules (no resource_type_unsupported)
func RunScorerConformance(t *testing.T, impl ScoreServer) {
	t.Helper()
	harness := NewScorerHarness(impl)
	harness.Start(t)
	defer harness.Stop()

	scenarios := scorerScenarios()
	if server, ok := impl.(scorerPluginInfoServer); ok {
		scenarios = append(scenarios, scorerScenario{"advertised_limits",
			func(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
				return scorerCheckServedLimits(ctx, client, server)
			}})
	} else if source, isSource := impl.(AdvertisedScorerMetadataSource); isSource {
		scenarios = append(scenarios, scorerScenario{"advertised_limits",
			func(ctx context.Context, client pbc.RecommendationScorerServiceClient) error {
				return scorerCheckAdvertisedLimits(ctx, client, source.AdvertisedScorerMetadata())
			}})
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), scorerScenarioTimeout)
			defer cancel()
			if err := s.run(ctx, harness.Client()); err != nil {
				t.Error(err)
			}
		})
	}
}
