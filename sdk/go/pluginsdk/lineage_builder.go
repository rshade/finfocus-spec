package pluginsdk

import (
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// LineageBuilder handles the construction of cost allocation lineage chains.
//
// The builder assembles a singly-linked list of LineageNode values from leaf
// (the resource itself) up toward the organizational root. The leaf node is
// created by NewLineageBuilder, and each WithParent call attaches one level
// above the current top of the chain.
//
// # Pass-Through Semantics
//
// The builder only assembles caller-supplied data:
//   - It does NOT infer missing parents or synthesize chain levels.
//   - It does NOT validate ids against provider APIs.
//   - It does NOT cross-check nodes against billing_account_id or
//     sub_account_id fields; FOCUS account fields remain canonical.
//   - Partial chains are valid: a resource node with no parents is a
//     well-formed lineage.
//   - Arbitrary depth is allowed by the builder. Practical protobuf recursion
//     limits apply at encode time for very deep chains; keep chains to a
//     reasonable organizational depth (typically under 10 levels).
//
// # Example
//
//	lineage := pluginsdk.NewLineageBuilder("i-0abc123", "web-server").
//	    WithMetadata("owner_email", "team@example.com").
//	    WithParent(pbc.LineageNodeType_LINEAGE_NODE_TYPE_SUB_ACCOUNT, "123456789012", "Production").
//	    WithMetadata("cost_center", "CC-ENG-001").
//	    WithParent(pbc.LineageNodeType_LINEAGE_NODE_TYPE_BILLING_ACCOUNT, "999988887777", "Payer").
//	    WithParent(pbc.LineageNodeType_LINEAGE_NODE_TYPE_ORGANIZATION, "o-exampleorg", "Example Org").
//	    Build()
//
// Build returns the leaf node; the chain walks up via Parent pointers.
//
// Thread Safety: NOT thread-safe. Do not call from multiple goroutines.
type LineageBuilder struct {
	leaf *pbc.LineageNode
	top  *pbc.LineageNode
}

// NewLineageBuilder creates a new builder with the resource (leaf) node.
// The leaf node is typed LINEAGE_NODE_TYPE_RESOURCE and carries the given
// resource id and name.
func NewLineageBuilder(resourceID, resourceName string) *LineageBuilder {
	leaf := &pbc.LineageNode{
		Type: pbc.LineageNodeType_LINEAGE_NODE_TYPE_RESOURCE,
		Id:   resourceID,
		Name: resourceName,
	}
	return &LineageBuilder{
		leaf: leaf,
		top:  leaf,
	}
}

// WithParent attaches a new node of nodeType above the current top of the
// chain and makes it the new top. Subsequent WithParent calls chain above
// this node; subsequent WithMetadata calls target this node.
func (b *LineageBuilder) WithParent(
	nodeType pbc.LineageNodeType,
	id, name string,
) *LineageBuilder {
	node := &pbc.LineageNode{
		Type: nodeType,
		Id:   id,
		Name: name,
	}
	b.top.Parent = node
	b.top = node
	return b
}

// WithMetadata sets a key/value pair on the most recently added node (the
// current top of the chain). Before any WithParent call, that is the resource
// (leaf) node itself.
func (b *LineageBuilder) WithMetadata(key, value string) *LineageBuilder {
	if b.top.Metadata == nil {
		b.top.Metadata = make(map[string]string)
	}
	b.top.Metadata[key] = value
	return b
}

// Build returns the leaf (resource) node of the constructed chain. The chain
// walks upward via Parent pointers; the topmost node's Parent is nil.
//
// Build performs no validation: the lineage is caller-supplied pass-through
// data, and partial chains are valid by design.
func (b *LineageBuilder) Build() *pbc.LineageNode {
	return b.leaf
}
