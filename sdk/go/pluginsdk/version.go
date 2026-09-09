// Package pluginsdk provides version information for the FinFocus specification.
package pluginsdk

import (
	"errors"
	"fmt"

	"github.com/rshade/finfocus-spec/sdk/go/internal/semver"
)

// init validates the SpecVersion constant at package initialization.
// This catches invalid versions immediately at startup rather than at runtime
// when GetPluginInfo is called, providing fail-fast behavior for developers.
//
//nolint:gochecknoinits // Intentional validation at package initialization
func init() {
	if err := ValidateSpecVersion(SpecVersion); err != nil {
		panic(fmt.Sprintf("pluginsdk: invalid SpecVersion constant %q: %v", SpecVersion, err))
	}
}

// SpecVersion is the version of the finfocus-spec protocol that this SDK implements.
// This constant is used by the default GetPluginInfo handler to report the spec version
// that plugins were compiled against.
//
// The version follows Semantic Versioning (https://semver.org/).
// Format: vMAJOR.MINOR.PATCH (e.g., "v0.4.11")
//
// IMPORTANT: This value should be updated when the spec version changes.
// It is typically synchronized with the repository's release tags.
const SpecVersion = "v0.6.1" // x-release-please-version

// ValidateSpecVersion validates that a version string is a valid semantic version.
// The version must be in the format vMAJOR.MINOR.PATCH where MAJOR, MINOR, and PATCH
// are non-negative integers without leading zeros (except for 0 itself).
//
// Valid examples: "v0.4.11", "v1.0.0", "v2.15.3"
// Invalid examples: "0.4.11" (no v prefix), "v1.2" (missing patch), "v01.2.3" (leading zero)
//
// Returns nil if the version is valid, or an error describing the validation failure.
func ValidateSpecVersion(version string) error {
	if version == "" {
		return errors.New("spec_version is required")
	}

	if !semver.IsValid(version) {
		return fmt.Errorf(
			"spec_version %q is not a valid semantic version (expected format: vMAJOR.MINOR.PATCH)",
			version,
		)
	}

	return nil
}

// IsValidSpecVersion returns true if the version string is a valid semantic version.
// This is a convenience wrapper around ValidateSpecVersion for boolean checks.
func IsValidSpecVersion(version string) bool {
	return ValidateSpecVersion(version) == nil
}
