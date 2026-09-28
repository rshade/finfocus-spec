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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"regexp"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	allocatorScenarioTimeout = 10 * time.Second
	conformanceUnknownField  = "conformance_unknown_field"
	conformanceBadVersion    = math.MaxInt32
	fixtureCurrency          = "USD"
	resourceKindCluster      = "cluster"
	subjectNamespace         = "namespace"
	subjectControllerKind    = "controller_kind"
	subjectController        = "controller"
	metricCPURequest         = "cpu_request"
	metricMemRequest         = "mem_request"
	metricCPUAllocatable     = "cpu_allocatable"
	metricMemAllocatable     = "mem_allocatable"
	unitCore                 = "core"
	unitGiB                  = "GiB"
	fixtureNamespacePayments = "payments"
	fixtureNamespaceWeb      = "web"
	fixturePodAPI            = "api"
	fixturePodFrontend       = "frontend"
	fixtureControlPlaneCost  = 3.0
)

// policyDigestPattern matches a lowercase hex SHA-256 digest.
var policyDigestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// AllocateServer is satisfied by any type with an Allocate method, including
// pbc.AllocatorServiceServer implementations and pluginsdk.AllocatorProvider.
type AllocateServer interface {
	Allocate(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error)
}

type allocateAdapter struct {
	pbc.UnimplementedAllocatorServiceServer

	impl AllocateServer
}

func (a *allocateAdapter) Allocate(ctx context.Context, req *pbc.AllocateRequest) (*pbc.AllocateResponse, error) {
	return a.impl.Allocate(ctx, req)
}

// AllocatorHarness serves an AllocateServer over an in-memory bufconn, so
// calls exercise proto serialization and the status codes clients really see.
type AllocatorHarness struct {
	server   *grpc.Server
	listener *bufconn.Listener
	client   pbc.AllocatorServiceClient
	conn     *grpc.ClientConn
}

// NewAllocatorHarness creates a harness serving impl as AllocatorService.
func NewAllocatorHarness(impl AllocateServer) *AllocatorHarness {
	listener := bufconn.Listen(bufSize)
	server := grpc.NewServer()
	pbc.RegisterAllocatorServiceServer(server, &allocateAdapter{impl: impl})

	go func() {
		_ = server.Serve(listener)
	}()

	return &AllocatorHarness{
		server:   server,
		listener: listener,
	}
}

// Start initializes the client connection to the in-memory server.
func (h *AllocatorHarness) Start(t testing.TB) {
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
	h.client = pbc.NewAllocatorServiceClient(conn)
}

// Stop closes the client connection and stops the server. It is safe to call
// more than once.
func (h *AllocatorHarness) Stop() {
	if h.conn != nil {
		_ = h.conn.Close()
	}
	if h.server != nil {
		h.server.Stop()
	}
}

// Client returns the AllocatorService client; call Start first.
func (h *AllocatorHarness) Client() pbc.AllocatorServiceClient {
	return h.client
}

// fixtureWorkload is one workload's requests on a fixture node.
type fixtureWorkload struct {
	namespace, pod string
	cpu, mem       float64
}

// fixtureNode is a node with its capacity, price, and workloads.
type fixtureNode struct {
	id        string
	cost      float64
	priced    bool
	cpu, mem  float64
	workloads []fixtureWorkload
}

// allocFixture describes a cluster; controlPlane > 0 adds a priced
// tags.kind=cluster resource with that cost.
type allocFixture struct {
	nodes        []fixtureNode
	controlPlane float64
}

func (n fixtureNode) usageRows() []*pbc.UsageRow {
	node := map[string]string{subjectKind: kindNode, subjectNode: n.id}
	rows := []*pbc.UsageRow{
		{Subject: node, Metric: metricCPUAllocatable, Amount: n.cpu, Unit: unitCore},
		{Subject: node, Metric: metricMemAllocatable, Amount: n.mem, Unit: unitGiB},
	}
	for _, w := range n.workloads {
		subject := map[string]string{
			subjectKind:           kindWorkload,
			subjectNamespace:      w.namespace,
			subjectPod:            w.pod,
			subjectNode:           n.id,
			subjectControllerKind: "Deployment",
			subjectController:     w.pod + "-deploy",
		}
		rows = append(rows,
			&pbc.UsageRow{Subject: subject, Metric: metricCPURequest, Amount: w.cpu, Unit: unitCore},
			&pbc.UsageRow{Subject: subject, Metric: metricMemRequest, Amount: w.mem, Unit: unitGiB},
		)
	}
	return rows
}

// request builds the AllocateRequest, first checking that the usage is what a
// valid usage source would return, so a fixture bug is never blamed on the
// allocator.
func (f allocFixture) request() (*pbc.AllocateRequest, error) {
	stats := &pbc.GetStatsResponse{Mode: pbc.StatsMode_STATS_MODE_RUN_RATE}
	req := &pbc.AllocateRequest{Mode: pbc.StatsMode_STATS_MODE_RUN_RATE}
	for _, n := range f.nodes {
		stats.Rows = append(stats.Rows, n.usageRows()...)
		resource := &pbc.ResourceDescriptor{
			Id:           n.id,
			Provider:     providerKubernetes,
			ResourceType: kindNode,
			Tags:         map[string]string{subjectKind: kindNode},
		}
		stats.Priceable = append(stats.Priceable, resource)
		entry := &pbc.PricedResource{Resource: resource, Priced: n.priced}
		if n.priced {
			entry.Cost, entry.Currency = n.cost, fixtureCurrency
		} else {
			entry.Note = "no price available for this node"
		}
		req.Priced = append(req.Priced, entry)
	}
	if f.controlPlane > 0 {
		resource := &pbc.ResourceDescriptor{
			Id: "control-plane", Provider: providerKubernetes, ResourceType: resourceKindCluster,
			Tags: map[string]string{subjectKind: resourceKindCluster},
		}
		stats.Priceable = append(stats.Priceable, resource)
		req.Priced = append(req.Priced,
			&pbc.PricedResource{Resource: resource, Cost: f.controlPlane, Currency: fixtureCurrency, Priced: true})
	}
	if err := ValidateStatsResponse(stats); err != nil {
		return nil, fmt.Errorf("conformance fixture usage is invalid (suite bug): %w", err)
	}
	req.Usage = stats.GetRows()
	if err := ValidateAllocateRequest(req); err != nil {
		return nil, fmt.Errorf("conformance fixture request is invalid (suite bug): %w", err)
	}
	return req, nil
}

// allocateAndVerify runs the common assertions: the call succeeds, the
// response passes ValidateAllocateResponse, and conservation holds. Both
// checks always run so a structural failure (such as a dropped idle row)
// still reports the cost shortfall it causes.
func allocateAndVerify(
	ctx context.Context, client pbc.AllocatorServiceClient, f allocFixture,
) (*pbc.AllocateResponse, error) {
	req, err := f.request()
	if err != nil {
		return nil, err
	}
	resp, err := client.Allocate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Allocate failed: %w", err)
	}
	if err = errors.Join(
		ValidateAllocateResponse(req, resp),
		CheckConservation(req, resp, DefaultConservationEpsilon),
	); err != nil {
		return nil, err
	}
	return resp, nil
}

func rowsWithKind(resp *pbc.AllocateResponse, kind string) []*pbc.AllocationRow {
	var rows []*pbc.AllocationRow
	for _, row := range resp.GetRows() {
		if row.GetSubject()[subjectKind] == kind {
			rows = append(rows, row)
		}
	}
	return rows
}

// Conformance fixtures. Nodes are listed as {id, cost USD, priced, allocatable
// cores, allocatable GiB, workloads}; workloads as {namespace, pod, requested
// cores, requested GiB}. The numbers are data, not tuning: any allocator must
// satisfy the invariants for them.

func singleNodeFixture() allocFixture {
	return allocFixture{nodes: []fixtureNode{
		{"n1", 10, true, 4, 16, []fixtureWorkload{
			{fixtureNamespacePayments, fixturePodAPI, 1, 4},
			{fixtureNamespaceWeb, fixturePodFrontend, 0.5, 2},
		}},
	}}
}

func threeNodeFixture(controlPlane float64) allocFixture {
	return allocFixture{controlPlane: controlPlane, nodes: []fixtureNode{
		{"n1", 10, true, 4, 16, []fixtureWorkload{
			{fixtureNamespacePayments, fixturePodAPI + "-n1", 1, 2},
			{fixtureNamespaceWeb, fixturePodFrontend + "-n1", 0.25, 1},
		}},
		{"n2", 6, true, 4, 16, []fixtureWorkload{
			{fixtureNamespacePayments, fixturePodAPI + "-n2", 1, 2},
			{fixtureNamespaceWeb, fixturePodFrontend + "-n2", 0.25, 1},
		}},
		{"n3", 4, true, 4, 16, []fixtureWorkload{
			{fixtureNamespacePayments, fixturePodAPI + "-n3", 1, 2},
			{fixtureNamespaceWeb, fixturePodFrontend + "-n3", 0.25, 1},
		}},
	}}
}

func emptyClusterFixture() allocFixture {
	return allocFixture{nodes: []fixtureNode{
		{"n1", 10, true, 4, 16, nil},
		{"n2", 6, true, 2, 8, nil},
	}}
}

// fullyPackedFixture requests exactly the node's allocatable capacity.
func fullyPackedFixture() allocFixture {
	return allocFixture{nodes: []fixtureNode{
		{"n1", 10, true, 4, 16, []fixtureWorkload{
			{fixtureNamespacePayments, fixturePodAPI, 2, 8},
			{fixtureNamespaceWeb, fixturePodFrontend, 2, 8},
		}},
	}}
}

func unpricedNodeFixture() allocFixture {
	return allocFixture{nodes: []fixtureNode{
		{"n1", 10, true, 4, 16, []fixtureWorkload{{fixtureNamespacePayments, fixturePodAPI, 1, 4}}},
		{"n2", 0, false, 4, 16, []fixtureWorkload{{fixtureNamespaceWeb, fixturePodFrontend, 1, 4}}},
	}}
}

// overRequestedFixture requests more CPU and memory than the node can allocate.
func overRequestedFixture() allocFixture {
	return allocFixture{nodes: []fixtureNode{
		{"n1", 10, true, 4, 16, []fixtureWorkload{
			{fixtureNamespacePayments, fixturePodAPI, 3, 8},
			{fixtureNamespaceWeb, fixturePodFrontend, 3, 12},
		}},
	}}
}

// pricedNodeTotal is the summed cost of the fixture's priced nodes.
func (f allocFixture) pricedNodeTotal() float64 {
	total := 0.0
	for _, n := range f.nodes {
		if n.priced {
			total += n.cost
		}
	}
	return total
}

func runVerified(f allocFixture) func(ctx context.Context, client pbc.AllocatorServiceClient) error {
	return func(ctx context.Context, client pbc.AllocatorServiceClient) error {
		_, err := allocateAndVerify(ctx, client, f)
		return err
	}
}

func scenarioEmptyCluster(ctx context.Context, client pbc.AllocatorServiceClient) error {
	f := emptyClusterFixture()
	resp, err := allocateAndVerify(ctx, client, f)
	if err != nil {
		return err
	}
	idle := 0.0
	for _, row := range rowsWithKind(resp, kindIdle) {
		idle += row.GetTotalCost()
	}
	expected := f.pricedNodeTotal()
	if math.Abs(idle-expected) > allocationTolerance(expected, DefaultConservationEpsilon) {
		return fmt.Errorf("%q rows total %g, want %g: with no workloads all node cost is idle",
			kindIdle, idle, expected)
	}
	return nil
}

func scenarioFullyPacked(ctx context.Context, client pbc.AllocatorServiceClient) error {
	resp, err := allocateAndVerify(ctx, client, fullyPackedFixture())
	if err != nil {
		return err
	}
	if len(rowsWithKind(resp, kindIdle)) != 1 {
		return fmt.Errorf("fully packed node n1 needs its %q row even at zero cost", kindIdle)
	}
	return nil
}

func scenarioControlPlane(ctx context.Context, client pbc.AllocatorServiceClient) error {
	resp, err := allocateAndVerify(ctx, client, threeNodeFixture(fixtureControlPlaneCost))
	if err != nil {
		return err
	}
	if len(rowsWithKind(resp, kindCluster)) == 0 {
		return fmt.Errorf("priced control plane produced no %q row", kindCluster)
	}
	return nil
}

func scenarioOverRequested(ctx context.Context, client pbc.AllocatorServiceClient) error {
	resp, err := allocateAndVerify(ctx, client, overRequestedFixture())
	if err != nil {
		return err
	}
	idle := rowsWithKind(resp, kindIdle)
	if len(idle) != 1 {
		return fmt.Errorf("over-requested node n1 needs exactly one %q row, got %d", kindIdle, len(idle))
	}
	if idle[0].GetTotalCost() < 0 || idle[0].GetCpuCost() < 0 || idle[0].GetMemCost() < 0 {
		return fmt.Errorf("over-requested node n1 has negative idle cost %v", idle[0].GetTotalCost())
	}
	return nil
}

// effectivePolicy calls Allocate with an empty request and returns the
// allocator's effective policy as a JSON object whose "version" is an integer.
func effectivePolicy(ctx context.Context, client pbc.AllocatorServiceClient) (map[string]any, error) {
	resp, err := client.Allocate(ctx, &pbc.AllocateRequest{})
	if err != nil {
		return nil, fmt.Errorf("Allocate with an empty request failed: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(resp.GetEffectivePolicyJson()))
	dec.UseNumber()
	var policy map[string]any
	if err = dec.Decode(&policy); err != nil || policy == nil {
		return nil, fmt.Errorf("effective_policy_json %q is not a JSON object", resp.GetEffectivePolicyJson())
	}
	version, ok := policy["version"].(json.Number)
	if !ok {
		return nil, fmt.Errorf(
			"effective policy has no numeric top-level \"version\": %s",
			resp.GetEffectivePolicyJson(),
		)
	}
	if _, err = version.Int64(); err != nil {
		return nil, fmt.Errorf("effective policy \"version\" %s is not an integer", version)
	}
	return policy, nil
}

// expectPolicyRejected sends policy and requires InvalidArgument whose
// message contains want (when non-empty).
func expectPolicyRejected(
	ctx context.Context, client pbc.AllocatorServiceClient, policy map[string]any, want string,
) error {
	doc, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("encode policy (suite bug): %w", err)
	}
	_, err = client.Allocate(ctx, &pbc.AllocateRequest{PolicyJson: doc})
	if err == nil {
		return fmt.Errorf("policy %s was accepted; an allocator must reject it, never fall back to defaults", doc)
	}
	st := status.Convert(err)
	if st.Code() != codes.InvalidArgument {
		return fmt.Errorf("policy %s: got %s, want %s", doc, st.Code(), codes.InvalidArgument)
	}
	if want != "" && !strings.Contains(st.Message(), want) {
		return fmt.Errorf("policy %s: error %q does not name %q", doc, st.Message(), want)
	}
	return nil
}

func clonePolicy(policy map[string]any) map[string]any {
	out := make(map[string]any, len(policy)+1)
	for k, v := range policy {
		out[k] = v
	}
	return out
}

func scenarioPolicyUnknownField(ctx context.Context, client pbc.AllocatorServiceClient) error {
	policy, err := effectivePolicy(ctx, client)
	if err != nil {
		return err
	}
	// Only the top level is probed: it is always the allocator's struct, while a
	// nested object may be a map field that legitimately accepts any key.
	top := clonePolicy(policy)
	top[conformanceUnknownField] = true
	return expectPolicyRejected(ctx, client, top, conformanceUnknownField)
}

func scenarioPolicyUnknownVersion(ctx context.Context, client pbc.AllocatorServiceClient) error {
	policy, err := effectivePolicy(ctx, client)
	if err != nil {
		return err
	}
	bad := clonePolicy(policy)
	bad["version"] = conformanceBadVersion
	return expectPolicyRejected(ctx, client, bad, "")
}

func scenarioEmptyRequest(ctx context.Context, client pbc.AllocatorServiceClient) error {
	resp, err := client.Allocate(ctx, &pbc.AllocateRequest{})
	if err != nil {
		return fmt.Errorf("Allocate with an empty request failed: %w", err)
	}
	if len(resp.GetRows()) != 0 {
		return fmt.Errorf("empty request produced %d rows, want 0", len(resp.GetRows()))
	}
	if !policyDigestPattern.MatchString(resp.GetPolicyDigest()) {
		return fmt.Errorf("policy_digest %q is not 64 lowercase hex characters", resp.GetPolicyDigest())
	}
	_, err = effectivePolicy(ctx, client)
	return err
}

func scenarioFingerprintStable(ctx context.Context, client pbc.AllocatorServiceClient) error {
	first, err := allocateAndVerify(ctx, client, singleNodeFixture())
	if err != nil {
		return err
	}
	second, err := allocateAndVerify(ctx, client, singleNodeFixture())
	if err != nil {
		return err
	}
	if first.GetPolicyDigest() != second.GetPolicyDigest() {
		return fmt.Errorf("policy_digest changed between identical calls: %q then %q",
			first.GetPolicyDigest(), second.GetPolicyDigest())
	}
	if !bytes.Equal(first.GetEffectivePolicyJson(), second.GetEffectivePolicyJson()) {
		return fmt.Errorf("effective_policy_json changed between identical calls: %s then %s",
			first.GetEffectivePolicyJson(), second.GetEffectivePolicyJson())
	}
	return nil
}

func scenarioEmptyEqualsBraces(ctx context.Context, client pbc.AllocatorServiceClient) error {
	empty, err := client.Allocate(ctx, &pbc.AllocateRequest{})
	if err != nil {
		return fmt.Errorf("Allocate with empty policy_json failed: %w", err)
	}
	braces, err := client.Allocate(ctx, &pbc.AllocateRequest{PolicyJson: []byte("{}")})
	if err != nil {
		return fmt.Errorf("Allocate with policy_json {} failed: %w", err)
	}
	if empty.GetPolicyDigest() != braces.GetPolicyDigest() {
		return fmt.Errorf("empty policy_json digest %q differs from {} digest %q",
			empty.GetPolicyDigest(), braces.GetPolicyDigest())
	}
	if !bytes.Equal(empty.GetEffectivePolicyJson(), braces.GetEffectivePolicyJson()) {
		return fmt.Errorf("empty policy_json effective policy %s differs from {} effective policy %s",
			empty.GetEffectivePolicyJson(), braces.GetEffectivePolicyJson())
	}
	return nil
}

type allocatorScenario struct {
	name string
	run  func(ctx context.Context, client pbc.AllocatorServiceClient) error
}

// allocatorScenarios lists the conformance scenarios in suite order.
func allocatorScenarios() []allocatorScenario {
	return []allocatorScenario{
		{name: "single_node", run: runVerified(singleNodeFixture())},
		{name: "three_nodes", run: runVerified(threeNodeFixture(0))},
		{name: "empty_cluster", run: scenarioEmptyCluster},
		{name: "fully_packed_node", run: scenarioFullyPacked},
		{name: "unpriced_node", run: runVerified(unpricedNodeFixture())},
		{name: "control_plane", run: scenarioControlPlane},
		{name: "over_requested_node", run: scenarioOverRequested},
		{name: "policy_unknown_field", run: scenarioPolicyUnknownField},
		{name: "policy_unknown_version", run: scenarioPolicyUnknownVersion},
		{name: "empty_request", run: scenarioEmptyRequest},
		{name: "fingerprint_stable", run: scenarioFingerprintStable},
		{name: "fingerprint_empty_equals_braces", run: scenarioEmptyEqualsBraces},
	}
}

func runAllocatorScenario(ctx context.Context, client pbc.AllocatorServiceClient, s allocatorScenario) error {
	ctx, cancel := context.WithTimeout(ctx, allocatorScenarioTimeout)
	defer cancel()
	err := s.run(ctx, client)
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("scenario timed out after %s: %w", allocatorScenarioTimeout, err)
	}
	return err
}

// runAllocatorScenarios runs every scenario against client and returns each
// scenario's error (nil on success) keyed by subtest name.
func runAllocatorScenarios(ctx context.Context, client pbc.AllocatorServiceClient) map[string]error {
	scenarios := allocatorScenarios()
	results := make(map[string]error, len(scenarios))
	for _, s := range scenarios {
		results[s.name] = runAllocatorScenario(ctx, client, s)
	}
	return results
}

// RunAllocatorConformance serves impl over an AllocatorHarness and runs the
// standard allocator scenarios as subtests. Assertions are policy-agnostic:
// no allocator-specific values are checked, and bad policies are derived from
// the allocator's own effective policy.
//
// Every allocation scenario requires that the call succeeds, that the
// response passes ValidateAllocateResponse, and that CheckConservation holds
// at DefaultConservationEpsilon. The subtests are:
//
//   - single_node: one priced node with two workloads
//   - three_nodes: three priced nodes with workloads on each
//   - empty_cluster: two priced nodes and no workloads; idle rows carry all cost
//   - fully_packed_node: requests equal allocatable; the idle row is still present
//   - unpriced_node: one priced and one unpriced node; only the priced one needs idle
//   - control_plane: three nodes plus a priced control plane; at least one cluster row
//   - over_requested_node: requests exceed allocatable; idle is present and non-negative
//   - policy_unknown_field: an unknown top-level key is rejected with
//     InvalidArgument naming it
//   - policy_unknown_version: version 2147483647 is rejected with InvalidArgument
//   - empty_request: no rows; 64-character lowercase hex digest; the effective
//     policy is an object with an integer "version"
//   - fingerprint_stable: identical requests yield identical digests and policies
//   - fingerprint_empty_equals_braces: empty policy_json and "{}" yield one digest
func RunAllocatorConformance(t *testing.T, impl AllocateServer) {
	t.Helper()
	harness := NewAllocatorHarness(impl)
	harness.Start(t)
	defer harness.Stop()

	for _, s := range allocatorScenarios() {
		t.Run(s.name, func(t *testing.T) {
			if err := runAllocatorScenario(context.Background(), harness.Client(), s); err != nil {
				t.Error(err)
			}
		})
	}
}
