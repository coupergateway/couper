# Go Guidelines

Go best practices and patterns for Couper development.

## Error Handling

### Always Handle Errors

Never ignore errors. Every error must be explicitly handled or returned.

```go
// Bad
data, _ := json.Marshal(v)

// Good
data, err := json.Marshal(v)
if err != nil {
    return fmt.Errorf("marshaling config: %w", err)
}
```

### Wrap Errors with Context

Use `%w` verb with `fmt.Errorf` to wrap errors, preserving the error chain for `errors.Is` and `errors.As`.

```go
// Bad - loses error chain
return fmt.Errorf("failed to load config: %v", err)

// Good - preserves error chain
return fmt.Errorf("loading config %q: %w", path, err)
```

### Use errors.Is and errors.As

Check error types using the standard library functions:

```go
// Check for specific error
if errors.Is(err, os.ErrNotExist) {
    // handle missing file
}

// Extract typed error
var pathErr *os.PathError
if errors.As(err, &pathErr) {
    // use pathErr.Path, pathErr.Op
}
```

### Don't Over-Wrap

Wrap errors when adding valuable context. Don't wrap at every level.

```go
// Bad - excessive wrapping
return fmt.Errorf("doThing: %w", fmt.Errorf("helper: %w", err))

// Good - wrap once with meaningful context
return fmt.Errorf("processing request %s: %w", reqID, err)
```

### Handle Errors Once

An error should be handled (logged, returned, recovered) exactly once. Logging and returning is handling twice.

```go
// Bad - handles twice
if err != nil {
    log.Error(err)
    return err
}

// Good - return and let caller decide
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

## Context Usage

### Context as First Argument

Always pass `context.Context` as the first parameter, named `ctx`:

```go
// Good
func ProcessRequest(ctx context.Context, req *Request) error {
    // ...
}

// Bad
func ProcessRequest(req *Request, ctx context.Context) error {
    // ...
}
```

### Never Store Context in Structs

Pass context explicitly to each function. Storing it prevents per-call deadlines and cancellation:

```go
// Bad
type Worker struct {
    ctx context.Context
}

// Good
type Worker struct {}

func (w *Worker) Process(ctx context.Context, data []byte) error {
    // ...
}
```

### Never Pass nil Context

Always provide a valid context:

```go
// Bad
doSomething(nil, data)

// Good
doSomething(context.Background(), data)
// Or better, propagate from caller
doSomething(ctx, data)
```

### Use context.Value Sparingly

`context.Value` is for request-scoped data crossing API boundaries, not for passing optional parameters:

```go
// Bad - optional parameters via context
ctx = context.WithValue(ctx, "debug", true)

// Good - explicit parameters
func Process(ctx context.Context, opts ProcessOptions) error
```

## Interface Design

### Keep Interfaces Small

Prefer single-method or few-method interfaces. They're easier to implement and compose:

```go
// Good - small, focused interface
type Reader interface {
    Read(p []byte) error
}

// Bad - fat interface
type Storage interface {
    Read(key string) ([]byte, error)
    Write(key string, data []byte) error
    Delete(key string) error
    List(prefix string) ([]string, error)
    Watch(key string) (<-chan Event, error)
    // ... many more methods
}
```

### Accept Interfaces, Return Structs

Functions should accept interfaces for flexibility but return concrete types for clarity:

```go
// Good
func NewHandler(backend http.RoundTripper) *Handler {
    return &Handler{backend: backend}
}

// Caller can pass any RoundTripper implementation
// Return type is concrete - no hidden costs
```

### Define Interfaces at Consumer Site

Define interfaces where they're used, not where they're implemented:

```go
// In handler package (consumer)
type Backend interface {
    RoundTrip(*http.Request) (*http.Response, error)
}

// Not in transport package (implementer)
```

### Don't Export Interfaces for Implementations

If you're providing both interface and implementation, the interface is often unnecessary:

```go
// Bad - interface just for the sake of it
type Service interface {
    DoThing() error
}
type serviceImpl struct{}

// Good - just export the struct
type Service struct{}
func (s *Service) DoThing() error { ... }
```

## Domain-Driven Design Patterns

### Separate Domain from Infrastructure

Keep business logic free from HTTP, database, and framework concerns:

```
domain/          # Pure business logic, no dependencies on infra
  ├── endpoint.go    # Domain types and rules
  └── backend.go     # Domain interfaces (ports)

adapters/        # Infrastructure implementations
  ├── http/          # HTTP handlers
  └── storage/       # Database implementations
```

### Define Ports in Domain

The domain defines interfaces (ports) for what it needs. Infrastructure implements them:

```go
// domain/repository.go - domain defines the port
type BackendRepository interface {
    Find(ctx context.Context, name string) (*Backend, error)
}

// adapters/storage/backends.go - infra implements
type ConfigBackendRepository struct {
    config *config.Couper
}

func (r *ConfigBackendRepository) Find(ctx context.Context, name string) (*Backend, error) {
    // implementation
}
```

### Use Value Objects

Encapsulate domain concepts with behavior, not just data:

```go
// Instead of primitive string
type Origin string

func ParseOrigin(s string) (Origin, error) {
    u, err := url.Parse(s)
    if err != nil {
        return "", fmt.Errorf("invalid origin: %w", err)
    }
    if u.Scheme == "" || u.Host == "" {
        return "", errors.New("origin requires scheme and host")
    }
    return Origin(u.Scheme + "://" + u.Host), nil
}

func (o Origin) Host() string { ... }
func (o Origin) Scheme() string { ... }
```

### Aggregates for Consistency Boundaries

Group related entities that must change together:

```go
// Endpoint is an aggregate root
type Endpoint struct {
    pattern  string
    proxies  []*Proxy
    requests []*Request
    response *Response
}

// Modifications go through the aggregate
func (e *Endpoint) AddProxy(p *Proxy) error {
    // validate, maintain invariants
}
```

### Repository Pattern

Encapsulate storage behind interfaces. This enables testing and swapping implementations:

```go
type AccessControlRepository interface {
    Get(ctx context.Context, name string) (AccessControl, error)
}

// In-memory for tests
type InMemoryACRepo struct {
    controls map[string]AccessControl
}

// Config-based for production
type ConfigACRepo struct {
    definitions *config.Definitions
}
```

### Anti-Corruption Layer

Protect domain from external models. Translate at boundaries:

```go
// external/oauth/client.go
type TokenResponse struct {
    AccessToken string `json:"access_token"`
    ExpiresIn   int    `json:"expires_in"`
}

// domain/auth/token.go
type Token struct {
    Value     string
    ExpiresAt time.Time
}

// adapters/oauth/translator.go - ACL
func ToToken(resp *external.TokenResponse) *domain.Token {
    return &domain.Token{
        Value:     resp.AccessToken,
        ExpiresAt: time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second),
    }
}
```

## General Best Practices

### Avoid Global State

Pass dependencies explicitly. Global state makes testing hard and hides dependencies:

```go
// Bad
var globalConfig *Config

func Handler(w http.ResponseWriter, r *http.Request) {
    // uses globalConfig
}

// Good
type Handler struct {
    config *Config
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // uses h.config
}
```

### Prefer Composition Over Inheritance

Go doesn't have inheritance. Use embedding and composition:

```go
// Composition via embedding
type LoggingBackend struct {
    http.RoundTripper
    logger *logrus.Entry
}

func (b *LoggingBackend) RoundTrip(req *http.Request) (*http.Response, error) {
    b.logger.Info("request", "url", req.URL)
    return b.RoundTripper.RoundTrip(req)
}
```

### Use Functional Options for Complex Construction

When constructors need many optional parameters:

```go
type Option func(*Server)

func WithTimeout(d time.Duration) Option {
    return func(s *Server) { s.timeout = d }
}

func WithLogger(l *logrus.Entry) Option {
    return func(s *Server) { s.logger = l }
}

func NewServer(addr string, opts ...Option) *Server {
    s := &Server{addr: addr, timeout: 30 * time.Second}
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

### Use Structured Logging

Use `log/slog` (Go 1.21+) or logrus with fields, not string formatting:

```go
// Bad
log.Printf("request failed: %s, status: %d", url, status)

// Good
logger.WithFields(logrus.Fields{
    "url":    url,
    "status": status,
}).Error("request failed")
```

### Document Exported Symbols

Every exported type, function, and method needs a doc comment:

```go
// Backend handles HTTP requests to upstream services.
// It implements http.RoundTripper with additional features
// like health checking and rate limiting.
type Backend struct {
    // ...
}

// RoundTrip executes a single HTTP transaction.
// It respects the context deadline and returns an error
// if the backend is unhealthy.
func (b *Backend) RoundTrip(req *http.Request) (*http.Response, error) {
    // ...
}
```
