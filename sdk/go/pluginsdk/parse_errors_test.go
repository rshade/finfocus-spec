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
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func costParseWireDetails(t *testing.T, err error, web bool) []proto.Message {
	t.Helper()
	if web {
		var wireErr *connect.Error
		require.ErrorAs(t, err, &wireErr)
		out := make([]proto.Message, 0, len(wireErr.Details()))
		for _, detail := range wireErr.Details() {
			value, detailErr := detail.Value()
			require.NoError(t, detailErr)
			out = append(out, value)
		}
		return out
	}
	var out []proto.Message
	for _, detail := range status.Convert(err).Details() {
		value, ok := detail.(proto.Message)
		require.True(t, ok, "unexpected status detail %T", detail)
		out = append(out, value)
	}
	return out
}

func TestCostParseErrors_RequestValidationAndPrecedence(t *testing.T) {
	with := func(change func(*pbc.ParseRequest)) *pbc.ParseRequest {
		req := costParseFixtureRequest()
		change(req)
		return req
	}
	hugeOptions, err := structpb.NewStruct(map[string]any{"value": strings.Repeat("x", 65536)})
	require.NoError(t, err)
	cases := []struct {
		name     string
		req      *pbc.ParseRequest
		modes    []string
		wantCode codes.Code
		field    string
		reason   string
	}{
		{"bad format", with(func(r *pbc.ParseRequest) { r.Format = "Fixture" }), nil,
			codes.InvalidArgument, "format", ""},
		{"missing input", &pbc.ParseRequest{Format: "fixture"}, nil,
			codes.InvalidArgument, "input", ""},
		{"empty root", with(func(r *pbc.ParseRequest) { r.Input = &pbc.ParseRequest_RootPath{} }), nil,
			codes.InvalidArgument, "root_path", ""},
		{"empty files", with(func(r *pbc.ParseRequest) { r.GetFiles().Files = nil }), nil,
			codes.InvalidArgument, "files", ""},
		{"unsafe second path", with(func(r *pbc.ParseRequest) {
			r.GetFiles().Files = append(r.GetFiles().Files, &pbc.ParseFile{Path: "../escape"})
		}), nil, codes.InvalidArgument, "files[1].path", ""},
		{"duplicate second path", with(func(r *pbc.ParseRequest) {
			r.GetFiles().Files = append(r.GetFiles().Files, &pbc.ParseFile{Path: "main.fixture"})
		}), nil, codes.InvalidArgument, "files[1].path", ""},
		{"oversized options", with(func(r *pbc.ParseRequest) { r.Options = hugeOptions }), nil,
			codes.InvalidArgument, "options", ""},
		{"unsupported format", with(func(r *pbc.ParseRequest) { r.Format = "other" }), nil,
			codes.FailedPrecondition, "format", "UNSUPPORTED_FORMAT"},
		{"unsupported path mode", with(func(r *pbc.ParseRequest) {
			r.Input = &pbc.ParseRequest_RootPath{RootPath: "/unused/root"}
		}), []string{"content"}, codes.FailedPrecondition, "input mode", "UNSUPPORTED_INPUT_MODE"},
		{"unsupported content mode", costParseFixtureRequest(), []string{"path"},
			codes.FailedPrecondition, "input mode", "UNSUPPORTED_INPUT_MODE"},
		{"total raw bytes over limit", with(func(r *pbc.ParseRequest) {
			r.GetFiles().Files = []*pbc.ParseFile{
				{Path: "a.fixture", Content: []byte("1234")}, {Path: "b.fixture", Content: []byte("56789")},
			}
		}), nil, codes.ResourceExhausted, "parse_max_input_bytes", ""},
		{"syntax before missing input", &pbc.ParseRequest{Format: "Fixture"}, nil,
			codes.InvalidArgument, "format", ""},
		{"missing input before unsupported format", &pbc.ParseRequest{Format: "other"}, nil,
			codes.InvalidArgument, "input", ""},
		{"unsafe path before unsupported format", with(func(r *pbc.ParseRequest) {
			r.Format = "other"
			r.GetFiles().Files[0].Path = "../escape"
		}), nil, codes.InvalidArgument, "files[0].path", ""},
		{"options before unsupported format", with(func(r *pbc.ParseRequest) {
			r.Format = "other"
			r.Options = hugeOptions
		}), nil, codes.InvalidArgument, "options", ""},
		{"options before unsupported mode", with(func(r *pbc.ParseRequest) { r.Options = hugeOptions }),
			[]string{"path"}, codes.InvalidArgument, "options", ""},
		{"format before mode", with(func(r *pbc.ParseRequest) { r.Format = "other" }), []string{"path"},
			codes.FailedPrecondition, "format", "UNSUPPORTED_FORMAT"},
		{"format before size", with(func(r *pbc.ParseRequest) {
			r.Format = "other"
			r.GetFiles().Files[0].Content = []byte("123456789")
		}), nil, codes.FailedPrecondition, "format", "UNSUPPORTED_FORMAT"},
		{"mode before size", with(func(r *pbc.ParseRequest) {
			r.GetFiles().Files[0].Content = []byte("123456789")
		}), []string{"path"}, codes.FailedPrecondition, "input mode", "UNSUPPORTED_INPUT_MODE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var messages []string
			for _, tr := range allocTransports {
				t.Run(tr.name, func(t *testing.T) {
					var calls atomic.Int32
					plugin := newCostParseTestPlugin(costParseFunc(func(
						context.Context, *pbc.ParseRequest,
					) (*pbc.ParseResponse, error) {
						calls.Add(1)
						return costParseFixtureResponse(), nil
					}))
					modes := tc.modes
					if modes == nil {
						modes = []string{"content", "path"}
					}
					_, call := startCostParseServer(t, plugin, tr.web, func(config *pluginsdk.ServeConfig) {
						config.PluginInfo = pluginsdk.NewPluginInfo("parse-test", "1.0.0",
							pluginsdk.WithParseLimits([]string{"fixture"}, modes, 8))
					})
					_, callErr := callCostParse(t, call, tc.req)
					code, message := allocWireCode(t, callErr, tr.web)
					assert.Equal(t, tc.wantCode, code)
					assert.Contains(t, message, tc.field)
					assert.Zero(t, calls.Load(), "invalid request reached provider")
					messages = append(messages, message)
					details := costParseWireDetails(t, callErr, tr.web)
					if tc.reason != "" {
						require.Len(t, details, 1)
						detail, ok := details[0].(*errdetails.ErrorInfo)
						require.True(t, ok)
						assert.Equal(t, tc.reason, detail.GetReason())
						assert.Equal(t, "finfocus.v1.CostParseService", detail.GetDomain())
					} else {
						assert.Empty(t, details)
					}
				})
			}
			require.Len(t, messages, 2)
			assert.Equal(t, messages[0], messages[1], "transport diagnostics differ")
		})
	}
}

func TestCostParseErrors_RawByteBoundary(t *testing.T) {
	for _, tr := range allocTransports {
		t.Run(tr.name, func(t *testing.T) {
			var calls atomic.Int32
			plugin := newCostParseTestPlugin(costParseFunc(func(
				context.Context, *pbc.ParseRequest,
			) (*pbc.ParseResponse, error) {
				calls.Add(1)
				return costParseFixtureResponse(), nil
			}))
			_, call := startCostParseServer(t, plugin, tr.web, func(config *pluginsdk.ServeConfig) {
				config.PluginInfo = pluginsdk.NewPluginInfo("parse-test", "1.0.0",
					pluginsdk.WithParseLimits([]string{"fixture"}, []string{"content"}, 8))
			})
			req := costParseFixtureRequest()
			req.GetFiles().Files = []*pbc.ParseFile{
				{Path: "long/path/does/not/count/toward/raw/bytes.fixture", Content: []byte("éé")},
				{Path: "second.fixture", Content: []byte("éé")}, {Path: "empty.fixture"},
			}
			_, err := callCostParse(t, call, req)
			require.NoError(t, err, "exact raw byte limit must succeed regardless of envelope size")
			assert.EqualValues(t, 1, calls.Load())
			req.GetFiles().Files[2].Content = []byte("x")
			_, err = callCostParse(t, call, req)
			code, message := allocWireCode(t, err, tr.web)
			assert.Equal(t, codes.ResourceExhausted, code)
			assert.Contains(t, message, "8")
			assert.EqualValues(t, 1, calls.Load(), "over-limit request reached provider")
		})
	}
}

func TestCostParseErrors_ProviderStatusesAndDetails(t *testing.T) {
	for code := codes.Canceled; code <= codes.Unauthenticated; code++ {
		if code == codes.Unknown || code == codes.Unimplemented {
			continue
		}
		t.Run(code.String(), func(t *testing.T) {
			details := &errdetails.ErrorInfo{
				Reason: "PROVIDER_REJECTED", Domain: "example.parser", Metadata: map[string]string{"key": "value"},
			}
			diagnostic := &errdetails.BadRequest{FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{Field: "files[0].content", Description: "main.fixture:7: malformed declaration"},
			}}
			st, err := status.New(code, "main.fixture:7: malformed declaration").WithDetails(details, diagnostic)
			require.NoError(t, err)
			for _, tr := range allocTransports {
				t.Run(tr.name, func(t *testing.T) {
					var calls atomic.Int32
					plugin := newCostParseTestPlugin(costParseFunc(func(
						context.Context, *pbc.ParseRequest,
					) (*pbc.ParseResponse, error) {
						calls.Add(1)
						return nil, fmt.Errorf("private backend connection secret: %w", st.Err())
					}))
					_, call := startCostParseServer(t, plugin, tr.web)
					_, callErr := callCostParse(t, call, costParseFixtureRequest())
					gotCode, message := allocWireCode(t, callErr, tr.web)
					assert.Equal(t, code, gotCode)
					assert.Equal(t, st.Message(), message)
					assert.NotContains(t, message, "secret")
					gotDetails := costParseWireDetails(t, callErr, tr.web)
					require.Len(t, gotDetails, 2)
					assert.True(t, proto.Equal(details, gotDetails[0]), "ErrorInfo changed")
					assert.True(t, proto.Equal(diagnostic, gotDetails[1]), "file/line field diagnostic changed")
					assert.EqualValues(t, 1, calls.Load())
				})
			}
		})
	}
}

func TestCostParseErrors_ProviderFailures(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response *pbc.ParseResponse
		err      error
		code     codes.Code
	}{
		{"ordinary error", nil, errors.New("secret backend credentials"), codes.Internal},
		{"unknown status", nil, status.Error(codes.Unknown, "secret backend credentials"), codes.Internal},
		{"nil success", nil, nil, codes.Internal},
		{
			"malformed success",
			&pbc.ParseResponse{Resources: []*pbc.ParsedResource{{SourceId: "secret"}}}, nil, codes.Internal,
		},
		{"unimplemented fallback", nil, status.Error(codes.Unimplemented, "secret backend credentials"), codes.Unimplemented},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var messages []string
			for _, tr := range allocTransports {
				t.Run(tr.name, func(t *testing.T) {
					var calls atomic.Int32
					plugin := newCostParseTestPlugin(costParseFunc(func(
						context.Context, *pbc.ParseRequest,
					) (*pbc.ParseResponse, error) {
						calls.Add(1)
						return tc.response, tc.err
					}))
					_, call := startCostParseServer(t, plugin, tr.web)
					_, err := callCostParse(t, call, costParseFixtureRequest())
					code, message := allocWireCode(t, err, tr.web)
					assert.Equal(t, tc.code, code)
					assert.NotContains(t, message, "secret")
					assert.NotContains(t, message, "credentials")
					assert.Empty(t, costParseWireDetails(t, err, tr.web))
					assert.EqualValues(t, 1, calls.Load())
					messages = append(messages, message)
				})
			}
			require.Len(t, messages, 2)
			assert.Equal(t, messages[0], messages[1])
		})
	}
}
