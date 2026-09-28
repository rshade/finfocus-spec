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
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// MetadataSupportsPerRequestCredentials is the GetPluginInfo metadata key
	// set to "true" when the plugin implements PerRequestCredentialConsumer.
	MetadataSupportsPerRequestCredentials = "supports_per_request_credentials" //nolint:gosec // G101: metadata key, not a secret

	// CredentialMetadataPrefix is the gRPC metadata and HTTP header prefix for
	// one credential entry. The header is this prefix plus the lowercase name.
	CredentialMetadataPrefix = "x-finfocus-credential-" //nolint:gosec // G101: header prefix, not a secret

	// credentialHTTPPrefix is CredentialMetadataPrefix in canonical HTTP form.
	credentialHTTPPrefix = "X-Finfocus-Credential-" //nolint:gosec // G101: header prefix, not a secret

	// MaxCredentialEntries is the maximum number of names in one set.
	MaxCredentialEntries = 16

	// MaxCredentialNameLen is the maximum name length after ASCII lowercasing.
	MaxCredentialNameLen = 64

	// MaxCredentialValueLen is the maximum length of one credential value.
	MaxCredentialValueLen = 4096

	// MaxCredentialBytes is the maximum sum of value lengths in one set.
	MaxCredentialBytes = 16384
)

var (
	// ErrEmptyCredentials is returned when NewCredentials is given no entries.
	ErrEmptyCredentials = errors.New("per-request credentials require at least one value")

	// ErrInvalidCredentials is returned when a set breaks the name, value, or size rules.
	// The text never includes the name or the value.
	ErrInvalidCredentials = errors.New("per-request credentials are not valid")

	// ErrMalformedCredentials is returned when credential headers arrived but could
	// not be read. The text never includes the name or the value.
	ErrMalformedCredentials = errors.New("per-request credentials are malformed")
)

// PerRequestCredentialConsumer is the opt-in for per-request cloud credentials.
// Implementing the method is the whole declaration. Serve does not call it.
// The plugin reads the call with ExtractCredentials.
type PerRequestCredentialConsumer interface {
	ConsumesPerRequestCredentials()
}

// Credentials is the named secret material attached to one call.
// The zero value means nothing is attached. String and GoString never
// include names or values.
type Credentials struct {
	entries map[string]string
}

// NewCredentials copies entries into an immutable set.
// Names are matched without regard to ASCII letter case. A nil or empty map
// returns ErrEmptyCredentials. Any other violation returns ErrInvalidCredentials.
func NewCredentials(entries map[string]string) (Credentials, error) {
	if len(entries) == 0 {
		return Credentials{}, ErrEmptyCredentials
	}
	if len(entries) > MaxCredentialEntries {
		return Credentials{}, ErrInvalidCredentials
	}

	copied := make(map[string]string, len(entries))
	total := 0
	for name, value := range entries {
		norm, ok := canonicalCredentialName(name)
		if !ok {
			return Credentials{}, ErrInvalidCredentials
		}
		if _, exists := copied[norm]; exists {
			return Credentials{}, ErrInvalidCredentials
		}
		if !validCredentialValue(value) {
			return Credentials{}, ErrInvalidCredentials
		}
		total += len(value)
		if total > MaxCredentialBytes {
			return Credentials{}, ErrInvalidCredentials
		}
		copied[norm] = value
	}
	return Credentials{entries: copied}, nil
}

// Get returns the value for name. The lookup ignores ASCII letter case.
func (c Credentials) Get(name string) (string, bool) {
	if len(c.entries) == 0 {
		return "", false
	}
	norm, ok := canonicalCredentialName(name)
	if !ok {
		return "", false
	}
	value, found := c.entries[norm]
	return value, found
}

// Len returns the number of entries. The zero value returns 0.
func (c Credentials) Len() int {
	return len(c.entries)
}

// Names returns lowercase names in ascending order.
// It returns nil for the zero value.
func (c Credentials) Names() []string {
	if len(c.entries) == 0 {
		return nil
	}
	names := make([]string, 0, len(c.entries))
	for name := range c.entries {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// String reports only how many entries are present.
func (c Credentials) String() string {
	if len(c.entries) == 0 {
		return "per-request credentials (none)"
	}
	return fmt.Sprintf("per-request credentials (%d redacted)", len(c.entries))
}

// GoString reports only how many entries are present.
func (c Credentials) GoString() string {
	return c.String()
}

type credentialCtxKey struct{}

type credentialErrCtxKey struct{}

// WithCredentials returns a child of ctx that carries creds for this call only.
// A zero Credentials value leaves ctx unchanged. ctx must be non-nil when creds
// is non-zero, matching context.WithValue. The value is not written to the
// process environment.
func WithCredentials(ctx context.Context, creds Credentials) context.Context {
	if creds.Len() == 0 {
		return ctx
	}
	return context.WithValue(ctx, credentialCtxKey{}, creds)
}

// ExtractCredentials reads the set attached to ctx.
// A nil context, or a context with no set, returns the zero value and a nil
// error. That is not a failure. Unusable wire material returns
// ErrMalformedCredentials and the zero value.
func ExtractCredentials(ctx context.Context) (Credentials, error) {
	if ctx == nil {
		return Credentials{}, nil
	}
	if _, failed := ctx.Value(credentialErrCtxKey{}).(error); failed {
		return Credentials{}, ErrMalformedCredentials
	}
	creds, ok := ctx.Value(credentialCtxKey{}).(Credentials)
	if !ok || creds.Len() == 0 {
		return Credentials{}, nil
	}
	return creds, nil
}

// CredentialUnaryClientInterceptor copies a valid set from ctx into outgoing
// gRPC metadata. Hosts that use pluginsdk.Client do not need it. It does not log.
func CredentialUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		creds, err := ExtractCredentials(ctx)
		if err != nil || creds.Len() == 0 {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		names := creds.Names()
		pairs := make([]string, 0, len(names)+len(names))
		for _, name := range names {
			value, ok := creds.Get(name)
			if !ok {
				continue
			}
			pairs = append(pairs, CredentialMetadataPrefix+name, value)
		}
		return invoker(metadata.AppendToOutgoingContext(ctx, pairs...), method, req, reply, cc, opts...)
	}
}

// attachCredentialsFromMetadata copies credential entries from md onto ctx.
// md must already be the incoming copy. No credential key returns ctx unchanged
// and allocates nothing. The function does not log.
func attachCredentialsFromMetadata(ctx context.Context, md metadata.MD) context.Context {
	if !metadataHasCredential(md) {
		return ctx
	}
	creds, err := credentialsFromMetadata(md)
	return contextWithCredentialResult(ctx, creds, err)
}

func metadataHasCredential(md metadata.MD) bool {
	for key := range md {
		prefix := CredentialMetadataPrefix
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func credentialsFromMetadata(md metadata.MD) (Credentials, error) {
	pairs := make(map[string]string)
	for key, values := range md {
		prefixLen := len(CredentialMetadataPrefix)
		if len(key) <= prefixLen || key[:prefixLen] != CredentialMetadataPrefix {
			continue
		}
		if len(values) != 1 {
			return Credentials{}, ErrMalformedCredentials
		}
		name := key[prefixLen:]
		if _, exists := pairs[name]; exists {
			return Credentials{}, ErrMalformedCredentials
		}
		pairs[name] = values[0]
	}
	return credentialsFromPairs(pairs)
}

func contextWithHTTPCredentials(ctx context.Context, header http.Header) (context.Context, bool) {
	if !headerHasCredential(header) {
		return ctx, false
	}
	creds, err := credentialsFromHTTPHeader(header)
	return contextWithCredentialResult(ctx, creds, err), true
}

func headerHasCredential(header http.Header) bool {
	for key := range header {
		if len(key) > len(credentialHTTPPrefix) && key[:len(credentialHTTPPrefix)] == credentialHTTPPrefix {
			return true
		}
	}
	return false
}

func credentialsFromHTTPHeader(header http.Header) (Credentials, error) {
	pairs := make(map[string]string)
	for key, values := range header {
		if len(key) <= len(credentialHTTPPrefix) || key[:len(credentialHTTPPrefix)] != credentialHTTPPrefix {
			continue
		}
		if len(values) != 1 {
			return Credentials{}, ErrMalformedCredentials
		}
		name, ok := canonicalCredentialName(key[len(credentialHTTPPrefix):])
		if !ok {
			return Credentials{}, ErrMalformedCredentials
		}
		if _, exists := pairs[name]; exists {
			return Credentials{}, ErrMalformedCredentials
		}
		pairs[name] = values[0]
	}
	return credentialsFromPairs(pairs)
}

func credentialsFromPairs(pairs map[string]string) (Credentials, error) {
	creds, err := NewCredentials(pairs)
	if err != nil {
		return Credentials{}, ErrMalformedCredentials
	}
	return creds, nil
}

func contextWithCredentialResult(ctx context.Context, creds Credentials, err error) context.Context {
	if err != nil {
		return context.WithValue(ctx, credentialErrCtxKey{}, ErrMalformedCredentials)
	}
	return WithCredentials(ctx, creds)
}

func applyCredentialHTTPHeaders(ctx context.Context, header http.Header) {
	if header == nil {
		return
	}
	creds, err := ExtractCredentials(ctx)
	if err != nil || creds.Len() == 0 {
		return
	}
	for _, name := range creds.Names() {
		value, ok := creds.Get(name)
		if !ok {
			continue
		}
		header.Set(credentialHTTPPrefix+name, value)
	}
}

func credentialContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ctx, ok := contextWithHTTPCredentials(r.Context(), r.Header); ok {
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

func canonicalCredentialName(name string) (string, bool) {
	if name == "" || len(name) > MaxCredentialNameLen {
		return "", false
	}
	buf := make([]byte, len(name))
	for i := range name {
		c := name[i]
		switch {
		case c >= 'A' && c <= 'Z':
			buf[i] = c + ('a' - 'A')
		case (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-':
			buf[i] = c
		default:
			return "", false
		}
	}
	if buf[0] < 'a' || buf[0] > 'z' {
		return "", false
	}
	return string(buf), true
}

func validCredentialValue(value string) bool {
	if value == "" || len(value) > MaxCredentialValueLen {
		return false
	}
	for i := range value {
		if value[i] < 0x20 || value[i] > 0x7E {
			return false
		}
	}
	return true
}
