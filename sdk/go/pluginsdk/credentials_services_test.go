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

package pluginsdk_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1/pbcconnect"
)

const optionalServiceCallTimeout = 5 * time.Second

type credentialServicesPlugin struct {
	*pluginsdk.BasePlugin

	seen chan pluginsdk.Credentials
}

func (p *credentialServicesPlugin) record(ctx context.Context) {
	creds, err := pluginsdk.ExtractCredentials(ctx)
	if err != nil {
		creds = pluginsdk.Credentials{}
	}
	p.seen <- creds
}

func (p *credentialServicesPlugin) GetStats(
	ctx context.Context, _ *pbc.GetStatsRequest,
) (*pbc.GetStatsResponse, error) {
	p.record(ctx)
	return &pbc.GetStatsResponse{}, nil
}

func (p *credentialServicesPlugin) Allocate(
	ctx context.Context, _ *pbc.AllocateRequest,
) (*pbc.AllocateResponse, error) {
	p.record(ctx)
	return &pbc.AllocateResponse{}, nil
}

func (p *credentialServicesPlugin) ScoreRecommendations(
	ctx context.Context, _ *pbc.ScoreRecommendationsRequest,
) (*pbc.ScoreRecommendationsResponse, error) {
	p.record(ctx)
	return &pbc.ScoreRecommendationsResponse{}, nil
}

func (p *credentialServicesPlugin) GetContractCommitments(
	ctx context.Context, _ *pbc.GetContractCommitmentsRequest,
) (*pbc.GetContractCommitmentsResponse, error) {
	p.record(ctx)
	return &pbc.GetContractCommitmentsResponse{}, nil
}

func (p *credentialServicesPlugin) GetBillingPeriods(
	ctx context.Context, _ *pbc.GetBillingPeriodsRequest,
) (*pbc.GetBillingPeriodsResponse, error) {
	p.record(ctx)
	return &pbc.GetBillingPeriodsResponse{}, nil
}

func (p *credentialServicesPlugin) GetInvoiceDetails(
	ctx context.Context, _ *pbc.GetInvoiceDetailsRequest,
) (*pbc.GetInvoiceDetailsResponse, error) {
	p.record(ctx)
	return &pbc.GetInvoiceDetailsResponse{}, nil
}

type optionalServiceCall struct {
	name string
	grpc func(conn *grpc.ClientConn, ctx context.Context) error
	web  func(baseURL string, ctx context.Context, header http.Header) error
}

func connectCall[Req, Resp any](
	ctx context.Context,
	header http.Header,
	msg *Req,
	call func(context.Context, *connect.Request[Req]) (*connect.Response[Resp], error),
) error {
	req := connect.NewRequest(msg)
	for key, values := range header {
		req.Header()[key] = values
	}
	_, err := call(ctx, req)
	return err
}

//nolint:gochecknoglobals // Table shared by the optional-service credential tests.
var optionalServiceCalls = []optionalServiceCall{
	{
		name: "GetStats",
		grpc: func(conn *grpc.ClientConn, ctx context.Context) error {
			_, err := pbc.NewUsageSourceServiceClient(conn).GetStats(ctx, &pbc.GetStatsRequest{})
			return err
		},
		web: func(baseURL string, ctx context.Context, header http.Header) error {
			client := pbcconnect.NewUsageSourceServiceClient(http.DefaultClient, baseURL)
			return connectCall(ctx, header, &pbc.GetStatsRequest{}, client.GetStats)
		},
	},
	{
		name: "Allocate",
		grpc: func(conn *grpc.ClientConn, ctx context.Context) error {
			_, err := pbc.NewAllocatorServiceClient(conn).Allocate(ctx, &pbc.AllocateRequest{})
			return err
		},
		web: func(baseURL string, ctx context.Context, header http.Header) error {
			client := pbcconnect.NewAllocatorServiceClient(http.DefaultClient, baseURL)
			return connectCall(ctx, header, &pbc.AllocateRequest{}, client.Allocate)
		},
	},
	{
		name: "ScoreRecommendations",
		grpc: func(conn *grpc.ClientConn, ctx context.Context) error {
			_, err := pbc.NewRecommendationScorerServiceClient(conn).
				ScoreRecommendations(ctx, &pbc.ScoreRecommendationsRequest{})
			return err
		},
		web: func(baseURL string, ctx context.Context, header http.Header) error {
			client := pbcconnect.NewRecommendationScorerServiceClient(http.DefaultClient, baseURL)
			return connectCall(ctx, header, &pbc.ScoreRecommendationsRequest{}, client.ScoreRecommendations)
		},
	},
	{
		name: "GetContractCommitments",
		grpc: func(conn *grpc.ClientConn, ctx context.Context) error {
			_, err := pbc.NewSupplementalDatasetServiceClient(conn).
				GetContractCommitments(ctx, &pbc.GetContractCommitmentsRequest{})
			return err
		},
		web: func(baseURL string, ctx context.Context, header http.Header) error {
			client := pbcconnect.NewSupplementalDatasetServiceClient(http.DefaultClient, baseURL)
			return connectCall(ctx, header, &pbc.GetContractCommitmentsRequest{}, client.GetContractCommitments)
		},
	},
	{
		name: "GetBillingPeriods",
		grpc: func(conn *grpc.ClientConn, ctx context.Context) error {
			_, err := pbc.NewSupplementalDatasetServiceClient(conn).
				GetBillingPeriods(ctx, &pbc.GetBillingPeriodsRequest{})
			return err
		},
		web: func(baseURL string, ctx context.Context, header http.Header) error {
			client := pbcconnect.NewSupplementalDatasetServiceClient(http.DefaultClient, baseURL)
			return connectCall(ctx, header, &pbc.GetBillingPeriodsRequest{}, client.GetBillingPeriods)
		},
	},
	{
		name: "GetInvoiceDetails",
		grpc: func(conn *grpc.ClientConn, ctx context.Context) error {
			_, err := pbc.NewSupplementalDatasetServiceClient(conn).
				GetInvoiceDetails(ctx, &pbc.GetInvoiceDetailsRequest{})
			return err
		},
		web: func(baseURL string, ctx context.Context, header http.Header) error {
			client := pbcconnect.NewSupplementalDatasetServiceClient(http.DefaultClient, baseURL)
			return connectCall(ctx, header, &pbc.GetInvoiceDetailsRequest{}, client.GetInvoiceDetails)
		},
	},
}

func startCredentialServicesServer(t *testing.T, web bool) (string, *credentialServicesPlugin) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	plugin := &credentialServicesPlugin{
		BasePlugin: pluginsdk.NewBasePlugin("optional-service-credentials"),
		seen:       make(chan pluginsdk.Credentials, 1),
	}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{
			Plugin:   plugin,
			Listener: listener,
			Web:      pluginsdk.WebConfig{Enabled: web},
		})
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(optionalServiceCallTimeout):
			t.Error("server did not shut down in time")
		}
	})
	return listener.Addr().String(), plugin
}

func awaitCredentials(t *testing.T, plugin *credentialServicesPlugin) pluginsdk.Credentials {
	t.Helper()
	select {
	case creds := <-plugin.seen:
		return creds
	case <-time.After(optionalServiceCallTimeout):
		t.Fatal("handler was not called")
		return pluginsdk.Credentials{}
	}
}

func TestPerRequestCredentialsOptionalServices(t *testing.T) {
	for _, tc := range optionalServiceCalls {
		t.Run("grpc/"+tc.name, func(t *testing.T) {
			addr, plugin := startCredentialServicesServer(t, false)
			conn, err := grpc.NewClient(
				addr,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithChainUnaryInterceptor(pluginsdk.CredentialUnaryClientInterceptor()),
			)
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.Close() })

			ctx, cancel := context.WithTimeout(context.Background(), optionalServiceCallTimeout)
			defer cancel()
			creds, err := pluginsdk.NewCredentials(map[string]string{"token": credentialFixture})
			require.NoError(t, err)

			_ = tc.grpc(conn, pluginsdk.WithCredentials(ctx, creds))
			got := awaitCredentials(t, plugin)
			value, ok := got.Get("token")
			assert.True(t, ok)
			assert.Equal(t, credentialFixture, value)

			_ = tc.grpc(conn, ctx)
			assert.Zero(t, awaitCredentials(t, plugin).Len(), "set leaked into the next call")
		})

		t.Run("connect/"+tc.name, func(t *testing.T) {
			addr, plugin := startCredentialServicesServer(t, true)
			ctx, cancel := context.WithTimeout(context.Background(), optionalServiceCallTimeout)
			defer cancel()
			header := http.Header{}
			header.Set(pluginsdk.CredentialMetadataPrefix+"token", credentialFixture)

			_ = tc.web("http://"+addr, ctx, header)
			got := awaitCredentials(t, plugin)
			value, ok := got.Get("token")
			assert.True(t, ok)
			assert.Equal(t, credentialFixture, value)

			_ = tc.web("http://"+addr, ctx, nil)
			assert.Zero(t, awaitCredentials(t, plugin).Len(), "set leaked into the next call")
		})
	}
}
