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

package testing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"
	"strings"
	"syscall"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	mockParseProvider        = "custom"
	mockParseUnknown         = "unknown"
	mockParseAction          = "action"
	mockParseDefaultMaxBytes = 65536
)

// MockCostParser is a separate deterministic parser for synthetic fixture input.
// Lines use "resource <source_id>" plus optional unknown, action, skip and notice lines. It never implements a vendor format.
// Instances are immutable after construction and safe for concurrent use.
type MockCostParser struct {
	limits     ParseLimits
	resource   *pbc.ResourceDescriptor
	expiration func() *timestamppb.Timestamp
}

// MockCostParserOption configures a synthetic parser fixture.
type MockCostParserOption func(*MockCostParser)

// WithParseResource clones the priceable descriptor used for each fixture resource.
func WithParseResource(resource *pbc.ResourceDescriptor) MockCostParserOption {
	return func(m *MockCostParser) { m.resource = proto.CloneOf(resource) }
}

// WithMockParseLimits clones the synthetic parser's advertised limits.
// Invalid limits are retained and rejected by shared validation.
func WithMockParseLimits(limits ParseLimits) MockCostParserOption {
	return func(m *MockCostParser) {
		m.limits = ParseLimits{
			Formats:       slices.Clone(limits.Formats),
			InputModes:    slices.Clone(limits.InputModes),
			MaxInputBytes: limits.MaxInputBytes,
		}
	}
}

// WithParseExpiration models a changing dependency with advisory expiration.
// The callback must be concurrency-safe when the mock is shared. Returned
// timestamps are cloned; pure-input fixtures omit this option and expiration.
func WithParseExpiration(expiration func() *timestamppb.Timestamp) MockCostParserOption {
	return func(m *MockCostParser) { m.expiration = expiration }
}

// NewMockCostParser creates a neutral parser accepting fixture content and roots.
// Path fixtures live in input.fixture beneath a caller-owned temporary root.
func NewMockCostParser(opts ...MockCostParserOption) *MockCostParser {
	m := &MockCostParser{
		limits: ParseLimits{
			Formats:       []string{"fixture"},
			InputModes:    []string{parseInputModeContent, parseInputModePath},
			MaxInputBytes: mockParseDefaultMaxBytes,
		},
		resource: &pbc.ResourceDescriptor{Provider: mockParseProvider, ResourceType: "custom:compute:Instance"},
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// GetPluginInfo advertises only parsing and the synthetic fixture limits.
func (m *MockCostParser) GetPluginInfo(context.Context, *pbc.GetPluginInfoRequest) (*pbc.GetPluginInfoResponse, error) {
	return &pbc.GetPluginInfoResponse{
		Name:         "mock-cost-parser",
		Version:      "1.0.0",
		SpecVersion:  "0.7.6",
		Capabilities: []pbc.PluginCapability{pbc.PluginCapability_PLUGIN_CAPABILITY_COST_PARSE},
		Metadata:     FormatParseLimits(m.limits.Formats, m.limits.InputModes, m.limits.MaxInputBytes),
	}, nil
}

// Parse interprets fixed synthetic fixture lines and clones every descriptor.
func (m *MockCostParser) Parse(ctx context.Context, req *pbc.ParseRequest) (*pbc.ParseResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateParseRequest(req, &m.limits); err != nil {
		return nil, err
	}
	for key := range req.GetOptions().GetFields() {
		return nil, newInvalidArgument(ErrInvalidParseRequest, "options.%s is unknown for fixture format", key)
	}
	files := req.GetFiles().GetFiles()
	if req.GetRootPath() != "" {
		var err error
		files, err = m.readParseFiles(ctx, req.GetRootPath())
		if err != nil {
			return nil, err
		}
	}

	resp := &pbc.ParseResponse{}
	sources := make(map[string]*pbc.ParsedResource)
	for _, file := range files {
		for line, text := range strings.Split(string(file.GetContent()), "\n") {
			if strings.TrimSpace(text) == "" {
				continue
			}
			if err := m.parseFixtureLine(resp, sources, text); err != nil {
				return nil, newInvalidArgument(ErrInvalidParseRequest, "%s:%d: %v", file.GetPath(), line+1, err)
			}
		}
	}

	if m.expiration != nil {
		resp.ExpiresAt = proto.CloneOf(m.expiration())
	}
	if err := ValidateParseResponse(req, resp); err != nil {
		return nil, fmt.Errorf("fixture response: %w", err)
	}
	return resp, nil
}

func (m *MockCostParser) readParseFiles(ctx context.Context, path string) ([]*pbc.ParseFile, error) {
	var files []*pbc.ParseFile
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, newInvalidArgument(ErrInvalidParseRequest, "root_path: %v", err)
	}

	remaining := m.limits.MaxInputBytes
	err = fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return newInvalidArgument(ErrInvalidParseRequest, "%s: %v", name, walkErr)
		}
		if contextErr := ctx.Err(); contextErr != nil {
			return contextErr
		}
		if entry.IsDir() || !strings.HasSuffix(name, ".fixture") {
			return nil
		}
		content, readErr := readParseFixture(root, name, remaining)
		if readErr != nil {
			return readErr
		}
		remaining -= int64(len(content))
		files = append(files, &pbc.ParseFile{Path: name, Content: content})
		return nil
	})
	err = errors.Join(err, root.Close())
	if err != nil {
		return nil, err
	}
	return files, nil
}

func readParseFixture(root *os.Root, name string, remaining int64) ([]byte, error) {
	// Nonblocking open prevents FIFOs/devices from waiting before descriptor checks,
	// including a regular-file entry replaced between directory walk and open.
	file, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, newInvalidArgument(ErrInvalidParseRequest, "%s: %v", name, err)
	}
	content, readErr := readOpenedParseFixture(file, name, remaining)
	if err = errors.Join(readErr, file.Close()); err != nil {
		return nil, err
	}
	return content, nil
}

func readOpenedParseFixture(file *os.File, name string, remaining int64) ([]byte, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, newInvalidArgument(ErrInvalidParseRequest, "%s: %v", name, err)
	}
	if !info.Mode().IsRegular() {
		return nil, newInvalidArgument(ErrInvalidParseRequest, "%s: expected regular file", name)
	}
	if info.Size() > remaining {
		return nil, status.Error(codes.ResourceExhausted, "consumed input exceeds parse_max_input_bytes limit")
	}
	// Snapshot length bounds every read even if another process grows the file.
	content, err := io.ReadAll(io.LimitReader(file, info.Size()))
	if err != nil {
		return nil, newInvalidArgument(ErrInvalidParseRequest, "%s: %v", name, err)
	}
	return content, nil
}

func (m *MockCostParser) parseFixtureLine(
	resp *pbc.ParseResponse,
	sources map[string]*pbc.ParsedResource,
	text string,
) error {
	parts := strings.Fields(text)
	switch {
	case len(parts) == 2 && (parts[0] == "resource" || parts[0] == "skip"):
		id := parts[1]
		if _, exists := sources[id]; exists {
			return fmt.Errorf("source_id %q is duplicated", id)
		}
		sources[id] = nil
		if parts[0] == "skip" {
			resp.Warnings = append(
				resp.GetWarnings(),
				&pbc.ParseWarning{
					Severity: pbc.ParseWarningSeverity_PARSE_WARNING_SEVERITY_WARNING,
					Code:     "resource_skipped",
					Message:  "synthetic source skipped",
					SourceId: id,
				},
			)
			return nil
		}
		resource := &pbc.ParsedResource{SourceId: id, Resource: proto.CloneOf(m.resource)}
		resp.Resources = append(resp.GetResources(), resource)
		sources[id] = resource
	case len(parts) == 3 && parts[0] == mockParseUnknown:
		resource := sources[parts[1]]
		if resource == nil {
			return errors.New("unknown must reference an earlier resource source_id")
		}
		if slices.Contains(resource.GetUnknownProperties(), parts[2]) {
			return fmt.Errorf("duplicate unknown_properties %q", parts[2])
		}
		if err := removeMockParseAttribute(resource.GetResource().GetAttributes(), parts[2]); err != nil {
			return err
		}
		resource.UnknownProperties = append(resource.GetUnknownProperties(), parts[2])
	case len(parts) == 3 && parts[0] == mockParseAction:
		return setMockParseAction(sources[parts[1]], parts[1], parts[2])
	case len(parts) >= 2 && parts[0] == "notice":
		resp.Warnings = append(
			resp.GetWarnings(),
			&pbc.ParseWarning{
				Severity: pbc.ParseWarningSeverity_PARSE_WARNING_SEVERITY_INFO,
				Code:     "document_notice",
				Message:  strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "notice")),
			},
		)
	default:
		return errors.New(
			"expected resource <source_id>, skip <source_id>, unknown <source_id> <path>, " +
				"action <source_id> <action> or notice <message>",
		)
	}
	return nil
}

func setMockParseAction(resource *pbc.ParsedResource, id, name string) error {
	if resource == nil {
		return errors.New("action must reference an earlier resource source_id")
	}
	action, known := pbc.ParseAction_value["PARSE_ACTION_"+strings.ToUpper(name)]
	if !known || action == int32(pbc.ParseAction_PARSE_ACTION_UNSPECIFIED) {
		return fmt.Errorf("action %q must be create, update, delete, replace or no_change", name)
	}
	if resource.GetAction() != pbc.ParseAction_PARSE_ACTION_UNSPECIFIED {
		return fmt.Errorf("duplicate action for source_id %q", id)
	}
	resource.Action = pbc.ParseAction(action)
	return nil
}

func removeMockParseAttribute(attributes *structpb.Struct, path string) error {
	if !absentParseAttribute(nil, path) {
		return fmt.Errorf("unknown_properties path %q is malformed", path)
	}
	segments := strings.Split(path, ".")
	for i, segment := range segments {
		if i == len(segments)-1 {
			delete(attributes.GetFields(), segment)
			return nil
		}
		value, present := attributes.GetFields()[segment]
		if !present {
			return nil
		}
		object, ok := value.GetKind().(*structpb.Value_StructValue)
		if !ok || object.StructValue == nil {
			return fmt.Errorf("unknown_properties ancestor %q is not an object", segment)
		}
		attributes = object.StructValue
	}
	return nil
}
