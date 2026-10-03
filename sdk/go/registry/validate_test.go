package registry_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-spec/sdk/go/registry"
)

func TestValidatePluginManifest_ValidManifests(t *testing.T) {
	validManifests := []string{
		// Minimal valid manifest
		`{
			"metadata": {
				"name": "test-plugin",
				"version": "1.0.0",
				"description": "Test plugin for validation",
				"author": "Test Author"
			},
			"specification": {
				"spec_version": "0.1.0",
				"supported_providers": ["custom"],
				"service_definition": {
					"service_name": "CostSourceService",
					"package_name": "finfocus.v1",
					"methods": ["Name"]
				}
			},
			"installation": {
				"installation_method": "binary"
			}
		}`,
		// Full AWS plugin example
		`{
			"metadata": {
				"name": "aws-cost-plugin",
				"version": "2.1.0",
				"description": "Comprehensive AWS cost source plugin supporting EC2, S3, Lambda, RDS, and DynamoDB with real-time and historical cost data retrieval",
				"author": "FinFocus Team"
			},
			"specification": {
				"spec_version": "0.1.0",
				"supported_providers": ["aws"],
				"service_definition": {
					"service_name": "CostSourceService",
					"package_name": "finfocus.v1",
					"methods": ["Name", "Supports", "GetActualCost", "GetProjectedCost", "GetPricingSpec"]
				}
			},
			"installation": {
				"installation_method": "container"
			}
		}`,
	}

	for i, manifestJSON := range validManifests {
		err := registry.ValidatePluginManifest([]byte(manifestJSON))
		if err != nil {
			t.Errorf("Valid manifest %d failed validation: %v", i+1, err)
		}
	}
}

func TestValidatePluginManifest_InvalidManifests(t *testing.T) {
	invalidManifests := []struct {
		name                  string
		manifest              string
		expectedErrorContains string
	}{
		{
			name:     "empty object",
			manifest: `{}`,
		},
		{
			name: "missing metadata",
			manifest: `{
				"specification": {
					"spec_version": "0.1.0",
					"supported_providers": ["custom"],
					"service_definition": {
						"service_name": "CostSourceService",
						"package_name": "finfocus.v1",
						"methods": ["Name"]
					}
				},
				"installation": {
					"installation_method": "binary"
				}
			}`,
		},
		{
			name: "invalid plugin name",
			manifest: `{
				"metadata": {
					"name": "Invalid_Plugin",
					"version": "1.0.0",
					"description": "Test plugin for validation",
					"author": "Test Author"
				},
				"specification": {
					"spec_version": "0.1.0",
					"supported_providers": ["custom"],
					"service_definition": {
						"service_name": "CostSourceService",
						"package_name": "finfocus.v1",
						"methods": ["Name"]
					}
				},
				"installation": {
					"installation_method": "binary"
				}
			}`,
		},
		{
			name: "invalid version",
			manifest: `{
				"metadata": {
					"name": "test-plugin",
					"version": "invalid-version",
					"description": "Test plugin for validation",
					"author": "Test Author"
				},
				"specification": {
					"spec_version": "0.1.0",
					"supported_providers": ["custom"],
					"service_definition": {
						"service_name": "CostSourceService",
						"package_name": "finfocus.v1",
						"methods": ["Name"]
					}
				},
				"installation": {
					"installation_method": "binary"
				}
			}`,
		},
		{
			name: "description too short",
			manifest: `{
				"metadata": {
					"name": "test-plugin",
					"version": "1.0.0",
					"description": "Short",
					"author": "Test Author"
				},
				"specification": {
					"spec_version": "0.1.0",
					"supported_providers": ["custom"],
					"service_definition": {
						"service_name": "CostSourceService",
						"package_name": "finfocus.v1",
						"methods": ["Name"]
					}
				},
				"installation": {
					"installation_method": "binary"
				}
			}`,
		},
		{
			name: "invalid provider",
			manifest: `{
				"metadata": {
					"name": "test-plugin",
					"version": "1.0.0",
					"description": "Test plugin for validation",
					"author": "Test Author"
				},
				"specification": {
					"spec_version": "0.1.0",
					"supported_providers": ["invalid-provider"],
					"service_definition": {
						"service_name": "CostSourceService",
						"package_name": "finfocus.v1",
						"methods": ["Name"]
					}
				},
				"installation": {
					"installation_method": "binary"
				}
			}`,
			expectedErrorContains: "must be one of: aws, azure, gcp, kubernetes, custom",
		},
		{
			name: "invalid service name",
			manifest: `{
				"metadata": {
					"name": "test-plugin",
					"version": "1.0.0",
					"description": "Test plugin for validation",
					"author": "Test Author"
				},
				"specification": {
					"spec_version": "0.1.0",
					"supported_providers": ["custom"],
					"service_definition": {
						"service_name": "invalidServiceName",
						"package_name": "finfocus.v1",
						"methods": ["Name"]
					}
				},
				"installation": {
					"installation_method": "binary"
				}
			}`,
		},
		{
			name: "invalid package name",
			manifest: `{
				"metadata": {
					"name": "test-plugin",
					"version": "1.0.0",
					"description": "Test plugin for validation",
					"author": "Test Author"
				},
				"specification": {
					"spec_version": "0.1.0",
					"supported_providers": ["custom"],
					"service_definition": {
						"service_name": "CostSourceService",
						"package_name": "Invalid.Package.Name",
						"methods": ["Name"]
					}
				},
				"installation": {
					"installation_method": "binary"
				}
			}`,
		},
		{
			name: "invalid method",
			manifest: `{
				"metadata": {
					"name": "test-plugin",
					"version": "1.0.0",
					"description": "Test plugin for validation",
					"author": "Test Author"
				},
				"specification": {
					"spec_version": "0.1.0",
					"supported_providers": ["custom"],
					"service_definition": {
						"service_name": "CostSourceService",
						"package_name": "finfocus.v1",
						"methods": ["InvalidMethod"]
					}
				},
				"installation": {
					"installation_method": "binary"
				}
			}`,
		},
		{
			name: "invalid installation method",
			manifest: `{
				"metadata": {
					"name": "test-plugin",
					"version": "1.0.0",
					"description": "Test plugin for validation",
					"author": "Test Author"
				},
				"specification": {
					"spec_version": "0.1.0",
					"supported_providers": ["custom"],
					"service_definition": {
						"service_name": "CostSourceService",
						"package_name": "finfocus.v1",
						"methods": ["Name"]
					}
				},
				"installation": {
					"installation_method": "invalid-method"
				}
			}`,
		},
	}

	for _, test := range invalidManifests {
		err := registry.ValidatePluginManifest([]byte(test.manifest))
		if err == nil {
			t.Errorf("Invalid manifest '%s' should have failed validation", test.name)
		} else if test.expectedErrorContains != "" {
			if !strings.Contains(err.Error(), test.expectedErrorContains) {
				t.Errorf("Invalid manifest '%s' error message should contain '%s', got: %v",
					test.name, test.expectedErrorContains, err)
			}
		}
	}
}

func TestValidatePluginManifest_InvalidJSON(t *testing.T) {
	invalidJSON := `{ "invalid": json }`
	err := registry.ValidatePluginManifest([]byte(invalidJSON))
	if err == nil {
		t.Errorf("Invalid JSON should have failed validation")
	}
}

func TestValidatePluginManifest_Examples(t *testing.T) {
	// Test that our example plugin manifests are valid
	exampleManifests := []string{
		// This should match the format from our examples directory
		`{
			"metadata": {
				"name": "aws-cost-plugin",
				"version": "2.1.0",
				"description": "Comprehensive AWS cost source plugin supporting EC2, S3, Lambda, RDS, and DynamoDB with real-time and historical cost data retrieval",
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
							"per_hour", "per_gb_month", "per_invocation",
							"per_rcu", "per_wcu", "reserved", "spot"
						],
						"regions": [
							"us-east-1", "us-west-2", "eu-west-1",
							"ap-southeast-1", "ap-northeast-1"
						]
					}
				},
				"capabilities": [
					"cost_retrieval", "cost_projection", "pricing_specs",
					"historical_data", "real_time_data", "caching", "filtering"
				],
				"service_definition": {
					"service_name": "CostSourceService",
					"package_name": "finfocus.v1",
					"methods": ["Name", "Supports", "GetActualCost", "GetProjectedCost", "GetPricingSpec"],
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
				"signature": "MEUCIQCx7HjRFkL3+Y8XrGQm4nW2V9iE2fP8jS6bK1qN7dR9yAIgD8sJ5tK2oP1mN3vB4cE5fG6hI7jK8lM9nO0pQ1rS2tU=",
				"public_key": "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1234567890abcdef...\n-----END PUBLIC KEY-----",
				"security_level": "verified",
				"permissions": [
					"network_access", "filesystem_read", "config_read", "temp_files"
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
			}
		}`,
	}

	for i, manifestJSON := range exampleManifests {
		// First check if it's valid JSON
		var manifest map[string]interface{}
		if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
			t.Errorf("Example manifest %d is not valid JSON: %v", i+1, err)
			continue
		}

		// Then validate against our validation function
		// Note: This validates against our simplified schema embedded in validate.go
		// The full schema validation would be done by the JSON schema validator
		err := registry.ValidatePluginManifest([]byte(manifestJSON))
		if err != nil {
			t.Errorf("Example manifest %d failed validation: %v", i+1, err)
		}
	}
}

// manifestCase is a manifest with one specification field overridden. An empty wantErr means the
// manifest is valid.
type manifestCase struct {
	name    string
	field   string
	value   any
	wantErr string
}

func (c manifestCase) manifest(t *testing.T) []byte {
	t.Helper()
	doc := map[string]any{
		"metadata": map[string]any{
			"name": "test-plugin", "version": "1.0.0",
			"description": "Test plugin for validation", "author": "Test Author",
		},
		"specification": map[string]any{
			"spec_version":        "0.1.0",
			"supported_providers": []any{"azure"},
			"service_definition": map[string]any{
				"service_name": "CostSourceService", "package_name": "finfocus.v1", "methods": []any{"Name"},
			},
			c.field: c.value,
		},
		"installation": map[string]any{"installation_method": "binary"},
	}
	data, err := json.Marshal(doc)
	require.NoError(t, err)
	return data
}

func resources(types ...any) map[string]any {
	return map[string]any{"azure": map[string]any{"resource_types": append([]any{}, types...)}}
}

func supportedResourcesCases() []manifestCase {
	const field = "supported_resources"
	const prefix = "specification.supported_resources.azure"
	token60 := "azure-native:compute:" + strings.Repeat("V", 60-len("azure-native:compute:"))
	return []manifestCase{
		{name: "60-character azure-native token", field: field, value: resources(token60)},
		{
			name: "53-character azure token", field: field,
			value: resources("azure:compute/linuxVirtualMachine:LinuxVirtualMachine"),
		},
		{name: "256-character token", field: field, value: resources(strings.Repeat("a", 256))},
		{
			name: "short names with billing modes and regions", field: field,
			value: map[string]any{
				"aws":   map[string]any{"resource_types": []any{"ec2"}, "billing_modes": []any{"per_hour", "spot"}},
				"azure": map[string]any{"resource_types": []any{"vm"}, "regions": []any{"eastus", "westeurope"}},
			},
		},
		{
			name: "not an object", field: field, value: []any{"azure"},
			wantErr: "specification.supported_resources must be an object",
		},
		{
			name: "package name as key", field: field,
			value:   map[string]any{"azure-native": map[string]any{"resource_types": []any{"vm"}}},
			wantErr: "specification.supported_resources: 'azure-native' is not a valid provider",
		},
		{
			name: "entry not an object", field: field, value: map[string]any{"azure": "vm"},
			wantErr: prefix + " must be an object",
		},
		{
			name: "unknown field", field: field,
			value:   map[string]any{"azure": map[string]any{"resource_types": []any{"vm"}, "skus": []any{"b1s"}}},
			wantErr: prefix + ": unknown field 'skus'",
		},
		{
			name: "missing resource_types", field: field,
			value:   map[string]any{"azure": map[string]any{"regions": []any{"eastus"}}},
			wantErr: prefix + ".resource_types is required",
		},
		{
			name: "empty resource_types", field: field, value: resources(),
			wantErr: prefix + ".resource_types must contain at least one resource type",
		},
		{
			name: "duplicate resource type", field: field, value: resources("vm", "vm"),
			wantErr: prefix + ".resource_types contains duplicate resource type 'vm'",
		},
		{
			name: "empty resource type", field: field, value: resources(""),
			wantErr: prefix + ".resource_types[0] must be 1 to 256 characters",
		},
		{
			name: "257-character token", field: field, value: resources(strings.Repeat("a", 257)),
			wantErr: prefix + ".resource_types[0] must be 1 to 256 characters",
		},
		{
			name: "resource type not a string", field: field, value: resources(42),
			wantErr: prefix + ".resource_types[0] must be a string",
		},
		{
			name: "unknown billing mode", field: field,
			value: map[string]any{"azure": map[string]any{
				"resource_types": []any{"vm"}, "billing_modes": []any{"hourly"},
			}},
			wantErr: prefix + ".billing_modes[0]: 'hourly' is not a valid billing mode",
		},
		{
			name: "duplicate billing mode", field: field,
			value: map[string]any{"azure": map[string]any{
				"resource_types": []any{"vm"}, "billing_modes": []any{"per_hour", "per_hour"},
			}},
			wantErr: prefix + ".billing_modes contains duplicate billing mode 'per_hour'",
		},
		{
			name: "region too short", field: field,
			value:   map[string]any{"azure": map[string]any{"resource_types": []any{"vm"}, "regions": []any{"x"}}},
			wantErr: prefix + ".regions[0] must be 2 to 30 characters",
		},
		{
			name: "region too long", field: field,
			value: map[string]any{"azure": map[string]any{
				"resource_types": []any{"vm"}, "regions": []any{strings.Repeat("r", 31)},
			}},
			wantErr: prefix + ".regions[0] must be 2 to 30 characters",
		},
		{
			name: "duplicate region", field: field,
			value: map[string]any{"azure": map[string]any{
				"resource_types": []any{"vm"}, "regions": []any{"eastus", "eastus"},
			}},
			wantErr: prefix + ".regions contains duplicate region 'eastus'",
		},
	}
}

func methodCapabilityCases() []manifestCase {
	allMethods := []any{
		"Name", "Supports", "GetActualCost", "GetProjectedCost", "GetPricingSpec", "EstimateCost",
		"GetRecommendations", "DismissRecommendation", "GetBudgets", "GetPluginInfo", "DryRun",
		"BatchCost", "ResolveResourceTypes",
	}
	protoCapabilities := []any{
		"projected_costs", "actual_costs", "carbon", "recommendations", "dry_run", "budgets", "energy",
		"water", "pricing_spec", "estimate_cost", "dismiss_recommendations", "batch_cost",
		"resolve_resource_types", "usage_stats", "allocation", "contract_commitments", "invoice_data",
		"recommendation_scoring",
	}
	serviceDefinition := func(methods ...any) map[string]any {
		return map[string]any{
			"service_name": "CostSourceService", "package_name": "finfocus.v1", "methods": methods,
		}
	}
	return []manifestCase{
		{name: "every CostSourceService RPC", field: "service_definition", value: serviceDefinition(allMethods...)},
		{
			name: "RPC of another service", field: "service_definition", value: serviceDefinition("Allocate"),
			wantErr: "specification.service_definition.methods[0]: 'Allocate' is not a valid method",
		},
		{name: "every protocol capability", field: "capabilities", value: protoCapabilities},
		{name: "existing capabilities", field: "capabilities", value: []any{"cost_retrieval", "caching"}},
		{
			name: "unknown capability", field: "capabilities", value: []any{"teleport"},
			wantErr: "specification.capabilities[0]: 'teleport' is not a valid capability",
		},
		{
			name: "proto enum name as capability", field: "capabilities", value: []any{"PLUGIN_CAPABILITY_DRY_RUN"},
			wantErr: "specification.capabilities[0]: 'PLUGIN_CAPABILITY_DRY_RUN' is not a valid capability",
		},
		{
			name: "capabilities not an array", field: "capabilities", value: "dry_run",
			wantErr: "specification.capabilities must be an array",
		},
		{
			name: "duplicate capability", field: "capabilities", value: []any{"dry_run", "dry_run"},
			wantErr: "specification.capabilities contains duplicate capability 'dry_run'",
		},
	}
}

func TestValidateSupportedResources(t *testing.T) {
	runManifestCases(t, supportedResourcesCases())
}

func TestValidateMethodsAndCapabilities(t *testing.T) {
	runManifestCases(t, methodCapabilityCases())
}

// runManifestCases checks each case against the SDK validator; an error must start with wantErr,
// which pins the field path.
func runManifestCases(t *testing.T, cases []manifestCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := registry.ValidatePluginManifest(tc.manifest(t))
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.True(
				t,
				strings.HasPrefix(err.Error(), tc.wantErr),
				"error %q should start with %q",
				err,
				tc.wantErr,
			)
		})
	}
}
