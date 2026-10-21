package pluginsdk_test

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1/pbcconnect"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func carbonRequest() *pbc.GetCarbonEmissionsRequest {
	return &pbc.GetCarbonEmissionsRequest{
		Start: timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		End:   timestamppb.New(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)),
	}
}
func carbonRecord() *pbc.CarbonEmissionRecord {
	r := carbonRequest()
	return &pbc.CarbonEmissionRecord{
		Id:           "stable",
		Provider:     "aws",
		Source:       "aws-sustainability",
		AccountId:    "123",
		Region:       "us-east-1",
		Service:      "AmazonEC2",
		Start:        r.GetStart(),
		End:          timestamppb.New(r.GetStart().AsTime().AddDate(0, 1, 0)),
		Method:       pbc.CarbonAccountingMethod_CARBON_ACCOUNTING_METHOD_MARKET_BASED,
		Scope:        pbc.CarbonEmissionsScope_CARBON_EMISSIONS_SCOPE_TOTAL,
		ModelVersion: "v1",
		Value:        0,
		Unit:         "gCO2e",
		RetrievedAt:  timestamppb.Now(),
	}
}
func TestHistoricalCarbonRequest(t *testing.T) {
	require.NoError(t, pluginsdk.ValidateGetCarbonEmissionsRequest(carbonRequest()))
	cases := map[string]func(*pbc.GetCarbonEmissionsRequest){
		"oversized token":   func(r *pbc.GetCarbonEmissionsRequest) { r.PageToken = strings.Repeat("x", 16385) },
		"missing":           func(r *pbc.GetCarbonEmissionsRequest) { r.Start = nil },
		"invalid timestamp": func(r *pbc.GetCarbonEmissionsRequest) { r.Start.Seconds = math.MaxInt64 },
		"partial month":     func(r *pbc.GetCarbonEmissionsRequest) { r.Start.Seconds++ },
		"reverse":           func(r *pbc.GetCarbonEmissionsRequest) { r.End = r.GetStart() },
		"negative page":     func(r *pbc.GetCarbonEmissionsRequest) { r.PageSize = -1 },
		"too large":         func(r *pbc.GetCarbonEmissionsRequest) { r.PageSize = 1001 },
		"method":            func(r *pbc.GetCarbonEmissionsRequest) { r.Method = 99 },
		"scope":             func(r *pbc.GetCarbonEmissionsRequest) { r.Scope = 99 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := carbonRequest()
			mutate(r)
			require.Error(t, pluginsdk.ValidateGetCarbonEmissionsRequest(r))
		})
	}
	r := carbonRequest()
	pluginsdk.ApplyGetCarbonEmissionsDefaults(r)
	require.Equal(t, int32(50), r.GetPageSize())
	require.Equal(t, pbc.CarbonAccountingMethod_CARBON_ACCOUNTING_METHOD_MARKET_BASED, r.GetMethod())
	require.Equal(t, pbc.CarbonEmissionsScope_CARBON_EMISSIONS_SCOPE_TOTAL, r.GetScope())
}
func TestHistoricalCarbonRecord(t *testing.T) {
	require.NoError(
		t,
		pluginsdk.ValidateGetCarbonEmissionsResponse(
			&pbc.GetCarbonEmissionsResponse{Records: []*pbc.CarbonEmissionRecord{carbonRecord()}},
		),
	)
	require.NoError(t, pluginsdk.ValidateGetCarbonEmissionsResponse(&pbc.GetCarbonEmissionsResponse{}))
	cases := map[string]func(*pbc.CarbonEmissionRecord){
		"nan":             func(r *pbc.CarbonEmissionRecord) { r.Value = math.NaN() },
		"inf":             func(r *pbc.CarbonEmissionRecord) { r.Value = math.Inf(1) },
		"negative":        func(r *pbc.CarbonEmissionRecord) { r.Value = -1 },
		"unit":            func(r *pbc.CarbonEmissionRecord) { r.Unit = "MTCO2e" },
		"model":           func(r *pbc.CarbonEmissionRecord) { r.ModelVersion = "" },
		"account":         func(r *pbc.CarbonEmissionRecord) { r.AccountId = "" },
		"region":          func(r *pbc.CarbonEmissionRecord) { r.Region = "" },
		"service":         func(r *pbc.CarbonEmissionRecord) { r.Service = "" },
		"source":          func(r *pbc.CarbonEmissionRecord) { r.Source = "" },
		"provider":        func(r *pbc.CarbonEmissionRecord) { r.Provider = "" },
		"id":              func(r *pbc.CarbonEmissionRecord) { r.Id = "" },
		"retrieved":       func(r *pbc.CarbonEmissionRecord) { r.RetrievedAt = nil },
		"method":          func(r *pbc.CarbonEmissionRecord) { r.Method = 0 },
		"scope":           func(r *pbc.CarbonEmissionRecord) { r.Scope = 0 },
		"multiple months": func(r *pbc.CarbonEmissionRecord) { r.End = carbonRequest().GetEnd() },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := carbonRecord()
			mutate(r)
			require.Error(t, pluginsdk.ValidateCarbonEmissionRecord(r))
		})
	}
	require.Error(t, pluginsdk.ValidateGetCarbonEmissionsResponse(nil))
	require.Error(t, pluginsdk.ValidateCarbonEmissionRecord(nil))
}
func TestHistoricalCarbonMissingProvider(t *testing.T) {
	s := pluginsdk.NewServer(pluginsdk.NewBasePlugin("old"))
	_, err := s.GetCarbonEmissions(context.Background(), carbonRequest())
	require.Equal(t, codes.Unimplemented, status.Code(err))
	desc := pbc.File_finfocus_v1_costsource_proto.Services().
		ByName("CostSourceService").
		Methods().
		ByName("GetCarbonEmissions")
	require.NotNil(t, desc)
	require.Equal(t, "finfocus.v1.GetCarbonEmissionsRequest", string(desc.Input().FullName()))
	require.NotEqual(
		t,
		pbc.PluginCapability_PLUGIN_CAPABILITY_CARBON,
		pbc.PluginCapability_PLUGIN_CAPABILITY_HISTORICAL_CARBON,
	)
}

type carbonPlugin struct{ *pluginsdk.BasePlugin }

func (p *carbonPlugin) GetCarbonEmissions(
	_ context.Context,
	req *pbc.GetCarbonEmissionsRequest,
) (*pbc.GetCarbonEmissionsResponse, error) {
	if req.GetMethod() != pbc.CarbonAccountingMethod_CARBON_ACCOUNTING_METHOD_MARKET_BASED ||
		req.GetScope() != pbc.CarbonEmissionsScope_CARBON_EMISSIONS_SCOPE_TOTAL ||
		req.GetPageSize() != 50 {
		return nil, status.Error(codes.InvalidArgument, "defaults missing")
	}
	return &pbc.GetCarbonEmissionsResponse{
		Records:       []*pbc.CarbonEmissionRecord{carbonRecord()},
		NextPageToken: "opaque",
	}, nil
}
func TestHistoricalCarbonCapability(t *testing.T) {
	plugin := &carbonPlugin{pluginsdk.NewBasePlugin("carbon")}
	server := pluginsdk.NewServer(plugin)
	require.NotContains(t, server.GetGlobalCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_HISTORICAL_CARBON)
	enabled := pluginsdk.NewServerWithOptions(
		plugin,
		nil,
		nil,
		pluginsdk.NewPluginInfo(
			"carbon",
			"v1.0.0",
			pluginsdk.WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_HISTORICAL_CARBON),
		),
	)
	require.Contains(t, enabled.GetGlobalCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_HISTORICAL_CARBON)
	require.NotContains(t, server.GetGlobalCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_CARBON)
	require.Equal(
		t,
		"supports_historical_carbon",
		pluginsdk.CapabilityToLegacyName(pbc.PluginCapability_PLUGIN_CAPABILITY_HISTORICAL_CARBON),
	)
	info := pluginsdk.NewPluginInfo(
		"carbon",
		"v1.0.0",
		pluginsdk.WithCapabilities(pbc.PluginCapability_PLUGIN_CAPABILITY_ACTUAL_COSTS),
	)
	explicit := pluginsdk.NewServerWithOptions(plugin, nil, nil, info)
	require.NotContains(t, explicit.GetGlobalCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_HISTORICAL_CARBON)
	resp, err := explicit.GetPluginInfo(context.Background(), &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	require.NotContains(t, resp.GetCapabilities(), pbc.PluginCapability_PLUGIN_CAPABILITY_HISTORICAL_CARBON)
}

//nolint:gocognit // Covers the transport/provider compatibility matrix in one test.
func TestHistoricalCarbonRoundTrip(t *testing.T) {
	for _, web := range []bool{false, true} {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("web=%t/provider=%t", web, enabled), func(t *testing.T) {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				var plugin pluginsdk.Plugin = pluginsdk.NewBasePlugin("old")
				if enabled {
					plugin = &carbonPlugin{pluginsdk.NewBasePlugin("carbon")}
				}
				ctx, cancel := context.WithCancel(context.Background())
				done := make(chan error, 1)
				go func() {
					done <- pluginsdk.Serve(ctx, pluginsdk.ServeConfig{Plugin: plugin, Listener: listener, Web: pluginsdk.WebConfig{Enabled: web}})
				}()
				t.Cleanup(func() {
					cancel()
					select {
					case serveErr := <-done:
						require.NoError(t, serveErr)
					case <-time.After(5 * time.Second):
						t.Error("shutdown timeout")
					}
				})
				callCtx, callCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer callCancel()
				req := carbonRequest()
				var resp *pbc.GetCarbonEmissionsResponse
				if web {
					client := pbcconnect.NewCostSourceServiceClient(
						http.DefaultClient,
						"http://"+listener.Addr().String(),
					)
					result, callErr := client.GetCarbonEmissions(callCtx, connect.NewRequest(req))
					err = callErr
					if result != nil {
						resp = result.Msg
					}
					if !enabled {
						require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
						return
					}
				} else {
					conn, connErr := grpc.NewClient(
						listener.Addr().String(),
						grpc.WithTransportCredentials(insecure.NewCredentials()),
					)
					require.NoError(t, connErr)
					defer conn.Close()
					resp, err = pbc.NewCostSourceServiceClient(conn).GetCarbonEmissions(callCtx, req)
					if !enabled {
						require.Equal(t, codes.Unimplemented, status.Code(err))
						return
					}
				}
				require.NoError(t, err)
				require.Len(t, resp.GetRecords(), 1)
				require.Equal(t, "opaque", resp.GetNextPageToken())
				require.Zero(t, resp.GetRecords()[0].GetValue())
				require.Equal(t, "aws-sustainability", resp.GetRecords()[0].GetSource())
				require.Equal(t, "v1", resp.GetRecords()[0].GetModelVersion())
				require.Zero(t, req.GetPageSize(), "SDK must not mutate caller request")
			})
		}
	}
}
