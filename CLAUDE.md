# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

gh-pr-review is a GitHub CLI extension (Go) that provides inline PR review comment management from the terminal. All output is structured JSON optimized for LLM consumption. It wraps `gh api` for authentication and host configuration — the `gh` CLI must be installed and authenticated.

## Build & Development Commands

```bash
# Run all tests (CGO must be disabled)
CGO_ENABLED=0 go test ./...

# Run a single test
CGO_ENABLED=0 go test ./cmd -run TestReviewStartCommand_GraphQLOnly

# Run linter
CGO_ENABLED=0 golangci-lint run --timeout=5m

# Build binary
go build -o gh-pr-review ./
```

## Architecture

Three-layer design: **CLI → Service → API**

- **`cmd/`** — Cobra command definitions. Each command file (review.go, comments.go, threads.go) contains flag parsing and calls into service layer. `deps.go` holds a `apiClientFactory` function variable for dependency injection in tests.
- **`internal/`** — Service packages, each owning a domain:
  - `review/` — Start, add-comment, edit-comment, delete-comment, submit, pending/latest review operations
  - `comments/` — Reply to review threads
  - `threads/` — List, resolve, unresolve threads
  - `report/` — Aggregate reviews into structured reports (domain.go for types, builder.go for assembly, query.go for GraphQL)
  - `preview/` — Preview pending comments with code context
  - `resolver/` — Parse PR URL/number into owner/repo/number
- **`ghcli/`** — Thin wrapper around `gh api` subprocess. Defines the `API` interface (REST + GraphQL methods) that all services depend on.

## Key Patterns

**Dependency injection for testing**: `cmd/deps.go` exports `apiClientFactory` as a package-level var. Tests override it with a `commandFakeAPI` that stubs GraphQL/REST calls by invocation count. See `cmd/review_test.go` for the pattern.

**GraphQL-first**: All GitHub interactions use GraphQL via `gh api graphql`. REST is available but unused in practice. GraphQL queries live in each service's `query.go` file.

**JSON output**: All commands write structured JSON to stdout via `cmd/output.go`. Fields use `omitempty` — no null values in output.

**PR selector resolution**: `internal/resolver` parses either a PR number or full GitHub URL into (owner, repo, number). The `-R owner/repo` flag and positional PR number are resolved together.

## Testing Conventions

- Tests use `testify/assert` and `testify/require`
- Command tests create a `commandFakeAPI` struct, set up `graphqlFunc`/`restFunc` closures that switch on call count, then execute `root.Execute()` with `SetArgs`
- Test fixtures live in `cmd/testdata/`
- All tests run with `CGO_ENABLED=0`

## Module Path

The Go module path is `github.com/agynio/gh-pr-review` (note: this differs from the GitHub remote `EurFelux/gh-pr-review`).
