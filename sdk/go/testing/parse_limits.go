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
	"strconv"
	"strings"
)

// ErrInvalidParseLimits identifies malformed or missing parser configuration.
var ErrInvalidParseLimits = errors.New("invalid parse limits")

const (
	parseInputModeContent = "content"
	parseInputModePath    = "path"
)

// ParseLimits are the parser's advertised formats, modes and raw input byte cap.
type ParseLimits struct {
	Formats       []string
	InputModes    []string
	MaxInputBytes int64
}

func validParseFormat(s string) bool {
	if len(s) < 1 || len(s) > 32 || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func validateParseLimits(limits *ParseLimits) error {
	if limits == nil || len(limits.Formats) == 0 || len(limits.InputModes) == 0 || limits.MaxInputBytes < 1 {
		return fmt.Errorf("%w: formats, input modes and positive max_input_bytes required", ErrInvalidParseLimits)
	}
	for i, f := range limits.Formats {
		if !validParseFormat(f) || slices.Contains(limits.Formats[:i], f) {
			return fmt.Errorf("%w: invalid or duplicate format %q", ErrInvalidParseLimits, f)
		}
	}
	for i, m := range limits.InputModes {
		if (m != parseInputModeContent && m != parseInputModePath) || slices.Contains(limits.InputModes[:i], m) {
			return fmt.Errorf("%w: invalid or duplicate input mode %q", ErrInvalidParseLimits, m)
		}
	}
	return nil
}

const (
	// ParseFormatsKey advertises comma-separated format names.
	ParseFormatsKey = "parse_formats"
	// ParseInputModesKey advertises content and/or path modes.
	ParseInputModesKey = "parse_input_modes"
	// ParseMaxInputBytesKey advertises the positive raw byte cap.
	ParseMaxInputBytesKey = "parse_max_input_bytes"
	// ParseErrorDomain identifies parsing ErrorInfo details.
	ParseErrorDomain = "finfocus.v1.CostParseService"
	// UnsupportedFormatReason identifies an unadvertised format.
	UnsupportedFormatReason = "UNSUPPORTED_FORMAT"
	// UnsupportedInputModeReason identifies an unadvertised input mode.
	UnsupportedInputModeReason = "UNSUPPORTED_INPUT_MODE"
)

// FormatParseLimits records supplied configuration without validating or repairing it.
func FormatParseLimits(formats, inputModes []string, maxInputBytes int64) map[string]string {
	return map[string]string{
		ParseFormatsKey:       strings.Join(formats, ","),
		ParseInputModesKey:    strings.Join(inputModes, ","),
		ParseMaxInputBytesKey: strconv.FormatInt(maxInputBytes, 10),
	}
}

// ParseParseLimits reads canonical parser metadata. No parse keys returns nil,nil;
// partial, duplicate or malformed values wrap ErrInvalidParseLimits.
func ParseParseLimits(metadata map[string]string) (*ParseLimits, error) {
	formats, hasFormats := metadata[ParseFormatsKey]
	modes, hasModes := metadata[ParseInputModesKey]
	raw, hasMax := metadata[ParseMaxInputBytesKey]
	if !hasFormats && !hasModes && !hasMax {
		return nil, nil //nolint:nilnil // No parse metadata means no advertised parser.
	}
	if !hasFormats || !hasModes || !hasMax {
		return nil, fmt.Errorf("%w: all three parse metadata keys required", ErrInvalidParseLimits)
	}
	maxBytes, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || maxBytes < 1 || strconv.FormatInt(maxBytes, 10) != raw {
		return nil, fmt.Errorf("%w: %s must be canonical positive int64", ErrInvalidParseLimits, ParseMaxInputBytesKey)
	}
	limits := &ParseLimits{
		Formats:       strings.Split(formats, ","),
		InputModes:    strings.Split(modes, ","),
		MaxInputBytes: maxBytes,
	}
	if err = validateParseLimits(limits); err != nil {
		return nil, err
	}
	return limits, nil
}
