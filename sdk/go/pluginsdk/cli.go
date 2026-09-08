package pluginsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	ax "github.com/rshade/ax-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/encoding/protojson"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	cliCommandServe   = "serve"
	cliCommandDryRun  = "dry-run"
	cliCommandSchema  = "__schema"
	cliCommandHelp    = "help"
	cliCommandVersion = "version"
	// cliCommandCompletion has no registered Cobra command (CompletionOptions.DisableDefaultCmd
	// disables it below); routing it here still sends "completion" through ax.Execute so it gets
	// a JSON "unknown command" error instead of being misparsed as a handshake positional arg.
	cliCommandCompletion = "completion"
)

const (
	errCodeInternal      = "internal_error"
	errCodeValidation    = "validation_error"
	errCodeUnimplemented = "unimplemented"
)

// cliFlagKinds is the single source of truth for CLI flag behavior: whether a
// flag takes a value (so firstPositional must skip it) and whether its mere
// presence routes the invocation through ax.Execute instead of the handshake
// path. Adding a new global flag only requires an entry here.
//
//nolint:gochecknoglobals // Read-only lookup table; avoids duplicating flag knowledge across functions.
var cliFlagKinds = map[string]struct {
	takesValue  bool
	triggersCLI bool
}{
	"--help":            {triggersCLI: true},
	"-h":                {triggersCLI: true},
	"--version":         {triggersCLI: true},
	"-v":                {triggersCLI: true},
	"--dry-run":         {triggersCLI: true},
	"--yes":             {triggersCLI: true},
	"--format":          {takesValue: true, triggersCLI: true},
	"--idempotency-key": {takesValue: true, triggersCLI: true},
	"--port":            {takesValue: true},
	"--provider":        {takesValue: true},
	"--resource-type":   {takesValue: true},
	"--region":          {takesValue: true},
	"--sku":             {takesValue: true},
}

// Run is the ax-go CLI entry point for plugin binaries. It returns a process
// exit code; callers should terminate with os.Exit(Run(config)).
//
// Handshake-sensitive invocations (no args, --port, or the serve subcommand
// without agent-discovery flags) call Serve() directly so stdout stays
// PORT=<n> only. --help, --version, dry-run, and __schema go through
// ax.Execute for structured envelopes and agent-safety flags.
//
// Serve() itself is unchanged and remains the programmatic gRPC server used
// by tests and hosts that already manage process lifecycle.
func Run(config ServeConfig) int {
	return runCLI(context.Background(), config, os.Args[1:], os.Stdout, os.Stderr, os.Stdin)
}

func runCLI(
	ctx context.Context,
	config ServeConfig,
	args []string,
	stdout, stderr io.Writer,
	stdin io.Reader,
) int {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	if stdin == nil {
		stdin = os.Stdin
	}

	if !isCLIInvocation(args) {
		return runHandshake(ctx, config, args, stdout, stderr)
	}

	root := newPluginCommand(config)
	root.SetArgs(args)
	version := pluginCLIVersion(config)
	return ax.Execute(
		ctx,
		root,
		ax.WithStdin(stdin),
		ax.WithStdout(stdout),
		ax.WithStderr(stderr),
		ax.WithVersion(version),
	)
}

func pluginCLIVersion(config ServeConfig) string {
	if config.PluginInfo != nil && config.PluginInfo.Version != "" {
		return config.PluginInfo.Version
	}
	return ax.ResolveVersion("")
}

func pluginCLIName(config ServeConfig) string {
	if config.PluginInfo != nil && config.PluginInfo.Name != "" {
		return config.PluginInfo.Name
	}
	if config.Plugin != nil && config.Plugin.Name() != "" {
		return config.Plugin.Name()
	}
	return "plugin"
}

func isCLIInvocation(args []string) bool {
	switch firstPositional(args) {
	case cliCommandDryRun, cliCommandVersion, cliCommandHelp, cliCommandSchema, cliCommandCompletion:
		return true
	}

	for _, arg := range args {
		if arg == "--" {
			break
		}
		name, _, _ := strings.Cut(arg, "=")
		if cliFlagKinds[name].triggersCLI {
			return true
		}
	}
	return false
}

func firstPositional(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		}
		if !strings.HasPrefix(arg, "-") {
			return arg
		}
		name, _, hasValue := strings.Cut(arg, "=")
		if hasValue {
			continue
		}
		if cliFlagKinds[name].takesValue && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			i++
		}
	}
	return ""
}

func parseHandshakeArgs(args []string) (int, error) {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == cliCommandServe {
			continue
		}
		filtered = append(filtered, arg)
	}

	fs := pflag.NewFlagSet("serve", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	port := fs.Int("port", 0, "TCP port for gRPC server (overrides FINFOCUS_PLUGIN_PORT)")
	if err := fs.Parse(filtered); err != nil {
		return 0, err
	}
	if rest := fs.Args(); len(rest) > 0 {
		return 0, fmt.Errorf("unexpected arguments: %s", strings.Join(rest, " "))
	}
	return *port, nil
}

func runHandshake(ctx context.Context, config ServeConfig, args []string, stdout, stderr io.Writer) int {
	if config.Plugin == nil {
		return writeCLIError(ctx, stderr, config, errCodeInternal, "ServeConfig.Plugin is required", ax.ExitInternal)
	}

	port, err := parseHandshakeArgs(args)
	if err != nil {
		return writeCLIError(
			ctx, stderr, config, errCodeValidation, err.Error(), ax.ExitValidation, ax.WithErrorCause(err),
		)
	}
	if port > 0 {
		config.Port = port
	}
	config.handshakeWriter = stdout

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if serveErr := Serve(ctx, config); serveErr != nil {
		if errors.Is(serveErr, context.Canceled) {
			return ax.ExitSuccess
		}
		return writeCLIError(
			ctx, stderr, config, errCodeInternal, serveErr.Error(), ax.ExitInternal, ax.WithErrorCause(serveErr),
		)
	}
	return ax.ExitSuccess
}

func writeCLIError(
	ctx context.Context,
	stderr io.Writer,
	config ServeConfig,
	code, message string,
	exitCode int,
	opts ...ax.ErrorOption,
) int {
	allOpts := append([]ax.ErrorOption{
		ax.WithErrorTool(pluginCLIName(config)),
		ax.WithErrorVersion(pluginCLIVersion(config)),
		ax.WithErrorExitCode(exitCode),
	}, opts...)
	axErr := ax.NewError(ctx, code, message, allOpts...)
	_ = ax.WriteError(stderr, axErr)
	return axErr.ExitCode()
}

func newPluginCommand(config ServeConfig) *cobra.Command {
	name := pluginCLIName(config)
	version := pluginCLIVersion(config)

	root := &cobra.Command{
		Use:     name,
		Short:   "FinFocus cost-source plugin",
		Long:    "Serve the plugin over gRPC, or inspect FOCUS field mappings with dry-run.",
		Version: version,
		Example: fmt.Sprintf(`  %s --port 50051
  %s --version
  %s dry-run --provider aws --resource-type ec2 --format=json`, name, name, name),
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if ax.DryRunFromContext(cmd.Context()) {
				return writeServePreview(cmd, config)
			}
			return cmd.Help()
		},
	}
	root.SetVersionTemplate("{{.Version}}\n")
	root.AddCommand(newServeCommand(config))
	root.AddCommand(newDryRunCommand(config))
	return root
}

func newServeCommand(config ServeConfig) *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:   cliCommandServe,
		Short: "Start the plugin gRPC server",
		Long: "Start the plugin gRPC server and announce PORT=<n> on stdout. " +
			"Host launches (`--port` or no subcommand) bypass ax.Execute so the handshake stays clean.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if ax.DryRunFromContext(cmd.Context()) {
				preview := config
				if port > 0 {
					preview.Port = port
				}
				return writeServePreview(cmd, preview)
			}
			return dryRunError(
				cmd.Context(),
				errCodeInternal,
				"serve is handshake-sensitive and must be invoked without agent-format flags",
				ax.ExitInternal,
				ax.WithActionableFix("run the plugin with --port or no subcommand"),
			)
		},
	}
	cmd.Flags().IntVar(&port, "port", 0, "TCP port for gRPC server (overrides FINFOCUS_PLUGIN_PORT)")
	return cmd
}

func writeServePreview(cmd *cobra.Command, config ServeConfig) error {
	port := config.Port
	if port == 0 {
		port = GetPort()
	}
	payload := map[string]any{
		"action": cliCommandServe,
		"port":   port,
		"plugin": pluginCLIName(config),
	}
	return ax.WriteJSON(cmd.OutOrStdout(), ax.NewEnvelope(cmd.Context(), payload))
}

func newDryRunCommand(config ServeConfig) *cobra.Command {
	var (
		provider     string
		resourceType string
		region       string
		sku          string
	)

	cmd := &cobra.Command{
		Use:   cliCommandDryRun,
		Short: "Inspect FOCUS field mappings without starting the gRPC server",
		Long:  "Call the plugin's DryRunHandler in-process and print an ax.Envelope JSON result.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDryRunCommand(cmd, config, provider, resourceType, region, sku)
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "cloud provider (required)")
	cmd.Flags().StringVar(&resourceType, "resource-type", "", "resource type (required)")
	cmd.Flags().StringVar(&region, "region", "", "optional region")
	cmd.Flags().StringVar(&sku, "sku", "", "optional SKU")
	return cmd
}

// dryRunError builds a structured ax.Error, folding the exit code into opts
// so call sites don't each repeat ax.WithErrorExitCode.
func dryRunError(ctx context.Context, code, message string, exitCode int, opts ...ax.ErrorOption) error {
	return ax.NewError(ctx, code, message, append(opts, ax.WithErrorExitCode(exitCode))...)
}

func runDryRunCommand(
	cmd *cobra.Command,
	config ServeConfig,
	provider, resourceType, region, sku string,
) error {
	ctx := cmd.Context()
	if strings.TrimSpace(provider) == "" || strings.TrimSpace(resourceType) == "" {
		return dryRunError(ctx, errCodeValidation, "--provider and --resource-type are required", ax.ExitValidation,
			ax.WithActionableFix("pass --provider and --resource-type"))
	}

	handler, ok := config.Plugin.(DryRunHandler)
	if !ok {
		return dryRunError(ctx, errCodeUnimplemented, "plugin does not implement DryRunHandler", ax.ExitInternal,
			ax.WithActionableFix("implement pluginsdk.DryRunHandler on the plugin"))
	}

	resp, err := handler.HandleDryRun(ctx, &pbc.DryRunRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     provider,
			ResourceType: resourceType,
			Region:       region,
			Sku:          sku,
		},
	})
	if err != nil {
		return dryRunError(ctx, errCodeInternal, err.Error(), ax.ExitInternal, ax.WithErrorCause(err))
	}
	if resp == nil {
		return dryRunError(ctx, errCodeInternal, "DryRunHandler returned a nil response", ax.ExitInternal)
	}

	raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: true}.Marshal(resp)
	if err != nil {
		return dryRunError(ctx, errCodeInternal, "failed to marshal dry-run response", ax.ExitInternal,
			ax.WithErrorCause(err))
	}

	compact := bytes.TrimSpace(raw)
	if !json.Valid(compact) {
		return dryRunError(ctx, errCodeInternal, "dry-run response was not valid JSON", ax.ExitInternal)
	}

	return ax.WriteJSON(cmd.OutOrStdout(), ax.NewEnvelope(ctx, json.RawMessage(compact)))
}
