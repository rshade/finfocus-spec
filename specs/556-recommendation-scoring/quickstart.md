# Quickstart: Validating Recommendation Scoring

Prerequisites: Go per `go.mod`, and `npm ci` in `sdk/typescript` for the TypeScript check.

## 1. Contract compiles and is additive

```bash
make generate
buf lint
git archive origin/main proto buf.yaml | tar -x -C "$(mktemp -d)"   # baseline for buf breaking
```

Run `buf breaking --against <baseline-dir>`. It passes because no existing message changes.

## 2. Go SDK

```bash
go test -race ./sdk/go/testing/ ./sdk/go/pluginsdk/ -run 'Scor|Capabilit'
```

## 3. Serve a scorer

```go
plugin := &myPlugin{BasePlugin: pluginsdk.NewBasePlugin("my-scorer")} // implements ScoreRecommendations
err := pluginsdk.Serve(ctx, pluginsdk.ServeConfig{Plugin: plugin})
```

## 4. Conformance

```go
func TestMyScorer(t *testing.T) {
    plugintesting.RunScorerConformance(t, &MyScorer{})
}
```

## 5. TypeScript

```bash
cd sdk/typescript/packages/client && npx vitest run test/recommendation-scorer.test.ts
```
