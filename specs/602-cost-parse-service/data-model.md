# Data Model: Cost Parse Service

**Status**: Implemented and validated; no persisted state or migrations.

## Parser advertisement

| Field | Representation | Invariant |
| --- | --- | --- |
| Capability | COST_PARSE, numeric value 19 | Legacy metadata name supports_cost_parse |
| parse_formats | Comma-separated names | Nonempty unique names, each matching format syntax |
| parse_input_modes | Comma-separated content/path | Nonempty unique subset of the two modes |
| parse_max_input_bytes | Decimal positive int64 string | 1 through 9223372036854775807 |

All three keys are required together. No keys is valid for a non-parser. Advertising COST_PARSE requires complete
metadata. Duplicate names, empty entries, whitespace, unknown modes, nondecimal and overflowing limits are invalid.
Canonical formatting uses lowercase names, no spaces, and a base-10 integer without a sign. Parsing requires that
canonical representation. Advertisement is the discovery authority: ParseResponse has no duplicated limits fields.

Go ParseLimits contains Formats []string, InputModes []string, MaxInputBytes int64. TypeScript uses the same concepts
with bigint for the byte limit. WithParseLimits stores the metadata; PluginInfo.Validate reports invalid values.

## Parse request and input

| Entity / field | Type | Validation / relationship |
| --- | --- | --- |
| ParseRequest.format | string | Required, 1–32 ASCII bytes, `[a-z][a-z0-9_-]*`, advertised |
| ParseRequest.files | ParseFileSet input arm | Exclusive with root_path, nonnil, nonempty collection |
| ParseRequest.root_path | string input arm | Exclusive with files, nonempty; may be absolute |
| ParseRequest.options | optional structured object | Encoded size <=65,536; provider checks format keys |
| ParseFileSet.files | ordered ParseFile collection | No nil entries; paths unique |
| ParseFile.path | string | Portable slash-relative name with safe nonempty segments |
| ParseFile.content | bytes | May be empty; contributes to total input bytes |

The server rejects both input arms before oneof decoding loses the conflict. Typed objects expose at most one arm.
Missing or nil input is invalid. Reject unsafe paths rather than cleaning them: absolute/UNC/drive forms, backslashes,
empty segments, `.` and `..` segments. Root-path filesystem policy is distinct from content-file naming.

Content size is the sum of file contents, accumulated without integer overflow. Path-mode size is all input bytes
consumed while reading; plugins enforce it and root containment as reads occur. Shared validators perform no file I/O.
Options use protobuf encoded size, not JSON text size or entry count. Supported known option keys are parameters for
cfn; release_name, namespace, values for helm; none for kustomize. Arbitrary future formats define their own keys.

## Parsed resource

| Field | Type | Validation / relationship |
| --- | --- | --- |
| resource | ResourceDescriptor | Required, passes existing testing.ValidateResourceDescriptor |
| source_id | string | Nonempty; unique across resources in one response |
| unknown_properties | string collection | Unique, well-formed dynamic attribute paths with absent values |

resource_type is the priceable Pulumi token. Attributes retain the pricing plugin's expected property names and casing.
For CloudFormation source_id is the logical ID; for Helm/Kustomize it is Kind/name. A format producing conflicting
native identities must reject or diagnose its input rather than return duplicate source identifiers.

Unknown paths are relative to attributes, composed of dot-separated nonempty key segments. Traversal uses Struct
objects, respects case, and never interprets paths as Recommendation protobuf fields. Any missing ancestor makes a
leaf absent; a scalar/list/null ancestor is not traversable and invalidates a nested path. A present leaf, including
null, is invalid. Lists have no indexing syntax. Literal dotted keys must be represented by an unresolved containing
property when possible; no escaping convention is introduced. Empty unknown-property collections are valid.

## Warning and result

| Entity / field | Type | Validation / relationship |
| --- | --- | --- |
| ParseWarning.severity | enum | INFO=1 or WARNING=2; UNSPECIFIED=0 and unknown numbers invalid |
| ParseWarning.code | string | Stable and nonempty; example resource_skipped |
| ParseWarning.message | string | Nonempty human-readable explanation |
| ParseWarning.source_id | string | Required for resource_skipped; optional for document-wide diagnostics |
| ParseResponse.resources | ParsedResource collection | No nil entries; may be empty |
| ParseResponse.warnings | ParseWarning collection | No nil entries; may be empty |
| ParseResponse.expires_at | optional Timestamp | Valid timestamp; advisory, may already be expired |

A warning source need not refer to returned resources because skipped resources are absent. Generic validators can
require source_id for the documented resource_skipped code, but cannot infer undocumented semantic skip codes;
providers remain responsible for explaining every actual skip.

A nil response is invalid. An empty nonnil response is valid. No response field changes the advertised limits.
Deterministic results mean stable resource/warning values and ordering for unchanged input/options/dependencies;
expiration is excluded from equality because it may advance with time. No retention or persistent state is introduced.

## Processing lifecycle and ownership

Discovery validates advertisement. Decoding enforces input exclusivity. Shared request validation checks syntax,
mode/format support, options size and content size. Provider validation checks option keys, reads/contains path input,
and interprets its format. Response validation checks structure, descriptors, identities, unknown paths and warnings.
The host may then price returned descriptors and optionally cache the result using input, options and parser identity.

Failed requests produce service errors. Successful requests may contain warnings and zero resources. Parsing does
not mutate infrastructure, calculate costs, merge parsers, or store cross-request state.
