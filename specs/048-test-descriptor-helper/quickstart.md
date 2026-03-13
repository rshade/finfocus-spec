# Quickstart: Using newTestDescriptor

## Overview

The `newTestDescriptor` helper creates `ResourceDescriptor` instances with standard test
defaults, reducing boilerplate in `helpers_test.go`.

## Usage Examples

### Basic Usage (ID and ARN)

```go
desc := newTestDescriptor("resource-001", "arn:aws:ec2:us-east-1:123456789012:instance/i-abc123")
// Result: provider="aws", resourceType="ec2", sku="t3.micro", region="us-east-1",
//         id="resource-001", arn="arn:aws:ec2:us-east-1:123456789012:instance/i-abc123"
```

### Defaults Only (no ID or ARN)

```go
desc := newTestDescriptor("", "")
// Result: provider="aws", resourceType="ec2", sku="t3.micro", region="us-east-1"
//         id and arn are NOT set (empty)
```

### With Additional Options

```go
desc := newTestDescriptor("id-001", "",
    pluginsdk.WithTags(map[string]string{"env": "prod"}),
)
// Result: standard defaults + id="id-001" + tags={"env": "prod"}
```

### Overriding a Default

```go
desc := newTestDescriptor("id-001", "",
    pluginsdk.WithRegion("eu-west-1"),  // Overrides default "us-east-1"
)
// Result: provider="aws", resourceType="ec2", sku="t3.micro", region="eu-west-1", id="id-001"
```

## When NOT to Use

- Tests that verify `NewResourceDescriptor` construction behavior — keep those explicit
- Tests using non-standard providers (e.g., `"gcp"`, `"azure"`, `"custom"`)
- Fuzz tests with intentionally generic parameters
