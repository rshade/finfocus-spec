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
	"math"
	"net"
	"sort"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// ErrInvalidStatsResponse is wrapped by every ValidateStatsResponse failure.
var ErrInvalidStatsResponse = errors.New("invalid GetStats response")

// Private copy of the pluginsdk Subject* and Kind* vocabulary: this package
// cannot import pluginsdk (pluginsdk/conformance.go imports it). The pluginsdk
// drift test compares the two through KnownSubjectKeys.
const (
	subjectKind  = "kind"
	subjectNode  = "node"
	subjectPod   = "pod"
	labelPrefix  = "label."
	kindWorkload = "workload"
	kindNode     = "node"
)

//nolint:gochecknoglobals // Zero-allocation lookup slices (registry pattern).
var (
	knownSubjectKeys = []string{
		"cluster",
		"namespace",
		"controller_kind",
		"controller",
		subjectPod,
		subjectNode,
		subjectKind,
	}
	validRowKinds = []string{kindWorkload, kindNode}
)

// KnownSubjectKeys returns a copy of the documented, non-label subject keys
// accepted by ValidateStatsResponse. It exists for the pluginsdk drift test.
func KnownSubjectKeys() []string {
	keys := make([]string, len(knownSubjectKeys))
	copy(keys, knownSubjectKeys)
	return keys
}

func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func isValidSubjectKey(key string) bool {
	if containsString(knownSubjectKeys, key) {
		return true
	}
	return strings.HasPrefix(key, labelPrefix) && len(key) > len(labelPrefix)
}

// rowIdentity returns a canonical key for a row's (subject, metric) pair.
func rowIdentity(row *pbc.UsageRow) string {
	subject := row.GetSubject()
	keys := make([]string, 0, len(subject))
	for k := range subject {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(subject[k])
		b.WriteByte(0)
	}
	b.WriteString(row.GetMetric())
	return b.String()
}

func validateUsageRow(i int, row *pbc.UsageRow) error {
	subject := row.GetSubject()
	kind, ok := subject[subjectKind]
	if !ok {
		return fmt.Errorf("%w: rows[%d]: subject missing required key %q", ErrInvalidStatsResponse, i, subjectKind)
	}
	if !containsString(validRowKinds, kind) {
		return fmt.Errorf("%w: rows[%d]: subject kind %q is not %q or %q",
			ErrInvalidStatsResponse, i, kind, kindWorkload, kindNode)
	}
	if kind == kindNode && subject[subjectNode] == "" {
		return fmt.Errorf("%w: rows[%d]: node row needs a non-empty %q subject key",
			ErrInvalidStatsResponse, i, subjectNode)
	}
	for key := range subject {
		if !isValidSubjectKey(key) {
			return fmt.Errorf("%w: rows[%d]: unknown subject key %q", ErrInvalidStatsResponse, i, key)
		}
	}
	amount := row.GetAmount()
	if amount < 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return fmt.Errorf("%w: rows[%d]: amount %v must be finite and non-negative", ErrInvalidStatsResponse, i, amount)
	}
	return nil
}

// ValidateStatsResponse returns nil if resp satisfies the usage-source
// contract, or the first violation found (rules V1–V10 in data-model.md).
// Every error wraps ErrInvalidStatsResponse and names the offending rows[i]
// or priceable[i] entry.
//
// Unit and metric consistency, and duplicate priceable IDs, are not checked.
func ValidateStatsResponse(resp *pbc.GetStatsResponse) error {
	if resp == nil {
		return fmt.Errorf("%w: response is nil", ErrInvalidStatsResponse)
	}
	if resp.GetMode() == pbc.StatsMode_STATS_MODE_UNSPECIFIED {
		return fmt.Errorf("%w: mode is %s", ErrInvalidStatsResponse, resp.GetMode())
	}

	seen := make(map[string]struct{}, len(resp.GetRows()))
	nodes := make(map[string]struct{})
	for i, row := range resp.GetRows() {
		if err := validateUsageRow(i, row); err != nil {
			return err
		}
		id := rowIdentity(row)
		if _, dup := seen[id]; dup {
			return fmt.Errorf("%w: rows[%d]: duplicate subject and metric %q",
				ErrInvalidStatsResponse, i, row.GetMetric())
		}
		seen[id] = struct{}{}
		if node := row.GetSubject()[subjectNode]; node != "" {
			nodes[node] = struct{}{}
		}
	}

	for i, entry := range resp.GetPriceable() {
		if entry == nil {
			return fmt.Errorf("%w: priceable[%d]: entry is nil", ErrInvalidStatsResponse, i)
		}
		if entry.GetId() == "" {
			return fmt.Errorf("%w: priceable[%d]: id is empty", ErrInvalidStatsResponse, i)
		}
		if entry.GetTags()[subjectKind] != kindNode {
			continue
		}
		if _, ok := nodes[entry.GetId()]; !ok {
			return fmt.Errorf("%w: priceable[%d]: node %q matches no row's %q subject",
				ErrInvalidStatsResponse, i, entry.GetId(), subjectNode)
		}
	}
	return nil
}

// UsageStatsServer is satisfied by any type with a GetStats method, including
// plugins implementing pluginsdk.UsageSourceProvider.
type UsageStatsServer interface {
	GetStats(ctx context.Context, req *pbc.GetStatsRequest) (*pbc.GetStatsResponse, error)
}

type usageStatsAdapter struct {
	pbc.UnimplementedUsageSourceServiceServer

	impl UsageStatsServer
}

func (a *usageStatsAdapter) GetStats(ctx context.Context, req *pbc.GetStatsRequest) (*pbc.GetStatsResponse, error) {
	return a.impl.GetStats(ctx, req)
}

// UsageSourceHarness serves a UsageStatsServer over an in-memory bufconn.
type UsageSourceHarness struct {
	server   *grpc.Server
	listener *bufconn.Listener
	client   pbc.UsageSourceServiceClient
	conn     *grpc.ClientConn
}

// NewUsageSourceHarness creates a harness serving impl as UsageSourceService.
func NewUsageSourceHarness(impl UsageStatsServer) *UsageSourceHarness {
	listener := bufconn.Listen(bufSize)
	server := grpc.NewServer()
	pbc.RegisterUsageSourceServiceServer(server, &usageStatsAdapter{impl: impl})

	go func() {
		_ = server.Serve(listener)
	}()

	return &UsageSourceHarness{
		server:   server,
		listener: listener,
	}
}

// Start initializes the client connection to the in-memory server.
func (h *UsageSourceHarness) Start(t testing.TB) {
	//nolint:staticcheck // grpc.NewClient doesn't work with bufconn
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return h.listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	h.conn = conn
	h.client = pbc.NewUsageSourceServiceClient(conn)
}

// Stop closes the client connection and stops the server.
func (h *UsageSourceHarness) Stop() {
	if h.conn != nil {
		_ = h.conn.Close()
	}
	if h.server != nil {
		h.server.Stop()
	}
}

// Client returns the UsageSourceService client; call Start first.
func (h *UsageSourceHarness) Client() pbc.UsageSourceServiceClient {
	return h.client
}
