# JSON-LD Serialization Package

High-performance JSON-LD serialization for FOCUS cost data, enabling enterprise knowledge graph
integration with Schema.org vocabulary mapping.

## Features

- **JSON-LD 1.1 Compliant** - Valid JSON-LD output with `@context`, `@type`, and `@id`
- **Schema.org Integration** - MonetaryAmount, DateTime, and PropertyValue type mappings
- **FOCUS Vocabulary** - Custom namespace for FinOps-specific fields
- **Streaming Support** - Bounded memory for large datasets (10,000+ records)
- **Customizable Context** - Enterprise ontology integration
- **Minimal Dependencies** - Only protobuf required, no JSON-LD libraries

## Installation

```go
import "github.com/rshade/finfocus-spec/sdk/go/jsonld"
```

## Quick Start

The examples below assume the following imports:

```go
import (
    "bytes"
    "context"
    "fmt"
    "log"
    "time"

    "google.golang.org/protobuf/types/known/timestamppb"
    pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
    "github.com/rshade/finfocus-spec/sdk/go/jsonld"
)
```

### Serialize a Single Record

```go
serializer := jsonld.NewSerializer()

record := &pbc.FocusCostRecord{
    BillingAccountId:  "123456789012",
    ChargePeriodStart: timestamppb.Now(),
    ServiceName:       "Amazon EC2",
    BilledCost:        125.50,
    BillingCurrency:   "USD",
}

output, err := serializer.Serialize(record)
if err != nil {
    log.Fatal(err)
}

fmt.Println(string(output))
```

### Batch Serialization (Streaming)

```go
serializer := jsonld.NewSerializer()

records := []*pbc.FocusCostRecord{...} // 10,000+ records

var buf bytes.Buffer
result, err := serializer.SerializeSlice(context.Background(), records, &buf)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Wrote %d records\n", result.RecordsWritten)
```

### Custom Context Configuration

```go
ctx := jsonld.NewContext().
    WithRemoteContext("https://your-org.com/ontology/v1").
    WithCustomMapping("billingAccountId", "yourOrg:accountIdentifier")

serializer := jsonld.NewSerializer(
    jsonld.WithContext(ctx),
)
```

### User-Provided IDs

```go
serializer := jsonld.NewSerializer(
    jsonld.WithUserIDField("invoice_id"),
)
```

## Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithContext(ctx)` | Default FOCUS context | Custom JSON-LD context |
| `WithOmitEmpty(bool)` | `true` | Omit empty/zero/nil fields |
| `WithIRIEnums(bool)` | `false` | Serialize enums as full IRIs |
| `WithDeprecated(bool)` | `true` | Include deprecated fields |
| `WithPrettyPrint(bool)` | `false` | Indent JSON output |
| `WithUserIDField(field)` | none | Field to use as @id |
| `WithIDPrefix(prefix)` | `urn:focus:cost:` | Prefix for generated IDs |

## Output Format

### FocusCostRecord

```json
{
  "@context": {
    "schema": "https://schema.org/",
    "focus": "https://focus.finops.org/v1#",
    "xsd": "http://www.w3.org/2001/XMLSchema#"
  },
  "@type": "focus:FocusCostRecord",
  "@id": "urn:focus:cost:a1b2c3d4...",
  "billingAccountId": "123456789012",
  "serviceName": "Amazon EC2",
  "billedCost": {
    "@type": "schema:MonetaryAmount",
    "value": 125.50,
    "currency": "USD"
  },
  "chargePeriodStart": "2025-01-01T00:00:00Z"
}
```

### ContractCommitment

```go
serializer := jsonld.NewSerializer()

commitment := &pbc.ContractCommitment{
    ContractCommitmentId:   "commit-001",
    ContractId:             "contract-001",
    ContractCommitmentCost: 10000.00,
    BillingCurrency:        "USD",
}

output, err := serializer.SerializeCommitment(commitment)
if err != nil {
    log.Fatal(err)
}

fmt.Println(string(output))
```

## Performance

| Operation | Records | Time | Allocations |
|-----------|---------|------|-------------|
| Single Record | 1 | ~15.3µs | ~134 |
| Batch | 100 | ~2ms | ~13K |
| Batch | 1,000 | ~18ms | ~134K |
| Batch | 10,000 | ~182ms | ~1.3M |
| Streaming | 10,000 | ~197ms | ~1.3M |

Run benchmarks:

```bash
go test -bench=. -benchmem ./sdk/go/jsonld/...
```

## Schema.org Mappings

| FOCUS Field | Schema.org Type |
|-------------|-----------------|
| billed_cost, list_cost, effective_cost | MonetaryAmount |
| charge_period_start, charge_period_end | DateTime (ISO 8601) |
| service_name | Service.name |
| resource_name | Thing.name |
| region_name | Place.name |
| tags | PropertyValue[] |

Fields without Schema.org equivalents use the FOCUS namespace (`focus:fieldName`).

## Error Handling

### Validation Errors

```go
output, err := serializer.Serialize(record)
if err != nil {
    if valErr, ok := err.(*jsonld.ValidationError); ok {
        fmt.Printf("Field: %s, Message: %s\n", valErr.Field, valErr.Message)
    }
}
```

### Streaming Errors

```go
result, err := serializer.SerializeSlice(context.Background(), records, &buf)
if result.HasErrors() {
    for _, e := range result.Errors {
        fmt.Printf("Index %d: %v\n", e.Index, e.Err)
    }
}
```

### Context Cancellation and Output Validation

When using context cancellation with streaming operations, the output may be corrupted if
cancellation occurs mid-stream. Use the helper methods to validate output safety:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

var buf bytes.Buffer
result, err := serializer.SerializeStream(ctx, recordsChan, &buf)

// Always check the error first
if err != nil {
    // For non-cancel errors (e.g., write failures), the output is unusable
    if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
        return fmt.Errorf("serialization failed: %w", err)
    }
    // Context was cancelled — check if the partial output is still valid
    if !result.IsOutputValid() {
        // ValidateOutputJSON checks if the buffer contains parseable JSON.
        // This only proves syntactic validity, not that serialization completed.
        if valErr := result.ValidateOutputJSON(buf.Bytes()); valErr != nil {
            // Output is corrupted - must retry from scratch
            return retry()
        }
        // Output is parseable JSON despite cancellation - safe to use
    }
}
// err == nil && result.IsOutputValid(): serialization completed successfully
```

**Helper Methods**:

- `IsOutputValid()` - Returns `false` if context cancellation may have corrupted the output
- `ValidateOutputJSON(data []byte) error` - Checks if the buffer contains parseable JSON
  when `CorruptedOnCancel` is true; does **not** guarantee serialization completed successfully

**Best Practices**:

1. Always check the `err` return value before using the output buffer
2. Check `IsOutputValid()` when context cancellation is possible
3. Use `ValidateOutputJSON()` to verify JSON parseability for partial output
4. Consider transactional writes (write to temp file, rename on success) for critical operations

## Testing

```bash
# Run all tests
go test -v ./sdk/go/jsonld/...

# Run benchmarks
go test -bench=. -benchmem ./sdk/go/jsonld/...

# Run conformance tests only
go test -v -run TestConformance ./sdk/go/jsonld/...
```

## Related Documentation

- [Example Outputs](../../../examples/jsonld/)
