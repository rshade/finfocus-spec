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

package registry_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-spec/sdk/go/registry"
)

func readSchemaDoc(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "schemas", name))
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc))
	return doc
}

func compileSchema(t *testing.T, name string) *jsonschema.Schema {
	t.Helper()
	c := jsonschema.NewCompiler()
	require.NoError(t, c.AddResource(name, readSchemaDoc(t, name)))
	s, err := c.Compile(name)
	require.NoError(t, err)
	return s
}

// schemaVerdict validates doc against a compiled schema.
func schemaVerdict(t *testing.T, s *jsonschema.Schema, doc []byte) error {
	t.Helper()
	var v any
	require.NoError(t, json.Unmarshal(doc, &v))
	return s.Validate(v)
}

// schemaNode walks a schema document by property names.
func schemaNode(t *testing.T, doc map[string]any, path ...string) map[string]any {
	t.Helper()
	node := doc
	for _, key := range path {
		next, ok := node[key].(map[string]any)
		require.True(t, ok, "schema has no object at %v (stopped at %q)", path, key)
		node = next
	}
	return node
}

func schemaEnum(t *testing.T, doc map[string]any, path ...string) []string {
	t.Helper()
	raw, ok := schemaNode(t, doc, path...)["enum"].([]any)
	require.True(t, ok, "schema has no enum at %v", path)
	out := make([]string, len(raw))
	for i, v := range raw {
		out[i], ok = v.(string)
		require.True(t, ok)
	}
	return out
}

func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func capabilityStrings() []string {
	caps := registry.AllPluginCapabilities()
	out := make([]string, len(caps))
	for i, c := range caps {
		out[i] = c.String()
	}
	return out
}

func TestSchemaAgreement(t *testing.T) {
	schema := compileSchema(t, "plugin_manifest.schema.json")
	cases := append(append([]manifestCase(nil), supportedResourcesCases()...), methodCapabilityCases()...)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := tc.manifest(t)
			sdkErr := registry.ValidatePluginManifest(doc)
			schemaErr := schemaVerdict(t, schema, doc)
			require.Equal(t, sdkErr == nil, schemaErr == nil,
				"SDK and schema disagree: sdk=%v schema=%v", sdkErr, schemaErr)
		})
	}
}

func TestSchemaDrift(t *testing.T) {
	manifest := readSchemaDoc(t, "plugin_manifest.schema.json")
	index := readSchemaDoc(t, "plugin_registry.schema.json")
	spec := []string{"properties", "specification", "properties"}
	resources := append(append([]string(nil), spec...),
		"supported_resources", "patternProperties", "^(aws|azure|gcp|kubernetes|custom)$", "properties")

	methods := append(append([]string(nil), spec...), "service_definition", "properties", "methods")
	require.Equal(t, sortedCopy(registry.AllServiceMethods()),
		sortedCopy(schemaEnum(t, manifest, append(methods, "items")...)),
		"manifest schema methods enum must list every CostSourceService RPC")
	require.NotContains(t, schemaNode(t, manifest, methods...), "maxItems")

	require.Equal(t, sortedCopy(capabilityStrings()),
		sortedCopy(schemaEnum(t, manifest, append(spec, "capabilities", "items")...)),
		"manifest schema capabilities enum must match registry.AllPluginCapabilities")
	require.Subset(t, schemaEnum(t, index, "$defs", "RegistryEntry", "properties", "capabilities", "items"),
		capabilityStrings(), "registry index schema must accept every manifest capability")

	require.Equal(t, registry.AllManifestBillingModes(),
		schemaEnum(t, manifest, append(resources, "billing_modes", "items")...))

	maxLength, ok := schemaNode(t, manifest, append(resources, "resource_types", "items")...)["maxLength"].(float64)
	require.True(t, ok)
	require.InDelta(t, float64(registry.MaxResourceTypeLength), maxLength, 0)
}

func TestExampleManifests(t *testing.T) {
	schema := compileSchema(t, "plugin_manifest.schema.json")
	for _, name := range []string{
		"aws-cost-plugin.json", "gcp-cost-plugin.json", "kubecost-plugin.json", "minimal-plugin.json",
	} {
		t.Run(name, func(t *testing.T) {
			doc, err := os.ReadFile(filepath.Join("..", "..", "..", "examples", "plugins", name))
			require.NoError(t, err)
			require.NoError(t, registry.ValidatePluginManifest(doc))
			require.NoError(t, schemaVerdict(t, schema, doc))
		})
	}
}
