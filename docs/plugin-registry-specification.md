# Plugin Registry Specification v0.1.0

## Overview

The FinFocus Plugin Registry Specification defines the comprehensive protocol for plugin discovery, registration, and
lifecycle management within the FinFocus ecosystem. This specification enables standardized plugin distribution,
installation, and management across different deployment environments.

## Table of Contents

- [Plugin Registry Specification v0.1.0](#plugin-registry-specification-v010)
  - [Overview](#overview)
  - [Table of Contents](#table-of-contents)
  - [Core Concepts](#core-concepts)
    - [Plugin Identity](#plugin-identity)
    - [Plugin Manifest](#plugin-manifest)
    - [Discovery Sources](#discovery-sources)
    - [Security Model](#security-model)
  - [Plugin Manifest Format](#plugin-manifest-format)
    - [Metadata Section](#metadata-section)
    - [Specification Section](#specification-section)
    - [Security Section](#security-section)
    - [Installation Section](#installation-section)
    - [Configuration Section](#configuration-section)
    - [Requirements Section](#requirements-section)
  - [Discovery Mechanisms](#discovery-mechanisms)
    - [Filesystem Discovery](#filesystem-discovery)
    - [Registry Discovery](#registry-discovery)
    - [URL Discovery](#url-discovery)
    - [Git Repository Discovery](#git-repository-discovery)
  - [Versioning and Compatibility](#versioning-and-compatibility)
    - [Semantic Versioning](#semantic-versioning)
    - [Specification Compatibility](#specification-compatibility)
    - [Dependency Management](#dependency-management)
    - [Version Constraints](#version-constraints)
  - [Security and Authentication](#security-and-authentication)
    - [Security Levels](#security-levels)
    - [Plugin Signing](#plugin-signing)
    - [Verification Process](#verification-process)
    - [Permission Model](#permission-model)
    - [Sandboxing](#sandboxing)
  - [Plugin Lifecycle Management](#plugin-lifecycle-management)
    - [Discovery Phase](#discovery-phase)
    - [Validation Phase](#validation-phase)
    - [Installation Phase](#installation-phase)
    - [Update Management](#update-management)
    - [Removal Process](#removal-process)
    - [Health Monitoring](#health-monitoring)
  - [gRPC Service Interface](#grpc-service-interface)
    - [PluginRegistryService](#pluginregistryservice)
    - [Service Methods](#service-methods)
    - [Error Handling](#error-handling)
  - [Implementation Guidelines](#implementation-guidelines)
    - [Registry Implementation](#registry-implementation)
    - [Client Implementation](#client-implementation)
    - [Plugin Developer Guidelines](#plugin-developer-guidelines)
  - [Examples](#examples)
    - [Complete Plugin Manifest](#complete-plugin-manifest)
    - [Discovery Configuration](#discovery-configuration)
    - [Installation Scripts](#installation-scripts)
  - [Appendices](#appendices)
    - [JSON Schema Reference](#json-schema-reference)
    - [Error Codes](#error-codes)
    - [Version History](#version-history)

## Core Concepts

### Plugin Identity

Every plugin in the FinFocus ecosystem is uniquely identified by:

- **Name**: A unique identifier following the pattern `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
- **Version**: Semantic version following SemVer 2.0.0 specification
- **Author**: The plugin author or organization

### Plugin Manifest

The plugin manifest is a JSON document that describes:

- Plugin metadata and identification
- Supported capabilities and providers
- Security and authentication requirements
- Installation and configuration instructions
- System and dependency requirements

### Discovery Sources

Plugins can be discovered from multiple sources:

- **Filesystem**: Local directory scanning for manifest files
- **Registry**: Remote registry API endpoints
- **URL**: Direct URL downloads of manifest files
- **Git**: Git repository-based plugin distribution

### Security Model

The security model operates on multiple trust levels:

- **Untrusted**: Requires explicit user approval
- **Community**: Community-verified plugins
- **Verified**: Officially verified by trusted authorities
- **Official**: Official FinFocus plugins

## Plugin Manifest Format

The plugin manifest follows the JSON schema defined in [`schemas/plugin_manifest.schema.json`](../schemas/plugin_manifest.schema.json).

### Metadata Section

Contains basic plugin information:

```json
{
  "metadata": {
    "name": "aws-cost-plugin",
    "version": "1.2.3",
    "description": "AWS cost source plugin for retrieving EC2, S3, and Lambda pricing",
    "author": "FinFocus Team",
    "homepage": "https://example.com/aws-plugin",
    "repository": "https://github.com/example/aws-plugin",
    "license": "Apache-2.0",
    "keywords": ["aws", "cost", "ec2", "s3", "lambda"],
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-03-15T12:30:00Z"
  }
}
```

**Required Fields**:

- `name`: Unique plugin identifier
- `version`: Semantic version
- `description`: Human-readable description (10-500 characters)
- `author`: Plugin author/organization

**Optional Fields**:

- `homepage`: Plugin homepage URL
- `repository`: Source code repository URL
- `license`: SPDX license identifier
- `keywords`: Searchable keywords (max 10)
- `created_at`, `updated_at`: ISO 8601 timestamps

### Specification Section

Defines plugin capabilities and interface compliance:

```json
{
  "specification": {
    "spec_version": "0.1.0",
    "supported_providers": ["aws"],
    "supported_resources": {
      "aws": {
        "resource_types": ["ec2", "s3", "lambda", "rds"],
        "billing_modes": [
          "per_hour",
          "per_gb_month",
          "per_invocation",
          "per_dtu"
        ],
        "regions": ["us-east-1", "us-west-2", "eu-west-1"]
      }
    },
    "capabilities": [
      "cost_retrieval",
      "cost_projection",
      "pricing_specs",
      "historical_data"
    ],
    "service_definition": {
      "service_name": "CostSourceService",
      "package_name": "finfocus.v1",
      "methods": [
        "Name",
        "Supports",
        "GetActualCost",
        "GetProjectedCost",
        "GetPricingSpec"
      ],
      "port": 50051,
      "health_check_path": "/health"
    },
    "observability_support": {
      "metrics_enabled": true,
      "tracing_enabled": true,
      "logging_enabled": true,
      "health_checks_enabled": true,
      "sli_support": false
    }
  }
}
```

**Key Components**:

- `spec_version`: FinFocus specification version supported
- `supported_providers`: Array of cloud providers (aws, azure, gcp, kubernetes, custom).
  Entries are clouds, not IaC packages: a plugin that prices `azure-native` resources lists `azure`
- `supported_resources`: Detailed resource support per provider. Keys are clouds, and a native
  package's resource types go under their cloud (`azure-native` types under `azure`)
- `service_definition`: gRPC service implementation details
- `observability_support`: Telemetry and monitoring capabilities

### Security Section

Contains security-related information:

```json
{
  "security": {
    "signature": "MEUCIQDx...",
    "public_key": "-----BEGIN PUBLIC KEY-----\n...",
    "certificate_chain": ["-----BEGIN CERTIFICATE-----\n..."],
    "security_level": "verified",
    "permissions": ["network_access", "filesystem_read", "config_read"],
    "sandbox_required": false
  }
}
```

**Security Fields**:

- `signature`: Base64-encoded plugin signature
- `public_key`: Public key for verification
- `certificate_chain`: Certificate chain for verification
- `security_level`: Trust level (untrusted, community, verified, official)
- `permissions`: Required system permissions
- `sandbox_required`: Whether sandboxing is required

### Installation Section

Provides installation instructions:

```json
{
  "installation": {
    "installation_method": "binary",
    "download_url": "https://releases.example.com/aws-plugin-v1.2.3-linux-amd64.tar.gz",
    "checksum": "sha256:a1b2c3d4e5f6...",
    "checksum_algorithm": "sha256",
    "install_script": "#!/bin/bash\ntar -xzf aws-plugin-v1.2.3-linux-amd64.tar.gz\n...",
    "pre_install_checks": ["check_system_requirements", "verify_dependencies"],
    "post_install_steps": [
      "create_config_directory",
      "set_permissions",
      "register_service"
    ]
  }
}
```

**Installation Methods**:

- `binary`: Direct binary download and installation
- `container`: Container image deployment
- `script`: Installation via script execution
- `package`: System package manager integration

### Configuration Section

Defines plugin configuration schema:

```json
{
  "configuration": {
    "schema": "{\"type\": \"object\", \"properties\": {\"api_key\": {\"type\": \"string\"}}}",
    "default_config": "{\"timeout\": 30, \"retry_count\": 3}",
    "required_fields": ["api_key", "region"],
    "examples": [
      {
        "name": "basic_configuration",
        "description": "Basic AWS configuration with API key",
        "config": "{\"api_key\": \"your-api-key\", \"region\": \"us-east-1\"}"
      }
    ]
  }
}
```

### Requirements Section

Specifies system and dependency requirements:

```json
{
  "requirements": {
    "min_spec_version": "0.1.0",
    "max_spec_version": "0.2.0",
    "dependencies": [
      {
        "name": "base-auth-plugin",
        "version_constraint": ">=1.0.0,<2.0.0",
        "optional": false
      }
    ],
    "system_requirements": {
      "min_memory_mb": 256,
      "min_disk_mb": 100,
      "supported_architectures": ["x86_64", "arm64"],
      "supported_os": ["linux", "darwin", "windows"]
    },
    "runtime_requirements": {
      "grpc_version": "1.50.0",
      "tls_required": true,
      "auth_methods": ["api_key", "jwt"],
      "timeout_seconds": 30
    }
  }
}
```

## Discovery Mechanisms

### Filesystem Discovery

Scans local filesystem directories for plugin manifests.

**Configuration**:

```json
{
  "discovery_sources": [
    {
      "type": "filesystem",
      "path": "/usr/local/plugins",
      "recursive": true,
      "manifest_filename": "plugin.json"
    }
  ]
}
```

**Behavior**:

- Recursively scans specified directories
- Looks for `plugin.json` or `manifest.json` files
- Validates manifest format and schema
- Caches results for performance

### Registry Discovery

Queries remote plugin registries via HTTP API.

**Configuration**:

```json
{
  "discovery_sources": [
    {
      "type": "registry",
      "url": "https://registry.finfocus.dev",
      "api_key": "optional-api-key",
      "timeout_seconds": 30,
      "cache_ttl_seconds": 300
    }
  ]
}
```

**API Endpoints**:

- `GET /plugins` - List available plugins
- `GET /plugins/{name}` - Get plugin details
- `GET /plugins/{name}/versions` - List plugin versions
- `GET /plugins/{name}/versions/{version}/manifest` - Get manifest

### URL Discovery

Downloads plugin manifests from direct URLs.

**Configuration**:

```json
{
  "discovery_sources": [
    {
      "type": "url",
      "urls": [
        "https://example.com/plugins/aws-plugin/manifest.json",
        "https://example.com/plugins/azure-plugin/manifest.json"
      ],
      "headers": {
        "Authorization": "Bearer token"
      }
    }
  ]
}
```

### Git Repository Discovery

Discovers plugins from Git repositories.

**Configuration**:

```json
{
  "discovery_sources": [
    {
      "type": "git",
      "repository": "https://github.com/example/plugins",
      "branch": "main",
      "path": "plugins/",
      "credentials": {
        "username": "token",
        "password": "github_pat_..."
      }
    }
  ]
}
```

## Versioning and Compatibility

### Semantic Versioning

All plugins must follow [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR**: Incompatible API changes
- **MINOR**: Backward-compatible functionality additions
- **PATCH**: Backward-compatible bug fixes

**Examples**:

- `1.0.0` - Initial release
- `1.1.0` - New features, backward compatible
- `1.1.1` - Bug fixes
- `2.0.0` - Breaking changes

### Specification Compatibility

Plugins declare compatibility with FinFocus specification versions:

```json
{
  "requirements": {
    "min_spec_version": "0.1.0",
    "max_spec_version": "0.2.0"
  }
}
```

**Compatibility Matrix**:

- Plugin `1.0.0` supports spec `0.1.0-0.2.0`
- Spec `0.1.5` is compatible with plugin `1.0.0`
- Spec `0.3.0` is NOT compatible with plugin `1.0.0`

### Dependency Management

Plugins can depend on other plugins:

```json
{
  "dependencies": [
    {
      "name": "auth-helper",
      "version_constraint": "^1.2.0",
      "optional": false
    },
    {
      "name": "metrics-collector",
      "version_constraint": ">=2.0.0,<3.0.0",
      "optional": true
    }
  ]
}
```

### Version Constraints

Supported version constraint syntax:

- `1.2.3` - Exact version
- `^1.2.3` - Compatible within major version
- `~1.2.3` - Compatible within minor version
- `>=1.2.3` - Greater than or equal
- `<2.0.0` - Less than
- `>=1.2.3,<2.0.0` - Range constraint

## Security and Authentication

### Security Levels

Four security trust levels:

1. **Untrusted** (`untrusted`)
   - Default for new plugins
   - Requires explicit user approval
   - May have restricted permissions
   - Manual verification required

2. **Community** (`community`)
   - Community-verified plugins
   - Basic security review completed
   - Automated testing passed
   - Lower friction installation

3. **Verified** (`verified`)
   - Officially verified by trusted authorities
   - Comprehensive security review
   - Code signing required
   - Enterprise deployment ready

4. **Official** (`official`)
   - Official FinFocus plugins
   - Maintained by core team
   - Highest trust level
   - Automatic updates allowed

### Plugin Signing

Plugins should be cryptographically signed for integrity verification:

**Signing Process**:

1. Generate plugin package (binary, manifest, assets)
2. Create SHA-256 hash of package
3. Sign hash with private key (RSA 2048+ or ECDSA P-256+)
4. Include signature and public key in manifest

**Signature Format**:

```json
{
  "security": {
    "signature": "MEUCIQD...",
    "public_key": "-----BEGIN PUBLIC KEY-----\nMII...",
    "certificate_chain": ["-----BEGIN CERTIFICATE-----\n..."]
  }
}
```

### Verification Process

Plugin verification follows these steps:

1. **Manifest Validation**: Validate against JSON schema
2. **Signature Verification**: Verify cryptographic signature
3. **Certificate Validation**: Validate certificate chain if present
4. **Dependency Check**: Verify all dependencies are available
5. **System Requirements**: Check system compatibility
6. **Security Review**: Apply security policies based on trust level

### Permission Model

Plugins declare required system permissions:

**Available Permissions**:

- `network_access` - Outbound network connections
- `filesystem_read` - Read filesystem access
- `filesystem_write` - Write filesystem access
- `environment_read` - Read environment variables
- `process_spawn` - Spawn child processes
- `system_info` - Access system information
- `temp_files` - Create temporary files
- `config_read` - Read configuration files
- `metrics_collect` - Collect system metrics

**Permission Grant Process**:

1. Plugin declares required permissions in manifest
2. Registry validates permission requirements
3. During installation, user is prompted to approve permissions
4. Runtime enforces granted permissions

### Sandboxing

High-security environments may require plugin sandboxing:

```json
{
  "security": {
    "sandbox_required": true,
    "permissions": ["network_access", "temp_files"]
  }
}
```

**Sandbox Implementation**:

- Container-based isolation
- Resource limits (CPU, memory, network)
- Restricted filesystem access
- Network policy enforcement

## Plugin Lifecycle Management

### Discovery Phase

1. **Source Scanning**: Query all configured discovery sources
2. **Manifest Collection**: Retrieve plugin manifests
3. **Initial Validation**: Validate manifest syntax and schema
4. **Indexing**: Build searchable plugin index
5. **Caching**: Cache results for performance

**Discovery Flow**:

```text
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ Discovery       │    │ Manifest         │    │ Plugin          │
│ Sources         │───▶│ Validation       │───▶│ Index           │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Validation Phase

Comprehensive validation before installation:

1. **Schema Validation**: Validate against plugin manifest schema
2. **Dependency Resolution**: Check all required dependencies
3. **System Compatibility**: Verify system requirements
4. **Security Assessment**: Apply security policies
5. **Version Compatibility**: Check spec version compatibility

**Validation Errors**:

- **Critical**: Prevent installation (schema errors, missing dependencies)
- **Warnings**: Allow installation with user confirmation (performance concerns)
- **Info**: Informational messages (recommendations)

### Installation Phase

Multi-step installation process:

1. **Pre-installation Checks**: Run validation and pre-install checks
2. **Download**: Retrieve plugin files from source
3. **Verification**: Verify checksums and signatures
4. **Extraction**: Extract plugin files to destination
5. **Configuration**: Apply default configuration
6. **Registration**: Register plugin with system
7. **Post-installation**: Run post-install steps
8. **Health Check**: Verify plugin health

**Installation Methods**:

**Binary Installation**:

```bash
# Download and extract
wget https://releases.example.com/plugin-v1.0.0.tar.gz
tar -xzf plugin-v1.0.0.tar.gz -C /usr/local/plugins/

# Set permissions
chmod +x /usr/local/plugins/plugin-v1.0.0/bin/plugin

# Create symlink
ln -s /usr/local/plugins/plugin-v1.0.0/bin/plugin /usr/local/bin/plugin
```

**Container Installation**:

```bash
# Pull image
docker pull registry.example.com/plugin:1.0.0

# Create container
docker run -d --name plugin \
  -p 50051:50051 \
  -v /etc/plugin:/config \
  registry.example.com/plugin:1.0.0
```

### Update Management

Plugin updates follow semantic versioning rules:

**Update Types**:

- **Patch Updates**: Automatic (bug fixes)
- **Minor Updates**: User confirmation (new features)
- **Major Updates**: Manual process (breaking changes)

**Update Process**:

1. **Update Detection**: Check for newer versions
2. **Compatibility Check**: Verify compatibility
3. **Backup**: Create backup of current version
4. **Download**: Retrieve new version
5. **Install**: Install new version
6. **Migration**: Run migration scripts if needed
7. **Verification**: Verify update success
8. **Rollback**: Rollback on failure

### Removal Process

Clean plugin removal:

1. **Dependency Check**: Verify no other plugins depend on this one
2. **Service Stop**: Stop running plugin services
3. **Data Backup**: Backup plugin data if requested
4. **File Removal**: Remove plugin files
5. **Configuration Cleanup**: Remove configuration files
6. **Registry Cleanup**: Remove from plugin registry
7. **Verification**: Verify complete removal

### Health Monitoring

Continuous plugin health monitoring:

**Health Metrics**:

- Service availability (gRPC health checks)
- Response time performance
- Error rates and types
- Resource utilization
- Dependency status

**Health States**:

- **Healthy**: Operating normally
- **Warning**: Performance degradation
- **Critical**: Service failures
- **Unknown**: Cannot determine status

## gRPC Service Interface

### PluginRegistryService

The registry implements the `PluginRegistryService` gRPC interface defined in [`proto/finfocus/v1/registry.proto`](../proto/finfocus/v1/registry.proto).

### Service Methods

**Discovery Methods**:

- `DiscoverPlugins`: Discover available plugins from configured sources
- `GetPluginManifest`: Retrieve specific plugin manifest

**Validation Methods**:

- `ValidatePlugin`: Validate plugin manifest and dependencies

**Lifecycle Methods**:

- `InstallPlugin`: Install plugin from source
- `UpdatePlugin`: Update existing plugin
- `RemovePlugin`: Remove installed plugin
- `ListInstalledPlugins`: List currently installed plugins

**Monitoring Methods**:

- `CheckPluginHealth`: Verify plugin health status

### Error Handling

Standard gRPC error codes with detailed error information:

**Common Error Codes**:

- `NOT_FOUND`: Plugin or version not found
- `INVALID_ARGUMENT`: Invalid request parameters
- `FAILED_PRECONDITION`: Requirements not met
- `PERMISSION_DENIED`: Insufficient permissions
- `UNAVAILABLE`: Service temporarily unavailable

**Error Response Format**:

```json
{
  "code": "PLUGIN_NOT_FOUND",
  "message": "Plugin 'aws-plugin' version '2.0.0' not found",
  "details": {
    "plugin_name": "aws-plugin",
    "requested_version": "2.0.0",
    "available_versions": ["1.0.0", "1.1.0", "1.2.0"]
  }
}
```

## Implementation Guidelines

### Registry Implementation

**Core Components**:

1. **Discovery Engine**: Manages multiple discovery sources
2. **Manifest Storage**: Caches plugin manifests
3. **Validation Engine**: Validates plugins and dependencies
4. **Installation Manager**: Handles plugin lifecycle
5. **Security Manager**: Handles signing and verification
6. **Health Monitor**: Monitors plugin health

**Storage Requirements**:

- Manifest cache with TTL
- Plugin binary storage
- Installation state tracking
- Dependency graph storage
- Health metrics storage

### Client Implementation

**Client Libraries** should provide:

1. **Discovery API**: Search and discover plugins
2. **Installation API**: Install, update, remove plugins
3. **Configuration API**: Manage plugin configuration
4. **Health API**: Monitor plugin health
5. **Security API**: Handle signatures and verification

**Example Client Usage**:

```go
// Create registry client
client := registry.NewClient("localhost:50051")

// Discover plugins
plugins, err := client.DiscoverPlugins(ctx, &registry.DiscoverPluginsRequest{
    Sources: []registry.DiscoverySource{registry.DISCOVERY_SOURCE_REGISTRY},
    Filter: "aws-*",
})

// Install plugin
result, err := client.InstallPlugin(ctx, &registry.InstallPluginRequest{
    Name: "aws-cost-plugin",
    Version: "1.2.3",
    Source: registry.DISCOVERY_SOURCE_REGISTRY,
    VerifySignature: true,
})
```

### Plugin Developer Guidelines

**Manifest Creation**:

1. Follow semantic versioning for releases
2. Declare all required dependencies
3. Specify minimum system requirements
4. Include comprehensive testing
5. Sign plugins for distribution

**Security Best Practices**:

1. Request minimal required permissions
2. Implement proper input validation
3. Use secure communication (TLS)
4. Follow principle of least privilege
5. Regularly update dependencies

**Testing Requirements**:

1. Unit tests for all functionality
2. Integration tests with FinFocus SDK
3. Security testing and vulnerability scanning
4. Performance benchmarking
5. Compatibility testing across versions

## Examples

### Complete Plugin Manifest

```json
{
  "metadata": {
    "name": "aws-comprehensive-plugin",
    "version": "2.1.0",
    "description": "Comprehensive AWS cost source plugin supporting EC2, S3, Lambda, RDS, and DynamoDB with real-time and historical cost data",
    "author": "FinFocus Team",
    "homepage": "https://finfocus.dev/plugins/aws",
    "repository": "https://github.com/finfocus/aws-plugin",
    "license": "Apache-2.0",
    "keywords": ["aws", "cost", "ec2", "s3", "lambda", "rds", "dynamodb"],
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-03-20T14:30:00Z"
  },
  "specification": {
    "spec_version": "0.1.0",
    "supported_providers": ["aws"],
    "supported_resources": {
      "aws": {
        "resource_types": ["ec2", "s3", "lambda", "rds", "dynamodb"],
        "billing_modes": [
          "per_hour",
          "per_gb_month",
          "per_invocation",
          "per_rcu",
          "per_wcu",
          "reserved",
          "spot"
        ],
        "regions": [
          "us-east-1",
          "us-west-2",
          "eu-west-1",
          "ap-southeast-1",
          "ap-northeast-1"
        ]
      }
    },
    "capabilities": [
      "cost_retrieval",
      "cost_projection",
      "pricing_specs",
      "historical_data",
      "real_time_data",
      "caching",
      "filtering"
    ],
    "service_definition": {
      "service_name": "CostSourceService",
      "package_name": "finfocus.v1",
      "methods": [
        "Name",
        "Supports",
        "GetActualCost",
        "GetProjectedCost",
        "GetPricingSpec"
      ],
      "port": 50051,
      "health_check_path": "/health"
    },
    "observability_support": {
      "metrics_enabled": true,
      "tracing_enabled": true,
      "logging_enabled": true,
      "health_checks_enabled": true,
      "sli_support": true
    }
  },
  "security": {
    "signature": "MEUCIQCx7HjRF...",
    "public_key": "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFA...",
    "security_level": "verified",
    "permissions": [
      "network_access",
      "filesystem_read",
      "config_read",
      "temp_files"
    ],
    "sandbox_required": false
  },
  "installation": {
    "installation_method": "binary",
    "download_url": "https://releases.finfocus.dev/aws-plugin/v2.1.0/aws-plugin-linux-amd64.tar.gz",
    "checksum": "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456",
    "checksum_algorithm": "sha256",
    "pre_install_checks": [
      "verify_aws_credentials",
      "check_network_connectivity",
      "validate_permissions"
    ],
    "post_install_steps": [
      "create_config_directory",
      "setup_log_rotation",
      "register_health_check"
    ]
  },
  "configuration": {
    "schema": "{\"type\":\"object\",\"required\":[\"aws_access_key_id\",\"aws_secret_access_key\",\"region\"],\"properties\":{\"aws_access_key_id\":{\"type\":\"string\"},\"aws_secret_access_key\":{\"type\":\"string\"},\"region\":{\"type\":\"string\",\"enum\":[\"us-east-1\",\"us-west-2\",\"eu-west-1\"]},\"timeout\":{\"type\":\"integer\",\"default\":30},\"cache_ttl\":{\"type\":\"integer\",\"default\":300}}}",
    "default_config": "{\"timeout\":30,\"cache_ttl\":300,\"enable_metrics\":true}",
    "required_fields": ["aws_access_key_id", "aws_secret_access_key", "region"],
    "examples": [
      {
        "name": "production_config",
        "description": "Production configuration with caching and metrics",
        "config": "{\"aws_access_key_id\":\"AKIA...\",\"aws_secret_access_key\":\"your-secret\",\"region\":\"us-east-1\",\"timeout\":60,\"cache_ttl\":600,\"enable_metrics\":true}"
      },
      {
        "name": "development_config",
        "description": "Development configuration with debug logging",
        "config": "{\"aws_access_key_id\":\"AKIA...\",\"aws_secret_access_key\":\"your-secret\",\"region\":\"us-west-2\",\"debug\":true,\"cache_ttl\":60}"
      }
    ]
  },
  "requirements": {
    "min_spec_version": "0.1.0",
    "max_spec_version": "0.2.0",
    "dependencies": [
      {
        "name": "auth-helper-plugin",
        "version_constraint": "^1.0.0",
        "optional": false
      }
    ],
    "system_requirements": {
      "min_memory_mb": 512,
      "min_disk_mb": 200,
      "supported_architectures": ["x86_64", "arm64"],
      "supported_os": ["linux", "darwin", "windows"]
    },
    "runtime_requirements": {
      "grpc_version": "1.50.0",
      "tls_required": true,
      "auth_methods": ["api_key", "jwt"],
      "timeout_seconds": 60
    }
  }
}
```

### Discovery Configuration

```json
{
  "registry_config": {
    "discovery_sources": [
      {
        "name": "official_registry",
        "type": "registry",
        "url": "https://registry.finfocus.dev",
        "priority": 1,
        "timeout_seconds": 30,
        "cache_ttl_seconds": 3600,
        "security_policy": {
          "min_security_level": "verified",
          "allow_untrusted": false,
          "require_signature": true
        }
      },
      {
        "name": "local_filesystem",
        "type": "filesystem",
        "path": "/usr/local/finfocus/plugins",
        "priority": 2,
        "recursive": true,
        "manifest_filename": "plugin.json",
        "watch_changes": true
      },
      {
        "name": "git_repository",
        "type": "git",
        "repository": "https://github.com/finfocus/community-plugins",
        "branch": "main",
        "path": "plugins/",
        "priority": 3,
        "sync_interval_seconds": 3600,
        "credentials": {
          "username": "token",
          "password": "${GITHUB_TOKEN}"
        }
      }
    ],
    "global_settings": {
      "concurrent_discoveries": 5,
      "default_timeout_seconds": 30,
      "cache_directory": "/var/cache/finfocus/plugins",
      "security_policy": {
        "default_security_level": "community",
        "prompt_for_untrusted": true,
        "verify_signatures": true
      }
    }
  }
}
```

### Installation Scripts

**Pre-install Check Script**:

```bash
#!/bin/bash
# pre_install_check.sh

set -e

echo "Running pre-installation checks for AWS Cost Plugin..."

# Check AWS credentials
if [ -z "$AWS_ACCESS_KEY_ID" ] && [ -z "$AWS_PROFILE" ]; then
    echo "ERROR: AWS credentials not configured"
    echo "Please set AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY or configure AWS_PROFILE"
    exit 1
fi

# Check network connectivity
if ! curl -s --max-time 10 https://aws.amazon.com > /dev/null; then
    echo "ERROR: Cannot reach AWS services"
    echo "Please check network connectivity"
    exit 1
fi

# Check system requirements
MEMORY_MB=$(free -m | awk 'NR==2{printf "%.0f", $2}')
if [ "$MEMORY_MB" -lt 512 ]; then
    echo "ERROR: Insufficient memory (${MEMORY_MB}MB < 512MB required)"
    exit 1
fi

# Check available disk space
DISK_MB=$(df /usr/local -BM | awk 'NR==2 {gsub(/M/,"", $4); print $4}')
if [ "$DISK_MB" -lt 200 ]; then
    echo "ERROR: Insufficient disk space (${DISK_MB}MB < 200MB required)"
    exit 1
fi

echo "Pre-installation checks completed successfully"
```

**Post-install Setup Script**:

```bash
#!/bin/bash
# post_install_setup.sh

set -e

PLUGIN_DIR="/usr/local/finfocus/plugins/aws-plugin"
CONFIG_DIR="/etc/finfocus/plugins/aws-plugin"
LOG_DIR="/var/log/finfocus/aws-plugin"

echo "Running post-installation setup..."

# Create directories
mkdir -p "$CONFIG_DIR"
mkdir -p "$LOG_DIR"
mkdir -p "/var/run/finfocus"

# Set permissions
chown -R finfocus:finfocus "$CONFIG_DIR"
chown -R finfocus:finfocus "$LOG_DIR"
chmod 750 "$CONFIG_DIR"
chmod 755 "$LOG_DIR"

# Create default configuration
cat > "$CONFIG_DIR/config.json" << EOF
{
  "timeout": 30,
  "cache_ttl": 300,
  "enable_metrics": true,
  "log_level": "info"
}
EOF

# Setup log rotation
cat > "/etc/logrotate.d/aws-plugin" << EOF
$LOG_DIR/*.log {
    daily
    missingok
    rotate 7
    compress
    notifempty
    create 0640 finfocus finfocus
    postrotate
        systemctl reload aws-plugin || true
    endscript
}
EOF

# Create systemd service
cat > "/etc/systemd/system/aws-plugin.service" << EOF
[Unit]
Description=FinFocus AWS Plugin
After=network.target

[Service]
Type=simple
User=finfocus
Group=finfocus
ExecStart=$PLUGIN_DIR/bin/aws-plugin --config $CONFIG_DIR/config.json
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
systemctl daemon-reload
systemctl enable aws-plugin
systemctl start aws-plugin

# Verify installation
sleep 5
if systemctl is-active --quiet aws-plugin; then
    echo "AWS Plugin installed and started successfully"
else
    echo "WARNING: AWS Plugin installation completed but service failed to start"
    systemctl status aws-plugin --no-pager
fi

echo "Post-installation setup completed"
```

## Appendices

### JSON Schema Reference

The complete JSON schema is defined in [`schemas/plugin_manifest.schema.json`](../schemas/plugin_manifest.schema.json).

Key validation rules:

- Plugin names must match `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
- Versions must follow semantic versioning pattern
- Required fields must be present and valid
- Enum values must match predefined lists
- URL fields must be valid URIs

### Error Codes

**Discovery Errors**:

- `DISCOVERY_SOURCE_UNREACHABLE`: Cannot connect to discovery source
- `MANIFEST_NOT_FOUND`: Plugin manifest not found
- `MANIFEST_INVALID`: Manifest fails schema validation

**Validation Errors**:

- `INVALID_VERSION`: Version format is invalid
- `DEPENDENCY_NOT_FOUND`: Required dependency not available
- `SYSTEM_REQUIREMENTS_NOT_MET`: System doesn't meet requirements
- `SIGNATURE_VERIFICATION_FAILED`: Digital signature verification failed

**Installation Errors**:

- `DOWNLOAD_FAILED`: Cannot download plugin files
- `CHECKSUM_MISMATCH`: File checksum doesn't match expected value
- `EXTRACTION_FAILED`: Cannot extract plugin files
- `PERMISSIONS_ERROR`: Insufficient permissions for installation

**Runtime Errors**:

- `PLUGIN_NOT_RESPONDING`: Plugin doesn't respond to health checks
- `CONFIGURATION_ERROR`: Plugin configuration is invalid
- `SERVICE_UNAVAILABLE`: Plugin service is not available

### Version History

**v0.1.0** (Current)

- Initial specification release
- Core plugin registry functionality
- Basic security model
- Filesystem and registry discovery
- gRPC service interface
- JSON schema validation

**Planned Future Versions**:

- v0.2.0: Enhanced security features, container deployment
- v0.3.0: Federation support, multi-registry synchronization
- v1.0.0: Production stability, comprehensive tooling

---

**Plugin Registry Specification v0.1.0** - Production-ready plugin management for the FinFocus ecosystem.
