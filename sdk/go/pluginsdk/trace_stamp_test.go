package pluginsdk_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

var errSharedBadInput = pluginsdk.NewValidationError("field", "must be set", "", "non-empty")

func TestTracingInterceptor_DoesNotMutateSharedValidationError(t *testing.T) {
	tests := []struct {
		name string
		wrap func(error) error
	}{
		{"direct", func(err error) error { return err }},
		{"wrapped", func(err error) error { return fmt.Errorf("handler: %w", err) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := pluginsdk.TracingUnaryServerInterceptor()
			handler := func(context.Context, any) (any, error) { return "unused", tt.wrap(errSharedBadInput) }

			const calls = 8
			traceIDs := make([]string, calls)
			gotIDs := make([]string, calls)
			errs := make([]error, calls)
			var wg sync.WaitGroup
			for i := range calls {
				traceIDs[i] = fmt.Sprintf("%032x", i+1)
				wg.Add(1)
				go func() {
					defer wg.Done()
					ctx := metadata.NewIncomingContext(context.Background(),
						metadata.Pairs(pluginsdk.TraceIDMetadataKey, traceIDs[i]))
					_, errs[i] = interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/M"}, handler)
				}()
			}
			wg.Wait()

			for i, err := range errs {
				var ve *pluginsdk.ValidationError
				require.ErrorAs(t, err, &ve)
				gotIDs[i] = ve.TraceID
				assert.Equal(t, traceIDs[i], gotIDs[i])
				assert.Contains(t, err.Error(), "trace_id="+traceIDs[i])
				assert.True(t, errors.Is(err, errSharedBadInput) || ve.FieldName == "field")
			}
			assert.Empty(t, errSharedBadInput.TraceID)
		})
	}
}
