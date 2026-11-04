# Quickstart: Verifying Decline Log Levels

```bash
cd sdk/go/pluginsdk
go test -run 'HandlerLogLevel|DeclineLevel|FaultLevel' -v .
go test -bench BenchmarkHandlerLogLevel -benchmem .
```

Expected: every decline code row passes with no `"level":"error"` entry; every fault row has one;
the benchmark reports 0 allocs/op.

Manual check: serve a plugin whose `Supports` returns `status.Error(codes.InvalidArgument, "x")`,
call it, and confirm the log shows no `"level":"error"` entry (raise to debug to see the decline entry).
