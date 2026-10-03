# Contract: Go SDK surface

## `pluginsdk`

```go
const (
    MaxTagValueLength  = 2048     // was 256
    MaxAttributesBytes = 64 << 10 // new
)

// ValidateResourceDescriptor additionally rejects attributes whose proto.Size
// exceeds MaxAttributesBytes (codes.InvalidArgument).
func ValidateResourceDescriptor(resource *pbc.ResourceDescriptor) error

// AttributeValue returns the value at a dot-separated path within attrs.
func AttributeValue(attrs *structpb.Struct, path string) (*structpb.Value, bool)
```

## `testing` (plugintesting)

```go
const (
    MaxTagValueLength  = 2048     // was 256
    MaxAttributesBytes = 64 << 10 // new
)

var ErrAttributesTooLarge = fmt.Errorf("attributes exceed maximum encoded size of %d bytes", MaxAttributesBytes)

// ValidateResourceDescriptor additionally returns
// NewContractError("attributes", size, ErrAttributesTooLarge).
func ValidateResourceDescriptor(resource *pbc.ResourceDescriptor) error
```

New conformance test: `RPCCorrectness_GetProjectedCostWithAttributes` (Basic). It also adds contract-suite
cases `ResourceDescriptor_AttributesAtLimitAccepted` and `ResourceDescriptor_AttributesOverLimitRejected`.

## TypeScript

`ResourceDescriptor.attributes?: JsonObject` (generated). There is no new hand-written API.
