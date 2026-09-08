//nolint:testpackage // Exercises unexported CLI routing and the runCLI test seam
package pluginsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	ax "github.com/rshade/ax-go"
	"github.com/rshade/ax-go/axtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

type cliDryRunPlugin struct {
	mockPlugin

	resp *pbc.DryRunResponse
	err  error
	got  *pbc.DryRunRequest
}

func (p *cliDryRunPlugin) HandleDryRun(_ context.Context, req *pbc.DryRunRequest) (*pbc.DryRunResponse, error) {
	p.got = req
	if p.err != nil {
		return nil, p.err
	}
	if p.resp != nil {
		return p.resp, nil
	}
	return NewDryRunResponse(
		WithResourceTypeSupported(true),
		WithConfigurationValid(true),
	), nil
}

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func testServeConfig(plugin Plugin, version string) ServeConfig {
	name := "test-plugin"
	if plugin != nil {
		name = plugin.Name()
	}
	return ServeConfig{
		Plugin:     plugin,
		PluginInfo: NewPluginInfo(name, version),
	}
}

func TestIsCLIInvocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "empty args is handshake", args: nil, want: false},
		{name: "port flag is handshake", args: []string{"--port", "50051"}, want: false},
		{name: "serve subcommand is handshake", args: []string{"serve"}, want: false},
		{name: "serve with port is handshake", args: []string{"serve", "--port", "50051"}, want: false},
		{name: "help flag is CLI", args: []string{"--help"}, want: true},
		{name: "short help is CLI", args: []string{"-h"}, want: true},
		{name: "version flag is CLI", args: []string{"--version"}, want: true},
		{name: "serve help is CLI", args: []string{"serve", "--help"}, want: true},
		{name: "dry-run subcommand is CLI", args: []string{"dry-run"}, want: true},
		{name: "schema command is CLI", args: []string{"__schema"}, want: true},
		{name: "format flag is CLI", args: []string{"--format=json"}, want: true},
		{name: "global dry-run flag is CLI", args: []string{"--dry-run"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isCLIInvocation(tt.args))
		})
	}
}

func TestParseHandshakeArgs(t *testing.T) {
	t.Parallel()

	t.Run("port flag", func(t *testing.T) {
		t.Parallel()
		port, err := parseHandshakeArgs([]string{"--port", "50051"})
		require.NoError(t, err)
		assert.Equal(t, 50051, port)
	})

	t.Run("serve subcommand with port", func(t *testing.T) {
		t.Parallel()
		port, err := parseHandshakeArgs([]string{"serve", "--port", "41234"})
		require.NoError(t, err)
		assert.Equal(t, 41234, port)
	})

	t.Run("empty args", func(t *testing.T) {
		t.Parallel()
		port, err := parseHandshakeArgs(nil)
		require.NoError(t, err)
		assert.Equal(t, 0, port)
	})

	t.Run("unknown flag", func(t *testing.T) {
		t.Parallel()
		_, err := parseHandshakeArgs([]string{"--format=json"})
		require.Error(t, err)
	})
}

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := testServeConfig(&mockPlugin{name: "help-plugin"}, "v1.2.3")

	code := runCLI(context.Background(), cfg, []string{"--help"}, &stdout, &stderr, bytes.NewReader(nil))

	assert.Equal(t, ax.ExitSuccess, code)
	assert.Contains(t, stdout.String(), "Usage:")
	assert.Contains(t, stdout.String(), "dry-run")
	assert.Contains(t, stdout.String(), "serve")
	assert.NotContains(t, stdout.String(), "PORT=")
	assert.NotContains(t, stdout.String(), `"error_code"`)
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := testServeConfig(&mockPlugin{name: "version-plugin"}, "v9.8.7")

	code := runCLI(context.Background(), cfg, []string{"--version"}, &stdout, &stderr, bytes.NewReader(nil))

	assert.Equal(t, ax.ExitSuccess, code)
	assert.Equal(t, "v9.8.7\n", stdout.String())
	assert.NotContains(t, stdout.String(), "PORT=")
}

func TestRunUnknownCommandJSONError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := testServeConfig(&mockPlugin{name: "err-plugin"}, "v1.0.0")

	code := runCLI(context.Background(), cfg, []string{"nope"}, &stdout, &stderr, bytes.NewReader(nil))

	assert.NotEqual(t, ax.ExitSuccess, code)
	assert.NotContains(t, stdout.String(), "PORT=")
	assert.Contains(t, stderr.String(), `"error_code"`)
}

func TestRunDryRunEnvelope(t *testing.T) {
	plugin := &cliDryRunPlugin{mockPlugin: mockPlugin{name: "dry-plugin"}}
	plugin.resp = NewDryRunResponse(
		WithResourceTypeSupported(true),
		WithConfigurationValid(true),
		WithFieldMappings(AllFieldsWithStatus(pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_SUPPORTED)),
	)
	cfg := testServeConfig(plugin, "v1.0.0")

	var stdout, stderr bytes.Buffer
	code := runCLI(
		context.Background(),
		cfg,
		[]string{"dry-run", "--provider", "aws", "--resource-type", "ec2", "--format=json"},
		&stdout,
		&stderr,
		bytes.NewReader(nil),
	)

	require.Equal(t, ax.ExitSuccess, code, "stderr=%s", stderr.String())
	require.NotEmpty(t, stdout.Bytes())

	var envelope ax.Envelope[json.RawMessage]
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &envelope))
	assert.Contains(t, string(envelope.Data), "resource_type_supported")
	assert.Contains(t, string(envelope.Data), "field_mappings")
	require.NotNil(t, plugin.got)
	require.NotNil(t, plugin.got.GetResource())
	assert.Equal(t, "aws", plugin.got.GetResource().GetProvider())
	assert.Equal(t, "ec2", plugin.got.GetResource().GetResourceType())
	assert.NotContains(t, stdout.String(), "PORT=")
}

func TestRunDryRunMissingFlags(t *testing.T) {
	plugin := &cliDryRunPlugin{mockPlugin: mockPlugin{name: "dry-plugin"}}
	cfg := testServeConfig(plugin, "v1.0.0")

	var stdout, stderr bytes.Buffer
	code := runCLI(
		context.Background(),
		cfg,
		[]string{"dry-run", "--format=json"},
		&stdout,
		&stderr,
		bytes.NewReader(nil),
	)

	assert.Equal(t, ax.ExitValidation, code)
	assert.Contains(t, stderr.String(), `"error_code"`)
	assert.Contains(t, stderr.String(), "validation_error")
	assert.NotContains(t, stdout.String(), "PORT=")
}

func TestRunDryRunUnimplemented(t *testing.T) {
	cfg := testServeConfig(&mockPlugin{name: "no-dry-run"}, "v1.0.0")

	var stdout, stderr bytes.Buffer
	code := runCLI(
		context.Background(),
		cfg,
		[]string{"dry-run", "--provider", "aws", "--resource-type", "ec2", "--format=json"},
		&stdout,
		&stderr,
		bytes.NewReader(nil),
	)

	assert.Equal(t, ax.ExitInternal, code)
	assert.Contains(t, stderr.String(), `"error_code"`)
	assert.Contains(t, stderr.String(), "unimplemented")
	assert.NotContains(t, stdout.String(), "PORT=")
}

func TestRunDryRunHandlerError(t *testing.T) {
	plugin := &cliDryRunPlugin{mockPlugin: mockPlugin{name: "dry-plugin"}}
	plugin.err = errors.New("boom")
	cfg := testServeConfig(plugin, "v1.0.0")

	var stdout, stderr bytes.Buffer
	code := runCLI(
		context.Background(),
		cfg,
		[]string{"dry-run", "--provider", "aws", "--resource-type", "ec2", "--format=json"},
		&stdout,
		&stderr,
		bytes.NewReader(nil),
	)

	assert.Equal(t, ax.ExitInternal, code)
	assert.Contains(t, stderr.String(), `"error_code"`)
	assert.Contains(t, stderr.String(), "internal_error")
	assert.Contains(t, stderr.String(), "boom")
	assert.NotContains(t, stdout.String(), "PORT=")
}

func TestRunHandshakeStdoutIsPortOnly(t *testing.T) {
	plugin := &mockPlugin{name: "handshake-plugin"}
	cfg := testServeConfig(plugin, "v1.0.0")

	var stdout syncBuffer
	var stderr syncBuffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan int, 1)
	go func() {
		done <- runCLI(ctx, cfg, []string{"--port", "0"}, &stdout, &stderr, bytes.NewReader(nil))
	}()

	require.Eventually(t, func() bool {
		return strings.HasPrefix(stdout.String(), "PORT=")
	}, 3*time.Second, 10*time.Millisecond, "stderr=%s", stderr.String())

	out := stdout.String()
	assert.Regexp(t, `^PORT=\d+\n$`, out)
	assert.NotContains(t, out, "{")
	assert.NotContains(t, out, "error_code")

	cancel()
	select {
	case code := <-done:
		assert.Equal(t, ax.ExitSuccess, code, "stderr=%s", stderr.String())
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not exit after context cancel")
	}
}

func TestRunHandshakeServeSubcommand(t *testing.T) {
	plugin := &mockPlugin{name: "serve-plugin"}
	cfg := testServeConfig(plugin, "v1.0.0")

	var stdout syncBuffer
	var stderr syncBuffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan int, 1)
	go func() {
		done <- runCLI(ctx, cfg, []string{"serve", "--port", "0"}, &stdout, &stderr, bytes.NewReader(nil))
	}()

	require.Eventually(t, func() bool {
		return strings.HasPrefix(stdout.String(), "PORT=")
	}, 3*time.Second, 10*time.Millisecond, "stderr=%s", stderr.String())

	assert.Regexp(t, `^PORT=\d+\n$`, stdout.String())
	cancel()
	select {
	case code := <-done:
		assert.Equal(t, ax.ExitSuccess, code, "stderr=%s", stderr.String())
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not exit after context cancel")
	}
}

func TestRunServeDryRunPreview(t *testing.T) {
	plugin := &mockPlugin{name: "serve-preview-plugin"}
	cfg := testServeConfig(plugin, "v1.0.0")

	var stdout, stderr bytes.Buffer
	code := runCLI(
		context.Background(),
		cfg,
		[]string{"serve", "--port", "50099", "--dry-run", "--format=json"},
		&stdout,
		&stderr,
		bytes.NewReader(nil),
	)

	require.Equal(t, ax.ExitSuccess, code, "stderr=%s", stderr.String())
	var envelope ax.Envelope[map[string]any]
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &envelope))
	assert.Equal(t, cliCommandServe, envelope.Data["action"])
	assert.InDelta(t, float64(50099), envelope.Data["port"], 0)
	assert.Equal(t, "serve-preview-plugin", envelope.Data["plugin"])
	assert.NotContains(t, stdout.String(), "PORT=")
}

func TestPluginCommandSchema(t *testing.T) {
	cfg := testServeConfig(&mockPlugin{name: "schema-plugin"}, "v2.0.0")
	root := newPluginCommand(cfg)

	result := axtest.Run(context.Background(), t, root, []string{"__schema"}, ax.WithVersion("v2.0.0"))
	require.Equal(t, ax.ExitSuccess, result.ExitCode, "stderr=%s", result.Stderr)
	assert.Contains(t, string(result.Stdout), "dry-run")
	assert.Contains(t, string(result.Stdout), "serve")
}

func TestRunNilPlugin(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLI(context.Background(), ServeConfig{}, []string{"--port", "0"}, &stdout, &stderr, bytes.NewReader(nil))

	assert.Equal(t, ax.ExitInternal, code)
	assert.Contains(t, stderr.String(), `"error_code"`)
	assert.NotContains(t, stdout.String(), "PORT=")
}

var _ io.Writer = (*syncBuffer)(nil)
