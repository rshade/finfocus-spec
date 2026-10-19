# SDK, Discovery, and Error Contract

**Status**: Implemented and validated; companion to [parse.proto](parse.proto).

## Go surfaces

- `pluginsdk.CostParseProvider`: Parse(context.Context, *pbc.ParseRequest) (*pbc.ParseResponse, error).
- `pluginsdk.WithParseLimits(formats []string, inputModes []string, maxInputBytes int64) PluginInfoOption`.
- `testing.ParseLimits`: Formats []string, InputModes []string, MaxInputBytes int64; pluginsdk exports an alias.
- `testing.FormatParseLimits(formats []string, inputModes []string, maxInputBytes int64) map[string]string`.
- `testing.ParseParseLimits(metadata map[string]string) (*ParseLimits, error)`; pluginsdk delegates.
- `testing.ValidateParseRequest(req *pbc.ParseRequest, limits *ParseLimits) error`; pluginsdk delegates.
- `testing.ValidateParseResponse(req *pbc.ParseRequest, resp *pbc.ParseResponse) error`; pluginsdk delegates.
- `testing.ErrInvalidParseLimits`, `ErrInvalidParseRequest`, `ErrInvalidParseResponse`: sentinel errors for errors.Is.

FormatParseLimits writes supplied configuration without silently correcting it; PluginInfo.Validate is the configuration
gate. ParseParseLimits returns nil,nil only when no parse keys exist. ValidateParseRequest requires nonnil valid limits;
missing provider configuration is a configuration error, not evidence of unsupported user input. Response validation
requires a nonnil request and response; full advertised-request validation happens separately because it needs limits.

Export godoc and corresponding TypeScript definitions. Constants for metadata keys and the service ErrorInfo domain
live beside limits. No new Serve parameter: detect CostParseProvider into optionalServices.parser.

## Dispatch and error mapping

| Condition | gRPC / Connect category | Detail |
| --- | --- | --- |
| Invalid format syntax, missing/conflicting input, bad content path | InvalidArgument | Name field/index |
| Options over encoded cap or unknown provider option | InvalidArgument | Name options/key |
| Unparseable format content | InvalidArgument | File and line when known |
| Valid syntax but unadvertised format | FailedPrecondition | ErrorInfo reason UNSUPPORTED_FORMAT |
| Unadvertised input mode | FailedPrecondition | ErrorInfo reason UNSUPPORTED_INPUT_MODE |
| Raw content/read total over advertised limit | ResourceExhausted | Identify size limit |
| No provider or provider returns Unimplemented | Unimplemented | Normal unsupported-service outcome |
| Provider returns recognized status | Preserve status/details | Apply handlerStatus / toConnectError |
| Provider returns ordinary error | Internal | Use established sanitized handler-error behavior |
| Provider returns nil response with nil error | Internal | Prevent successful malformed response |

ErrorInfo domain is `finfocus.v1.CostParseService`. Validate basic syntax/input/path/options structure before checking
advertised format, then mode, then content size. Unknown format-specific keys require provider interpretation.
Known malformed requests must not reach provider execution. Shared request validation returns status-carrying errors;
all InvalidArgument failures wrap ErrInvalidParseRequest following newInvalidArgument. Response contract failures wrap
ErrInvalidParseResponse; server adapters convert invalid provider output to Internal rather than blaming the request.

The shared parser dispatch path validates requests using configured limits, invokes the provider, normalizes handler
errors, then validates successful responses. A missing provider returns Unimplemented before parsing validation.
Malformed configured limits fail startup, not a user request. Stub methods returning Unimplemented keep that status.
Generated method embedding is indistinguishable from implementation at discovery; health lists registered interfaces
and does not probe Parse or promise a successful parse for every input.

## Strict oneof transport contract

Normal generated binary decoding uses the last oneof arm. A parse-specific raw scan must detect both top-level
fields 2 and 3 before standard decoding. The gRPC server-local codec delegates to encoding.GetCodecV2("proto");
all messages other than ParseRequest pass through unchanged. After successfully decoding a conflicting request,
clear Input so application validation returns InvalidArgument naming input. Direct codec errors become Internal in
gRPC, so the semantic conflict must reach the adapter as an invalid typed request instead. Never invoke the provider
for that request. No process-global codec replacement or side-channel request map is permitted.

Connect's parsing handler overrides only its binary codec with equivalent conflict handling. Normal JSON decoding
rejects both members before invocation. Preserve protobuf unknown-field compatibility when scanning and allow repeated
occurrences of the same input arm under standard protobuf semantics. Test conflicts in both orders and with an empty
root arm. Typed-validator tests cover no arm, typed-nil wrapper, nil file set, empty collection/root, and both valid arms.

## Discovery and compatibility

Add capability 19, legacy supports_cost_parse, registry/manifest cost_parse enum values, maxValidCapability, inference,
optionalCapabilities preallocation count (11 to 12), and bounds/drift tests. Parse metadata is mandatory for advertised
parsers and otherwise validated whenever present. Infer capability from the provider interface. Explicit
WithCapabilities(COST_PARSE) is the supported parser-only override; startup warns if optional-only operation retains
inferred base pricing capabilities. Existing MockPlugin and SourceFormat remain unchanged.

## TypeScript and gateway

CostParseClient accepts ClientConfig and exposes `parse(request: ParseRequest): Promise<ParseResponse>`. Use generated
CostParseService and createClient; honor injected transport and the existing default JSON Connect transport. Export
client, service, messages, schemas, severity and parseParseLimits from the client package. ParseLimits.maxInputBytes is
bigint; metadata parsing must reject the same malformed sets as Go without unsafe Number conversion.

RESTGatewayConfig gains optional costParseClient. Register POST /finfocus.v1.CostParseService/Parse when configured;
without the client it follows existing 404 behavior. Reuse fromJson/toJson, including base64 bytes and RFC 3339 Timestamp.
Keep existing error sanitization and HTTP status mapping. Existing REST errors do not expose google.rpc.ErrorInfo;
clients needing stable unsupported-format/mode reasons use the Connect/gRPC client. Do not add or claim REST detail
propagation as part of this feature. The existing 1 MiB raw-body cap remains, independently of the plugin's raw input limit.

## Mock and conformance

Provide MockCostParser separately from MockPlugin. It advertises a synthetic format and uses deterministic fixed rules
with a configurable valid descriptor; it implements the same validators and provider-owned checks. Caller-supplied
format fixtures let real parsers participate without accepting a synthetic format or vendor templates in this repository.

Define ParseServer by its Parse method. NewCostParseHarness(ParseServer) uses the existing bufconn helper and generated
client. RunCostParseConformance accepts testing.T, ParseServer, and explicit configuration containing limits, a valid
request fixture for the provider's format/mode, and optional mode-specific valid fixtures. Fixtures are cloned before
mutation. Providers may supply GetPluginInfo for cross-checking served metadata. Use local temporary roots for mock
path tests; do not inject arbitrary inaccessible paths into real-provider conformance.

Required scenario names: format_rejected, input_mode_xor, unsupported_input_mode, path_traversal_rejected,
source_ids_unique, unknown_not_absent, descriptors_valid, deterministic. Input XOR tests assert typed missing input;
raw binary/JSON conflicting arms are exercised separately in SDK transport integration tests. Unsupported mode tests
skip with an explanation only when every mode is advertised. Content-path scenarios require a content-mode fixture.
Determinism compares resources and warnings over 100 calls with unchanged dependencies, excluding expiration.
Each call uses a 10-second deadline. Do not allocate an unbounded advertised-limit-plus-one probe; large boundary
cases are validator tests with controlled configured limits. Negative response tests must prove validators reject bad
responses; real-provider structural conformance does not force providers to emit unknowns or skipped resources.
