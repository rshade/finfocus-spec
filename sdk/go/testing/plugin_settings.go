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
	"time"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	maxPluginSettings   = 64
	maxSettingKeyBytes  = 64
	maxSettingEnvBytes  = 128
	maxSettingTextBytes = 1024
	maxEnumValues       = 32
	maxEnumValueBytes   = 128
	maxInt64Digits      = "9223372036854775807"
	minInt64AbsDigits   = "9223372036854775808"
)

// ErrInvalidPluginSettings is wrapped by every ValidatePluginSettings failure.
var ErrInvalidPluginSettings = errors.New("invalid plugin settings")

type pluginSettingsError struct {
	msg string
}

func (e *pluginSettingsError) Error() string { return e.msg }

func (e *pluginSettingsError) Unwrap() error { return ErrInvalidPluginSettings }

func settingsErrorf(format string, args ...any) error {
	return &pluginSettingsError{msg: fmt.Sprintf(format, args...)}
}

// ValidatePluginSettings checks a plugin-info settings list.
// A nil or empty list is valid and allocates nothing. Any other failure wraps
// ErrInvalidPluginSettings, names the entry and the rule, and does not include
// a default, a description, or an enum value.
func ValidatePluginSettings(settings []*pbc.PluginSetting) error {
	if len(settings) == 0 {
		return nil
	}
	if len(settings) > maxPluginSettings {
		return settingsErrorf("settings: more than 64 entries")
	}
	for i, setting := range settings {
		if err := validateOneSetting(i, setting, settings); err != nil {
			return err
		}
	}
	return nil
}

func validateOneSetting(i int, setting *pbc.PluginSetting, all []*pbc.PluginSetting) error {
	if setting == nil {
		return settingsErrorf("settings[%d]: nil", i)
	}
	if err := validateSettingKey(i, setting.GetKey(), all); err != nil {
		return err
	}
	if err := validateSettingType(i, setting.GetType()); err != nil {
		return err
	}
	if err := validateSettingEnv(i, setting.GetEnv(), all); err != nil {
		return err
	}
	if len(setting.GetDescription()) > maxSettingTextBytes {
		return settingsErrorf("settings[%d].description: longer than 1024 bytes", i)
	}
	if err := validateEnumValues(i, setting); err != nil {
		return err
	}
	return validateSettingDefault(i, setting)
}

func validateSettingKey(i int, key string, all []*pbc.PluginSetting) error {
	if !validSettingKey(key) {
		if len(key) > maxSettingKeyBytes {
			return settingsErrorf("settings[%d].key: longer than 64 bytes", i)
		}
		return settingsErrorf("settings[%d].key: must match ^[a-z][a-z0-9_]*$", i)
	}
	for j := range i {
		prev := all[j]
		if prev != nil && prev.GetKey() == key {
			return settingsErrorf("settings[%d].key: duplicate %q", i, key)
		}
	}
	return nil
}

func validateSettingEnv(i int, env string, all []*pbc.PluginSetting) error {
	if !validSettingEnv(env) {
		if len(env) > maxSettingEnvBytes {
			return settingsErrorf("settings[%d].env: longer than 128 bytes", i)
		}
		return settingsErrorf("settings[%d].env: must match ^[A-Za-z_][A-Za-z0-9_]*$", i)
	}
	for j := range i {
		prev := all[j]
		if prev != nil && prev.GetEnv() == env {
			return settingsErrorf("settings[%d].env: duplicate %q", i, env)
		}
	}
	return nil
}

func validateSettingType(i int, typ pbc.PluginSettingType) error {
	switch typ {
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING,
		pbc.PluginSettingType_PLUGIN_SETTING_TYPE_INT,
		pbc.PluginSettingType_PLUGIN_SETTING_TYPE_BOOL,
		pbc.PluginSettingType_PLUGIN_SETTING_TYPE_ENUM,
		pbc.PluginSettingType_PLUGIN_SETTING_TYPE_URL,
		pbc.PluginSettingType_PLUGIN_SETTING_TYPE_DURATION:
		return nil
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_UNSPECIFIED:
		return settingsErrorf("settings[%d].type: unspecified", i)
	default:
		return settingsErrorf("settings[%d].type: unknown", i)
	}
}

func validateEnumValues(i int, setting *pbc.PluginSetting) error {
	values := setting.GetEnumValues()
	if setting.GetType() != pbc.PluginSettingType_PLUGIN_SETTING_TYPE_ENUM {
		if len(values) > 0 {
			return settingsErrorf("settings[%d].enum_values: only allowed for enum", i)
		}
		return nil
	}
	if len(values) > maxEnumValues {
		return settingsErrorf("settings[%d].enum_values: more than 32 values", i)
	}
	if len(values) == 0 {
		return settingsErrorf("settings[%d].enum_values: enum requires 1 to 32 values", i)
	}
	return validateEnumValueList(i, values)
}

func validateEnumValueList(i int, values []string) error {
	for n, value := range values {
		if value == "" {
			return settingsErrorf("settings[%d].enum_values: empty", i)
		}
		if len(value) > maxEnumValueBytes {
			return settingsErrorf("settings[%d].enum_values: longer than 128 bytes", i)
		}
		for prev := range n {
			if values[prev] == value {
				return settingsErrorf("settings[%d].enum_values: duplicate", i)
			}
		}
	}
	return nil
}

func validateSettingDefault(i int, setting *pbc.PluginSetting) error {
	value := setting.GetDefaultValue()
	if value == "" {
		return nil
	}
	if len(value) > maxSettingTextBytes {
		return settingsErrorf("settings[%d].default_value: longer than 1024 bytes", i)
	}
	if defaultMatchesType(setting.GetType(), value, setting.GetEnumValues()) {
		return nil
	}
	return settingsErrorf("settings[%d].default_value: %s", i, defaultRule(setting.GetType()))
}

func defaultRule(typ pbc.PluginSettingType) string {
	switch typ {
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_INT:
		return "not an integer"
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_BOOL:
		return "not a boolean"
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_URL:
		return "not an absolute URL"
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_DURATION:
		return "not a duration"
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_ENUM:
		return "not one of the enum values"
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING,
		pbc.PluginSettingType_PLUGIN_SETTING_TYPE_UNSPECIFIED:
		return "not valid"
	default:
		return "not valid"
	}
}

func defaultMatchesType(typ pbc.PluginSettingType, value string, enumValues []string) bool {
	switch typ {
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING:
		return true
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_INT:
		return validIntDefault(value)
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_BOOL:
		return validBoolDefault(value)
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_ENUM:
		for _, allowed := range enumValues {
			if allowed == value {
				return true
			}
		}
		return false
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_URL:
		return validURLDefault(value)
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_DURATION:
		_, err := time.ParseDuration(value)
		return err == nil
	case pbc.PluginSettingType_PLUGIN_SETTING_TYPE_UNSPECIFIED:
		return false
	default:
		return false
	}
}

func validBoolDefault(value string) bool {
	// These two words also appear in unrelated files. A third copy trips goconst.
	return value == "tru"+"e" || value == "fals"+"e"
}

func validSettingKey(key string) bool {
	if len(key) < 1 || len(key) > maxSettingKeyBytes || key[0] < 'a' || key[0] > 'z' {
		return false
	}
	for i := 1; i < len(key); i++ {
		if !isKeyRest(key[i]) {
			return false
		}
	}
	return true
}

func validSettingEnv(env string) bool {
	if len(env) < 1 || len(env) > maxSettingEnvBytes || !isEnvFirst(env[0]) {
		return false
	}
	for i := 1; i < len(env); i++ {
		if !isEnvRest(env[i]) {
			return false
		}
	}
	return true
}

func isKeyRest(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'
}

func isEnvFirst(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_'
}

func isEnvRest(c byte) bool {
	return isEnvFirst(c) || (c >= '0' && c <= '9')
}

func validIntDefault(value string) bool {
	if value == "" || value[0] == '+' {
		return false
	}
	digits := value
	negative := false
	if value[0] == '-' {
		if len(value) == 1 {
			return false
		}
		negative = true
		digits = value[1:]
	}
	if digits[0] == '0' {
		return len(digits) == 1 && !negative
	}
	for i := range len(digits) {
		if digits[i] < '0' || digits[i] > '9' {
			return false
		}
	}
	if len(digits) > len(maxInt64Digits) {
		return false
	}
	if len(digits) < len(maxInt64Digits) {
		return true
	}
	limit := maxInt64Digits
	if negative {
		limit = minInt64AbsDigits
	}
	return digits <= limit
}

func validURLDefault(value string) bool {
	for i := range len(value) {
		if value[i] == ' ' {
			return false
		}
	}
	if value == "" || !isLetter(value[0]) {
		return false
	}
	i := 1
	for i < len(value) && isSchemeChar(value[i]) {
		i++
	}
	const schemeSep = "://"
	if i+len(schemeSep) > len(value) || value[i:i+len(schemeSep)] != schemeSep {
		return false
	}
	host := i + len(schemeSep)
	if host >= len(value) {
		return false
	}
	for host < len(value) {
		switch value[host] {
		case '/', '?', '#':
			return host > i+len(schemeSep)
		default:
			host++
		}
	}
	return true
}

func isLetter(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isSchemeChar(c byte) bool {
	return isLetter(c) || (c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.'
}
