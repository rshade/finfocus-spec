// Copyright 2024-2026 Richard Shade
// Licensed under the Apache License, Version 2.0

package pluginsdk

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// typeRegistryFile is the JSON schema accepted by LoadMappingsFromJSON/LoadMappingsFromFile.
// One source_format per file; see
// specs/050-resolve-resource-types-hardening/contracts/type_registry_mapping_file.schema.json
// for the full contract. finfocus-spec does not ship or maintain any file conforming to
// this schema -- it is a mechanism for plugin- or community-maintained mapping data.
type typeRegistryFile struct {
	SourceFormat string                           `json:"source_format"`
	Mappings     map[string]typeRegistryFileEntry `json:"mappings"`
}

type typeRegistryFileEntry struct {
	PulumiToken      string            `json:"pulumi_token"`
	Supported        bool              `json:"supported"`
	PropertyMappings map[string]string `json:"property_mappings,omitempty"`
}

// sourceFormatByName matches a mapping file's source_format string case-insensitively
// against the SourceFormat enum names.
//
//nolint:gochecknoglobals // Intentional optimization for zero-allocation lookup
var sourceFormatByName = map[string]pbc.SourceFormat{
	"TERRAFORM":      pbc.SourceFormat_SOURCE_FORMAT_TERRAFORM,
	"CLOUDFORMATION": pbc.SourceFormat_SOURCE_FORMAT_CLOUDFORMATION,
}

// LoadMappingsFromJSON reads a JSON-encoded mapping file from reader and registers its
// entries into the registry, merging with any existing mappings for that source_format
// (a later entry with the same key overwrites an earlier one -- last write wins). Must
// be called during initialization, before the gRPC server starts.
//
// Returns a descriptive error (registering nothing) for malformed JSON, an
// unrecognized/missing source_format, or an entry missing pulumi_token.
func (r *TypeRegistry) LoadMappingsFromJSON(reader io.Reader) error {
	var file typeRegistryFile
	if err := json.NewDecoder(reader).Decode(&file); err != nil {
		return fmt.Errorf("pluginsdk: failed to decode type registry mapping file: %w", err)
	}

	format, ok := sourceFormatByName[strings.ToUpper(file.SourceFormat)]
	if !ok {
		return fmt.Errorf(
			"pluginsdk: unrecognized source_format %q; must be one of TERRAFORM, CLOUDFORMATION",
			file.SourceFormat,
		)
	}

	for sourceType, entry := range file.Mappings {
		if entry.PulumiToken == "" {
			return fmt.Errorf("pluginsdk: mapping entry %q is missing required field pulumi_token", sourceType)
		}
	}

	if r.mappings[format] == nil {
		r.mappings[format] = make(map[string]*pbc.ResourceTypeMapping, len(file.Mappings))
	}
	for sourceType, entry := range file.Mappings {
		r.mappings[format][sourceType] = &pbc.ResourceTypeMapping{
			PulumiToken:      entry.PulumiToken,
			Supported:        entry.Supported,
			PropertyMappings: entry.PropertyMappings,
		}
	}

	return nil
}

// LoadMappingsFromFile opens path and delegates to LoadMappingsFromJSON.
func (r *TypeRegistry) LoadMappingsFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("pluginsdk: failed to open type registry mapping file %q: %w", path, err)
	}
	defer f.Close()

	return r.LoadMappingsFromJSON(f)
}
