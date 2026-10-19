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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	pt "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func parseRequest() *pbc.ParseRequest {
	return &pbc.ParseRequest{
		Format: "fixture",
		Input: &pbc.ParseRequest_Files{
			Files: &pbc.ParseFileSet{Files: []*pbc.ParseFile{{Path: "input.fixture", Content: []byte("abc")}}},
		},
	}
}
func parseLimits() *pt.ParseLimits {
	return &pt.ParseLimits{
		Formats:       []string{"fixture", "a", strings.Repeat("a", 32)},
		InputModes:    []string{"content", "path"},
		MaxInputBytes: 3,
	}
}
func TestCostParseRequest(t *testing.T) {
	cases := []struct {
		name   string
		change func(*pbc.ParseRequest)
		code   codes.Code
		field  string
	}{
		{"valid", func(*pbc.ParseRequest) {}, codes.OK, ""},
		{"one byte format", func(r *pbc.ParseRequest) { r.Format = "a" }, codes.OK, ""},
		{"32 byte format", func(r *pbc.ParseRequest) { r.Format = strings.Repeat("a", 32) }, codes.OK, ""},
		{
			"33 byte format",
			func(r *pbc.ParseRequest) { r.Format = strings.Repeat("a", 33) },
			codes.InvalidArgument,
			"format",
		},
		{"uppercase", func(r *pbc.ParseRequest) { r.Format = "Fixture" }, codes.InvalidArgument, "format"},
		{"missing", func(r *pbc.ParseRequest) { r.Input = nil }, codes.InvalidArgument, "input"},
		{
			"nil wrapper",
			func(r *pbc.ParseRequest) { r.Input = (*pbc.ParseRequest_Files)(nil) },
			codes.InvalidArgument,
			"input",
		},
		{
			"nil root wrapper",
			func(r *pbc.ParseRequest) { r.Input = (*pbc.ParseRequest_RootPath)(nil) },
			codes.InvalidArgument,
			"input",
		},
		{"nil set", func(r *pbc.ParseRequest) { r.Input = &pbc.ParseRequest_Files{} }, codes.InvalidArgument, "files"},
		{"empty set", func(r *pbc.ParseRequest) { r.GetFiles().Files = nil }, codes.InvalidArgument, "files"},
		{"nil file", func(r *pbc.ParseRequest) { r.GetFiles().Files[0] = nil }, codes.InvalidArgument, "files[0]"},
		{
			"duplicate path",
			func(r *pbc.ParseRequest) {
				r.GetFiles().Files = append(r.GetFiles().GetFiles(), r.GetFiles().GetFiles()[0])
			},
			codes.InvalidArgument,
			"path",
		},
		{
			"empty root",
			func(r *pbc.ParseRequest) { r.Input = &pbc.ParseRequest_RootPath{} },
			codes.InvalidArgument,
			"root_path",
		},
		{
			"absolute nonexistent root",
			func(r *pbc.ParseRequest) { r.Input = &pbc.ParseRequest_RootPath{RootPath: "/does/not/exist"} },
			codes.OK,
			"",
		},
		{"unsupported format", func(r *pbc.ParseRequest) { r.Format = "other" }, codes.FailedPrecondition, "format"},
		{
			"over limit",
			func(r *pbc.ParseRequest) { r.GetFiles().Files[0].Content = []byte("abcd") },
			codes.ResourceExhausted,
			"bytes",
		},
		{"empty content", func(r *pbc.ParseRequest) { r.GetFiles().Files[0].Content = nil }, codes.OK, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := parseRequest()
			c.change(r)
			before := proto.Clone(r)
			err := pt.ValidateParseRequest(r, parseLimits())
			require.Equal(t, c.code, status.Code(err))
			if err != nil {
				require.ErrorIs(t, err, pt.ErrInvalidParseRequest)
				require.ErrorContains(t, err, c.field)
			}
			require.True(t, proto.Equal(r, before), "mutated request")
		})
	}
	for _, path := range []string{
		"", "/a", "//host/a", "C:/a", "a:b", "values:prod.yaml", "charts/app:v1/x.yaml",
		`a\b`, "a//b", "a/./b", "a/../b", "a/", "../a", "a\x00b",
	} {
		t.Run("path "+path, func(t *testing.T) {
			r := parseRequest()
			r.GetFiles().Files[0].Path = path
			if status.Code(pt.ValidateParseRequest(r, parseLimits())) != codes.InvalidArgument {
				t.Fatal(path)
			}
		})
	}
	require.ErrorIs(t, pt.ValidateParseRequest(nil, parseLimits()), pt.ErrInvalidParseRequest)
	require.ErrorIs(t, pt.ValidateParseRequest(parseRequest(), nil), pt.ErrInvalidParseLimits)
	modeRequest := parseRequest()
	limits := parseLimits()
	limits.InputModes = []string{"path"}
	modeErr := pt.ValidateParseRequest(modeRequest, limits)
	st := status.Convert(modeErr)
	if st.Code() != codes.FailedPrecondition {
		t.Fatal(modeErr)
	}
	details := st.Details()
	if len(details) != 1 {
		t.Fatal(details)
	}
	info := details[0].(*errdetails.ErrorInfo)
	if info.GetReason() != "UNSUPPORTED_INPUT_MODE" || info.GetDomain() != "finfocus.v1.CostParseService" {
		t.Fatal(info)
	}
	for _, n := range []int{65524, 65525} {
		r := parseRequest()
		r.Options = &structpb.Struct{
			Fields: map[string]*structpb.Value{"x": structpb.NewStringValue(strings.Repeat("a", n))},
		}
		size := proto.Size(r.GetOptions())
		err := pt.ValidateParseRequest(r, parseLimits())
		if (size <= 65536) != (err == nil) {
			t.Fatalf("options size %d: %v", size, err)
		}
	}
}
