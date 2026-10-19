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

package testing_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	pt "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func costParsePolicyLimits(maxBytes int64) pt.ParseLimits {
	return pt.ParseLimits{
		Formats: []string{"fixture"}, InputModes: []string{"content", "path"}, MaxInputBytes: maxBytes,
	}
}

func costParsePolicyFiles(files ...*pbc.ParseFile) *pbc.ParseRequest {
	return &pbc.ParseRequest{
		Format: "fixture", Input: &pbc.ParseRequest_Files{Files: &pbc.ParseFileSet{Files: files}},
	}
}

func TestCostParseMockPolicyOptions(t *testing.T) {
	for _, key := range []string{"unexpected", "parameters", "release_name"} {
		t.Run(key, func(t *testing.T) {
			req := costParseMockRequest()
			req.Options = &structpb.Struct{Fields: map[string]*structpb.Value{key: structpb.NewNullValue()}}
			before := proto.Clone(req)
			limits := costParsePolicyLimits(1024)
			// The generic validator cannot interpret a format's option allowlist.
			if err := pt.ValidateParseRequest(req, &limits); err != nil {
				t.Fatalf("shared validator rejected provider-owned options: %v", err)
			}
			resp, err := pt.NewMockCostParser().Parse(context.Background(), req)
			if status.Code(err) != codes.InvalidArgument || resp != nil {
				t.Fatalf("unknown option must fail: response=%v err=%v", resp, err)
			}
			if !strings.Contains(err.Error(), "options") || !strings.Contains(err.Error(), key) {
				t.Fatalf("option error lacks field/key: %v", err)
			}
			require.True(t, proto.Equal(req, before), "provider mutated caller options")
		})
	}
}

func TestCostParseMockPolicyFileLineDiagnostics(t *testing.T) {
	for _, malformed := range []string{"resource", "resource Too Many", "unrecognized Item"} {
		t.Run(malformed, func(t *testing.T) {
			req := costParsePolicyFiles(&pbc.ParseFile{
				Path: "nested/broken.fixture", Content: []byte("resource Valid\n\n" + malformed + "\n"),
			})
			resp, err := pt.NewMockCostParser().Parse(context.Background(), req)
			if status.Code(err) != codes.InvalidArgument || resp != nil {
				t.Fatalf("malformed fixture must fail: response=%v err=%v", resp, err)
			}
			if !strings.Contains(err.Error(), "nested/broken.fixture:3") {
				t.Fatalf("missing known file and physical line: %v", err)
			}
		})
	}
}

func costParsePolicyInput(t *testing.T, mode string, files []*pbc.ParseFile, limits pt.ParseLimits) *pbc.ParseRequest {
	t.Helper()
	req := costParsePolicyFiles(files...)
	if mode == "path" {
		root := t.TempDir()
		for _, file := range files {
			require.NoError(t, os.WriteFile(filepath.Join(root, file.GetPath()), file.GetContent(), 0o600))
		}
		req.Input = &pbc.ParseRequest_RootPath{RootPath: root}
		require.NoError(t, pt.ValidateParseRequest(req, &limits),
			"shared validator must leave path byte accounting to the provider")
	}
	return req
}

func TestCostParseMockPolicyByteLimits(t *testing.T) {
	cases := []struct {
		name     string
		mode     string
		multiple bool
		over     bool
	}{
		{"content/single/exact", "content", false, false},
		{"content/single/over", "content", false, true},
		{"content/multiple/exact", "content", true, false},
		{"content/multiple/over", "content", true, true},
		{"path/single/exact", "path", false, false},
		{"path/single/over", "path", false, true},
		{"path/multiple/exact", "path", true, false},
		{"path/multiple/over", "path", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := []*pbc.ParseFile{{Path: "input.fixture", Content: []byte("resource First\n")}}
			wantIDs := []string{"First"}
			if tc.multiple {
				files = append(files, &pbc.ParseFile{Path: "second.fixture", Content: []byte("resource Second\n")})
				wantIDs = append(wantIDs, "Second")
			}
			var total int64
			for _, file := range files {
				total += int64(len(file.GetContent()))
			}
			limit := total
			if tc.over {
				limit--
			}
			limits := costParsePolicyLimits(limit)
			req := costParsePolicyInput(t, tc.mode, files, limits)
			parser := pt.NewMockCostParser(pt.WithMockParseLimits(limits))
			resp, err := parser.Parse(context.Background(), req)
			if tc.over {
				require.Equal(t, codes.ResourceExhausted, status.Code(err), "total %d exceeds %d", total, limit)
				require.Nil(t, resp)
				require.Regexp(t, "limit|max_input_bytes", err.Error(), "error must identify the size limit")
				return
			}
			require.NoError(t, err, "exact byte limit must succeed")
			require.NoError(t, pt.ValidateParseResponse(req, resp))
			var gotIDs []string
			for _, parsed := range resp.GetResources() {
				gotIDs = append(gotIDs, parsed.GetSourceId())
			}
			require.Equal(t, wantIDs, gotIDs, "all fixtures must be consumed in path order")
		})
	}
}

func TestCostParseMockPolicyEscapingSymlink(t *testing.T) {
	for _, entry := range []string{"input.fixture", "second.fixture"} {
		t.Run(entry, func(t *testing.T) {
			root := t.TempDir()
			outside := filepath.Join(t.TempDir(), "outside.fixture")
			require.NoError(t, os.WriteFile(outside, []byte("resource Outside\n"), 0o600))
			if entry != "input.fixture" {
				if err := os.WriteFile(
					filepath.Join(root, "input.fixture"),
					[]byte("resource Inside\n"),
					0o600,
				); err != nil {
					t.Fatal(err)
				}
			}
			require.NoError(t, os.Symlink(outside, filepath.Join(root, entry)))
			req := &pbc.ParseRequest{Format: "fixture", Input: &pbc.ParseRequest_RootPath{RootPath: root}}
			limits := costParsePolicyLimits(1024)
			if err := pt.ValidateParseRequest(req, &limits); err != nil {
				t.Fatalf("generic validator must not inspect filesystem containment: %v", err)
			}
			resp, err := pt.NewMockCostParser().Parse(context.Background(), req)
			if status.Code(err) != codes.InvalidArgument || resp != nil {
				t.Fatalf("escaping symlink must fail: response=%v err=%v", resp, err)
			}
		})
	}
}
