# Research: Omitted Fields

- **Decision**: dotted proto field names resolved with `protoreflect`. **Rationale**: no second
  schema of field names to keep in sync, and it follows `FocusFieldNames` parity thinking.
  **Alternatives**: a hand-maintained allowlist of names (drifts); JSON names (camelCase differs by
  transport); map keys and indexes in paths (hosts clear whole maps, per-key paths add no value yet).
- **Decision**: advisory list, request-wide. **Alternatives**: per-recommendation lists (larger
  payloads, no consumer need: core allowlists are operator-wide).
- **Decision**: strict validation of unknown paths. **Alternatives**: ignore unknown (hides host
  bugs); the scorer may skip the validator to be lenient.
- **Decision**: field 5. Verified against `../finfocus-spec-574` (request 4, response 5) read-only.
