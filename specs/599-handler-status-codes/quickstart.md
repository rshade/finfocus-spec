# Quickstart: Validate Handler Status Codes

Run from the repository root.

```bash
go test -run 'TestHandlerStatus|TestStubEmbedding' -v ./sdk/go/pluginsdk/
go test ./sdk/go/pluginsdk/ ./sdk/go/testing/
```

Expected:

1. Each of the eight RPCs returns a handler's `InvalidArgument` code and message, over gRPC and Connect.
2. A plain handler error still returns `Internal` with the old message and without the error text.
3. A stub-embedding plugin and a plain plugin return the same code and response from the seven
   optional RPCs, and the stub-embedding plugin passes `RPCCorrectness_GetBudgetsRPC`.
4. Existing tests that expect `Internal` for plain errors pass without edits.

See [contracts/server-errors.md](contracts/server-errors.md).
