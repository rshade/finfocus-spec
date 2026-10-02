# Research: Cross-Batch Scoring Sessions

## Decision 1: Request-level session id

- **Decision**: Add `session_id` to the request, echoed in the response.
- **Rationale**: One concept fixes both group ids and token stability. The scorer stays stateless.
  Group ids can be derived from `(session_id, duplicate key)`, so hosts merge by equality.
- **Alternatives**: A stable namespace alone leaves tokens unfixed and gives no way to scope ids per
  operation. A dedupe RPC over identifiers adds a service, a capability, and a second round trip,
  and identifiers-only grouping loses the action type signal the scorer groups on.

## Decision 2: Derivation, not storage

- **Decision**: The contract requires determinism, not a hash. The mock uses SHA-256 over
  session id, resource id, and action type, truncated to 16 hex characters.
- **Rationale**: A stateful scorer would need expiry and cross-replica sharing. Determinism
  survives restarts and load balancing.
- **Alternatives**: Scorer-issued tokens returned in batch 1 and sent back in batch 2 need a second
  field and state.

## Decision 3: Singleton relaxation

- **Decision**: Without a session the two-member rule stays. With a session a single-member group is
  valid.
- **Rationale**: In a cross-batch group a batch can hold one member. Clearing it would defeat the
  feature.

## Decision 4: Echo to detect support

- **Decision**: The response echoes `session_id`. A mismatch is invalid; an empty echo for a
  non-empty request means the scorer declines sessions.
- **Rationale**: Without it a host cannot tell whether ids are mergeable. Hosts that see no echo
  treat ids as per-response.

## Decision 5: Cached items

- **Decision**: Document, do not add a field. An item absent from the session is never grouped; the
  host re-sends cached items when grouping matters.
- **Rationale**: Score caching is docs-only in 556. A "known ids" side channel would leak
  identifiers and adds surface the issue does not ask for.

## Decision 6: Session id grammar

- **Decision**: 1 to 128 printable ASCII characters (0x21 to 0x7E) when set.
- **Rationale**: Keeps ids loggable and hashable; avoids embedding payload in it.
