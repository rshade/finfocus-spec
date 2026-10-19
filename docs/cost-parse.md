# Cost parsing

CostParseService is an optional plugin service for cloud-native and Kubernetes inputs.
It returns existing priceable ResourceDescriptor messages. CloudFormation, Helm, Kustomize and future formats
are implemented by individual plugins. Terraform and Pulumi ingestion remain core-native. Parsing never calculates prices.

## Discover and submit input

Discover COST_PARSE (19), legacy supports_cost_parse, and all three metadata keys:
parse_formats, parse_input_modes and parse_max_input_bytes. Formats are unique lowercase names matching
`[a-z][a-z0-9_-]*` in 1–32 ASCII bytes. Modes are content, path, or both. The byte limit is a canonical positive int64.
Metadata is mandatory for a parser and validated whenever present. WithParseLimits records configuration;
PluginInfo.Validate rejects malformed configuration. Automatic inference appends parsing to the existing pricing
capabilities. Parser-only plugins use an explicit WithCapabilities(COST_PARSE) override.

Select exactly one input: a nonempty named file collection or a nonempty host-exposed root_path.
Binary requests containing both fields are rejected before ordinary oneof decoding can discard evidence of the conflict.
JSON also rejects conflicting members. Repeated occurrences of the same binary arm retain protobuf semantics.
Content paths are portable slash-relative names: no absolute/UNC/drive form, backslash, colon, NUL byte,
empty segment, dot or parent segment. Colons are excluded everywhere, including Windows alternate-stream forms.
Duplicates are invalid. Empty file contents are allowed at the contract layer; format rules may reject them.
Root paths may be absolute. Shared validation never reads the filesystem.

Options have a separate maximum of 65,536 encoded protobuf bytes. Providers document and enforce format-specific
allowlists and reject unknown keys. Known keys are parameters for cfn; release_name, namespace and values for helm;
there are no kustomize keys. The SDK does not interpret these keys or implement vendor parsers.

## Input safety and diagnostics

Treat all input as untrusted. Hosts expose only an intended root; providers enforce root containment, including
escaping symlinks, and account for bytes consumed from every file or dependency. Providers stop reads before
crossing their advertised limit. Generic path validation alone cannot enforce filesystem containment.

Content-mode size is the sum of raw file contents, accumulated without overflow; filenames and encoding overhead
do not count. Path-mode size is the total raw bytes actually read. Options have their independent encoded cap.
Transport receive ceilings can reject requests before provider execution: the effective limit includes those
ceilings and the advertised limit. Base64 expands binary content in protobuf JSON. The REST gateway retains its
independent 1 MiB raw-body limit; this feature does not increase any transport ceiling.

| Condition | Category | Detail |
| --- | --- | --- |
| Bad syntax, missing/conflicting input, unsafe path, invalid options | InvalidArgument | Field/index or option key |
| Unparseable content | InvalidArgument | File and physical line when known |
| Unadvertised format | FailedPrecondition | UNSUPPORTED_FORMAT |
| Unadvertised mode | FailedPrecondition | UNSUPPORTED_INPUT_MODE |
| Excess raw bytes | ResourceExhausted | Size limit |
| Missing provider or generated stub | Unimplemented | Parsing unavailable |
| Invalid successful provider output | Internal | Sanitized response failure |

ErrorInfo uses domain finfocus.v1.CostParseService. Validate syntax/input/path/options first, then format, then
mode, then content size. Recognized provider statuses and details survive transport adaptation; ordinary errors
are sanitized as Internal. Connect/gRPC clients expose structured details. The existing REST error envelope
does not expose ErrorInfo. Health reports registered interfaces and never probes Parse; an embedded generated
stub can therefore be registered while returning Unimplemented.

## Synthetic fixtures

MockCostParser is a separate neutral fixture provider. It advertises fixture format and both modes.
Each nonempty line is `resource <source_id>` (other line kinds are listed below);
path mode reads .fixture files in lexical walk order beneath a caller-owned root.
Default descriptors use provider custom and type custom:compute:Instance.
WithParseResource clones a caller-supplied descriptor. Existing MockPlugin capabilities are unchanged.

## Partial results and source identity

Each ParsedResource has a response-unique nonempty source_id, such as a CloudFormation logical ID or
Deployment/api for Kubernetes Kind/name. The descriptor's resource_type is the priceable Pulumi type token;
it can enter pricing directly without ResolveResourceTypes. Attribute names retain the plugin's pricing shape,
including CloudFormation property casing.

For example, a returned Deployment/api can retain known Sizing.Memory=0 and Label="" while listing
Sizing.Cpu as unknown. The Cpu value is absent from attributes. An absent property without an unknown marker
is simply absent; known zero, false and empty strings are resolved values. Null is a present value and cannot
represent a listed unknown. Missing object ancestors permit an unknown leaf; scalar, list and null ancestors
do not. Paths are relative to attributes, case sensitive and dot separated; list indexing and escaped literal
dotted keys are unsupported. Report an unresolved containing property where those forms would be needed.

Warnings are separate from resources. Match stable warning codes, never explanation text. A resource_skipped
warning supplies a nonempty source_id even when that source is absent from resources. Document-wide notices
may omit source_id. Only INFO=1 and WARNING=2 are valid; empty code/message or nil entries are invalid.
Empty resource collections and all-skipped results are successful responses when diagnostics explain the skips.

The synthetic mock adds `unknown <earlier_source_id> <attribute.path>`, `skip <source_id>` and
`notice <message>` fixture lines. Unknown paths remove values from cloned fixture descriptors. Repeated sources,
including skipped sources, and repeated unknown paths are rejected with the fixture file and physical line.

## Optional action hint

ParsedResource.action (field 4, enum ParseAction) is an optional hint for plan-style inputs that report a change per
resource: CREATE, UPDATE, DELETE, REPLACE or NO_CHANGE. UNSPECIFIED (zero, the default) means the parser reports a
declared resource with no action. Parsers of declared templates, such as CloudFormation templates, Helm and
Kustomize, leave it unset. A future plugin that parses a plan-like artifact, for example a change set, can report it.
A host-side parser for plan-style inputs can use the same result shape.

Hosts must not require the field. Unlike warning severity, zero is valid here, but unknown numbers are invalid in the
same way: shared response validation rejects a ParsedResource whose action is not one of the six defined values.
The field carries no pricing meaning by itself. Parsing still never calculates prices, and how a host prices a
DELETE, REPLACE or NO_CHANGE resource is the host's concern. Existing consumers are unaffected: the field and enum
are additive, and an absent field decodes as UNSPECIFIED.

The synthetic mock adds `action <earlier_source_id> <name>` where name is create, update, delete, replace or
no_change. It must reference an earlier returned resource and may appear once per resource. Fixtures without action
lines return UNSPECIFIED.

## Provider selection and caching

Choose an explicitly selected parser first. Otherwise use an unambiguous provider match among parsers that
advertise the requested format and mode. If multiple providers still match, fail as ambiguous and ask for a
selection. Combining parser outputs is undefined. This is host integration guidance; the SDK does not implement
selection, merging or a cache.

For a pure-input result, cache by a hash of file contents, format, options and parser identity/version.
For example, changing the namespace option or parser version changes the key even when file bytes are identical.
For path input, include current consumed-file/dependency contents; the text of root_path alone is insufficient.
Account for external dependencies and invalidate when they change. An optional expires_at is advisory freshness
guidance for results depending on changing data; omission provides no expiration guidance. Already-expired but
valid timestamps remain structurally valid. Expiration may advance while resource and warning results stay equal.

Run [provider-neutral Go conformance](../sdk/go/testing/README.md#cost-parser-conformance) with valid fixtures
for your own format. See the [validation quickstart](../specs/602-cost-parse-service/quickstart.md) for transport,
compatibility and performance commands.
