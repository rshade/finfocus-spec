package pluginsdk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// ValidationErrors represents multiple validation errors.
type ValidationErrors []*pbc.ValidationError

func (errs ValidationErrors) Error() string {
	if len(errs) == 0 {
		return "no validation errors"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "validation failed with %d error(s):", len(errs))
	for _, err := range errs {
		fmt.Fprintf(&b, "\n  - %s", err.GetMessage()) // Assuming pbc.ValidationError has a GetMessage method
	}
	return b.String()
}

// LoadManifest reads a plugin manifest from path and decodes it as YAML (.yaml, .yml) or JSON (.json)
// based on the file extension.
//
// It reads the canonical form that SaveManifest writes and the forms earlier SDK versions wrote:
// protojson camelCase keys, full enum value names (INSTALLATION_METHOD_BINARY), integer enums, and
// YAML with lowercased Go field names. Unknown keys are ignored.
//
// LoadManifest does not validate the manifest; call registry.ValidatePluginManifest on the canonical
// JSON (MarshalManifestJSON) to do that. It returns an error if the file cannot be read, the extension
// is unsupported, decoding fails, or an enum value matches none of the accepted forms.
func LoadManifest(path string) (*pbc.PluginManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest file: %w", err)
	}

	switch ext := filepath.Ext(path); ext {
	case ".yaml", ".yml":
		manifest, decodeErr := decodeManifest(data, true)
		if decodeErr != nil {
			return nil, fmt.Errorf("parsing YAML manifest: %w", decodeErr)
		}
		return manifest, nil
	case ".json":
		manifest, decodeErr := decodeManifest(data, false)
		if decodeErr != nil {
			return nil, fmt.Errorf("parsing JSON manifest: %w", decodeErr)
		}
		return manifest, nil
	default:
		return nil, fmt.Errorf("unsupported manifest file extension: %s (supported: .yaml, .yml, .json)", ext)
	}
}

// SaveManifest writes a plugin manifest to path as JSON (.json) or YAML (.yaml, .yml), creating the
// parent directory if needed.
//
// The output is the canonical manifest form that schemas/plugin_manifest.schema.json and
// registry.ValidatePluginManifest expect: snake_case keys, schema enum strings (binary, verified),
// RFC 3339 timestamps, and keys sorted at every level. The same manifest always produces the same
// bytes. See MarshalManifestJSON and MarshalManifestYAML.
func SaveManifest(path string, m *pbc.PluginManifest) error {
	var data []byte
	var err error

	switch ext := filepath.Ext(path); ext {
	case ".yaml", ".yml":
		data, err = MarshalManifestYAML(m)
		if err != nil {
			return fmt.Errorf("marshaling to YAML: %w", err)
		}
	case ".json":
		data, err = MarshalManifestJSON(m)
		if err != nil {
			return fmt.Errorf("marshaling to JSON: %w", err)
		}
	default:
		return fmt.Errorf("unsupported manifest file extension: %s (supported: .yaml, .yml, .json)", ext)
	}

	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if mkdirErr := os.MkdirAll(dir, 0o750); mkdirErr != nil {
			return fmt.Errorf("creating manifest directory: %w", mkdirErr)
		}
	}
	if writeErr := os.WriteFile(path, data, 0o600); writeErr != nil {
		return fmt.Errorf("writing manifest file: %w", writeErr)
	}

	return nil
}

// MarshalManifestJSON returns the canonical JSON form of a plugin manifest, byte for byte what
// SaveManifest writes to a .json path. Pass the result to registry.ValidatePluginManifest to
// validate a manifest before saving it.
func MarshalManifestJSON(m *pbc.PluginManifest) ([]byte, error) {
	return encodeManifestJSON(m)
}

// MarshalManifestYAML returns the canonical YAML form of a plugin manifest, byte for byte what
// SaveManifest writes to a .yaml or .yml path. It carries the same keys, values, and order as
// MarshalManifestJSON; strings that YAML would read as another type are quoted.
func MarshalManifestYAML(m *pbc.PluginManifest) ([]byte, error) {
	return encodeManifestYAML(m)
}
