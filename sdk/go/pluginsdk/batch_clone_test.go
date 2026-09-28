//nolint:testpackage // Tests the unexported descriptorClone implementation.
package pluginsdk

import (
	"testing"

	"google.golang.org/protobuf/proto"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func descriptorCloneTestCases() map[string]*pbc.ResourceDescriptor {
	utilization := 0.75
	zero := 0.0
	growthRate := 0.05

	return map[string]*pbc.ResourceDescriptor{
		"nil":   nil,
		"empty": {},
		"scalars only": {
			Provider: "aws", ResourceType: "ec2", Sku: "m5.large",
			Region: "us-east-1", Id: "web-1",
			Arn: "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
		},
		"all fields": {
			Provider: "aws", ResourceType: "ec2", Sku: "m5.large",
			Region: "us-east-1",
			Tags: map[string]string{
				"app": "web",
				"env": "production",
			},
			UtilizationPercentage: &utilization,
			Id:                    "urn:pulumi:prod::myapp::aws:ec2/instance:Instance::web",
			Arn:                   "arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0",
			GrowthType:            pbc.GrowthType_GROWTH_TYPE_EXPONENTIAL,
			GrowthRate:            &growthRate,
		},
		"optional explicit zero preserved": {
			Provider:              "gcp",
			ResourceType:          "compute_engine",
			UtilizationPercentage: &zero,
			GrowthRate:            &zero,
		},
	}
}

// TestDescriptorCloneMatchesProtoClone verifies the manual copy produces a
// message equal to both the original and the old proto.Clone result, across
// representative field combinations.
func TestDescriptorCloneMatchesProtoClone(t *testing.T) {
	for name, original := range descriptorCloneTestCases() {
		t.Run(name, func(t *testing.T) {
			got := descriptorClone(original)
			if original == nil {
				if got != nil {
					t.Fatalf("descriptorClone(nil) = %v, want nil", got)
				}
				return
			}
			if got == original {
				t.Fatal("descriptorClone returned the input pointer, want a copy")
			}
			if !proto.Equal(original, got) {
				t.Fatalf("clone not equal to original:\noriginal: %v\nclone:    %v", original, got)
			}
			//nolint:forcetypeassert // Type assertion guaranteed by proto.Clone contract.
			want := proto.Clone(original).(*pbc.ResourceDescriptor)
			if !proto.Equal(want, got) {
				t.Fatalf("clone differs from proto.Clone result:\nproto.Clone: %v\nmanual:      %v", want, got)
			}
		})
	}
}

// TestDescriptorCloneIndependence verifies mutating the clone (map entries and
// optional pointers) never leaks back into the original descriptor.
func TestDescriptorCloneIndependence(t *testing.T) {
	utilization := 0.75
	growthRate := 0.05
	original := &pbc.ResourceDescriptor{
		Provider:              "aws",
		ResourceType:          "ec2",
		Tags:                  map[string]string{"env": "production"},
		UtilizationPercentage: &utilization,
		GrowthRate:            &growthRate,
	}

	clone := descriptorClone(original)
	clone.GetTags()["env"] = "staging"
	clone.GetTags()["team"] = "platform"
	*clone.UtilizationPercentage = 0.1
	*clone.GrowthRate = 0.9

	if original.GetTags()["env"] != "production" || len(original.GetTags()) != 1 {
		t.Fatalf("original tags mutated via clone: %v", original.GetTags())
	}
	if original.GetUtilizationPercentage() != utilization {
		t.Fatalf("original utilization mutated via clone: %v", original.GetUtilizationPercentage())
	}
	if original.GetGrowthRate() != growthRate {
		t.Fatalf("original growth rate mutated via clone: %v", original.GetGrowthRate())
	}
}

// TestDescriptorClonePreservesUnknownFields verifies unknown wire fields
// survive the manual copy, matching proto.Clone behavior.
func TestDescriptorClonePreservesUnknownFields(t *testing.T) {
	original := &pbc.ResourceDescriptor{Provider: "aws", ResourceType: "ec2"}
	// Field 99, varint 7 — not part of the ResourceDescriptor schema.
	unknown := []byte{0xF8, 0x06, 0x07}
	original.ProtoReflect().SetUnknown(unknown)

	clone := descriptorClone(original)
	if got := clone.ProtoReflect().GetUnknown(); string(got) != string(unknown) {
		t.Fatalf("unknown fields not preserved: got %x, want %x", got, unknown)
	}
	if !proto.Equal(original, clone) {
		t.Fatalf("clone with unknown fields not equal to original: %v", clone)
	}
}
