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
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestCostParseContract(t *testing.T) {
	if pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE != 19 {
		t.Fatal("capability must be additive value 19")
	}
	messages := []struct {
		message proto.Message
		fields  map[protoreflect.Name]protoreflect.FieldNumber
	}{
		{
			&pbc.ParseRequest{},
			map[protoreflect.Name]protoreflect.FieldNumber{"format": 1, "files": 2, "root_path": 3, "options": 4},
		},
		{&pbc.ParseFileSet{}, map[protoreflect.Name]protoreflect.FieldNumber{"files": 1}},
		{&pbc.ParseFile{}, map[protoreflect.Name]protoreflect.FieldNumber{"path": 1, "content": 2}},
		{
			&pbc.ParseResponse{},
			map[protoreflect.Name]protoreflect.FieldNumber{"resources": 1, "warnings": 2, "expires_at": 3},
		},
		{
			&pbc.ParsedResource{},
			map[protoreflect.Name]protoreflect.FieldNumber{"resource": 1, "source_id": 2, "unknown_properties": 3},
		},
		{
			&pbc.ParseWarning{},
			map[protoreflect.Name]protoreflect.FieldNumber{"severity": 1, "code": 2, "message": 3, "source_id": 4},
		},
	}
	for _, m := range messages {
		for name, number := range m.fields {
			f := m.message.ProtoReflect().Descriptor().Fields().ByName(name)
			if f == nil || f.Number() != number {
				t.Fatalf("%T field %s != %d", m.message, name, number)
			}
		}
	}
	method := pbc.File_finfocus_v1_parse_proto.Services().ByName("CostParseService").Methods().ByName("Parse")
	if method.Input().FullName() != "finfocus.v1.ParseRequest" ||
		method.Output().FullName() != "finfocus.v1.ParseResponse" {
		t.Fatal("service shape")
	}
	for _, req := range []*pbc.ParseRequest{
		{Format: "fixture", Input: &pbc.ParseRequest_Files{Files: &pbc.ParseFileSet{Files: []*pbc.ParseFile{{Path: "input.fixture", Content: []byte{0, 255, 65}}}}}},
		{Format: "fixture", Input: &pbc.ParseRequest_RootPath{RootPath: "/tmp/root"}},
	} {
		b, err := proto.Marshal(req)
		if err != nil {
			t.Fatal(err)
		}
		out := new(pbc.ParseRequest)
		if err = proto.Unmarshal(b, out); err != nil || !proto.Equal(req, out) {
			t.Fatalf("request round trip: %v", err)
		}
	}
	response := &pbc.ParseResponse{
		Resources: []*pbc.ParsedResource{
			{
				Resource: &pbc.ResourceDescriptor{Provider: "fixture", ResourceType: "fixture:compute:Instance"},
				SourceId: "Instance",
			},
		},
		Warnings: []*pbc.ParseWarning{
			{
				Severity: pbc.ParseWarningSeverity_PARSE_WARNING_SEVERITY_WARNING,
				Code:     "resource_skipped",
				Message:  "unsupported",
				SourceId: "Skipped",
			},
		},
		ExpiresAt: timestamppb.New(time.Unix(1800000000, 123456789)),
	}
	b, err := proto.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	out := new(pbc.ParseResponse)
	if err = proto.Unmarshal(b, out); err != nil || !proto.Equal(response, out) {
		t.Fatalf("response round trip: %v", err)
	}
}
