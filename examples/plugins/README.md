# Plugin Manifest Examples

This directory contains comprehensive examples of plugin manifests for the FinFocus Plugin Registry system.

## Overview

These examples demonstrate various plugin configurations, installation methods, and feature sets across different cloud
providers and deployment scenarios.

## Examples

### Cloud Provider Plugins

#### AWS Cost Plugin (`aws-cost-plugin.json`)

- **Full-featured example** with all manifest sections
- **Binary installation** method
- **Comprehensive security** with signature verification
- **Multi-service support**: EC2, S3, Lambda, RDS, DynamoDB
- **Advanced features**: Caching, filtering, observability
- **Configuration examples** for production and development

#### Azure Cost Plugin (`azure-cost-plugin.json`)

- **Container-based deployment** using Docker
- **OAuth2 authentication** with service principal
- **Multi-service support**: VMs, Storage, SQL Database, App Services
- **Community security level** example
- **Simplified configuration** with minimal required fields

#### GCP Cost Plugin (`gcp-cost-plugin.json`)

- **Package manager installation** (apt/deb)
- **Service account authentication**
- **Multi-service support**: Compute Engine, Cloud Storage, BigQuery, Cloud SQL
- **Commitment-based pricing** support (preemptible, committed use)
- **Flexible authentication** (service account file or metadata)

#### Kubecost Plugin (`kubecost-plugin.json`)

- **Script-based installation** method
- **Kubernetes-native** resource types (namespace, pod, node)
- **Real-time cost tracking** capabilities
- **Optional dependencies** example
- **Multiple configuration scenarios** (in-cluster, external)

#### GreenOps Plugin (`greenops-plugin.json`)

- **Protocol capabilities** in a manifest: `projected_costs`, `carbon`, `energy`, `water`, `dry_run`
  (the `PluginCapability` names without the `PLUGIN_CAPABILITY_` prefix)
- **Methods beyond the core five**: `GetPluginInfo` and `DryRun`
- Per-resource metric support is reported at runtime through `SupportsResponse.supported_metrics`,
  not in the manifest

### Minimal Example

#### Minimal Plugin (`minimal-plugin.json`)

- **Bare minimum required fields** demonstration
- **Custom provider** support
- **Binary installation** with default settings
- **Useful for plugin development** starting point

## Validation

All examples are validated against the plugin manifest schema:

```bash
# Validate all examples
for file in examples/plugins/*.json; do
    npx ajv validate --spec=draft2020 --strict=false \
        -s schemas/plugin_manifest.schema.json -d "$file"
done
```

## Usage Patterns

### Installation Methods Demonstrated

1. **Binary** (`aws-cost-plugin.json`, `minimal-plugin.json`)
   - Direct executable download and installation
   - Checksum verification
   - Pre/post-install scripts

2. **Container** (`azure-cost-plugin.json`)
   - Docker image deployment
   - Registry-based distribution, with `download_url` as an `oci://` image reference (the schema
     requires a URI)
   - Containerized runtime isolation

3. **Script** (`kubecost-plugin.json`)
   - Custom installation script execution
   - Complex installation logic
   - Environment-specific setup

4. **Package** (`gcp-cost-plugin.json`)
   - System package manager integration
   - Platform-specific packages
   - Automated dependency resolution

### Security Levels

- **Official**: Not demonstrated (reserved for FinFocus team)
- **Verified**: `aws-cost-plugin.json`, `kubecost-plugin.json`
  - Digital signatures required
  - Comprehensive security review
  - Enterprise deployment ready

- **Community**: `azure-cost-plugin.json`, `gcp-cost-plugin.json`
  - Community verification
  - Basic security requirements
  - Lower friction installation

- **Untrusted**: `minimal-plugin.json` (default)
  - User approval required
  - Minimal security validation
  - Development/testing use

### Configuration Complexity

- **Simple**: `minimal-plugin.json` - No configuration required
- **Basic**: `kubecost-plugin.json` - Single required endpoint
- **Standard**: `azure-cost-plugin.json`, `gcp-cost-plugin.json` - Authentication credentials
- **Advanced**: `aws-cost-plugin.json` - Multiple examples, feature flags, performance tuning

## Plugin Development Guidelines

When creating your own plugin manifest:

1. **Start with minimal example** and add features incrementally
2. **Choose appropriate security level** based on your verification status
3. **Select installation method** that fits your distribution model
4. **Provide comprehensive configuration examples** for common scenarios
5. **Validate manifest** against schema before distribution
6. **Include observability features** for production deployments

## Schema Reference

All examples conform to the plugin manifest schema defined in:
`../../schemas/plugin_manifest.schema.json`

For detailed field descriptions and validation rules, refer to the schema documentation.
