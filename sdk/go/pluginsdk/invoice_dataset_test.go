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

//nolint:testpackage // Exercises the unexported supplemental adapters through Serve
package pluginsdk

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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1/pbcconnect"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

type getBillingPeriodsFunc func(
	ctx context.Context, req *pbc.GetBillingPeriodsRequest,
) (*pbc.GetBillingPeriodsResponse, error)

type getInvoiceDetailsFunc func(
	ctx context.Context, req *pbc.GetInvoiceDetailsRequest,
) (*pbc.GetInvoiceDetailsResponse, error)

type invoiceCalls struct {
	periods getBillingPeriodsFunc
	details getInvoiceDetailsFunc
	commit  getCommitmentsFunc
}

type invoiceTestPlugin struct {
	*BasePlugin

	source *plugintesting.MockInvoiceDatasetSource
	err    error
}

func newInvoiceTestPlugin(t testing.TB) *invoiceTestPlugin {
	t.Helper()
	at := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	periods := make([]*pbc.BillingPeriod, 0, 3)
	lines := make([]*pbc.InvoiceDetail, 0, 3)
	for i, issuer := range []string{"A", "B", "C"} {
		start := at.AddDate(0, i, 0)
		period, err := NewBillingPeriodBuilder().
			WithWindow(start, start.AddDate(0, 1, 0)).
			WithStatus(pbc.FocusBillingPeriodStatus_FOCUS_BILLING_PERIOD_STATUS_CLOSED).
			WithInvoiceIssuerName(issuer).
			WithCreated(start).
			WithLastUpdated(start).
			Build()
		require.NoError(t, err)
		periods = append(periods, period)
		line, err := NewInvoiceDetailBuilder().
			WithBaseline(start).
			WithIdentity("detail-"+issuer, "invoice-1", issuer, "account-1").
			Build()
		require.NoError(t, err)
		lines = append(lines, line)
	}
	source, err := plugintesting.NewMockInvoiceDatasetSource(periods, lines)
	require.NoError(t, err)
	return &invoiceTestPlugin{BasePlugin: NewBasePlugin("invoices"), source: source}
}

func (p *invoiceTestPlugin) GetBillingPeriods(
	ctx context.Context, req *pbc.GetBillingPeriodsRequest,
) (*pbc.GetBillingPeriodsResponse, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.source.GetBillingPeriods(ctx, req)
}

func (p *invoiceTestPlugin) GetInvoiceDetails(
	ctx context.Context, req *pbc.GetInvoiceDetailsRequest,
) (*pbc.GetInvoiceDetailsResponse, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.source.GetInvoiceDetails(ctx, req)
}

func startInvoiceServer(t *testing.T, plugin Plugin, web bool, info ...*PluginInfo) (string, invoiceCalls) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	cfg := ServeConfig{Plugin: plugin, Listener: listener, Web: WebConfig{Enabled: web}}
	if len(info) > 0 {
		cfg.PluginInfo = info[0]
	}
	go func() {
		errCh <- Serve(ctx, cfg)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(usageCallTimeout):
			t.Error("server did not shut down in time")
		}
	})
	if web {
		client := pbcconnect.NewSupplementalDatasetServiceClient(http.DefaultClient, "http://"+addr)
		return addr, invoiceCalls{
			periods: func(ctx context.Context, req *pbc.GetBillingPeriodsRequest) (*pbc.GetBillingPeriodsResponse, error) {
				resp, callErr := client.GetBillingPeriods(ctx, connect.NewRequest(req))
				if callErr != nil {
					return nil, callErr
				}
				return resp.Msg, nil
			},
			details: func(ctx context.Context, req *pbc.GetInvoiceDetailsRequest) (*pbc.GetInvoiceDetailsResponse, error) {
				resp, callErr := client.GetInvoiceDetails(ctx, connect.NewRequest(req))
				if callErr != nil {
					return nil, callErr
				}
				return resp.Msg, nil
			},
			commit: func(
				ctx context.Context, req *pbc.GetContractCommitmentsRequest,
			) (*pbc.GetContractCommitmentsResponse, error) {
				resp, callErr := client.GetContractCommitments(ctx, connect.NewRequest(req))
				if callErr != nil {
					return nil, callErr
				}
				return resp.Msg, nil
			},
		}
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	client := pbc.NewSupplementalDatasetServiceClient(conn)
	return addr, invoiceCalls{
		periods: func(
			ctx context.Context, req *pbc.GetBillingPeriodsRequest,
		) (*pbc.GetBillingPeriodsResponse, error) {
			return client.GetBillingPeriods(ctx, req)
		},
		details: func(
			ctx context.Context, req *pbc.GetInvoiceDetailsRequest,
		) (*pbc.GetInvoiceDetailsResponse, error) {
			return client.GetInvoiceDetails(ctx, req)
		},
		commit: func(
			ctx context.Context, req *pbc.GetContractCommitmentsRequest,
		) (*pbc.GetContractCommitmentsResponse, error) {
			return client.GetContractCommitments(ctx, req)
		},
	}
}

func TestInvoiceDatasetServeTransportParity(t *testing.T) {
	req := &pbc.GetBillingPeriodsRequest{PageSize: 2}
	detailReq := &pbc.GetInvoiceDetailsRequest{PageSize: 2}
	periodResponses := make(map[string]*pbc.GetBillingPeriodsResponse, len(usageTransports))
	detailResponses := make(map[string]*pbc.GetInvoiceDetailsResponse, len(usageTransports))
	for _, tr := range usageTransports {
		_, calls := startInvoiceServer(t, newInvoiceTestPlugin(t), tr.web)
		ctx, cancel := context.WithTimeout(context.Background(), usageCallTimeout)
		periods, err := calls.periods(ctx, req)
		require.NoError(t, err, tr.name)
		require.NoError(t, ValidateGetBillingPeriodsResponse(req, periods), tr.name)
		assert.Len(t, periods.GetBillingPeriods(), 2, tr.name)
		assert.Equal(t, int32(3), periods.GetTotalCount(), tr.name)
		details, err := calls.details(ctx, detailReq)
		require.NoError(t, err, tr.name)
		require.NoError(t, ValidateGetInvoiceDetailsResponse(detailReq, details), tr.name)
		assert.Len(t, details.GetInvoiceDetails(), 2, tr.name)
		cancel()
		periodResponses[tr.name] = periods
		detailResponses[tr.name] = details
	}
	assert.True(t, proto.Equal(periodResponses["grpc"], periodResponses["connect"]))
	assert.True(t, proto.Equal(detailResponses["grpc"], detailResponses["connect"]))
}

func TestInvoiceDatasetMissingProviderIsUnimplemented(t *testing.T) {
	for _, tr := range usageTransports {
		t.Run("commitment plugin/"+tr.name, func(t *testing.T) {
			_, calls := startInvoiceServer(t, newCommitmentTestPlugin(t), tr.web)
			ctx, cancel := context.WithTimeout(context.Background(), usageCallTimeout)
			defer cancel()
			_, err := calls.periods(ctx, &pbc.GetBillingPeriodsRequest{})
			code, msg := wireCode(t, err, tr.web)
			assert.Equal(t, codes.Unimplemented, code)
			assert.Equal(t, "method GetBillingPeriods not implemented", msg)
			_, err = calls.details(ctx, &pbc.GetInvoiceDetailsRequest{})
			code, msg = wireCode(t, err, tr.web)
			assert.Equal(t, codes.Unimplemented, code)
			assert.Equal(t, "method GetInvoiceDetails not implemented", msg)
			_, err = calls.commit(ctx, &pbc.GetContractCommitmentsRequest{})
			require.NoError(t, err)
		})
		t.Run("invoice plugin/"+tr.name, func(t *testing.T) {
			addr, calls := startInvoiceServer(t, newInvoiceTestPlugin(t), tr.web)
			ctx, cancel := context.WithTimeout(context.Background(), usageCallTimeout)
			defer cancel()
			_, err := calls.commit(ctx, &pbc.GetContractCommitmentsRequest{})
			code, msg := wireCode(t, err, tr.web)
			assert.Equal(t, codes.Unimplemented, code)
			assert.Equal(t, "method GetContractCommitments not implemented", msg)
			_, err = calls.periods(ctx, &pbc.GetBillingPeriodsRequest{})
			require.NoError(t, err)
			if tr.web {
				resp, healthErr := checkCommitmentHealth(t, addr)
				require.NoError(t, healthErr)
				assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.GetStatus())
			}
		})
	}
}

func TestInvoiceDatasetErrorParity(t *testing.T) {
	start := timestamppb.New(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))
	req := &pbc.GetInvoiceDetailsRequest{Start: start}
	var messages [2]string
	var got [2]codes.Code
	for i, tr := range usageTransports {
		_, calls := startInvoiceServer(t, newInvoiceTestPlugin(t), tr.web)
		ctx, cancel := context.WithTimeout(context.Background(), usageCallTimeout)
		_, err := calls.details(ctx, req)
		cancel()
		got[i], messages[i] = wireCode(t, err, tr.web)
		assert.Equal(t, codes.InvalidArgument, got[i], tr.name)
		assert.NotContains(t, messages[i], "rpc error:", tr.name)
	}
	assert.Equal(t, got[0], got[1])
	assert.Equal(t, messages[0], messages[1])
}

type bothDatasetsPlugin struct {
	*invoiceTestPlugin

	commitments *commitmentTestPlugin
}

func (p *bothDatasetsPlugin) GetContractCommitments(
	ctx context.Context, req *pbc.GetContractCommitmentsRequest,
) (*pbc.GetContractCommitmentsResponse, error) {
	return p.commitments.GetContractCommitments(ctx, req)
}

func TestBothDatasetsServeTogether(t *testing.T) {
	for _, tr := range usageTransports {
		t.Run(tr.name, func(t *testing.T) {
			plugin := &bothDatasetsPlugin{
				invoiceTestPlugin: newInvoiceTestPlugin(t),
				commitments:       newCommitmentTestPlugin(t),
			}
			_, calls := startInvoiceServer(t, plugin, tr.web)
			ctx, cancel := context.WithTimeout(context.Background(), usageCallTimeout)
			defer cancel()
			_, err := calls.periods(ctx, &pbc.GetBillingPeriodsRequest{PageSize: 1})
			require.NoError(t, err)
			_, err = calls.details(ctx, &pbc.GetInvoiceDetailsRequest{PageSize: 1})
			require.NoError(t, err)
			_, err = calls.commit(ctx, &pbc.GetContractCommitmentsRequest{PageSize: 1})
			require.NoError(t, err)
			if tr.web {
				return
			}
			server := NewServerWithOptions(
				plugin, nil, nil, &PluginInfo{Name: "both", Version: "1.0.0", SpecVersion: "1.0.0"},
			)
			resp, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
			require.NoError(t, err)
			assert.Contains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_INVOICE_DATA)
			assert.Contains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS)
		})
	}
}

func TestExplicitCapabilitiesStillServesInvoiceRPCs(t *testing.T) {
	info := NewPluginInfo("invoices", "v1.0.0",
		WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS))
	_, calls := startInvoiceServer(t, newInvoiceTestPlugin(t), false, info)
	ctx, cancel := context.WithTimeout(context.Background(), usageCallTimeout)
	defer cancel()
	_, err := calls.periods(ctx, &pbc.GetBillingPeriodsRequest{PageSize: 1})
	require.NoError(t, err)
}

func TestInferCapabilities_InvoiceData(t *testing.T) {
	server := NewServerWithOptions(newInvoiceTestPlugin(t), nil, nil,
		&PluginInfo{Name: "invoices", Version: "1.0.0", SpecVersion: "1.0.0"})
	resp, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	assert.Contains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_INVOICE_DATA)
	assert.NotContains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS)
	assert.Equal(t, "true", resp.GetMetadata()["supports_invoice_data"])
}

func TestExplicitCapabilities_OverrideInvoiceData(t *testing.T) {
	info := NewPluginInfo("invoices", "v1.0.0", WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS))
	server := NewServerWithOptions(newInvoiceTestPlugin(t), nil, nil, info)
	resp, err := server.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	assert.Equal(t, []pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS}, resp.GetCapabilities())
}

func TestInvoiceDatasetWrappersDelegate(t *testing.T) {
	bad := &pbc.GetBillingPeriodsRequest{PageSize: -1}
	assert.Equal(t, plugintesting.ValidateGetBillingPeriodsRequest(bad).Error(),
		ValidateGetBillingPeriodsRequest(bad).Error())
	detailBad := &pbc.GetInvoiceDetailsRequest{PageSize: -1}
	assert.Equal(t, plugintesting.ValidateGetInvoiceDetailsRequest(detailBad).Error(),
		ValidateGetInvoiceDetailsRequest(detailBad).Error())
}
