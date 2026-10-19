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
	"errors"
	"fmt"
	"slices"
	"strings"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

var (
	// ErrInvalidParseRequest identifies invalid or unsupported parsing input.
	ErrInvalidParseRequest = errors.New("invalid parse request")
	// ErrInvalidParseResponse identifies a provider response violating the contract.
	ErrInvalidParseResponse = errors.New("invalid parse response")
)

const maxParseOptionsBytes = 65536

type parseStatusError struct {
	wrapped error
	st      *status.Status
}

// Error returns the client-safe validation message.
func (e *parseStatusError) Error() string { return e.st.Message() }

// Unwrap exposes the validation sentinel for errors.Is.
func (e *parseStatusError) Unwrap() error { return e.wrapped }

// GRPCStatus preserves the required category and structured details.
func (e *parseStatusError) GRPCStatus() *status.Status { return e.st }

func unsupportedParse(reason, field, value string) error {
	st := status.New(
		codes.FailedPrecondition,
		fmt.Sprintf("%s: unsupported %s %q", ErrInvalidParseRequest, field, value),
	)
	detailed, err := st.WithDetails(&errdetails.ErrorInfo{Reason: reason, Domain: ParseErrorDomain})
	if err == nil {
		st = detailed
	}
	return &parseStatusError{wrapped: ErrInvalidParseRequest, st: st}
}

// ValidateParseRequest checks syntax, advertised support and raw content bytes.
// It performs no filesystem I/O and leaves format-specific options to providers.
// A nil or malformed limits value returns ErrInvalidParseLimits.
func ValidateParseRequest(req *pbc.ParseRequest, limits *ParseLimits) error {
	if err := validateParseLimits(limits); err != nil {
		return err
	}
	if req == nil {
		return newInvalidArgument(ErrInvalidParseRequest, "request is nil")
	}
	if !validParseFormat(req.GetFormat()) {
		return newInvalidArgument(ErrInvalidParseRequest, "format must match [a-z][a-z0-9_-]* in 1..32 ASCII bytes")
	}
	mode, inputErr := validateParseInput(req)
	if inputErr != nil {
		return inputErr
	}
	if proto.Size(req.GetOptions()) > maxParseOptionsBytes {
		return newInvalidArgument(ErrInvalidParseRequest, "options exceed %d encoded bytes", maxParseOptionsBytes)
	}
	if !slices.Contains(limits.Formats, req.GetFormat()) {
		return unsupportedParse(UnsupportedFormatReason, "format", req.GetFormat())
	}
	if !slices.Contains(limits.InputModes, mode) {
		return unsupportedParse(UnsupportedInputModeReason, "input mode", mode)
	}
	if mode == parseInputModeContent {
		remaining := limits.MaxInputBytes
		for _, file := range req.GetFiles().GetFiles() {
			size := int64(len(file.GetContent()))
			if size > remaining {
				return &parseStatusError{
					wrapped: ErrInvalidParseRequest,
					st: status.New(
						codes.ResourceExhausted,
						fmt.Sprintf("input bytes exceed parse_max_input_bytes %d", limits.MaxInputBytes),
					),
				}
			}
			remaining -= size
		}
	}
	return nil
}

func validateParseInput(req *pbc.ParseRequest) (string, error) {
	switch input := req.GetInput().(type) {
	case *pbc.ParseRequest_Files:
		if input == nil {
			return "", newInvalidArgument(ErrInvalidParseRequest, "input is nil")
		}
		if input.Files == nil || len(input.Files.GetFiles()) == 0 {
			return "", newInvalidArgument(ErrInvalidParseRequest, "files must be nonempty")
		}
		if err := validateParseFiles(input.Files.GetFiles()); err != nil {
			return "", err
		}
		return parseInputModeContent, nil
	case *pbc.ParseRequest_RootPath:
		if input == nil {
			return "", newInvalidArgument(ErrInvalidParseRequest, "input is nil")
		}
		if input.RootPath == "" {
			return "", newInvalidArgument(ErrInvalidParseRequest, "root_path is empty")
		}
		return parseInputModePath, nil
	default:
		return "", newInvalidArgument(ErrInvalidParseRequest, "input must select files or root_path exclusively")
	}
}

func validateParseFiles(files []*pbc.ParseFile) error {
	paths := make(map[string]struct{}, len(files))
	for i, file := range files {
		if file == nil {
			return newInvalidArgument(ErrInvalidParseRequest, "files[%d] is nil", i)
		}
		if !validParseFilePath(file.GetPath()) {
			return newInvalidArgument(ErrInvalidParseRequest, "files[%d].path is unsafe", i)
		}
		if _, ok := paths[file.GetPath()]; ok {
			return newInvalidArgument(ErrInvalidParseRequest, "files[%d].path is duplicated", i)
		}
		paths[file.GetPath()] = struct{}{}
	}
	return nil
}

func validParseFilePath(path string) bool {
	if path == "" || strings.ContainsAny(path, "\\:\x00") {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

// ValidateParseResponse checks descriptors, source identities, absent unknown
// attribute paths, known optional actions, diagnostics and advisory expiration without mutating input.
// Full request validation is separate because it needs advertised limits.
func ValidateParseResponse(req *pbc.ParseRequest, resp *pbc.ParseResponse) error {
	if req == nil || resp == nil {
		return fmt.Errorf("%w: request and response are required", ErrInvalidParseResponse)
	}
	sources := make(map[string]struct{}, len(resp.GetResources()))
	for i, r := range resp.GetResources() {
		if r == nil {
			return fmt.Errorf("%w: resources[%d] is nil", ErrInvalidParseResponse, i)
		}
		if err := ValidateResourceDescriptor(r.GetResource()); err != nil {
			return fmt.Errorf("%w: resources[%d].resource: %w", ErrInvalidParseResponse, i, err)
		}
		if r.GetSourceId() == "" {
			return fmt.Errorf("%w: resources[%d].source_id is empty", ErrInvalidParseResponse, i)
		}
		if _, ok := sources[r.GetSourceId()]; ok {
			return fmt.Errorf("%w: duplicate source_id %q", ErrInvalidParseResponse, r.GetSourceId())
		}
		sources[r.GetSourceId()] = struct{}{}
		if err := validateParseUnknowns(r, i); err != nil {
			return err
		}
		if _, ok := pbc.ParseAction_name[int32(r.GetAction())]; !ok {
			return fmt.Errorf(
				"%w: resources[%d].action %d is not a known action",
				ErrInvalidParseResponse,
				i,
				r.GetAction(),
			)
		}
	}
	if err := validateParseWarnings(resp.GetWarnings()); err != nil {
		return err
	}
	if expiry := resp.GetExpiresAt(); expiry != nil {
		if err := expiry.CheckValid(); err != nil {
			return fmt.Errorf("%w: expires_at: %w", ErrInvalidParseResponse, err)
		}
	}
	return nil
}

func validateParseWarnings(warnings []*pbc.ParseWarning) error {
	for i, w := range warnings {
		if w == nil ||
			(w.GetSeverity() != pbc.ParseWarningSeverity_PARSE_WARNING_SEVERITY_INFO && w.GetSeverity() != pbc.ParseWarningSeverity_PARSE_WARNING_SEVERITY_WARNING) ||
			w.GetCode() == "" ||
			w.GetMessage() == "" ||
			(w.GetCode() == "resource_skipped" && w.GetSourceId() == "") {
			return fmt.Errorf(
				"%w: warnings[%d] has invalid severity, code, message or source_id",
				ErrInvalidParseResponse,
				i,
			)
		}
	}
	return nil
}

func validateParseUnknowns(r *pbc.ParsedResource, i int) error {
	paths := make(map[string]struct{}, len(r.GetUnknownProperties()))
	for _, path := range r.GetUnknownProperties() {
		if _, ok := paths[path]; ok {
			return fmt.Errorf("%w: duplicate unknown_properties %q", ErrInvalidParseResponse, path)
		}
		paths[path] = struct{}{}
		if !absentParseAttribute(r.GetResource().GetAttributes(), path) {
			return fmt.Errorf(
				"%w: resources[%d].unknown_properties %q must be well formed and absent",
				ErrInvalidParseResponse,
				i,
				path,
			)
		}
	}
	return nil
}

func absentParseAttribute(attributes *structpb.Struct, path string) bool {
	if path == "" || strings.ContainsAny(path, "[]\\") {
		return false
	}
	segments := strings.Split(path, ".")
	for _, segment := range segments {
		if segment == "" {
			return false
		}
	}
	for i, segment := range segments {
		value, present := attributes.GetFields()[segment]
		if !present {
			return true
		}
		if i == len(segments)-1 {
			return false
		}
		object, ok := value.GetKind().(*structpb.Value_StructValue)
		if !ok || object.StructValue == nil {
			return false
		}
		attributes = object.StructValue
	}
	return false
}
