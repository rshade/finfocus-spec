# Research: Include Dismissed Recommendations

## R1. Field number and type

**Decision**: `bool include_dismissed = 8` on `GetRecommendationsRequest`.

**Rationale**: `usage_profile` is field 7. A bool matches the issue text. Proto3 defaults it to
false, which is today's "omit plugin-side dismissals" behavior. Unknown fields are preserved and
ignored by older plugins, so a new host can send the field to an old plugin safely.

**Alternatives considered**: A new RPC for dismissed recommendations (that is rshade/finfocus#546).
An enum with unspecified/false/true (no third state is required). Reusing
`excluded_recommendation_ids` by leaving it empty (that cannot tell a plugin to reveal its own
dismissals).

## R2. Exclusion IDs win

**Decision**: `excluded_recommendation_ids` is applied after the dismissal filter, and it applies
when `include_dismissed` is true.

**Rationale**: Hosts today hide local dismissals by sending those IDs. If the new field overrode
that list, an old host that set both, or a new host that still sends the list, would show
recommendations the operator had excluded. rshade/finfocus keeps sending the list when
`--include-dismissed` is set so old plugins, which ignore the new field, keep hiding those IDs.

**Alternatives considered**: `include_dismissed` clears the exclusion list (breaks old plugins and
the explicit omit). Two flags, one for plugin dismissals and one for host exclusions (the exclusion
list is already that second control).

## R3. Where the filter runs

**Decision**: `pluginsdk.ApplyRecommendationVisibility` is the helper plugin authors call. The mock
plugin has its own copy of the same rules because `pluginsdk` imports `sdk/go/testing`. The mock
filters before pagination. The gRPC server does not filter; it does not know the plugin's dismissal
store.

**Rationale**: The import cycle is existing and documented on the mock's filter. Filtering before
pagination keeps `page_size` honest. Automatic filtering in `Server.GetRecommendations` would need
a dismissal store the server does not have.

**Alternatives considered**: A third package both sides import (new package for one function).
Filtering inside the server with a new plugin interface (larger than this field).

## R4. No status on the recommendation

**Decision**: The response message is unchanged. A recommendation returned because
`include_dismissed` is true is not labeled dismissed on the wire.

**Rationale**: Issue #545 adds a request field only. Labeling is the host's local dismissal record.

**Alternatives considered**: A `dismissed` bool on `Recommendation` (a second protocol change, not
in the issue).
