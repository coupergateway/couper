# GitHub Copilot Instructions

Instructions for GitHub Copilot when working with the Couper codebase.

## Project Overview

Couper is a lightweight API gateway written in Go. Configuration is done via HCL files. See the shared documentation for details:

- **[Architecture](../docs/ARCHITECTURE.md)**: Configuration flow, key packages, the Inline pattern, request processing pipeline
- **[Go Guidelines](../docs/GO_GUIDELINES.md)**: Error handling, context usage, interface design, DDD patterns

## Build Commands

```bash
make build          # Build binary with race detection
make test           # Run all tests
make coverage       # Run tests with coverage
make generate       # Generate code (assets, error types)
make generate-docs  # Generate documentation from struct tags
```

Run single test:
```bash
go test -v -timeout 90s -race -count=1 ./path/to/package -run TestName
```

## Key Patterns

### The Inline Interface

Config structs implement `config.Inline` to separate parse-time attributes (struct fields) from runtime attributes (evaluated per-request from `Remain hcl.Body`). See [Architecture - Inline Pattern](../docs/ARCHITECTURE.md#the-inline-interface-pattern-core-architecture).

### Testing Philosophy

Tests are **real integration tests** - no mocking. They:
- Open real ports and listeners
- Use actual HTTP clients/servers
- Load test fixtures from `server/testdata/*.hcl`

Mocks are rare exceptions for special unit test cases only.

## Code Style

Follow the [Go Guidelines](../docs/GO_GUIDELINES.md):

- **Errors**: Always handle, wrap with `%w`, use `errors.Is`/`errors.As`
- **Context**: First argument, named `ctx`, never store in structs
- **Interfaces**: Keep small, accept interfaces return structs, define at consumer site
- **Dependencies**: Pass explicitly, avoid global state

## Documentation

- Struct tags generate docs at https://docs.couper.io
- Use `hcl`, `docs`, `type`, `default` tags
- Run `make generate-docs` after changing config structs
