package pluginsdk_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// lineageLevel describes one expected level when walking a chain leaf-to-root.
type lineageLevel struct {
	nodeType pbc.LineageNodeType
	id       string
	name     string
}

// walkLineage flattens a chain into levels, leaf first, failing on cycles or
// chains longer than maxDepth.
func walkLineage(t *testing.T, leaf *pbc.LineageNode, maxDepth int) []lineageLevel {
	t.Helper()
	var levels []lineageLevel
	for node := leaf; node != nil; node = node.GetParent() {
		if len(levels) >= maxDepth {
			t.Fatalf("chain exceeds %d levels (cycle or runaway chain)", maxDepth)
		}
		levels = append(levels, lineageLevel{
			nodeType: node.GetType(),
			id:       node.GetId(),
			name:     node.GetName(),
		})
	}
	return levels
}

// roundTripProtojson marshals want to JSON, unmarshals into empty, and
// asserts full proto equality.
func roundTripProtojson(t *testing.T, want, empty proto.Message) {
	t.Helper()
	data, err := protojson.Marshal(want)
	if err != nil {
		t.Fatalf("protojson.Marshal failed: %v", err)
	}
	err = protojson.Unmarshal(data, empty)
	if err != nil {
		t.Fatalf("protojson.Unmarshal failed: %v", err)
	}
	if diff := cmp.Diff(want, empty, protocmp.Transform()); diff != "" {
		t.Errorf("round trip mismatch (-want +got):\n%s", diff)
	}
}

// buildFourLevelChain builds the 4-level chain from the issue example:
// resource -> sub-account -> billing account -> organization, with per-level
// metadata on the resource and sub-account nodes.
func buildFourLevelChain() *pbc.LineageNode {
	return pluginsdk.NewLineageBuilder("i-0abc123def456", "web-server-prod-01").
		WithMetadata("region", "us-east-1").
		WithParent(pbc.LineageNodeType_LINEAGE_NODE_TYPE_SUB_ACCOUNT, "123456789012", "Production").
		WithMetadata("cost_center", "CC-ENG-001").
		WithParent(pbc.LineageNodeType_LINEAGE_NODE_TYPE_BILLING_ACCOUNT, "999988887777", "Example Payer").
		WithParent(pbc.LineageNodeType_LINEAGE_NODE_TYPE_ORGANIZATION, "o-exampleorg", "Example Org").
		Build()
}

// wantFourLevelChain returns the expected leaf-to-root ordering of
// buildFourLevelChain.
func wantFourLevelChain() []lineageLevel {
	return []lineageLevel{
		{pbc.LineageNodeType_LINEAGE_NODE_TYPE_RESOURCE, "i-0abc123def456", "web-server-prod-01"},
		{pbc.LineageNodeType_LINEAGE_NODE_TYPE_SUB_ACCOUNT, "123456789012", "Production"},
		{pbc.LineageNodeType_LINEAGE_NODE_TYPE_BILLING_ACCOUNT, "999988887777", "Example Payer"},
		{pbc.LineageNodeType_LINEAGE_NODE_TYPE_ORGANIZATION, "o-exampleorg", "Example Org"},
	}
}

// TestLineageBuilderChainOrder builds the 4-level chain from the issue
// example and asserts walking Parent pointers yields ids/names/types in
// exact leaf-to-root order.
func TestLineageBuilderChainOrder(t *testing.T) {
	leaf := buildFourLevelChain()

	got := walkLineage(t, leaf, 10)
	if diff := cmp.Diff(wantFourLevelChain(), got, cmp.AllowUnexported(lineageLevel{})); diff != "" {
		t.Errorf("chain order mismatch (-want +got):\n%s", diff)
	}

	// The root's Parent must be nil (top of the reported chain).
	root := leaf.GetParent().GetParent().GetParent()
	if root.GetParent() != nil {
		t.Errorf("expected root node to have nil Parent, got %v", root.GetParent())
	}
}

// TestLineageBuilderMetadataPlacement verifies metadata lands on the most
// recently added node.
func TestLineageBuilderMetadataPlacement(t *testing.T) {
	testCases := []struct {
		name  string
		build func() *pbc.LineageNode
		// wantMetadata[i] is the expected metadata map at chain level i (leaf first).
		wantMetadata []map[string]string
	}{
		{
			name: "metadata before any WithParent lands on the resource node",
			build: func() *pbc.LineageNode {
				return pluginsdk.NewLineageBuilder("res-1", "resource").
					WithMetadata("owner_email", "team@example.com").
					WithParent(pbc.LineageNodeType_LINEAGE_NODE_TYPE_SUB_ACCOUNT, "sub-1", "sub").
					Build()
			},
			wantMetadata: []map[string]string{
				{"owner_email": "team@example.com"},
				nil,
			},
		},
		{
			name: "metadata after WithParent lands on the new top, not above it",
			build: func() *pbc.LineageNode {
				return pluginsdk.NewLineageBuilder("res-1", "resource").
					WithParent(pbc.LineageNodeType_LINEAGE_NODE_TYPE_SUB_ACCOUNT, "sub-1", "sub").
					WithMetadata("cost_center", "CC-ENG-001").
					WithParent(pbc.LineageNodeType_LINEAGE_NODE_TYPE_BILLING_ACCOUNT, "bill-1", "billing").
					Build()
			},
			wantMetadata: []map[string]string{
				nil,
				{"cost_center": "CC-ENG-001"},
				nil,
			},
		},
		{
			name: "metadata on every level",
			build: func() *pbc.LineageNode {
				return pluginsdk.NewLineageBuilder("res-1", "resource").
					WithMetadata("region", "us-east-1").
					WithParent(pbc.LineageNodeType_LINEAGE_NODE_TYPE_SUB_ACCOUNT, "sub-1", "sub").
					WithMetadata("cost_center", "CC-ENG-001").
					WithParent(pbc.LineageNodeType_LINEAGE_NODE_TYPE_ORGANIZATION, "org-1", "org").
					WithMetadata("owner_email", "finops@example.com").
					Build()
			},
			wantMetadata: []map[string]string{
				{"region": "us-east-1"},
				{"cost_center": "CC-ENG-001"},
				{"owner_email": "finops@example.com"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			leaf := tc.build()
			var got []map[string]string
			for node := leaf; node != nil; node = node.GetParent() {
				got = append(got, node.GetMetadata())
			}
			if diff := cmp.Diff(tc.wantMetadata, got); diff != "" {
				t.Errorf("metadata placement mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestLineageBuilderPartialChain verifies a resource-only chain (no parents)
// is valid: Build returns a non-nil leaf with nil Parent.
func TestLineageBuilderPartialChain(t *testing.T) {
	leaf := pluginsdk.NewLineageBuilder("res-only", "lonely-resource").Build()

	if leaf == nil {
		t.Fatal("Build() returned nil for resource-only chain")
	}
	if leaf.GetParent() != nil {
		t.Errorf("expected nil Parent for partial chain, got %v", leaf.GetParent())
	}
	if leaf.GetType() != pbc.LineageNodeType_LINEAGE_NODE_TYPE_RESOURCE {
		t.Errorf("leaf type = %v, want LINEAGE_NODE_TYPE_RESOURCE", leaf.GetType())
	}
	if leaf.GetId() != "res-only" || leaf.GetName() != "lonely-resource" {
		t.Errorf("leaf id/name = %q/%q, want %q/%q",
			leaf.GetId(), leaf.GetName(), "res-only", "lonely-resource")
	}
}

// TestLineageBuilderCustomAndDeepChain verifies LINEAGE_NODE_TYPE_CUSTOM
// levels and a 10-level chain work end to end.
func TestLineageBuilderCustomAndDeepChain(t *testing.T) {
	builder := pluginsdk.NewLineageBuilder("res-0", "resource")

	const depth = 9 // parents above the leaf -> 10 levels total
	want := []lineageLevel{
		{pbc.LineageNodeType_LINEAGE_NODE_TYPE_RESOURCE, "res-0", "resource"},
	}
	for i := 1; i <= depth; i++ {
		nodeType := pbc.LineageNodeType_LINEAGE_NODE_TYPE_CUSTOM
		if i == depth {
			nodeType = pbc.LineageNodeType_LINEAGE_NODE_TYPE_ORGANIZATION
		}
		builder.WithParent(nodeType, fmt.Sprintf("level-%d", i), fmt.Sprintf("Level %d", i))
		want = append(want, lineageLevel{nodeType, fmt.Sprintf("level-%d", i), fmt.Sprintf("Level %d", i)})
	}

	leaf := builder.Build()
	got := walkLineage(t, leaf, depth+1)
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(lineageLevel{})); diff != "" {
		t.Errorf("deep chain mismatch (-want +got):\n%s", diff)
	}
}

// TestLineageBuilderJSONRoundTrip verifies a built chain and an
// ActualCostResult carrying it survive a protojson round trip with full
// fidelity.
func TestLineageBuilderJSONRoundTrip(t *testing.T) {
	result := &pbc.ActualCostResult{
		Timestamp: timestamppb.Now(),
		Cost:      42.17,
		Source:    "test",
		Lineage:   buildFourLevelChain(),
	}

	roundTripProtojson(t, result, &pbc.ActualCostResult{})
}

// TestLineageBuilderIntegrationRoundTrip verifies lineage on both
// ActualCostResult and ResourceDescriptor survives a protojson round trip
// with order, ids, names, and per-level metadata intact.
func TestLineageBuilderIntegrationRoundTrip(t *testing.T) {
	t.Run("ActualCostResult with 4-level lineage", func(t *testing.T) {
		result := &pbc.ActualCostResult{
			Timestamp: timestamppb.Now(),
			Cost:      42.17,
			Source:    "test",
			Lineage:   buildFourLevelChain(),
		}
		roundTripProtojson(t, result, &pbc.ActualCostResult{})
	})

	t.Run("ResourceDescriptor with 4-level lineage", func(t *testing.T) {
		descriptor := &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
			Lineage:      buildFourLevelChain(),
		}
		roundTripProtojson(t, descriptor, &pbc.ResourceDescriptor{})
	})

	t.Run("lineage order, ids, names, and per-level metadata intact", func(t *testing.T) {
		result := &pbc.ActualCostResult{
			Timestamp: timestamppb.Now(),
			Cost:      42.17,
			Source:    "test",
			Lineage:   buildFourLevelChain(),
		}
		data, err := protojson.Marshal(result)
		if err != nil {
			t.Fatalf("protojson.Marshal failed: %v", err)
		}
		got := &pbc.ActualCostResult{}
		err = protojson.Unmarshal(data, got)
		if err != nil {
			t.Fatalf("protojson.Unmarshal failed: %v", err)
		}

		levels := walkLineage(t, got.GetLineage(), 10)
		if diff := cmp.Diff(wantFourLevelChain(), levels, cmp.AllowUnexported(lineageLevel{})); diff != "" {
			t.Errorf("chain levels mismatch (-want +got):\n%s", diff)
		}

		var gotMetadata []map[string]string
		for node := got.GetLineage(); node != nil; node = node.GetParent() {
			gotMetadata = append(gotMetadata, node.GetMetadata())
		}
		wantMetadata := []map[string]string{
			{"region": "us-east-1"},
			{"cost_center": "CC-ENG-001"},
			nil,
			nil,
		}
		if diff := cmp.Diff(wantMetadata, gotMetadata); diff != "" {
			t.Errorf("per-level metadata mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestActualCostResultLineageDisagreementRoundTrip is a negative case: a
// lineage that omits/disagrees with the FOCUS billing_account_id and
// sub_account_id fields still round-trips without error, because lineage is
// pass-through data with no consistency validation.
func TestActualCostResultLineageDisagreementRoundTrip(t *testing.T) {
	// The lineage stops at a custom "cost center" level: no billing account
	// node exists to agree or disagree with the FOCUS fields below. Both
	// omissions and disagreements are valid pass-through data.
	lineage := pluginsdk.NewLineageBuilder("res-9", "unattached-resource").
		WithParent(pbc.LineageNodeType_LINEAGE_NODE_TYPE_CUSTOM, "cc-override", "Override Cost Center").
		Build()

	result := &pbc.ActualCostResult{
		Timestamp: timestamppb.Now(),
		Cost:      7.50,
		Source:    "test",
		Lineage:   lineage,
		FocusRecord: &pbc.FocusCostRecord{
			BillingAccountId: "billing-from-focus",
			SubAccountId:     "sub-from-focus",
		},
	}

	got := &pbc.ActualCostResult{}
	roundTripProtojson(t, result, got)

	// The FOCUS fields remain canonical and untouched by the lineage.
	if got.GetFocusRecord().GetBillingAccountId() != "billing-from-focus" {
		t.Errorf("billing_account_id = %q, want %q",
			got.GetFocusRecord().GetBillingAccountId(), "billing-from-focus")
	}
	if got.GetLineage().GetParent().GetType() != pbc.LineageNodeType_LINEAGE_NODE_TYPE_CUSTOM {
		t.Errorf("parent type = %v, want LINEAGE_NODE_TYPE_CUSTOM",
			got.GetLineage().GetParent().GetType())
	}
}
