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

package testing_test

import (
	"errors"
	"strings"
	"testing"

	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	hiddenDefault = "s3cret-value"
	hiddenText    = "do-not-echo"
	hiddenChoice  = "hidden-choice"
)

func TestValidatePluginSettings(t *testing.T) {
	t.Parallel()

	stringType := pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING
	intType := pbc.PluginSettingType_PLUGIN_SETTING_TYPE_INT
	boolType := pbc.PluginSettingType_PLUGIN_SETTING_TYPE_BOOL
	enumType := pbc.PluginSettingType_PLUGIN_SETTING_TYPE_ENUM
	urlType := pbc.PluginSettingType_PLUGIN_SETTING_TYPE_URL
	durationType := pbc.PluginSettingType_PLUGIN_SETTING_TYPE_DURATION

	key := func(value string) *pbc.PluginSetting {
		return &pbc.PluginSetting{Key: value, Type: stringType, Env: "TYPESAFE_API_KEY"}
	}
	enumSetting := func(values []string, def string) *pbc.PluginSetting {
		return &pbc.PluginSetting{
			Key: "region", Type: enumType, Env: "JEV_REGION", EnumValues: values, DefaultValue: def,
		}
	}

	tests := []struct {
		name     string
		settings []*pbc.PluginSetting
		want     string
	}{
		{name: "nil slice"},
		{name: "empty slice", settings: []*pbc.PluginSetting{}},
		{name: "required string without default", settings: []*pbc.PluginSetting{
			{Key: "api_key", Type: stringType, Required: true, Secret: true, Env: "TYPESAFE_API_KEY"},
		}},
		{name: "required with default", settings: []*pbc.PluginSetting{
			{Key: "timeout", Type: durationType, Required: true, DefaultValue: "30s", Env: "JEV_TIMEOUT"},
		}},
		{name: "integer zero", settings: []*pbc.PluginSetting{
			{Key: "retries", Type: intType, DefaultValue: "0", Env: "JEV_RETRIES"},
		}},
		{name: "integer negative", settings: []*pbc.PluginSetting{
			{Key: "offset", Type: intType, DefaultValue: "-1", Env: "JEV_OFFSET"},
		}},
		{name: "integer max", settings: []*pbc.PluginSetting{
			{Key: "limit", Type: intType, DefaultValue: "9223372036854775807", Env: "JEV_LIMIT"},
		}},
		{name: "integer min", settings: []*pbc.PluginSetting{
			{Key: "limit", Type: intType, DefaultValue: "-9223372036854775808", Env: "JEV_LIMIT"},
		}},
		{name: "bool true", settings: []*pbc.PluginSetting{
			{Key: "enabled", Type: boolType, DefaultValue: "true", Env: "JEV_ENABLED"},
		}},
		{name: "bool false", settings: []*pbc.PluginSetting{
			{Key: "enabled", Type: boolType, DefaultValue: "false", Env: "JEV_ENABLED"},
		}},
		{name: "durations", settings: []*pbc.PluginSetting{
			{Key: "timeout", Type: durationType, DefaultValue: "1h30m", Env: "JEV_TIMEOUT"},
			{Key: "zero", Type: durationType, DefaultValue: "0s", Env: "JEV_ZERO"},
		}},
		{name: "url", settings: []*pbc.PluginSetting{
			{Key: "endpoint", Type: urlType, DefaultValue: "https://example.com", Env: "JEV_ENDPOINT"},
		}},
		{name: "enum default", settings: []*pbc.PluginSetting{enumSetting([]string{"us", "eu"}, "us")}},
		{
			name:     "nil entry",
			settings: []*pbc.PluginSetting{nil},
			want:     "settings[0]: nil",
		},
		{
			name:     "too many",
			settings: make([]*pbc.PluginSetting, 65),
			want:     "settings: more than 64 entries",
		},
		{
			name:     "duplicate key",
			settings: []*pbc.PluginSetting{key("api_key"), key("api_key")},
			want:     `settings[1].key: duplicate "api_key"`,
		},
		{
			name: "duplicate env",
			settings: []*pbc.PluginSetting{
				{Key: "api_key", Type: stringType, Env: "TYPESAFE_API_KEY"},
				{Key: "other", Type: stringType, Env: "TYPESAFE_API_KEY"},
			},
			want: `settings[1].env: duplicate "TYPESAFE_API_KEY"`,
		},
		{
			name:     "empty key",
			settings: []*pbc.PluginSetting{key("")},
			want:     "settings[0].key: must match ^[a-z][a-z0-9_]*$",
		},
		{
			name:     "upper key",
			settings: []*pbc.PluginSetting{key("Api")},
			want:     "settings[0].key: must match ^[a-z][a-z0-9_]*$",
		},
		{
			name:     "digit key",
			settings: []*pbc.PluginSetting{key("1abc")},
			want:     "settings[0].key: must match ^[a-z][a-z0-9_]*$",
		},
		{
			name:     "long key",
			settings: []*pbc.PluginSetting{key(strings.Repeat("a", 65))},
			want:     "settings[0].key: longer than 64 bytes",
		},
		{
			name: "unspecified type",
			settings: []*pbc.PluginSetting{
				{Key: "api_key", Env: "TYPESAFE_API_KEY", DefaultValue: hiddenDefault},
			},
			want: "settings[0].type: unspecified",
		},
		{
			name:     "empty env",
			settings: []*pbc.PluginSetting{{Key: "api_key", Type: stringType}},
			want:     "settings[0].env: must match ^[A-Za-z_][A-Za-z0-9_]*$",
		},
		{
			name:     "digit env",
			settings: []*pbc.PluginSetting{{Key: "api_key", Type: stringType, Env: "1ENV"}},
			want:     "settings[0].env: must match ^[A-Za-z_][A-Za-z0-9_]*$",
		},
		{
			name:     "hyphen env",
			settings: []*pbc.PluginSetting{{Key: "api_key", Type: stringType, Env: "API-KEY"}},
			want:     "settings[0].env: must match ^[A-Za-z_][A-Za-z0-9_]*$",
		},
		{
			name:     "long env",
			settings: []*pbc.PluginSetting{{Key: "api_key", Type: stringType, Env: strings.Repeat("A", 129)}},
			want:     "settings[0].env: longer than 128 bytes",
		},
		{
			name: "enum values on string",
			settings: []*pbc.PluginSetting{
				{Key: "api_key", Type: stringType, Env: "TYPESAFE_API_KEY", EnumValues: []string{hiddenChoice}},
			},
			want: "settings[0].enum_values: only allowed for enum",
		},
		{
			name:     "enum without values",
			settings: []*pbc.PluginSetting{enumSetting(nil, "")},
			want:     "settings[0].enum_values: enum requires 1 to 32 values",
		},
		{
			name:     "duplicate enum value",
			settings: []*pbc.PluginSetting{enumSetting([]string{hiddenChoice, hiddenChoice}, "")},
			want:     "settings[0].enum_values: duplicate",
		},
		{
			name:     "long enum value",
			settings: []*pbc.PluginSetting{enumSetting([]string{strings.Repeat("a", 129)}, "")},
			want:     "settings[0].enum_values: longer than 128 bytes",
		},
		{
			name:     "too many enum values",
			settings: []*pbc.PluginSetting{enumSetting(repeatValues(33), "")},
			want:     "settings[0].enum_values: more than 32 values",
		},
		{name: "plus int", settings: intDefault("+1"), want: "settings[0].default_value: not an integer"},
		{name: "neg zero", settings: intDefault("-0"), want: "settings[0].default_value: not an integer"},
		{name: "leading zero", settings: intDefault("01"), want: "settings[0].default_value: not an integer"},
		{
			name:     "int overflow",
			settings: intDefault("9223372036854775808"),
			want:     "settings[0].default_value: not an integer",
		},
		{name: "bool word", settings: boolDefault("True"), want: "settings[0].default_value: not a boolean"},
		{name: "bool digit", settings: boolDefault("1"), want: "settings[0].default_value: not a boolean"},
		{name: "url bare", settings: urlDefault("example.com"), want: "settings[0].default_value: not an absolute URL"},
		{
			name:     "url empty host",
			settings: urlDefault("https://"),
			want:     "settings[0].default_value: not an absolute URL",
		},
		{
			name:     "url space",
			settings: urlDefault("https://ex ample.com"),
			want:     "settings[0].default_value: not an absolute URL",
		},
		{name: "url path", settings: urlDefault("/path"), want: "settings[0].default_value: not an absolute URL"},
		{name: "duration bare", settings: durationDefault("30"), want: "settings[0].default_value: not a duration"},
		{name: "duration words", settings: durationDefault("1 day"), want: "settings[0].default_value: not a duration"},
		{
			name:     "enum default miss",
			settings: []*pbc.PluginSetting{enumSetting([]string{"us"}, hiddenDefault)},
			want:     "settings[0].default_value: not one of the enum values",
		},
		{
			name: "long description",
			settings: []*pbc.PluginSetting{{
				Key: "api_key", Type: stringType, Env: "TYPESAFE_API_KEY",
				Description: hiddenText + strings.Repeat("x", 1024),
			}},
			want: "settings[0].description: longer than 1024 bytes",
		},
		{
			name: "long default",
			settings: []*pbc.PluginSetting{{
				Key: "api_key", Type: stringType, Env: "TYPESAFE_API_KEY",
				DefaultValue: hiddenDefault + strings.Repeat("x", 1024),
			}},
			want: "settings[0].default_value: longer than 1024 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertSettingResult(t, plugintesting.ValidatePluginSettings(tt.settings), tt.want)
		})
	}
}

func assertSettingResult(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		if err != nil {
			t.Fatalf("ValidatePluginSettings() error = %v", err)
		}
		return
	}
	if !errors.Is(err, plugintesting.ErrInvalidPluginSettings) {
		t.Fatalf("error = %v, want ErrInvalidPluginSettings", err)
	}
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
	if strings.Contains(err.Error(), hiddenDefault) ||
		strings.Contains(err.Error(), hiddenText) ||
		strings.Contains(err.Error(), hiddenChoice) {
		t.Fatalf("error %q quotes a hidden value", err.Error())
	}
}

func TestValidatePluginSettings_Allocs(t *testing.T) {
	valid := []*pbc.PluginSetting{
		{Key: "api_key", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_STRING, Env: "TYPESAFE_API_KEY"},
		{Key: "retries", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_INT, DefaultValue: "0", Env: "JEV_RETRIES"},
		{
			Key:          "enabled",
			Type:         pbc.PluginSettingType_PLUGIN_SETTING_TYPE_BOOL,
			DefaultValue: "true",
			Env:          "JEV_ENABLED",
		},
		{
			Key: "region", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_ENUM, Env: "JEV_REGION",
			EnumValues: []string{"us", "eu"}, DefaultValue: "us",
		},
		{
			Key: "endpoint", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_URL,
			DefaultValue: "https://example.com", Env: "JEV_ENDPOINT",
		},
	}
	cases := []struct {
		name     string
		settings []*pbc.PluginSetting
	}{
		{name: "nil"},
		{name: "empty", settings: []*pbc.PluginSetting{}},
		{name: "valid", settings: valid},
	}
	for _, tt := range cases {
		allocs := testing.AllocsPerRun(100, func() {
			if err := plugintesting.ValidatePluginSettings(tt.settings); err != nil {
				t.Fatalf("ValidatePluginSettings() error = %v", err)
			}
		})
		if allocs != 0 {
			t.Fatalf("%s: got %v allocs/op, want 0", tt.name, allocs)
		}
	}
}

func BenchmarkValidatePluginSettings_Empty(b *testing.B) {
	for b.Loop() {
		if err := plugintesting.ValidatePluginSettings(nil); err != nil {
			b.Fatal(err)
		}
	}
}

func intDefault(value string) []*pbc.PluginSetting {
	return []*pbc.PluginSetting{{
		Key: "retries", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_INT,
		DefaultValue: value, Env: "JEV_RETRIES",
	}}
}

func boolDefault(value string) []*pbc.PluginSetting {
	return []*pbc.PluginSetting{{
		Key: "enabled", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_BOOL,
		DefaultValue: value, Env: "JEV_ENABLED", Description: hiddenText,
	}}
}

func urlDefault(value string) []*pbc.PluginSetting {
	return []*pbc.PluginSetting{{
		Key: "endpoint", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_URL,
		DefaultValue: value, Env: "JEV_ENDPOINT",
	}}
}

func durationDefault(value string) []*pbc.PluginSetting {
	return []*pbc.PluginSetting{{
		Key: "timeout", Type: pbc.PluginSettingType_PLUGIN_SETTING_TYPE_DURATION,
		DefaultValue: value, Env: "JEV_TIMEOUT",
	}}
}

func repeatValues(n int) []string {
	values := make([]string, n)
	for i := range values {
		values[i] = strings.Repeat("v", i+1)
	}
	return values
}
