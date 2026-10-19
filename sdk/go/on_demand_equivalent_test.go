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

package finfocus_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// legacyCostMessage deliberately knows only the pre-existing currency and
// monthly-cost fields. It represents an old decoder without loading old code.
func legacyCostMessage(t *testing.T, name string, currencyNumber, monthlyNumber int32) *dynamicpb.Message {
	t.Helper()
	optional := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	stringType := descriptorpb.FieldDescriptorProto_TYPE_STRING
	doubleType := descriptorpb.FieldDescriptorProto_TYPE_DOUBLE
	file := &descriptorpb.FileDescriptorProto{
		Name: proto.String("legacy_cost.proto"), Package: proto.String("finfocus.v1"),
		Syntax: proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String(name),
			Field: []*descriptorpb.FieldDescriptorProto{
				{Name: proto.String("currency"), Number: &currencyNumber, Label: &optional, Type: &stringType},
				{Name: proto.String("monthly"), Number: &monthlyNumber, Label: &optional, Type: &doubleType},
			},
		}},
	}
	desc, err := protodesc.NewFile(file, nil)
	require.NoError(t, err)
	return dynamicpb.NewMessage(desc.Messages().ByName(protoreflect.Name(name)))
}

func TestOnDemandEquivalentOldDecoder(t *testing.T) {
	zero := 0.0
	tests := []struct {
		name          string
		currencyField int32
		monthlyField  int32
		newResponse   proto.Message
		emptyResponse proto.Message
		presenceCheck func(proto.Message) bool
	}{
		{"GetProjectedCostResponse", 2, 3,
			&pbc.GetProjectedCostResponse{Currency: "USD", CostPerMonth: 80, OnDemandEquivalentMonthlyCost: &zero},
			&pbc.GetProjectedCostResponse{},
			func(v proto.Message) bool {
				return v.(*pbc.GetProjectedCostResponse).OnDemandEquivalentMonthlyCost != nil
			}},
		{"EstimateCostResponse", 1, 2,
			&pbc.EstimateCostResponse{Currency: "USD", CostMonthly: 80, OnDemandEquivalentMonthlyCost: &zero},
			&pbc.EstimateCostResponse{},
			func(v proto.Message) bool { return v.(*pbc.EstimateCostResponse).OnDemandEquivalentMonthlyCost != nil }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			old := legacyCostMessage(t, tc.name, tc.currencyField, tc.monthlyField)
			wire, err := proto.Marshal(tc.newResponse)
			require.NoError(t, err)
			require.NoError(t, (proto.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(wire, old))
			fields := old.Descriptor().Fields()
			assert.Equal(t, "USD", old.Get(fields.ByNumber(protoreflect.FieldNumber(tc.currencyField))).String())
			assert.InDelta(t, 80, old.Get(fields.ByNumber(protoreflect.FieldNumber(tc.monthlyField))).Float(), 0)

			oldWire, err := proto.Marshal(old)
			require.NoError(t, err)
			require.NoError(t, proto.Unmarshal(oldWire, tc.emptyResponse))
			assert.False(t, tc.presenceCheck(tc.emptyResponse))
		})
	}
}

func TestOnDemandEquivalentFieldNumbersAndPresence(t *testing.T) {
	zero := 0.0
	for _, tc := range []struct {
		name  string
		empty proto.Message
		set   proto.Message
		field protoreflect.FieldNumber
	}{
		{"projected", &pbc.GetProjectedCostResponse{},
			&pbc.GetProjectedCostResponse{OnDemandEquivalentMonthlyCost: &zero}, 18},
		{"estimate", &pbc.EstimateCostResponse{},
			&pbc.EstimateCostResponse{OnDemandEquivalentMonthlyCost: &zero}, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field := tc.set.ProtoReflect().Descriptor().Fields().ByName("on_demand_equivalent_monthly_cost")
			require.NotNil(t, field)
			assert.Equal(t, tc.field, field.Number())
			assert.True(t, field.HasPresence())
			assert.False(t, tc.empty.ProtoReflect().Has(field))
			assert.True(t, tc.set.ProtoReflect().Has(field))
		})
	}
}
