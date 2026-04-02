# Agent Guidelines for FinFocus Spec

## Build/Test Commands

- **All tests**: `make test` or `go test ./...`
- **Single test**: `go test -run TestName ./path/to/package`
- **Validate all**: `make validate` (tests + linting + schemas)
- **Generate protobuf**: `make generate`
- **Lint Go**: `make lint` or `golangci-lint run`
- **Validate schemas**: `npm run validate:schema && npm run validate:examples`

## Code Style

- **Go version**: 1.25.5
- **Formatting**: `goimports` + `golines` (120 char lines)
- **Linting**: 120+ linters via golangci-lint (see `.golangci.yml`)
- **Imports**: Standard library first, then third-party, then local
- **Naming**: CamelCase (protobuf snake_case → Go CamelCase)
- **Functions**: Small, composable; add `String()` for new enums
- **Error handling**: Check all errors; use table-driven tests
- **Tests**: Separate `_test` packages; cover error paths
- **Markdown/YAML**: Follow `.markdownlint.json` and `.yamllint`

## Key Rules

- Never edit generated code (`sdk/go/proto/`, `bin/buf`)
- Run `make validate` before commits
- Use conventional commits: `feat:`, `fix:`, `chore:`
- Sanitize secrets in examples

## Active Technologies

- Go 1.25.5 + Go standard library (errors, fmt) (047-validation-error-integration)
- N/A (in-memory validation only) (047-validation-error-integration)

- Documentation (Markdown, JSON) - No code implementation + markdownlint-cli2 (for validation), JSON schema validators (001-migration-docs)
- Files - Repository documentation files (MIGRATION.md, CHANGELOG.md, README.md) (001-migration-docs)

- Go 1.25.5+ + gRPC, Protobuf, Buf, ConnectRPC, RS/Zerolog (035-project-rename-finfocus)
- N/A (Protobuf definitions and Go SDK) (035-project-rename-finfocus)

- Go 1.25.5 + gRPC, protobuf, buf v1.32.1 (034-sdk-polish)

- Go 1.25.5 (as specified in go.mod) + gRPC/protobuf (existing), buf v1.32.1 (existing for proto management) (001-sdk-polish-release)
- N/A (SDK does not manage persistent storage) (001-sdk-polish-release)

- Go 1.25.5 (per go.mod) + gRPC, protobuf, buf v1.32.1 (001-get-budgets-rpc)
- JSON Schema (Draft 2020-12) for PricingSpec and BudgetSpec validation (001-get-budgets-rpc)

## Recent Changes

- 001-get-budgets-rpc: Added Go 1.25.5 (per go.mod) + gRPC, protobuf, buf v1.32.1

## Common Issues & Solutions

- Issue: `make lint` and `make validate` may time out on this project.
  Solution: Run `golangci-lint run` directly for faster Go linting results, or `make test` for unit tests.

## Workflow Optimizations

- For CodeRabbit fixes: Always verify `git log` and file content first; reviews may reference older
  commits that have already been fixed by subsequent pushes.

## Project-Specific Patterns

- `pluginsdk.Serve`: Tests dealing with `Serve` should prefer injecting a `net.Listener` (via
  `ServeConfig.Listener`) rather than relying on `Port` and `listenOnLoopback` to avoid race
  conditions and ensure predictable port binding.

## CI Variance

GitHub Actions CI runners exhibit high-performance variability (up to 2x for sub-microsecond
benchmarks). Benchmark alerts are informational and should not fail builds (`fail-on-alert: false`).
The alert threshold is set to 150% to reduce noise.
