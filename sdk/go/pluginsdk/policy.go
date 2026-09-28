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
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrInvalidPolicy is wrapped by every DecodePolicy input error.
var ErrInvalidPolicy = errors.New("invalid allocation policy")

//nolint:gochecknoglobals // Reflection types computed once for the opaque-type check.
var (
	rawMessageType      = reflect.TypeOf(json.RawMessage(nil))
	jsonUnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

// policyError is a plain error carrying codes.InvalidArgument, so allocators
// can return DecodePolicy errors directly without an "rpc error:" prefix.
type policyError struct {
	msg string
}

func (e *policyError) Error() string { return e.msg }

func (e *policyError) Unwrap() error { return ErrInvalidPolicy }

// GRPCStatus reports the error as codes.InvalidArgument with the same message.
func (e *policyError) GRPCStatus() *status.Status {
	return status.New(codes.InvalidArgument, e.msg)
}

func policyErrorf(format string, args ...any) error {
	return &policyError{msg: ErrInvalidPolicy.Error() + ": " + fmt.Sprintf(format, args...)}
}

// DecodePolicy applies the JSON policy document data onto target, which must
// be a non-nil pointer already holding the allocator's defaults. Empty,
// whitespace-only, and null documents leave target unchanged. Unknown fields
// (exact, case-sensitive match), malformed JSON, trailing data, and type
// mismatches return an error that wraps ErrInvalidPolicy, names the JSON path
// (for example "node_split.cpu"), and carries codes.InvalidArgument. Nested
// objects, including struct values held in maps, merge field by field; arrays
// replace the default wholesale. On error, target may be partially updated.
//
// Fields of type any, json.RawMessage, or types implementing json.Unmarshaler
// or encoding.TextUnmarshaler are opaque: they accept any value. A nil or
// non-pointer target is a programming error and returns a plain error with
// no status code.
func DecodePolicy(data []byte, target any) error {
	rv := reflect.ValueOf(target)
	if target == nil || rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("pluginsdk: DecodePolicy target must be a non-nil pointer, got %T", target)
	}

	doc := bytes.TrimSpace(data)
	if len(doc) == 0 || bytes.Equal(doc, []byte("null")) {
		return nil
	}

	parsed, err := parsePolicyDocument(doc)
	if err != nil {
		return err
	}
	if walkErr := walkPolicy(parsed, rv.Elem().Type(), rv.Elem(), ""); walkErr != nil {
		return walkErr
	}
	merged, err := json.Marshal(parsed)
	if err != nil {
		return fmt.Errorf("pluginsdk: re-encode policy document: %w", err)
	}
	return unmarshalPolicy(merged, target)
}

// parsePolicyDocument parses exactly one JSON value and rejects trailing data.
func parsePolicyDocument(doc []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(doc))
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return nil, policyErrorf("malformed JSON: %v", err)
	}
	if err := dec.Decode(&json.RawMessage{}); !errors.Is(err, io.EOF) {
		return nil, policyErrorf("trailing data after the policy document")
	}
	parsed, err := decodeJSONNumbers(raw)
	if err != nil {
		return nil, policyErrorf("malformed JSON: %v", err)
	}
	return parsed, nil
}

// decodeJSONNumbers keeps numbers as json.Number so re-encoding is lossless.
func decodeJSONNumbers(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var parsed any
	if err := dec.Decode(&parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func unmarshalPolicy(doc []byte, target any) error {
	err := json.Unmarshal(doc, target)
	if err == nil {
		return nil
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		path := typeErr.Field
		if path == "" {
			path = "(root)"
		}
		return policyErrorf("%s: cannot use JSON %s as %s", path, typeErr.Value, typeErr.Type)
	}
	return policyErrorf("%v", err)
}

func joinPolicyPath(parent, key string) string {
	if parent == "" {
		return key
	}
	return parent + "." + key
}

func isOpaquePolicyType(t reflect.Type) bool {
	if t == rawMessageType || t.Kind() == reflect.Interface {
		return true
	}
	pt := reflect.PointerTo(t)
	return t.Implements(jsonUnmarshalerType) || pt.Implements(jsonUnmarshalerType) ||
		t.Implements(textUnmarshalerType) || pt.Implements(textUnmarshalerType)
}

// walkPolicy checks doc against t for unknown fields, building JSON paths.
// v is the matching part of the target when it exists (it may be invalid);
// slices the document supplies are reset to nil in v so they are replaced
// wholesale rather than merged element by element, and object-valued map
// entries have v's existing entry merged into doc in place, because
// json.Unmarshal decodes map values into fresh zero values. Shape mismatches
// are left for json.Unmarshal to report as type errors.
func walkPolicy(doc any, t reflect.Type, v reflect.Value, path string) error {
	if doc == nil || isOpaquePolicyType(t) {
		return nil
	}
	//nolint:exhaustive // Only container kinds need walking; scalars are checked by json.Unmarshal.
	switch t.Kind() {
	case reflect.Pointer:
		var elem reflect.Value
		if v.IsValid() && !v.IsNil() {
			elem = v.Elem()
		}
		return walkPolicy(doc, t.Elem(), elem, path)
	case reflect.Struct:
		return walkPolicyStruct(doc, t, v, path)
	case reflect.Map:
		return walkPolicyMap(doc, t, v, path)
	case reflect.Slice, reflect.Array:
		return walkPolicyList(doc, t, v, path)
	default:
		return nil
	}
}

func walkPolicyStruct(doc any, t reflect.Type, v reflect.Value, path string) error {
	obj, ok := doc.(map[string]any)
	if !ok {
		return nil
	}
	fields := policyFields(t)
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fieldPath := joinPolicyPath(path, key)
		index, known := fields[key]
		if !known {
			return policyErrorf("unknown field %q", fieldPath)
		}
		field := t.FieldByIndex(index)
		var fv reflect.Value
		if v.IsValid() {
			// An error means a nil embedded pointer; walk by type only.
			fv, _ = v.FieldByIndexErr(index)
		}
		if err := walkPolicy(obj[key], field.Type, fv, fieldPath); err != nil {
			return err
		}
	}
	return nil
}

func walkPolicyMap(doc any, t reflect.Type, v reflect.Value, path string) error {
	obj, ok := doc.(map[string]any)
	if !ok {
		return nil
	}
	defaults, err := policyMapDefaults(v, path)
	if err != nil {
		return err
	}
	for key, override := range obj {
		if def, found := defaults[key]; found {
			obj[key] = mergePolicyDefaults(def, override, t.Elem())
		}
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if walkErr := walkPolicy(obj[key], t.Elem(), reflect.Value{}, joinPolicyPath(path, key)); walkErr != nil {
			return walkErr
		}
	}
	return nil
}

// policyMapDefaults returns the existing entries of map v in JSON form, keyed
// as encoding/json encodes map keys.
func policyMapDefaults(v reflect.Value, path string) (map[string]any, error) {
	if !v.IsValid() || v.IsNil() || v.Len() == 0 || !v.CanInterface() {
		return nil, nil //nolint:nilnil // A nil map means "no defaults to merge".
	}
	encoded, err := json.Marshal(v.Interface())
	if err != nil {
		return nil, fmt.Errorf("pluginsdk: encode default policy map %q: %w", path, err)
	}
	decoded, err := decodeJSONNumbers(encoded)
	if err != nil {
		return nil, fmt.Errorf("pluginsdk: decode default policy map %q: %w", path, err)
	}
	defaults, _ := decoded.(map[string]any)
	return defaults, nil
}

// mergePolicyDefaults overlays override onto def when both are JSON objects
// decoded into a non-opaque struct or map type t; anything else, including
// arrays, is replaced by override. Unknown struct keys are kept so walkPolicy
// still rejects them.
func mergePolicyDefaults(def, override any, t reflect.Type) any {
	defObj, defIsObj := def.(map[string]any)
	overObj, overIsObj := override.(map[string]any)
	if !defIsObj || !overIsObj || isOpaquePolicyType(t) {
		return override
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	var fieldType func(key string) (reflect.Type, bool)
	//nolint:exhaustive // Only struct and map values can hold nested defaults.
	switch t.Kind() {
	case reflect.Struct:
		fields := policyFields(t)
		fieldType = func(key string) (reflect.Type, bool) {
			index, known := fields[key]
			if !known {
				return nil, false
			}
			return t.FieldByIndex(index).Type, true
		}
	case reflect.Map:
		fieldType = func(string) (reflect.Type, bool) { return t.Elem(), true }
	default:
		return override
	}

	merged := make(map[string]any, len(defObj))
	maps.Copy(merged, defObj)
	for key, value := range overObj {
		if ft, known := fieldType(key); known {
			merged[key] = mergePolicyDefaults(defObj[key], value, ft)
		} else {
			merged[key] = value
		}
	}
	return merged
}

func walkPolicyList(doc any, t reflect.Type, v reflect.Value, path string) error {
	items, ok := doc.([]any)
	if !ok {
		return nil
	}
	if t.Kind() == reflect.Slice && v.IsValid() && v.CanSet() {
		v.Set(reflect.Zero(t))
	}
	for i, item := range items {
		var elem reflect.Value
		if t.Kind() == reflect.Array && v.IsValid() && i < v.Len() {
			elem = v.Index(i)
		}
		if err := walkPolicy(item, t.Elem(), elem, path+"["+strconv.Itoa(i)+"]"); err != nil {
			return err
		}
	}
	return nil
}

// policyFields maps the JSON names encoding/json would use for t's fields to
// their index paths. Direct fields take precedence over promoted ones.
func policyFields(t reflect.Type) map[string][]int {
	fields := make(map[string][]int, t.NumField())
	var embedded []reflect.StructField
	for i := range t.NumField() {
		f := t.Field(i)
		name, include := policyFieldName(f)
		if !include {
			continue
		}
		if f.Anonymous && name == "" {
			embedded = append(embedded, f)
			continue
		}
		if name == "" {
			name = f.Name
		}
		fields[name] = f.Index
	}
	for _, f := range embedded {
		promotePolicyFields(fields, f)
	}
	return fields
}

// promotePolicyFields adds the fields of embedded field f to fields, without
// overriding names already taken at a shallower depth.
func promotePolicyFields(fields map[string][]int, f reflect.StructField) {
	ft := f.Type
	if ft.Kind() == reflect.Pointer {
		ft = ft.Elem()
	}
	if ft.Kind() != reflect.Struct {
		if f.IsExported() {
			fields[f.Name] = f.Index
		}
		return
	}
	for name, index := range policyFields(ft) {
		if _, taken := fields[name]; !taken {
			fields[name] = append(append([]int{}, f.Index...), index...)
		}
	}
}

// policyFieldName returns the tag name ("" when untagged) and whether
// encoding/json would consider the field at all.
func policyFieldName(f reflect.StructField) (string, bool) {
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", false
	}
	if !f.IsExported() && !f.Anonymous {
		return "", false
	}
	name, _, _ := strings.Cut(tag, ",")
	return name, true
}
