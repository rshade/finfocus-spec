# Go SDK API: Recommendation Scoring

## pluginsdk

```go
type RecommendationScorerProvider interface {
    ScoreRecommendations(
        ctx context.Context, req *pbc.ScoreRecommendationsRequest,
    ) (*pbc.ScoreRecommendationsResponse, error)
}

func ValidateScoreRecommendationsRequest(req *pbc.ScoreRecommendationsRequest, maxBatchSize int32) error
func ValidateScoreRecommendationsResponse(
    req *pbc.ScoreRecommendationsRequest, resp *pbc.ScoreRecommendationsResponse,
) error
```

Request failures wrap `plugintesting.ErrInvalidScoreRequest` and carry `codes.InvalidArgument`.
Response failures wrap `plugintesting.ErrInvalidScoreResponse`.

## sdk/go/testing

```go
var ErrInvalidScoreRequest, ErrInvalidScoreResponse error

type ScoreServer interface { ScoreRecommendations(...) }
type MockRecommendationScorer struct{}
func NewMockRecommendationScorer(opts ...MockScorerOption) *MockRecommendationScorer
func WithScorerMaxBatchSize(n int32) MockScorerOption
func WithScorerSignals(signals ...pbc.ScoreSignal) MockScorerOption
const DefaultMockScorerBatchSize = 25

func NewScorerHarness(impl ScoreServer) *ScorerHarness
func RunScorerConformance(t *testing.T, impl ScoreServer)
```

## Generated

`pbc.RecommendationScorerServiceClient`, `pbc.RegisterRecommendationScorerServiceServer`,
`pbcconnect.NewRecommendationScorerServiceHandler`, `pbcconnect.RecommendationScorerServiceName`,
`pbc.PluginCapability_PLUGIN_CAPABILITY_RECOMMENDATION_SCORING`.
