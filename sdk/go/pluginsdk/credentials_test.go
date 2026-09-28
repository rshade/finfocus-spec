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

package pluginsdk_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

const credentialFixture = "test-secret-value"

func TestCredentialHTTPPrefix(t *testing.T) {
	t.Parallel()
	if got := http.CanonicalHeaderKey(pluginsdk.CredentialMetadataPrefix); got != "X-Finfocus-Credential-" {
		t.Fatalf("canonical prefix = %q", got)
	}
	if pluginsdk.ErrEmptyCredentials.Error() != "per-request credentials require at least one value" {
		t.Fatalf("empty text = %q", pluginsdk.ErrEmptyCredentials.Error())
	}
	if pluginsdk.ErrInvalidCredentials.Error() != "per-request credentials are not valid" {
		t.Fatalf("invalid text = %q", pluginsdk.ErrInvalidCredentials.Error())
	}
	if pluginsdk.ErrMalformedCredentials.Error() != "per-request credentials are malformed" {
		t.Fatalf("malformed text = %q", pluginsdk.ErrMalformedCredentials.Error())
	}
}

func assertNewCredentials(
	t *testing.T,
	entries map[string]string,
	wantErr error,
	wantName, wantGet string,
) {
	t.Helper()
	creds, err := pluginsdk.NewCredentials(entries)
	if wantErr != nil {
		if !errors.Is(err, wantErr) {
			t.Fatalf("error = %v, want %v", err, wantErr)
		}
		if err != nil && strings.Contains(err.Error(), credentialFixture) {
			t.Fatalf("error text contains fixture: %s", err.Error())
		}
		return
	}
	if err != nil {
		t.Fatalf("NewCredentials: %v", err)
	}
	got, ok := creds.Get(wantName)
	if !ok || got != wantGet {
		t.Fatalf("Get(%s) = %q, %v", wantName, got, ok)
	}
	rendered := creds.String()
	goRendered := creds.GoString()
	if strings.Contains(rendered, credentialFixture) || strings.Contains(goRendered, credentialFixture) {
		t.Fatalf("string form leaked fixture: %s / %s", rendered, goRendered)
	}
}

func TestNewCredentials(t *testing.T) {
	t.Parallel()

	longName := "a" + strings.Repeat("b", pluginsdk.MaxCredentialNameLen-1)
	tooLongName := longName + "c"
	maxValue := strings.Repeat("v", pluginsdk.MaxCredentialValueLen)
	overValue := maxValue + "x"

	tests := []struct {
		name     string
		entries  map[string]string
		wantErr  error
		wantName string
		wantGet  string
	}{
		{
			name:    "nil",
			entries: nil,
			wantErr: pluginsdk.ErrEmptyCredentials,
		},
		{
			name:    "empty",
			entries: map[string]string{},
			wantErr: pluginsdk.ErrEmptyCredentials,
		},
		{
			name:     "one entry",
			entries:  map[string]string{"Token": credentialFixture},
			wantName: "token",
			wantGet:  credentialFixture,
		},
		{
			name:    "case duplicate",
			entries: map[string]string{"Token": "one", "token": "two"},
			wantErr: pluginsdk.ErrInvalidCredentials,
		},
		{
			name:    "empty name",
			entries: map[string]string{"": credentialFixture},
			wantErr: pluginsdk.ErrInvalidCredentials,
		},
		{
			name:    "name starts with digit",
			entries: map[string]string{"1token": credentialFixture},
			wantErr: pluginsdk.ErrInvalidCredentials,
		},
		{
			name:    "name with space",
			entries: map[string]string{"access key": credentialFixture},
			wantErr: pluginsdk.ErrInvalidCredentials,
		},
		{
			name:     "name at limit",
			entries:  map[string]string{longName: "ok"},
			wantName: longName,
			wantGet:  "ok",
		},
		{
			name:    "name over limit",
			entries: map[string]string{tooLongName: credentialFixture},
			wantErr: pluginsdk.ErrInvalidCredentials,
		},
		{
			name:    "empty value",
			entries: map[string]string{"token": ""},
			wantErr: pluginsdk.ErrInvalidCredentials,
		},
		{
			name:    "value with newline",
			entries: map[string]string{"token": "abc\n" + credentialFixture},
			wantErr: pluginsdk.ErrInvalidCredentials,
		},
		{
			name:     "value at limit",
			entries:  map[string]string{"token": maxValue},
			wantName: "token",
			wantGet:  maxValue,
		},
		{
			name:    "value over limit",
			entries: map[string]string{"token": overValue},
			wantErr: pluginsdk.ErrInvalidCredentials,
		},
		{
			name:    "non ascii value",
			entries: map[string]string{"token": credentialFixture + "é"},
			wantErr: pluginsdk.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertNewCredentials(t, tt.entries, tt.wantErr, tt.wantName, tt.wantGet)
		})
	}
}

func TestNewCredentialsLimitsAndCopy(t *testing.T) {
	t.Parallel()

	tooMany := make(map[string]string, pluginsdk.MaxCredentialEntries+1)
	for i := range pluginsdk.MaxCredentialEntries + 1 {
		tooMany["k"+strings.Repeat("a", i)] = "v"
	}
	if _, err := pluginsdk.NewCredentials(tooMany); !errors.Is(err, pluginsdk.ErrInvalidCredentials) {
		t.Fatalf("17 entries error = %v", err)
	}

	// 16384/4096 = 4, so five max-sized values exceed the total.
	over := map[string]string{
		"one":   strings.Repeat("a", pluginsdk.MaxCredentialValueLen),
		"two":   strings.Repeat("b", pluginsdk.MaxCredentialValueLen),
		"three": strings.Repeat("c", pluginsdk.MaxCredentialValueLen),
		"four":  strings.Repeat("d", pluginsdk.MaxCredentialValueLen),
		"five":  "z",
	}
	if _, err := pluginsdk.NewCredentials(over); !errors.Is(err, pluginsdk.ErrInvalidCredentials) {
		t.Fatalf("total bytes error = %v", err)
	}

	src := map[string]string{"token": credentialFixture}
	creds, err := pluginsdk.NewCredentials(src)
	if err != nil {
		t.Fatalf("NewCredentials: %v", err)
	}
	src["token"] = "changed"
	got, ok := creds.Get("TOKEN")
	if !ok || got != credentialFixture {
		t.Fatalf("copy = %q, %v", got, ok)
	}
	if creds.Len() != 1 {
		t.Fatalf("Len = %d", creds.Len())
	}
	if names := creds.Names(); len(names) != 1 || names[0] != "token" {
		t.Fatalf("Names = %v", names)
	}
	if creds.String() != "per-request credentials (1 redacted)" {
		t.Fatalf("String = %q", creds.String())
	}
}

func TestExtractCredentialsContext(t *testing.T) {
	t.Parallel()

	var missing context.Context
	if creds, err := pluginsdk.ExtractCredentials(missing); err != nil || creds.Len() != 0 {
		t.Fatalf("nil context = %+v, %v", creds, err)
	}
	if creds, err := pluginsdk.ExtractCredentials(context.Background()); err != nil || creds.Len() != 0 {
		t.Fatalf("empty context = %+v, %v", creds, err)
	}

	ctx := pluginsdk.WithCredentials(context.Background(), pluginsdk.Credentials{})
	if creds, err := pluginsdk.ExtractCredentials(ctx); err != nil || creds.Len() != 0 {
		t.Fatalf("zero credentials stored something: %+v, %v", creds, err)
	}

	creds, err := pluginsdk.NewCredentials(map[string]string{"token": credentialFixture, "other": "second"})
	if err != nil {
		t.Fatalf("NewCredentials: %v", err)
	}
	parent := context.Background()
	first := pluginsdk.WithCredentials(parent, creds)
	other, err := pluginsdk.NewCredentials(map[string]string{"token": "other-value"})
	if err != nil {
		t.Fatalf("NewCredentials other: %v", err)
	}
	second := pluginsdk.WithCredentials(parent, other)

	got, err := pluginsdk.ExtractCredentials(first)
	if err != nil {
		t.Fatalf("first extract: %v", err)
	}
	value, ok := got.Get("token")
	if !ok || value != credentialFixture {
		t.Fatalf("first token = %q, %v", value, ok)
	}
	got, err = pluginsdk.ExtractCredentials(second)
	if err != nil {
		t.Fatalf("second extract: %v", err)
	}
	value, ok = got.Get("token")
	if !ok || value != "other-value" {
		t.Fatalf("second token = %q, %v", value, ok)
	}
	if _, err = pluginsdk.ExtractCredentials(parent); err != nil {
		t.Fatalf("parent gained credentials: %v", err)
	}
	if parentCreds, parentErr := pluginsdk.ExtractCredentials(parent); parentErr != nil || parentCreds.Len() != 0 {
		t.Fatalf("parent = %+v, %v", parentCreds, parentErr)
	}
}

func TestCredentialHeaderRoundTrip(t *testing.T) {
	t.Parallel()

	creds, err := pluginsdk.NewCredentials(map[string]string{"token": credentialFixture})
	if err != nil {
		t.Fatalf("NewCredentials: %v", err)
	}
	ctx := pluginsdk.WithCredentials(context.Background(), creds)
	var gotMD metadata.MD
	interceptor := pluginsdk.CredentialUnaryClientInterceptor()
	err = interceptor(ctx, "/finfocus.v1.CostSource/GetActualCost", nil, nil, nil,
		func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			gotMD, _ = metadata.FromOutgoingContext(ctx)
			return nil
		})
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	values := gotMD.Get(pluginsdk.CredentialMetadataPrefix + "token")
	if len(values) != 1 || values[0] != credentialFixture {
		t.Fatalf("outgoing = %v", values)
	}

	bare := pluginsdk.CredentialUnaryClientInterceptor()
	err = bare(context.Background(), "/finfocus.v1.CostSource/GetActualCost", nil, nil, nil,
		func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			if md, ok := metadata.FromOutgoingContext(ctx); ok {
				if len(md.Get(pluginsdk.CredentialMetadataPrefix+"token")) != 0 {
					t.Fatal("absent context wrote credential metadata")
				}
			}
			return nil
		})
	if err != nil {
		t.Fatalf("bare interceptor: %v", err)
	}
}
