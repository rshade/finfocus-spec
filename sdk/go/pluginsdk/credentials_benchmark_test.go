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

//nolint:testpackage // Measures unexported attachCredentialsFromMetadata, which must not allocate.
package pluginsdk

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestMalformedHTTPCredentialDoesNotEcho(t *testing.T) {
	header := make(http.Header)
	header.Add(credentialHTTPPrefix+"Token", "test-secret-value")
	header.Add(credentialHTTPPrefix+"Token", "test-secret-value")
	ctx, ok := contextWithHTTPCredentials(context.Background(), header)
	if !ok {
		t.Fatal("malformed headers were ignored")
	}
	_, err := ExtractCredentials(ctx)
	if !errors.Is(err, ErrMalformedCredentials) {
		t.Fatalf("error = %v", err)
	}
	if err != nil && strings.Contains(err.Error(), "test-secret-value") {
		t.Fatalf("error contains fixture: %s", err.Error())
	}
}

func TestCredentialHTTPPrefixMatchesCanonical(t *testing.T) {
	if credentialHTTPPrefix != http.CanonicalHeaderKey(CredentialMetadataPrefix) {
		t.Fatalf("prefix %q is not canonical", credentialHTTPPrefix)
	}
}

func TestCredentialAbsentAllocs(t *testing.T) {
	md := metadata.Pairs(TraceIDMetadataKey, "abcdef1234567890abcdef1234567890")
	ctx := context.Background()
	allocs := testing.AllocsPerRun(200, func() {
		if attachCredentialsFromMetadata(ctx, md) != ctx {
			t.Fatal("absent credentials replaced the context")
		}
	})
	if allocs != 0 {
		t.Fatalf("absent parse allocs = %v, want 0", allocs)
	}
}

func BenchmarkCredentialAbsent(b *testing.B) {
	md := metadata.Pairs(TraceIDMetadataKey, "abcdef1234567890abcdef1234567890")
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = attachCredentialsFromMetadata(ctx, md)
	}
}

func BenchmarkNewCredentialsOneEntry(b *testing.B) {
	entries := map[string]string{"token": "example-value"}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := NewCredentials(entries); err != nil {
			b.Fatal(err)
		}
	}
}
