# Contract: Session Semantics

1. A host chooses one `session_id` per operation and sends it with every batch of that operation.
   It never reuses a `session_id` for another operation or another pseudonymization key.
2. A host using `IDENTIFIER_MODE_PSEUDONYMIZED` draws one key per session. A resource then has one
   token in every batch.
3. A scorer that supports sessions echoes `session_id` and derives each `duplicate_group_id` from
   the session id and the group's duplicate key, with no state between calls.
4. Equal non-empty `duplicate_group_id` values across the responses of one session mean the same
   group. Ids from different sessions carry no meaning together.
5. A response may hold one member of a group when `session_id` is set.
6. A recommendation never sent in any batch of the session is never grouped. A host that wants a
   cached item grouped sends it in a batch and discards its other scores.
7. `IDENTIFIER_MODE_OMITTED` leaves grouping unreliable, with or without a session.
8. `duplicate_group_id` is still never cached per recommendation.
9. Requests without `session_id` behave as before.
