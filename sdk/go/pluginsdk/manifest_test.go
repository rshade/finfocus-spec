package pluginsdk_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/rshade/finfocus-spec/sdk/go/registry"
)

//nolint:gochecknoglobals // Standard golden-file update flag.
var updateGolden = flag.Bool("update", false, "rewrite the manifest golden files in testdata")

// newTestManifest returns a manifest that sets every section with values the manifest schema accepts.
func newTestManifest() *pbc.PluginManifest {
	now := timestamppb.New(time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC))
	return &pbc.PluginManifest{
		Metadata: &pbc.PluginMetadata{
			Name:        "test-plugin",
			Version:     "1.0.0",
			Description: "A test plugin for FinFocus",
			Author:      "Test Author",
			Homepage:    "https://example.com/plugins?name=test&tab=docs",
			Repository:  "https://github.com/example/test-plugin",
			License:     "Apache-2.0",
			Keywords:    []string{"cost", "cloud", "alpha"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		Specification: &pbc.PluginSpecification{
			SpecVersion:        "0.1.0",
			SupportedProviders: []string{"azure", "aws"},
			SupportedResources: map[string]*pbc.ProviderResources{
				"azure": {
					ResourceTypes: []string{"azure-native:compute:VirtualMachine"},
					BillingModes:  []string{"per_hour"},
					Regions:       []string{"eastus"},
				},
				"aws": {
					ResourceTypes: []string{"aws:ec2/instance:Instance", "aws:s3/bucket:Bucket"},
					BillingModes:  []string{"per_hour", "per_gb_month"},
					Regions:       []string{"us-east-1", "us-west-2"},
				},
			},
			Capabilities: []string{"cost_projection", "cost_retrieval"},
			ServiceDefinition: &pbc.ServiceDefinition{
				ServiceName:     "CostSourceService",
				PackageName:     "finfocus.v1",
				Methods:         []string{"GetProjectedCost", "GetActualCost"},
				Port:            50051,
				HealthCheckPath: "/healthz",
			},
			ObservabilitySupport: &pbc.ObservabilitySupport{
				MetricsEnabled:      true,
				TracingEnabled:      true,
				LoggingEnabled:      true,
				HealthChecksEnabled: true,
			},
		},
		Security: &pbc.PluginSecurity{
			Signature:        "c2lnbmF0dXJl",
			PublicKey:        "some-public-key",
			CertificateChain: []string{"cert1", "cert2"},
			SecurityLevel:    pbc.SecurityLevel_SECURITY_LEVEL_VERIFIED,
			Permissions:      []string{"network_access"},
			SandboxRequired:  true,
		},
		Installation: &pbc.InstallationSpec{
			InstallationMethod: pbc.InstallationMethod_INSTALLATION_METHOD_BINARY,
			DownloadUrl:        "https://example.com/download",
			Checksum:           "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
			ChecksumAlgorithm:  "sha256",
			InstallScript:      "install.sh",
			PreInstallChecks:   []string{"check_docker"},
			PostInstallSteps:   []string{"restart_service"},
		},
		Configuration: &pbc.ConfigurationSpec{
			Schema:         "{}",
			DefaultConfig:  "{}",
			RequiredFields: []string{"api_key"},
			Examples: []*pbc.ConfigurationExample{
				{
					Name:        "example-config",
					Description: "A basic configuration",
					Config:      `{"api_key": "123"}`,
				},
			},
		},
	}
}

// compileManifestSchema compiles schemas/plugin_manifest.schema.json.
func compileManifestSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "schemas", "plugin_manifest.schema.json"))
	require.NoError(t, err)
	var doc any
	require.NoError(t, json.Unmarshal(data, &doc))
	c := jsonschema.NewCompiler()
	require.NoError(t, c.AddResource("plugin_manifest.schema.json", doc))
	s, err := c.Compile("plugin_manifest.schema.json")
	require.NoError(t, err)
	return s
}

// manifestJSONDoc returns the JSON form of a saved manifest file, converting YAML to JSON.
func manifestJSONDoc(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	if filepath.Ext(path) == ".json" {
		return data
	}
	var v any
	require.NoError(t, yaml.Unmarshal(data, &v))
	out, err := json.Marshal(v)
	require.NoError(t, err)
	return out
}

func saveTo(t *testing.T, ext string, m *pbc.PluginManifest) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest"+ext)
	require.NoError(t, pluginsdk.SaveManifest(path, m))
	return path
}

func TestManifestSaveLoad(t *testing.T) {
	want := newTestManifest()
	for _, ext := range []string{".json", ".yaml", ".yml"} {
		t.Run(ext, func(t *testing.T) {
			got, err := pluginsdk.LoadManifest(saveTo(t, ext, want))
			require.NoError(t, err)
			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Errorf("loaded manifest mismatch (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("unsupported extension", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "manifest.txt")
		err := pluginsdk.SaveManifest(path, want)
		require.ErrorContains(t, err, "unsupported manifest file extension")
		_, statErr := os.Stat(path)
		require.True(t, os.IsNotExist(statErr), "nothing should be written")
	})
}

func TestSaveManifestPassesValidation(t *testing.T) {
	schema := compileManifestSchema(t)
	for _, ext := range []string{".json", ".yaml", ".yml"} {
		t.Run(ext, func(t *testing.T) {
			doc := manifestJSONDoc(t, saveTo(t, ext, newTestManifest()))
			require.NoError(t, registry.ValidatePluginManifest(doc))

			var v map[string]any
			require.NoError(t, json.Unmarshal(doc, &v))
			require.NoError(t, schema.Validate(v))

			spec, ok := v["specification"].(map[string]any)
			require.True(t, ok)
			require.Contains(t, spec, "spec_version")
			require.Contains(t, spec, "supported_resources")
			install, ok := v["installation"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, "binary", install["installation_method"])
			security, ok := v["security"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, "verified", security["security_level"])
		})
	}
}

func TestSaveManifestDeterministic(t *testing.T) {
	for _, ext := range []string{".json", ".yaml"} {
		t.Run(ext, func(t *testing.T) {
			first, err := os.ReadFile(saveTo(t, ext, newTestManifest()))
			require.NoError(t, err)
			second, err := os.ReadFile(saveTo(t, ext, newTestManifest()))
			require.NoError(t, err)
			require.Equal(t, string(first), string(second))
			require.True(t, strings.HasSuffix(string(first), "\n"), "output must end with a newline")
			require.Contains(t, string(first), "name=test&tab=docs", "& must not be HTML-escaped")

			var v map[string]any
			require.NoError(t, json.Unmarshal(manifestJSONDoc(t, saveTo(t, ext, newTestManifest())), &v))
			metadata, ok := v["metadata"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, []any{"cost", "cloud", "alpha"}, metadata["keywords"], "list order must be kept")
		})
	}

	t.Run("json indent", func(t *testing.T) {
		data, err := pluginsdk.MarshalManifestJSON(newTestManifest())
		require.NoError(t, err)
		require.True(t, strings.HasPrefix(string(data), "{\n  \"configuration\": {\n    \""))
	})
}

func TestManifestGolden(t *testing.T) {
	cases := map[string]func(*pbc.PluginManifest) ([]byte, error){
		"manifest.golden.json": pluginsdk.MarshalManifestJSON,
		"manifest.golden.yaml": pluginsdk.MarshalManifestYAML,
	}
	for name, marshal := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := marshal(newTestManifest())
			require.NoError(t, err)
			path := filepath.Join("testdata", name)
			if *updateGolden {
				require.NoError(t, os.MkdirAll("testdata", 0o750))
				require.NoError(t, os.WriteFile(path, got, 0o600))
			}
			want, err := os.ReadFile(path)
			require.NoError(t, err, "run go test -run TestManifestGolden -update to create the golden file")
			require.Equal(t, string(want), string(got))
		})
	}
}

func TestMarshalManifestMatchesSave(t *testing.T) {
	cases := map[string]func(*pbc.PluginManifest) ([]byte, error){
		".json": pluginsdk.MarshalManifestJSON,
		".yaml": pluginsdk.MarshalManifestYAML,
		".yml":  pluginsdk.MarshalManifestYAML,
	}
	for ext, marshal := range cases {
		t.Run(ext, func(t *testing.T) {
			saved, err := os.ReadFile(saveTo(t, ext, newTestManifest()))
			require.NoError(t, err)
			marshaled, err := marshal(newTestManifest())
			require.NoError(t, err)
			require.Equal(t, string(saved), string(marshaled))
		})
	}
}

func writeManifestFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return path
}

func TestLoadManifestLegacyFormats(t *testing.T) {
	want := newTestManifest()

	legacyJSON, err := protojson.MarshalOptions{Indent: "  "}.Marshal(want)
	require.NoError(t, err)
	//nolint:musttag // Reproduces the previous writer, which marshaled the proto struct directly.
	legacyYAML, err := yaml.Marshal(want)
	require.NoError(t, err)
	require.Contains(t, string(legacyYAML), "specversion", "fixture must be in the old YAML shape")

	minimalWant := &pbc.PluginManifest{
		Metadata: &pbc.PluginMetadata{Name: "legacy-plugin"},
		Installation: &pbc.InstallationSpec{
			InstallationMethod: pbc.InstallationMethod_INSTALLATION_METHOD_CONTAINER,
		},
		Security: &pbc.PluginSecurity{SecurityLevel: pbc.SecurityLevel_SECURITY_LEVEL_OFFICIAL},
	}

	cases := []struct {
		name string
		file string
		data []byte
		want *pbc.PluginManifest
	}{
		{"protojson camelCase JSON", "m.json", legacyJSON, want},
		{"yaml of the proto struct", "m.yaml", legacyYAML, want},
		{
			"numeric enums in JSON", "m.json",
			[]byte(`{"metadata":{"name":"legacy-plugin"},"installation":{"installation_method":2},` +
				`"security":{"security_level":4}}`),
			minimalWant,
		},
		{
			"prefixed enum names in YAML",
			"m.yml",
			[]byte(
				"metadata:\n  name: legacy-plugin\ninstallation:\n  installationMethod: INSTALLATION_METHOD_CONTAINER\n" +
					"security:\n  securitylevel: SECURITY_LEVEL_OFFICIAL\n",
			),
			minimalWant,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, loadErr := pluginsdk.LoadManifest(writeManifestFile(t, tc.file, tc.data))
			require.NoError(t, loadErr)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("loaded manifest mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLoadManifestInvalidEnum(t *testing.T) {
	cases := map[string]string{
		"m.json": `{"installation":{"installation_method":"zip"}}`,
		"m.yaml": "installation:\n  installation_method: zip\n",
	}
	for file, content := range cases {
		t.Run(file, func(t *testing.T) {
			_, err := pluginsdk.LoadManifest(writeManifestFile(t, file, []byte(content)))
			require.ErrorContains(t, err, "installation_method")
			require.ErrorContains(t, err, "zip")
		})
	}
}

// Two spellings of one field must be rejected, as protojson rejects them, so a loaded manifest
// cannot differ from what registry.ValidatePluginManifest saw in the same file.
func TestLoadManifestConflictingKeys(t *testing.T) {
	cases := map[string]string{
		"m.json": `{"specification":{"spec_version":"0.1.0","specVersion":"9.9.9"}}`,
		"m.yaml": "specification:\n  spec_version: 0.1.0\n  specversion: 9.9.9\n",
		"m.yml":  "installation:\n  installation_method: binary\n  installationMethod: script\n",
	}
	for file, content := range cases {
		t.Run(file, func(t *testing.T) {
			_, err := pluginsdk.LoadManifest(writeManifestFile(t, file, []byte(content)))
			require.ErrorContains(t, err, "set the same field")
		})
	}
}

func BenchmarkMarshalManifestJSON(b *testing.B) {
	m := newTestManifest()
	b.ReportAllocs()
	for range b.N {
		if _, err := pluginsdk.MarshalManifestJSON(m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalManifestYAML(b *testing.B) {
	m := newTestManifest()
	b.ReportAllocs()
	for range b.N {
		if _, err := pluginsdk.MarshalManifestYAML(m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadManifest(b *testing.B) {
	for _, ext := range []string{".json", ".yaml"} {
		b.Run(ext, func(b *testing.B) {
			path := filepath.Join(b.TempDir(), "manifest"+ext)
			if err := pluginsdk.SaveManifest(path, newTestManifest()); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if _, err := pluginsdk.LoadManifest(path); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestManifestYAMLKeepsStrings(t *testing.T) {
	want := &pbc.PluginManifest{
		Metadata: &pbc.PluginMetadata{Name: "test-plugin", Version: "1.0", Keywords: []string{"true", "null", "0x10"}},
		Specification: &pbc.PluginSpecification{
			SupportedResources: map[string]*pbc.ProviderResources{
				"aws": {ResourceTypes: []string{"123"}, Regions: []string{"no", "on", "1e3"}},
			},
		},
	}
	got, err := pluginsdk.LoadManifest(saveTo(t, ".yaml", want))
	require.NoError(t, err)
	if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
		t.Errorf("YAML round trip changed values (-want +got):\n%s", diff)
	}
}

func TestLoadManifestErrors(t *testing.T) {
	testCases := []struct {
		name        string
		filePath    string
		fileContent string
		expectError string // Substring to expect in error message
	}{
		{
			name:        "non-existent file",
			filePath:    "non-existent.yaml",
			expectError: "reading manifest file",
		},
		{
			name:        "invalid YAML content",
			filePath:    "invalid.yaml",
			fileContent: "invalid: yaml: content: [",
			expectError: "parsing YAML manifest",
		},
		{
			name:        "invalid JSON content",
			filePath:    "invalid.json",
			fileContent: "{ \"metadata\": { \"name\": 123 } }", // Name should be string, not int
			expectError: "parsing JSON manifest",
		},
		{
			name:        "unsupported file extension",
			filePath:    "test.txt",
			fileContent: "some content",
			expectError: "unsupported manifest file extension",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			fullPath := filepath.Join(tmpDir, tc.filePath)

			if tc.fileContent != "" {
				err := os.WriteFile(fullPath, []byte(tc.fileContent), 0o600)
				if err != nil {
					t.Fatalf("Failed to write file for test %q: %v", tc.name, err)
				}
			}

			_, err := pluginsdk.LoadManifest(fullPath)
			if err == nil || !ErrorContains(err, tc.expectError) {
				t.Errorf("LoadManifest() expected error containing %q, got %v", tc.expectError, err)
			}
		})
	}
}

// ErrorContains checks if an error's message contains a specific substring.
// This is a helper function to avoid direct string comparison on error messages.
func ErrorContains(err error, s string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), s)
}

func TestValidationErrors(t *testing.T) {
	// Test empty errors
	var emptyErrs pluginsdk.ValidationErrors
	if emptyErrs.Error() != "no validation errors" {
		t.Errorf("Expected 'no validation errors', got %q", emptyErrs.Error())
	}

	// Test with single error
	singleErr := pluginsdk.ValidationErrors{
		&pbc.ValidationError{Message: "first error"},
	}
	errMsg := singleErr.Error()
	if !strings.Contains(errMsg, "validation failed with 1 error(s)") {
		t.Errorf("Expected '1 error(s)' in message, got %q", errMsg)
	}
	if !strings.Contains(errMsg, "first error") {
		t.Errorf("Expected 'first error' in message, got %q", errMsg)
	}

	// Test with multiple errors
	multiErr := pluginsdk.ValidationErrors{
		&pbc.ValidationError{Message: "error one"},
		&pbc.ValidationError{Message: "error two"},
	}
	errMsg = multiErr.Error()
	if !strings.Contains(errMsg, "validation failed with 2 error(s)") {
		t.Errorf("Expected '2 error(s)' in message, got %q", errMsg)
	}
	if !strings.Contains(errMsg, "error one") || !strings.Contains(errMsg, "error two") {
		t.Errorf("Expected both errors in message, got %q", errMsg)
	}
}
