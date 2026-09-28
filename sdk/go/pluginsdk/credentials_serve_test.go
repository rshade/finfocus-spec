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
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const credentialCallTimeout = 3 * time.Second

type credentialSeen struct {
	value  string
	absent bool
	err    error
}

type credentialPlugin struct {
	*pluginsdk.BasePlugin

	ch        chan credentialSeen
	unrelated bool
}

func (p *credentialPlugin) GetActualCost(
	ctx context.Context,
	_ *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	creds, err := pluginsdk.ExtractCredentials(ctx)
	seen := credentialSeen{err: err}
	if err == nil && creds.Len() == 0 {
		seen.absent = true
	}
	if err == nil {
		seen.value, _ = creds.Get("token")
	}
	p.ch <- seen
	if err != nil {
		return nil, err
	}
	if p.unrelated {
		return nil, errors.New("cost unavailable")
	}
	return &pbc.GetActualCostResponse{}, nil
}

type optInCredentialPlugin struct {
	*credentialPlugin
}

func (p *optInCredentialPlugin) ConsumesPerRequestCredentials() {}

type plainInfoProvider struct {
	*pluginsdk.BasePlugin

	metadata map[string]string
}

func (p *plainInfoProvider) GetPluginInfo(
	context.Context,
	*pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	return &pbc.GetPluginInfoResponse{
		Name:        p.Name(),
		Version:     "v1.0.0",
		SpecVersion: pluginsdk.SpecVersion,
		Metadata:    p.metadata,
	}, nil
}

type optInInfoProvider struct {
	*optInCredentialPlugin

	metadata map[string]string
}

func (p *optInInfoProvider) GetPluginInfo(
	context.Context,
	*pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	return &pbc.GetPluginInfoResponse{
		Name:        p.Name(),
		Version:     "v1.0.0",
		SpecVersion: pluginsdk.SpecVersion,
		Metadata:    p.metadata,
	}, nil
}

func startCredentialServer(
	t *testing.T,
	plugin pluginsdk.Plugin,
	web bool,
	info *pluginsdk.PluginInfo,
	extra []grpc.UnaryServerInterceptor,
	logger *zerolog.Logger,
) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	cfg := pluginsdk.ServeConfig{
		Plugin:            plugin,
		Listener:          listener,
		Web:               pluginsdk.WebConfig{Enabled: web},
		PluginInfo:        info,
		UnaryInterceptors: extra,
		Logger:            logger,
	}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- pluginsdk.Serve(ctx, cfg) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(credentialCallTimeout):
			t.Error("server did not shut down")
		}
	})
	return addr
}

func dialCredentialGRPC(t *testing.T, addr string) pbc.CostSourceServiceClient {
	t.Helper()
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(pluginsdk.CredentialUnaryClientInterceptor()),
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return pbc.NewCostSourceServiceClient(conn)
}

func callActual(
	ctx context.Context,
	t *testing.T,
	client pbc.CostSourceServiceClient,
) error {
	t.Helper()
	deadline := time.Now().Add(credentialCallTimeout)
	var err error
	for time.Now().Before(deadline) {
		callCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		_, err = client.GetActualCost(callCtx, &pbc.GetActualCostRequest{ResourceId: "res-1"})
		cancel()
		if status.Code(err) == codes.Unavailable {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		return err
	}
	return err
}

func callPluginInfo(t *testing.T, client pbc.CostSourceServiceClient) (*pbc.GetPluginInfoResponse, error) {
	t.Helper()
	deadline := time.Now().Add(credentialCallTimeout)
	var (
		resp *pbc.GetPluginInfoResponse
		err  error
	)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		resp, err = client.GetPluginInfo(ctx, &pbc.GetPluginInfoRequest{})
		cancel()
		if status.Code(err) == codes.Unavailable {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		return resp, err
	}
	return resp, err
}

func credentialSet(t *testing.T) pluginsdk.Credentials {
	t.Helper()
	creds, err := pluginsdk.NewCredentials(map[string]string{"token": credentialFixture})
	if err != nil {
		t.Fatalf("NewCredentials: %v", err)
	}
	return creds
}

func requireNoFixture(t *testing.T, label, text string) {
	t.Helper()
	if strings.Contains(text, credentialFixture) {
		t.Fatalf("%s contains fixture", label)
	}
}

func TestPerRequestCredentialsNativeRoundTrip(t *testing.T) {
	plugin := &optInCredentialPlugin{credentialPlugin: &credentialPlugin{
		BasePlugin: pluginsdk.NewBasePlugin("opt-in"),
		ch:         make(chan credentialSeen, 2),
	}}
	addr := startCredentialServer(t, plugin, false, pluginsdk.NewPluginInfo("opt-in", "v1.0.0"), nil, nil)
	client := dialCredentialGRPC(t, addr)

	ctx := pluginsdk.WithCredentials(context.Background(), credentialSet(t))
	if err := callActual(ctx, t, client); err != nil {
		t.Fatalf("first call: %v", err)
	}
	first := <-plugin.ch
	if first.err != nil || first.value != credentialFixture {
		t.Fatalf("first seen = %+v", first)
	}

	if err := callActual(context.Background(), t, client); err != nil {
		t.Fatalf("second call: %v", err)
	}
	second := <-plugin.ch
	if second.err != nil || !second.absent || second.value != "" {
		t.Fatalf("second seen = %+v", second)
	}
}

func TestPerRequestCredentialsDefaultUnchanged(t *testing.T) {
	plugin := pluginsdk.NewBasePlugin("plain")
	info := pluginsdk.NewPluginInfo("plain", "v1.0.0",
		pluginsdk.WithMetadata(pluginsdk.MetadataSupportsPerRequestCredentials, pluginsdk.ValueTrue),
	)
	addr := startCredentialServer(t, plugin, false, info, nil, nil)
	client := dialCredentialGRPC(t, addr)

	before := append([]string(nil), os.Environ()...)
	slices.Sort(before)
	err := callActual(pluginsdk.WithCredentials(context.Background(), credentialSet(t)), t, client)
	if err == nil {
		t.Fatal("expected the default actual-cost error, got nil")
	}
	requireNoFixture(t, "default error", err.Error())
	after := append([]string(nil), os.Environ()...)
	slices.Sort(after)
	if !slices.Equal(before, after) {
		t.Fatal("process environment changed")
	}

	infoResp, err := callPluginInfo(t, client)
	if err != nil {
		t.Fatalf("GetPluginInfo: %v", err)
	}
	if _, ok := infoResp.GetMetadata()[pluginsdk.MetadataSupportsPerRequestCredentials]; ok {
		t.Fatal("non-opt-in plugin advertised acceptance")
	}

	opt := &optInCredentialPlugin{credentialPlugin: &credentialPlugin{
		BasePlugin: pluginsdk.NewBasePlugin("opt-absent"),
		ch:         make(chan credentialSeen, 1),
	}}
	optAddr := startCredentialServer(t, opt, false, pluginsdk.NewPluginInfo("opt-absent", "v1.0.0"), nil, nil)
	optClient := dialCredentialGRPC(t, optAddr)
	if callErr := callActual(context.Background(), t, optClient); callErr != nil {
		t.Fatalf("absent call: %v", callErr)
	}
	seen := <-opt.ch
	if seen.err != nil || !seen.absent {
		t.Fatalf("absent seen = %+v", seen)
	}
}

func TestPerRequestCredentialsRedaction(t *testing.T) {
	var logs bytes.Buffer
	logger := zerolog.New(&logs)
	plugin := &optInCredentialPlugin{credentialPlugin: &credentialPlugin{
		BasePlugin: pluginsdk.NewBasePlugin("redact"),
		ch:         make(chan credentialSeen, 3),
		unrelated:  false,
	}}
	metrics := pluginsdk.NewPluginMetrics("redact-plugin")
	addr := startCredentialServer(
		t,
		plugin,
		false,
		pluginsdk.NewPluginInfo("redact", "v1.0.0"),
		[]grpc.UnaryServerInterceptor{pluginsdk.MetricsInterceptorWithRegistry(metrics)},
		&logger,
	)
	client := dialCredentialGRPC(t, addr)
	creds := credentialSet(t)

	if err := callActual(pluginsdk.WithCredentials(context.Background(), creds), t, client); err != nil {
		t.Fatalf("success call: %v", err)
	}
	<-plugin.ch

	plugin.unrelated = true
	err := callActual(pluginsdk.WithCredentials(context.Background(), creds), t, client)
	if err == nil || !strings.Contains(err.Error(), "cost unavailable") {
		t.Fatalf("unrelated error = %v", err)
	}
	requireNoFixture(t, "unrelated error", err.Error())
	<-plugin.ch

	malformedCtx := metadata.AppendToOutgoingContext(
		context.Background(),
		pluginsdk.CredentialMetadataPrefix+"token", credentialFixture,
		pluginsdk.CredentialMetadataPrefix+"token", credentialFixture,
	)
	err = callActual(malformedCtx, t, client)
	malformedText := pluginsdk.ErrMalformedCredentials.Error()
	malformed := err != nil &&
		(errors.Is(err, pluginsdk.ErrMalformedCredentials) || strings.Contains(err.Error(), malformedText))
	if !malformed {
		t.Fatalf("malformed error = %v", err)
	}
	requireNoFixture(t, "malformed error", err.Error())
	seen := <-plugin.ch
	if !errors.Is(seen.err, pluginsdk.ErrMalformedCredentials) {
		t.Fatalf("handler malformed = %v", seen.err)
	}
	requireNoFixture(t, "handler error", seen.err.Error())
	requireNoFixture(t, "logs", logs.String())

	families, gatherErr := metrics.Registry.Gather()
	if gatherErr != nil {
		t.Fatalf("gather: %v", gatherErr)
	}
	for _, family := range families {
		requireNoFixture(t, "metric name", family.GetName())
		requireNoFixture(t, "metric help", family.GetHelp())
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				requireNoFixture(t, "metric label", label.GetName()+"="+label.GetValue())
			}
		}
	}
}

func TestPerRequestCredentialsPluginInfo(t *testing.T) {
	plain := pluginsdk.NewBasePlugin("plain-info")
	plainAddr := startCredentialServer(t, plain, false, pluginsdk.NewPluginInfo("plain-info", "v1.0.0"), nil, nil)
	plainClient := dialCredentialGRPC(t, plainAddr)
	plainInfo, err := callPluginInfo(t, plainClient)
	if err != nil {
		t.Fatalf("plain info: %v", err)
	}

	opt := &optInCredentialPlugin{credentialPlugin: &credentialPlugin{
		BasePlugin: pluginsdk.NewBasePlugin("opt-info"),
		ch:         make(chan credentialSeen, 1),
	}}
	optAddr := startCredentialServer(t, opt, false, pluginsdk.NewPluginInfo("opt-info", "v1.0.0"), nil, nil)
	optClient := dialCredentialGRPC(t, optAddr)
	optInfo, err := callPluginInfo(t, optClient)
	if err != nil {
		t.Fatalf("opt info: %v", err)
	}
	if optInfo.GetMetadata()[pluginsdk.MetadataSupportsPerRequestCredentials] != pluginsdk.ValueTrue {
		t.Fatalf("opt-in metadata = %v", optInfo.GetMetadata())
	}
	if len(plainInfo.GetCapabilities()) == 0 || len(plainInfo.GetCapabilities()) != len(optInfo.GetCapabilities()) {
		t.Fatalf("capabilities plain=%v opt=%v", plainInfo.GetCapabilities(), optInfo.GetCapabilities())
	}
	for i := range plainInfo.GetCapabilities() {
		if plainInfo.GetCapabilities()[i] != optInfo.GetCapabilities()[i] {
			t.Fatalf("capability %d differs", i)
		}
	}

	denier := &optInInfoProvider{
		optInCredentialPlugin: &optInCredentialPlugin{credentialPlugin: &credentialPlugin{
			BasePlugin: pluginsdk.NewBasePlugin("denier"),
			ch:         make(chan credentialSeen, 1),
		}},
		metadata: map[string]string{
			pluginsdk.MetadataSupportsPerRequestCredentials: "false",
		},
	}
	denierAddr := startCredentialServer(t, denier, false, nil, nil, nil)
	denierInfo, err := callPluginInfo(t, dialCredentialGRPC(t, denierAddr))
	if err != nil {
		t.Fatalf("denier info: %v", err)
	}
	if denierInfo.GetMetadata()[pluginsdk.MetadataSupportsPerRequestCredentials] != pluginsdk.ValueTrue {
		t.Fatalf("provider false stuck: %v", denierInfo.GetMetadata())
	}

	claimer := &plainInfoProvider{
		BasePlugin: pluginsdk.NewBasePlugin("claimer"),
		metadata: map[string]string{
			pluginsdk.MetadataSupportsPerRequestCredentials: pluginsdk.ValueTrue,
		},
	}
	claimerAddr := startCredentialServer(t, claimer, false, nil, nil, nil)
	claimerInfo, err := callPluginInfo(t, dialCredentialGRPC(t, claimerAddr))
	if err != nil {
		t.Fatalf("claimer info: %v", err)
	}
	if _, ok := claimerInfo.GetMetadata()[pluginsdk.MetadataSupportsPerRequestCredentials]; ok {
		t.Fatal("hand-written acceptance survived")
	}
}

func TestPerRequestCredentialsConnectRoundTrip(t *testing.T) {
	if strings.Contains(strings.ToLower(pluginsdk.DefaultAllowedHeaders), "finfocus-credential") {
		t.Fatal("default CORS allow-list includes credential headers")
	}
	plugin := &optInCredentialPlugin{credentialPlugin: &credentialPlugin{
		BasePlugin: pluginsdk.NewBasePlugin("connect-opt"),
		ch:         make(chan credentialSeen, 2),
	}}
	addr := startCredentialServer(t, plugin, true, pluginsdk.NewPluginInfo("connect-opt", "v1.0.0"), nil, nil)
	client := pluginsdk.NewConnectClient("http://" + addr)
	t.Cleanup(client.Close)

	ctx := pluginsdk.WithCredentials(context.Background(), credentialSet(t))
	var err error
	deadline := time.Now().Add(credentialCallTimeout)
	for time.Now().Before(deadline) {
		callCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		_, err = client.GetActualCost(callCtx, &pbc.GetActualCostRequest{ResourceId: "res-1"})
		cancel()
		if err != nil && strings.Contains(err.Error(), "connection refused") {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		break
	}
	if err != nil {
		t.Fatalf("connect call: %v", err)
	}
	first := <-plugin.ch
	if first.err != nil || first.value != credentialFixture {
		t.Fatalf("connect seen = %+v", first)
	}

	if _, err = client.GetActualCost(context.Background(), &pbc.GetActualCostRequest{ResourceId: "res-1"}); err != nil {
		t.Fatalf("connect absent: %v", err)
	}
	second := <-plugin.ch
	if second.err != nil || !second.absent {
		t.Fatalf("connect absent seen = %+v", second)
	}
}
