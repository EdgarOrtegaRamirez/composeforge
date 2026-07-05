# AGENTS.md

## Project Overview
ComposeForge is a Docker Compose analysis and management toolkit in Go. It provides CLI tools for validating, analyzing, diffing, and merging Docker Compose files.

## Architecture
- `pkg/parser/` — YAML parsing, data models, memory size parsing
- `pkg/validator/` — Security checks, best practices, cross-reference validation
- `pkg/analyzer/` — Dependency graph analysis, topological sort, network topology
- `pkg/differ/` — Semantic diff between compose files with field-level tracking
- `pkg/merger/` — Multi-file merge with override semantics
- `cmd/composeforge/` — CLI entry point using Cobra

## Key Algorithms
1. **Topological Sort (Kahn's Algorithm)** — `pkg/analyzer/analyzer.go:topologicalSort()`
2. **DFS Cycle Detection** — `pkg/analyzer/analyzer.go:hasCycle()`
3. **BFS Depth Calculation** — `pkg/analyzer/analyzer.go:calculateDepths()`
4. **Semantic Diff** — `pkg/differ/differ.go:diffService()`
5. **Deep Merge** — `pkg/merger/merger.go:Merge()`

## Building & Testing
```bash
# Build
go build -o composeforge ./cmd/composeforge

# Test all
go test ./tests/...

# Test specific package
go test ./tests/parser/...
go test ./tests/validator/...

# Vet
go vet ./...
```

## Adding New Validation Rules
1. Add rule to `pkg/validator/validator.go`
2. Use `addIssue(result, severity, category, message, service)` 
3. Add test in `tests/validator/validator_test.go`

## Adding New Analysis Features
1. Add to `pkg/analyzer/analyzer.go`
2. Update `ComposeAnalysis` struct if needed
3. Add test in `tests/analyzer/analyzer_test.go`

## Code Style
- Use `gofmt` for formatting
- Run `go vet` before committing
- Tests use standard `testing` package
- Error handling: return errors, don't panic
