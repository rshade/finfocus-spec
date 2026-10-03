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

package pluginsdk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const timestampFullName protoreflect.FullName = "google.protobuf.Timestamp"

var errManifestNotObject = errors.New("manifest must be an object")

// encodeManifestJSON returns the canonical manifest JSON: proto field names, schema enum strings,
// sorted keys, two-space indent, no HTML escaping, and a trailing newline.
func encodeManifestJSON(m *pbc.PluginManifest) ([]byte, error) {
	raw, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(m)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var obj map[string]any
	if decodeErr := dec.Decode(&obj); decodeErr != nil {
		return nil, decodeErr
	}
	shortenEnums(m.ProtoReflect().Descriptor(), obj)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if encodeErr := enc.Encode(obj); encodeErr != nil {
		return nil, encodeErr
	}
	return buf.Bytes(), nil
}

// encodeManifestYAML returns the canonical manifest as YAML with the same keys, values, and order as
// encodeManifestJSON.
func encodeManifestYAML(m *pbc.PluginManifest) ([]byte, error) {
	data, err := encodeManifestJSON(m)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if unmarshalErr := yaml.Unmarshal(data, &doc); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	if styleErr := blockStyle(&doc); styleErr != nil {
		return nil, styleErr
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2) //nolint:mnd // Two-space indent matches the JSON form.
	if encodeErr := enc.Encode(&doc); encodeErr != nil {
		return nil, encodeErr
	}
	if closeErr := enc.Close(); closeErr != nil {
		return nil, closeErr
	}
	return buf.Bytes(), nil
}

// blockStyle converts the flow-style nodes that JSON decodes into block style. A string keeps
// quotes only when yaml.v3 would quote it, so values such as "1.0" or "no" stay strings.
func blockStyle(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode && n.Tag == "!!str" {
		out, err := yaml.Marshal(n.Value)
		if err != nil {
			return err
		}
		if out[0] != '"' && out[0] != '\'' && out[0] != '|' && out[0] != '>' {
			n.Style = 0
		}
		return nil
	}
	n.Style = 0
	for _, child := range n.Content {
		if err := blockStyle(child); err != nil {
			return err
		}
	}
	return nil
}

// shortenEnums rewrites enum values in a protojson object (proto field names) from the full value
// name to the schema string: INSTALLATION_METHOD_BINARY becomes binary.
func shortenEnums(md protoreflect.MessageDescriptor, obj map[string]any) {
	for key, val := range obj {
		fd := md.Fields().ByName(protoreflect.Name(key))
		if fd == nil {
			continue
		}
		switch {
		case fd.IsMap():
			if fd.MapValue().Kind() != protoreflect.MessageKind {
				continue
			}
			entries, _ := val.(map[string]any)
			for _, entry := range entries {
				if child, ok := entry.(map[string]any); ok {
					shortenEnums(fd.MapValue().Message(), child)
				}
			}
		case fd.Kind() == protoreflect.EnumKind:
			obj[key] = shortenEnumValue(fd.Enum(), val)
		case fd.Kind() == protoreflect.MessageKind:
			forEachObject(val, func(child map[string]any) { shortenEnums(fd.Message(), child) })
		}
	}
}

func shortenEnumValue(ed protoreflect.EnumDescriptor, val any) any {
	if list, ok := val.([]any); ok {
		for i, item := range list {
			list[i] = shortenEnumValue(ed, item)
		}
		return list
	}
	name, ok := val.(string)
	if !ok {
		return val
	}
	return strings.ToLower(strings.TrimPrefix(name, enumPrefix(ed)))
}

// enumPrefix returns the value-name prefix shared by an enum's values, taken from its zero value:
// INSTALLATION_METHOD_UNSPECIFIED gives INSTALLATION_METHOD_.
func enumPrefix(ed protoreflect.EnumDescriptor) string {
	zero := ed.Values().ByNumber(0)
	if zero == nil {
		return ""
	}
	return strings.TrimSuffix(string(zero.Name()), "UNSPECIFIED")
}

func forEachObject(val any, fn func(map[string]any)) {
	switch v := val.(type) {
	case map[string]any:
		fn(v)
	case []any:
		for _, item := range v {
			if child, ok := item.(map[string]any); ok {
				fn(child)
			}
		}
	}
}

// decodeManifest parses a manifest in any form the SDK has written: the canonical form, protojson
// camelCase JSON, and YAML marshaled from the proto struct (lowercased Go field names, integer enums,
// timestamps as seconds and nanos).
func decodeManifest(data []byte, isYAML bool) (*pbc.PluginManifest, error) {
	var raw any
	if isYAML {
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
	} else {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
	}

	manifest := &pbc.PluginManifest{}
	if raw == nil {
		return manifest, nil
	}
	obj, ok := toStringMap(raw)
	if !ok {
		return nil, errManifestNotObject
	}
	normalized, err := normalizeMessage(manifest.ProtoReflect().Descriptor(), obj, "")
	if err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(normalized)
	if err != nil {
		return nil, err
	}
	unmarshaler := protojson.UnmarshalOptions{AllowPartial: true, DiscardUnknown: true}
	if unmarshalErr := unmarshaler.Unmarshal(canonical, manifest); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	return manifest, nil
}

// normalizeMessage rewrites obj into the form protojson reads: keys become proto field names and
// enum values become full value names. Keys that match no field are dropped. Two keys that name the
// same field (spec_version and specVersion) are an error, as they are for protojson; picking one
// would depend on map order and could differ from what a validator read.
func normalizeMessage(md protoreflect.MessageDescriptor, obj map[string]any, path string) (map[string]any, error) {
	out := make(map[string]any, len(obj))
	sourceKeys := make(map[string]string, len(obj))
	for key, val := range obj {
		fd := findField(md, key)
		if fd == nil {
			continue
		}
		name := string(fd.Name())
		fieldPath := name
		if path != "" {
			fieldPath = path + "." + name
		}
		if previous, seen := sourceKeys[name]; seen {
			first, second := min(previous, key), max(previous, key)
			return nil, fmt.Errorf("%s: keys %q and %q set the same field", fieldPath, first, second)
		}
		sourceKeys[name] = key
		normalized, err := normalizeField(fd, val, fieldPath)
		if err != nil {
			return nil, err
		}
		out[name] = normalized
	}
	return out, nil
}

// findField matches a key by proto name, JSON name, or the proto name without underscores compared
// case-insensitively (the form yaml.v3 wrote for proto structs).
func findField(md protoreflect.MessageDescriptor, key string) protoreflect.FieldDescriptor {
	fields := md.Fields()
	if fd := fields.ByName(protoreflect.Name(key)); fd != nil {
		return fd
	}
	if fd := fields.ByJSONName(key); fd != nil {
		return fd
	}
	folded := strings.ReplaceAll(key, "_", "")
	for i := range fields.Len() {
		fd := fields.Get(i)
		if strings.EqualFold(strings.ReplaceAll(string(fd.Name()), "_", ""), folded) {
			return fd
		}
	}
	return nil
}

func normalizeField(fd protoreflect.FieldDescriptor, val any, path string) (any, error) {
	if fd.IsMap() {
		return normalizeMap(fd, val, path)
	}
	if fd.IsList() {
		list, ok := val.([]any)
		if !ok {
			return val, nil
		}
		out := make([]any, len(list))
		for i, item := range list {
			normalized, err := normalizeSingular(fd, item, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			out[i] = normalized
		}
		return out, nil
	}
	return normalizeSingular(fd, val, path)
}

func normalizeMap(fd protoreflect.FieldDescriptor, val any, path string) (any, error) {
	entries, ok := toStringMap(val)
	if !ok || fd.MapValue().Kind() != protoreflect.MessageKind {
		return val, nil
	}
	out := make(map[string]any, len(entries))
	for key, entry := range entries {
		child, isObject := toStringMap(entry)
		if !isObject {
			out[key] = entry
			continue
		}
		normalized, err := normalizeMessage(fd.MapValue().Message(), child, path+"."+key)
		if err != nil {
			return nil, err
		}
		out[key] = normalized
	}
	return out, nil
}

func normalizeSingular(fd protoreflect.FieldDescriptor, val any, path string) (any, error) {
	if fd.Kind() == protoreflect.EnumKind {
		return normalizeEnum(fd.Enum(), val, path)
	}
	if fd.Kind() != protoreflect.MessageKind && fd.Kind() != protoreflect.GroupKind {
		return val, nil
	}
	if fd.Message().FullName() == timestampFullName {
		return normalizeTimestamp(val), nil
	}
	child, ok := toStringMap(val)
	if !ok {
		return val, nil
	}
	return normalizeMessage(fd.Message(), child, path)
}

// normalizeEnum accepts the schema string (binary), the full value name (INSTALLATION_METHOD_BINARY),
// or the value number, and returns the full value name. A null value passes through.
func normalizeEnum(ed protoreflect.EnumDescriptor, val any, path string) (any, error) {
	if val == nil {
		return val, nil
	}
	values := ed.Values()
	if name, ok := val.(string); ok {
		if values.ByName(protoreflect.Name(name)) != nil {
			return name, nil
		}
		full := enumPrefix(ed) + strings.ToUpper(name)
		if values.ByName(protoreflect.Name(full)) != nil {
			return full, nil
		}
	} else if n, isInt := toInt64(val); isInt && n >= math.MinInt32 && n <= math.MaxInt32 {
		if value := values.ByNumber(protoreflect.EnumNumber(n)); value != nil {
			return string(value.Name()), nil
		}
	}
	return nil, fmt.Errorf("%s: %v is not a valid %s value", path, val, ed.Name())
}

// normalizeTimestamp converts the forms a YAML file may hold (a time value, or an object with
// seconds and nanos) to RFC 3339; other values pass through for protojson to judge.
func normalizeTimestamp(val any) any {
	switch v := val.(type) {
	case time.Time:
		return v.UTC().Format(time.RFC3339Nano)
	case map[string]any:
		seconds, _ := toInt64(v["seconds"])
		nanos, _ := toInt64(v["nanos"])
		return time.Unix(seconds, nanos).UTC().Format(time.RFC3339Nano)
	default:
		return val
	}
}

func toInt64(val any) (int64, bool) {
	switch v := val.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	default:
		return 0, false
	}
}

// toStringMap returns val as a map with string keys. yaml.v3 decodes a mapping with any non-string
// key as map[any]any.
func toStringMap(val any) (map[string]any, bool) {
	switch v := val.(type) {
	case map[string]any:
		return v, true
	case map[any]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[fmt.Sprint(key)] = item
		}
		return out, true
	default:
		return nil, false
	}
}
