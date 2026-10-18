package pluginsdk

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// RecommendationHistoryProvider is an optional interface for plugins that maintain
// server-side recommendation lifecycle history. Plugins without this interface
// return Unimplemented, allowing hosts to fall back to local history.
type RecommendationHistoryProvider interface {
	GetRecommendationHistory(ctx context.Context, req *pbc.GetRecommendationHistoryRequest) (
		*pbc.GetRecommendationHistoryResponse, error)
}

// GetRecommendationHistory retrieves the plugin's lifecycle events for a recommendation.
// It returns Unimplemented when the plugin does not provide server-side history.
func (s *Server) GetRecommendationHistory(
	ctx context.Context,
	req *pbc.GetRecommendationHistoryRequest,
) (*pbc.GetRecommendationHistoryResponse, error) {
	if req.GetRecommendationId() == "" {
		return nil, status.Error(codes.InvalidArgument, "recommendation_id is required")
	}
	provider, ok := s.plugin.(RecommendationHistoryProvider)
	if !ok {
		return nil, recommendationHistoryUnsupported()
	}
	resp, err := provider.GetRecommendationHistory(ctx, req)
	if err != nil {
		useDefault, clientErr := handlerStatus(err, "plugin failed to execute GetRecommendationHistory")
		if useDefault {
			s.logHandlerFallback("GetRecommendationHistory")
			return nil, recommendationHistoryUnsupported()
		}
		s.logger.Error().Err(err).Msg("GetRecommendationHistory handler error")
		return nil, clientErr
	}
	if resp == nil {
		s.logger.Error().Msg("GetRecommendationHistory handler returned a nil response")
		return nil, status.Error(codes.Internal, "plugin returned a nil response")
	}
	return resp, nil
}

func recommendationHistoryUnsupported() error {
	return status.Error(codes.Unimplemented, "plugin does not support GetRecommendationHistory")
}
