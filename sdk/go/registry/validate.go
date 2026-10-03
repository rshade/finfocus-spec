package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	// MinDescriptionLength defines the minimum required length for plugin descriptions.
	MinDescriptionLength = 10
	// MaxDescriptionLength defines the maximum allowed length for plugin descriptions.
	MaxDescriptionLength = 500
	// MinAuthorLength defines the minimum required length for plugin author names.
	MinAuthorLength = 2
	// MaxAuthorLength defines the maximum allowed length for plugin author names.
	MaxAuthorLength = 100
	// MaxResourceTypeLength defines the maximum length of a supported_resources resource_types entry,
	// long enough for full Pulumi and Terraform type tokens such as
	// "azure-native:compute:VirtualMachine".
	MaxResourceTypeLength = 256
)

// ValidatePluginManifest validates a plugin manifest JSON document against the schema.
func ValidatePluginManifest(manifestJSON []byte) error {
	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate basic structure
	if err := validateBasicStructure(manifest); err != nil {
		return err
	}

	// Validate metadata section
	if err := validateMetadata(manifest); err != nil {
		return err
	}

	// Validate specification section
	if err := validateSpecification(manifest); err != nil {
		return err
	}

	// Validate installation section
	if err := validateInstallation(manifest); err != nil {
		return err
	}

	return nil
}

// validateBasicStructure validates the basic structure of the manifest.
func validateBasicStructure(manifest map[string]interface{}) error {
	requiredFields := []string{"metadata", "specification", "installation"}

	for _, field := range requiredFields {
		if _, exists := manifest[field]; !exists {
			return fmt.Errorf("required field '%s' is missing", field)
		}
	}

	return nil
}

// validateMetadata validates the metadata section.
func validateMetadata(manifest map[string]interface{}) error {
	metadata, ok := manifest["metadata"].(map[string]interface{})
	if !ok {
		return errors.New("metadata must be an object")
	}

	// Validate required fields
	requiredFields := []string{"name", "version", "description", "author"}
	for _, field := range requiredFields {
		if _, exists := metadata[field]; !exists {
			return fmt.Errorf("metadata.%s is required", field)
		}
	}

	// Validate name
	name, ok := metadata["name"].(string)
	if !ok {
		return errors.New("metadata.name must be a string")
	}
	if err := ValidatePluginName(name); err != nil {
		return fmt.Errorf("metadata.name: %w", err)
	}

	// Validate version (semantic version)
	version, ok := metadata["version"].(string)
	if !ok {
		return errors.New("metadata.version must be a string")
	}
	if err := validateSemanticVersion(version); err != nil {
		return fmt.Errorf("metadata.version: %w", err)
	}

	// Validate description
	description, ok := metadata["description"].(string)
	if !ok {
		return errors.New("metadata.description must be a string")
	}
	if len(description) < MinDescriptionLength {
		return fmt.Errorf("metadata.description must be at least %d characters long", MinDescriptionLength)
	}
	if len(description) > MaxDescriptionLength {
		return fmt.Errorf("metadata.description must be no more than %d characters long", MaxDescriptionLength)
	}

	// Validate author
	author, ok := metadata["author"].(string)
	if !ok {
		return errors.New("metadata.author must be a string")
	}
	if len(author) < MinAuthorLength {
		return fmt.Errorf("metadata.author must be at least %d characters long", MinAuthorLength)
	}
	if len(author) > MaxAuthorLength {
		return fmt.Errorf("metadata.author must be no more than %d characters long", MaxAuthorLength)
	}

	return nil
}

// validateSpecification validates the specification section.
func validateSpecification(manifest map[string]interface{}) error {
	specification, ok := manifest["specification"].(map[string]interface{})
	if !ok {
		return errors.New("specification must be an object")
	}

	// Validate required fields
	requiredFields := []string{"spec_version", "supported_providers", "service_definition"}
	for _, field := range requiredFields {
		if _, exists := specification[field]; !exists {
			return fmt.Errorf("specification.%s is required", field)
		}
	}

	// Validate spec_version
	specVersion, ok := specification["spec_version"].(string)
	if !ok {
		return errors.New("specification.spec_version must be a string")
	}
	if err := validateSemanticVersion(specVersion); err != nil {
		return fmt.Errorf("specification.spec_version: %w", err)
	}

	// Validate supported_providers
	if err := validateSupportedProviders(specification); err != nil {
		return err
	}

	if err := validateSupportedResources(specification); err != nil {
		return err
	}

	if capabilities, exists := specification["capabilities"]; exists {
		if err := validateStringList("specification.capabilities", capabilities, capabilityRule); err != nil {
			return err
		}
	}

	// Validate service_definition
	if err := validateServiceDefinition(specification); err != nil {
		return err
	}

	return nil
}

// validateSupportedProviders validates the supported_providers array.
func validateSupportedProviders(specification map[string]interface{}) error {
	providersInterface, exists := specification["supported_providers"]
	if !exists {
		return errors.New("specification.supported_providers is required")
	}

	return validateStringList("specification.supported_providers", providersInterface, providerRule)
}

// listRule describes a JSON array of unique strings: the noun used in messages, whether the array
// must be non-empty, and a check that returns "" for a valid item or the message suffix for an
// invalid one.
type listRule struct {
	noun     string
	nonEmpty bool
	check    func(item string) string
}

//nolint:gochecknoglobals // Shared, immutable validation rules.
var (
	providerRule = listRule{noun: "provider", nonEmpty: true, check: func(item string) string {
		if IsValidProvider(item) {
			return ""
		}
		return fmt.Sprintf(": '%s' is not a valid provider, must be one of: %s",
			item, strings.Join(getAllProviderStrings(), ", "))
	}}
	methodRule     = listRule{noun: "method", nonEmpty: true, check: enumCheck("method", IsValidServiceMethod)}
	capabilityRule = listRule{noun: "capability", check: enumCheck("capability", IsValidPluginCapability)}
	resourceRule   = listRule{noun: "resource type", nonEmpty: true, check: lengthCheck(1, MaxResourceTypeLength)}
	billingRule    = listRule{noun: "billing mode", check: enumCheck("billing mode", IsValidManifestBillingMode)}
	regionRule     = listRule{noun: "region", check: lengthCheck(minRegionLength, maxRegionLength)}
)

const (
	minRegionLength = 2
	maxRegionLength = 30
)

func enumCheck(noun string, valid func(string) bool) func(string) string {
	return func(item string) string {
		if valid(item) {
			return ""
		}
		return fmt.Sprintf(": '%s' is not a valid %s", item, noun)
	}
}

// lengthCheck counts characters (not bytes), as JSON Schema minLength and maxLength do.
func lengthCheck(minLen, maxLen int) func(string) string {
	return func(item string) string {
		if n := utf8.RuneCountInString(item); n >= minLen && n <= maxLen {
			return ""
		}
		return fmt.Sprintf(" must be %d to %d characters", minLen, maxLen)
	}
}

// validateStringList validates a JSON array of unique strings at path.
func validateStringList(path string, raw interface{}, rule listRule) error {
	items, ok := raw.([]interface{})
	if !ok {
		return fmt.Errorf("%s must be an array", path)
	}
	if rule.nonEmpty && len(items) == 0 {
		return fmt.Errorf("%s must contain at least one %s", path, rule.noun)
	}

	seen := make(map[string]bool, len(items))
	for i, rawItem := range items {
		item, isString := rawItem.(string)
		if !isString {
			return fmt.Errorf("%s[%d] must be a string", path, i)
		}
		if msg := rule.check(item); msg != "" {
			return fmt.Errorf("%s[%d]%s", path, i, msg)
		}
		if seen[item] {
			return fmt.Errorf("%s contains duplicate %s '%s'", path, rule.noun, item)
		}
		seen[item] = true
	}
	return nil
}

// validateSupportedResources validates the optional supported_resources object. Providers are
// checked in sorted order so the first reported error does not depend on map iteration.
func validateSupportedResources(specification map[string]interface{}) error {
	raw, exists := specification["supported_resources"]
	if !exists {
		return nil
	}
	resources, ok := raw.(map[string]interface{})
	if !ok {
		return errors.New("specification.supported_resources must be an object")
	}

	providers := make([]string, 0, len(resources))
	for provider := range resources {
		providers = append(providers, provider)
	}
	sort.Strings(providers)

	for _, provider := range providers {
		if !IsValidProvider(provider) {
			return fmt.Errorf("specification.supported_resources: '%s' is not a valid provider, must be one of: %s",
				provider, strings.Join(getAllProviderStrings(), ", "))
		}
		if err := validateProviderResources("specification.supported_resources."+provider,
			resources[provider]); err != nil {
			return err
		}
	}
	return nil
}

// validateProviderResources validates one supported_resources entry.
func validateProviderResources(path string, raw interface{}) error {
	entry, ok := raw.(map[string]interface{})
	if !ok {
		return fmt.Errorf("%s must be an object", path)
	}

	rules := map[string]listRule{"resource_types": resourceRule, "billing_modes": billingRule, "regions": regionRule}
	fields := make([]string, 0, len(entry))
	for field := range entry {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	for _, field := range fields {
		if _, known := rules[field]; !known {
			return fmt.Errorf("%s: unknown field '%s'", path, field)
		}
	}

	if _, exists := entry["resource_types"]; !exists {
		return fmt.Errorf("%s.resource_types is required", path)
	}
	for _, field := range fields {
		if err := validateStringList(path+"."+field, entry[field], rules[field]); err != nil {
			return err
		}
	}
	return nil
}

// validateServiceDefinition validates the service_definition section.
func validateServiceDefinition(specification map[string]interface{}) error {
	serviceDefInterface, exists := specification["service_definition"]
	if !exists {
		return errors.New("specification.service_definition is required")
	}

	serviceDef, ok := serviceDefInterface.(map[string]interface{})
	if !ok {
		return errors.New("specification.service_definition must be an object")
	}

	// Validate required fields
	requiredFields := []string{"service_name", "package_name", "methods"}
	for _, field := range requiredFields {
		if _, fieldExists := serviceDef[field]; !fieldExists {
			return fmt.Errorf("specification.service_definition.%s is required", field)
		}
	}

	// Validate service_name
	serviceName, ok := serviceDef["service_name"].(string)
	if !ok {
		return errors.New("specification.service_definition.service_name must be a string")
	}
	serviceNamePattern := regexp.MustCompile(`^[A-Z][a-zA-Z0-9]*Service$`)
	if !serviceNamePattern.MatchString(serviceName) {
		return errors.New("specification.service_definition.service_name must match pattern " +
			"'^[A-Z][a-zA-Z0-9]*Service$'")
	}

	// Validate package_name
	packageName, ok := serviceDef["package_name"].(string)
	if !ok {
		return errors.New("specification.service_definition.package_name must be a string")
	}
	packageNamePattern := regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)*(\\.v[0-9]+)?$`)
	if !packageNamePattern.MatchString(packageName) {
		return errors.New(
			"specification.service_definition.package_name must match pattern '^[a-z][a-z0-9]*(\\.[a-z][a-z0-9]*)*(\\.v[0-9]+)?$'",
		)
	}

	// Validate methods
	if err := validateServiceMethods(serviceDef); err != nil {
		return err
	}

	return nil
}

// validateServiceMethods validates the methods array in service definition.
func validateServiceMethods(serviceDef map[string]interface{}) error {
	methodsInterface, exists := serviceDef["methods"]
	if !exists {
		return errors.New("specification.service_definition.methods is required")
	}

	return validateStringList("specification.service_definition.methods", methodsInterface, methodRule)
}

// validateInstallation validates the installation section.
func validateInstallation(manifest map[string]interface{}) error {
	installation, ok := manifest["installation"].(map[string]interface{})
	if !ok {
		return errors.New("installation must be an object")
	}

	// Validate installation_method
	methodInterface, exists := installation["installation_method"]
	if !exists {
		return errors.New("installation.installation_method is required")
	}

	method, ok := methodInterface.(string)
	if !ok {
		return errors.New("installation.installation_method must be a string")
	}

	if !IsValidInstallationMethod(method) {
		return fmt.Errorf("installation.installation_method: '%s' is not a valid installation method", method)
	}

	return nil
}

// validateSemanticVersion validates that a version string follows semantic versioning.
func validateSemanticVersion(version string) error {
	// Semantic version regex pattern
	semverPattern := regexp.MustCompile(
		`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
			`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
			`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`,
	)

	if !semverPattern.MatchString(version) {
		return fmt.Errorf("version '%s' is not a valid semantic version", version)
	}

	return nil
}
