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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// costParseRawCodec bypasses client serialization only. The server still uses
// its normal proto content subtype and must inspect the bytes it receives.
type costParseRawCodec struct{}

func (costParseRawCodec) Name() string { return "proto" }
func (costParseRawCodec) Marshal(v any) ([]byte, error) {
	if raw, ok := v.([]byte); ok {
		return raw, nil
	}
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("not a protobuf message: %T", v)
	}
	return proto.Marshal(msg)
}
func (costParseRawCodec) Unmarshal(raw []byte, v any) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("not a protobuf message: %T", v)
	}
	return proto.Unmarshal(raw, msg)
}

// costParseRawCall submits a literal protobuf/JSON body at the decoding boundary.
func costParseRawCall(
	t *testing.T, addr, transport, method string, raw []byte, result proto.Message,
) (codes.Code, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), allocCallTimeout)
	defer cancel()
	if transport == "grpc" {
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer func() { require.NoError(t, conn.Close()) }()
		err = conn.Invoke(ctx, method, raw, result, grpc.ForceCodec(costParseRawCodec{}))
		return status.Code(err), status.Convert(err).Message()
	}
	contentType := "application/proto"
	if transport == "connect-json" {
		contentType = "application/json"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+addr+method, bytes.NewReader(raw))
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Connect-Protocol-Version", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	if resp.StatusCode == http.StatusOK {
		if transport != "connect-json" {
			require.NoError(t, proto.Unmarshal(body, result))
		}
		return codes.OK, ""
	}
	var wireError struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(body, &wireError))
	switch wireError.Code {
	case "invalid_argument":
		return codes.InvalidArgument, wireError.Message
	case "internal":
		return codes.Internal, wireError.Message
	default:
		t.Fatalf("unexpected Connect HTTP %d error: %s", resp.StatusCode, body)
	}
	return codes.Unknown, wireError.Message
}

func costParseWireField(number protowire.Number, value []byte) []byte {
	return protowire.AppendBytes(protowire.AppendTag(nil, number, protowire.BytesType), value)
}

func costParseWireInputs(t *testing.T, root string) ([]byte, []byte, []byte) {
	t.Helper()
	content, err := proto.Marshal(costParseFixtureRequest().GetFiles())
	require.NoError(t, err)
	return costParseWireField(1, []byte("fixture")), costParseWireField(2, content), costParseWireField(3, []byte(root))
}

func TestCostParseCodec_ConflictingBinaryInputs(t *testing.T) {
	for _, transport := range []string{"grpc", "connect-proto"} {
		for _, emptyRoot := range []bool{false, true} {
			for _, rootFirst := range []bool{false, true} {
				t.Run(
					fmt.Sprintf("%s/empty_root=%t/root_first=%t", transport, emptyRoot, rootFirst),
					func(t *testing.T) {
						root := t.TempDir()
						if emptyRoot {
							root = ""
						}
						format, files, path := costParseWireInputs(t, root)
						raw := append([]byte(nil), format...)
						raw = append(raw, files...)
						raw = append(raw, path...)
						if rootFirst {
							raw = append(append(format, path...), files...)
						}
						var calls atomic.Int32
						plugin := newCostParseTestPlugin(costParseFunc(func(
							context.Context, *pbc.ParseRequest,
						) (*pbc.ParseResponse, error) {
							calls.Add(1)
							return costParseFixtureResponse(), nil
						}))
						addr, _ := startCostParseServer(t, plugin, transport != "grpc")
						code, message := costParseRawCall(t, addr, transport, pbc.CostParseService_Parse_FullMethodName,
							raw, &pbc.ParseResponse{})
						assert.Equal(t, codes.InvalidArgument, code)
						assert.Contains(t, message, "input")
						assert.Zero(t, calls.Load(), "conflicting oneof arms reached provider")
					},
				)
			}
		}
	}
}

func TestCostParseCodec_ConflictingJSONInputs(t *testing.T) {
	for _, emptyRoot := range []bool{false, true} {
		for _, rootFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("empty_root=%t/root_first=%t", emptyRoot, rootFirst), func(t *testing.T) {
				root := t.TempDir()
				if emptyRoot {
					root = ""
				}
				path, err := json.Marshal(root)
				require.NoError(t, err)
				files := `"files":{"files":[{"path":"main.fixture","content":"aW5zdGFuY2U="}]}`
				rootMember := `"rootPath":` + string(path)
				members := files + "," + rootMember
				if rootFirst {
					members = rootMember + "," + files
				}
				raw := []byte(`{"format":"fixture",` + members + `}`)
				var calls atomic.Int32
				plugin := newCostParseTestPlugin(costParseFunc(func(
					context.Context, *pbc.ParseRequest,
				) (*pbc.ParseResponse, error) {
					calls.Add(1)
					return costParseFixtureResponse(), nil
				}))
				addr, _ := startCostParseServer(t, plugin, true)
				code, message := costParseRawCall(t, addr, "connect-json", pbc.CostParseService_Parse_FullMethodName,
					raw, &pbc.ParseResponse{})
				assert.Equal(t, codes.InvalidArgument, code)
				assert.Contains(t, message, "input")
				assert.Zero(t, calls.Load(), "JSON conflict reached provider")
			})
		}
	}
}

func TestCostParseCodec_ValidBinaryCompatibility(t *testing.T) {
	for _, transport := range []string{"grpc", "connect-proto"} {
		for _, scenario := range []string{"repeated_files", "repeated_root", "unknown_fields"} {
			t.Run(transport+"/"+scenario, func(t *testing.T) {
				format, files, root := costParseWireInputs(t, t.TempDir())
				raw := append([]byte(nil), format...)
				raw = append(raw, files...)
				switch scenario {
				case "repeated_files":
					second, err := proto.Marshal(&pbc.ParseFileSet{Files: []*pbc.ParseFile{{
						Path: "second.fixture", Content: []byte("second"),
					}}})
					require.NoError(t, err)
					raw = append(raw, costParseWireField(2, second)...)
				case "repeated_root":
					raw = append(append(format, root...), costParseWireField(3, []byte(t.TempDir()))...)
				case "unknown_fields":
					// Nested tags 2/3 inside unknown bytes or groups are not top-level input arms.
					raw = append(raw, costParseWireField(100, append(files, root...))...)
					raw = protowire.AppendTag(raw, 101, protowire.VarintType)
					raw = protowire.AppendVarint(raw, 42)
					raw = protowire.AppendTag(raw, 102, protowire.Fixed32Type)
					raw = protowire.AppendFixed32(raw, 42)
					raw = protowire.AppendTag(raw, 103, protowire.Fixed64Type)
					raw = protowire.AppendFixed64(raw, 42)
					raw = protowire.AppendTag(raw, 104, protowire.StartGroupType)
					raw = append(raw, files...)
					raw = append(raw, root...)
					raw = protowire.AppendTag(raw, 104, protowire.EndGroupType)
				}
				expected := &pbc.ParseRequest{}
				require.NoError(t, proto.Unmarshal(raw, expected))
				var calls atomic.Int32
				plugin := newCostParseTestPlugin(costParseFunc(func(
					_ context.Context, got *pbc.ParseRequest,
				) (*pbc.ParseResponse, error) {
					calls.Add(1)
					assert.True(t, proto.Equal(expected, got), "parser codec changed standard protobuf semantics")
					return costParseFixtureResponse(), nil
				}))
				addr, _ := startCostParseServer(t, plugin, transport != "grpc")
				result := &pbc.ParseResponse{}
				code, _ := costParseRawCall(t, addr, transport, pbc.CostParseService_Parse_FullMethodName, raw, result)
				assert.Equal(t, codes.OK, code)
				assert.True(t, proto.Equal(costParseFixtureResponse(), result))
				assert.EqualValues(t, 1, calls.Load())
			})
		}
	}
}

func TestCostParseCodec_MalformedBinary(t *testing.T) {
	for _, transport := range []string{"grpc", "connect-proto"} {
		for name, raw := range map[string][]byte{
			"truncated_length": {0x12, 0x05, 0x01},
			"invalid_tag":      {0},
			"truncated_varint": {0x80},
		} {
			t.Run(transport+"/"+name, func(t *testing.T) {
				var calls atomic.Int32
				plugin := newCostParseTestPlugin(costParseFunc(func(
					context.Context, *pbc.ParseRequest,
				) (*pbc.ParseResponse, error) {
					calls.Add(1)
					return costParseFixtureResponse(), nil
				}))
				addr, _ := startCostParseServer(t, plugin, transport != "grpc")
				code, _ := costParseRawCall(t, addr, transport, pbc.CostParseService_Parse_FullMethodName,
					raw, &pbc.ParseResponse{})
				want := codes.InvalidArgument
				if transport == "grpc" {
					want = codes.Internal
				}
				assert.Equal(t, want, code, "malformed protobuf keeps the transport's normal decode error")
				assert.Zero(t, calls.Load())
			})
		}
	}
}

func TestCostParseCodec_OtherRPCUnchanged(t *testing.T) {
	for _, transport := range []string{"grpc", "connect-proto"} {
		for _, parserPresent := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/parser=%t", transport, parserPresent), func(t *testing.T) {
				var calls atomic.Int32
				var plugin pluginsdk.Plugin = pluginsdk.NewBasePlugin("plain")
				if parserPresent {
					plugin = newCostParseTestPlugin(costParseFunc(func(
						context.Context, *pbc.ParseRequest,
					) (*pbc.ParseResponse, error) {
						calls.Add(1)
						return costParseFixtureResponse(), nil
					}))
				}
				addr, _ := startCostParseServer(t, plugin, transport != "grpc")
				// These same numbers are unknown fields on GetPluginInfoRequest, whose
				// ordinary decoder must accept them even when the Parse codec is installed.
				raw := append(costParseWireField(2, []byte("unknown")), costParseWireField(3, nil)...)
				result := &pbc.GetPluginInfoResponse{}
				code, _ := costParseRawCall(
					t,
					addr,
					transport,
					pbc.CostSourceService_GetPluginInfo_FullMethodName,
					raw,
					result,
				)
				assert.Equal(t, codes.OK, code)
				assert.Equal(t, "parse-test", result.GetName())
				assert.Zero(t, calls.Load())
				code, _ = costParseRawCall(t, addr, transport, pbc.CostSourceService_GetPluginInfo_FullMethodName,
					[]byte{0x12, 0x05, 0x01}, &pbc.GetPluginInfoResponse{})
				want := codes.InvalidArgument
				if transport == "grpc" {
					want = codes.Internal
				}
				assert.Equal(t, want, code)
			})
		}
	}
}
