# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Couper is a lightweight API gateway written in Go. Configuration is done via HCL (HashiCorp Configuration Language) files. It provides access control, request/response manipulation, and observability features.

## Shared Documentation

Detailed documentation is maintained in shared files:

- **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)**: Configuration flow, key packages, the Inline pattern, request processing pipeline, HCL evaluation context, improvement notes
- **[docs/GO_GUIDELINES.md](docs/GO_GUIDELINES.md)**: Error handling, context usage, interface design, DDD patterns, Go best practices

## Build and Development Commands

```bash
# Build the binary (with race detection)
make build

# Run with a configuration file
./couper run -f public/couper.hcl

# Run all tests
make test

# Run a single test
go test -v -timeout 90s -race -count=1 ./path/to/package -run TestName

# Run tests in a specific package
go test -v -timeout 90s -race -count=1 ./server/...

# Run tests with coverage
make coverage

# Generate code (assets and error types)
make generate

# Generate documentation (writes to docs/website/content/configuration/block/*.md)
make generate-docs

# Serve documentation locally
make serve-docs

# Update dependencies
make update-modules
```

## Testing Philosophy

Tests are **real integration tests** - no mocking of HTTP, networking, or Couper internals:

- Open real ports and listeners
- Use actual HTTP clients and servers
- Load test fixture HCL files from `server/testdata/`
- Exercise the full request/response pipeline

**Mocks are rare exceptions** - only for special unit test situations where isolation is absolutely necessary.

### Test Structure

- **`server/http_integration_test.go`**: Main integration test suite
- **`server/testdata/*.hcl`**: Test configuration fixtures
- **`internal/test/backend.go`**: Test backend server (real HTTP server)
- **`internal/test/`**: Helper utilities for test setup and assertions

## Quick Reference

### Key Patterns

| Pattern | Location | Purpose |
|---------|----------|---------|
| Inline Interface | `config/inline.go` | Separates parse-time vs runtime attributes |
| Modifier Attributes | `config/meta/attributes.go` | Shared request/response modifiers |
| HCL Evaluation | `eval/context.go` | Expression evaluation with request context |
| Sequence Execution | `handler/endpoint.go` | Parallel/sequential backend requests |
| Error Hierarchy | `errors/` | Error types for `error_handler` blocks |

### Code Generation

- **Error types**: Generated from `errors/generate/types.go` via `go generate`
- **Documentation**: Generated from struct tags via `make generate-docs`
- **Assets**: Embedded via `go:embed`

### Custom Libraries

The project uses forked versions of `hashicorp/hcl/v2` and `zclconf/go-cty` (see `go.mod` replace directives).

## Essential Reading

Before making significant changes, read:

1. **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** - Especially the Inline pattern section
2. **[docs/GO_GUIDELINES.md](docs/GO_GUIDELINES.md)** - Error handling and interface design
3. **`config/inline.go`** - Core interfaces
4. **`eval/http.go`** - Request/response context application
5. **`handler/endpoint.go`** - Request orchestration
