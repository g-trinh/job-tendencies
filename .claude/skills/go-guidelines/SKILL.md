---
name: go-guidelines
description: >
  DEPENDENCY — never triggered directly. This skill is imported exclusively by
  software-development. It contains Go-specific implementation rules: package
  design, struct and constructor conventions, interface discipline, dependency
  injection wiring, context propagation, concurrency patterns, error handling,
  and Go-specific code smells. Do NOT use standalone. Trigger only when
  software-development explicitly says "read go-guidelines/SKILL.md".
---

# Go Implementation Guidelines

> **This file is a dependency of `software-development`.** It is never used standalone.
> After reading this file, return to `software-development` and continue from where you left off.

---

## 1. Package Design

- Design packages around **domain concepts**, not technical roles.
  Prefer `internal/domain/order` over `internal/models` or `internal/entities`.
- A package's exported API is its contract — keep it minimal and intentional.
  Every exported symbol is a promise; every unnecessary export is a liability.
- If two packages import each other, one of them is wrong.
  Resolve the cycle with an interface, a shared type package, or by merging the packages.
- Avoid `init()` — it makes dependency order implicit and untestable.
  Express startup dependencies explicitly in `main`.

---

## 2. Structs & Constructors

- Always provide a constructor for non-trivial structs: `func NewOrder(...) (*Order, error)`.
- Constructors **must** validate inputs and return an error if invariants are not met.
  Never return a zero-value struct that is invalid by design.
- Use unexported fields by default. Expose only what callers genuinely need.
- Prefer value receivers for small, immutable types; pointer receivers for everything else.
  **Never mix** receiver types on the same type — pick one and be consistent.
- **Return a single result struct** instead of multiple return values (beyond `error`).
  Two or more non-error return values should be grouped into a named struct.
  This makes call sites clearer, avoids positional mistakes, and is easier to extend.
  Exception: stdlib-style pairs like `(value, ok)` or `(n, err)` are fine.

```go
// Good — constructor enforces invariants
func NewOrder(customerID string, lines []OrderLine) (*Order, error) {
    if customerID == "" {
        return nil, fmt.Errorf("customerID is required")
    }
    if len(lines) == 0 {
        return nil, fmt.Errorf("order must have at least one line")
    }
    return &Order{customerID: customerID, lines: lines}, nil
}

// Bad — caller can create an invalid Order silently
order := Order{lines: nil}
```

---

## 3. Interfaces

- Define interfaces in the **consumer** package, not the implementor's package.
  The consumer owns the contract; the implementor satisfies it implicitly.
- Accept interfaces, return concrete types (in most cases).
- Name single-method interfaces with the `-er` suffix: `OrderStorer`, `EventPublisher`, `MetricRecorder`.
- Do not define an interface until you have **two concrete implementations** or a **test boundary** that requires one.
  Premature interfaces add indirection without value.
- Keep interfaces small. A 7-method interface is almost always a sign the abstraction is wrong.

```go
// consumer package (internal/app/order)
type OrderRepository interface {
    Save(ctx context.Context, o *order.Order) error
    FindByID(ctx context.Context, id order.ID) (*order.Order, error)
}

// implementor package (internal/infra/postgres)
// No interface declared here — it satisfies the consumer's interface implicitly.
type orderRepository struct { db *sql.DB }
func (r *orderRepository) Save(ctx context.Context, o *order.Order) error { ... }
func (r *orderRepository) FindByID(ctx context.Context, id order.ID) (*order.Order, error) { ... }
```

---

## 4. Dependency Injection

- Wire **all** dependencies in `cmd/<entrypoint>/main.go`.
  This is the only place allowed to import and instantiate concrete implementations.
- Pass dependencies explicitly via constructors — never via global variables,
  package-level `var`, or `sync.Once` singletons outside of `main`.
- For large services, add a `wire.go` (or equivalent) to `cmd/` to document the dependency
  graph in one place. Manual wiring is preferred over reflection-based frameworks unless the
  graph is genuinely large (20+ dependencies).
- `main.go` should read like a wiring diagram, not business logic:

```go
func main() {
    cfg := config.Load()
    db  := postgres.Connect(cfg.DSN)

    orderRepo    := postgres.NewOrderRepository(db)
    eventBus     := kafka.NewEventBus(cfg.Kafka)
    orderService := order.NewService(orderRepo, eventBus)
    orderHandler := http.NewOrderHandler(orderService)

    http.ListenAndServe(cfg.Addr, orderHandler)
}
```

---

## 5. Context Propagation

- Every function that performs I/O, calls an external service, or could block **must** accept
  `context.Context` as its **first parameter**, named `ctx`.
- **Never** store a `context.Context` in a struct field. Contexts are request-scoped; structs are not.
- Propagate cancellation actively: check `ctx.Err()` in loops and between I/O operations.
- Never ignore a cancelled context and continue doing work — respect the caller's intent.

```go
// Good
func (r *orderRepository) FindByID(ctx context.Context, id order.ID) (*order.Order, error) {
    rows, err := r.db.QueryContext(ctx, query, id)
    ...
}

// Bad — context ignored, long-running work cannot be cancelled
func (r *orderRepository) FindByID(id order.ID) (*order.Order, error) {
    rows, err := r.db.QueryContext(context.Background(), query, id)
    ...
}
```

---

## 6. Concurrency

- Prefer **channels** for ownership transfer between goroutines.
  Prefer **mutexes** for protecting shared state accessed from multiple goroutines.
- Every goroutine must have a **clear owner** responsible for its lifecycle (start and stop).
  Anonymous goroutines launched with no owner are a maintenance hazard.
- Never launch a goroutine without a way to wait for it to finish.
  Use `sync.WaitGroup` for fire-and-forget fans, `golang.org/x/sync/errgroup` for
  goroutines that can fail.
- Document intentionally accepted data races with a comment explaining why they are safe.
  Never leave a race silent.

```go
// Good — goroutines are owned, lifecycle is managed
g, ctx := errgroup.WithContext(ctx)

g.Go(func() error { return processOrders(ctx, orders) })
g.Go(func() error { return publishEvents(ctx, events) })

if err := g.Wait(); err != nil {
    return fmt.Errorf("processing batch: %w", err)
}
```

---

## 7. Error Handling

These rules extend the base-guidelines error handling section with Go-specific patterns.

- **Always wrap errors with call-site context** using `fmt.Errorf("doing X: %w", err)`.
  The error message should read as a chain: `"placing order: saving to db: connection refused"`.
- **Sentinel errors** for cases callers must branch on: `var ErrNotFound = errors.New("not found")`.
  Define them in the `domain` package, not in `infra`.
- **Typed errors** when callers need structured data from the error:
  ```go
  type ValidationError struct {
      Field   string
      Message string
  }
  func (e *ValidationError) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Message) }
  ```
- Use `errors.Is` for sentinel comparison, `errors.As` for typed error extraction.
  Never compare error strings.
- At service boundaries (HTTP handler, gRPC server, message consumer), translate domain errors
  to transport errors. Do not leak internal error details to external callers.

---

## 8. Go Doc

- **Always add Go docs** whenever it is possible
- **Content** must give context and example, along with an explanation of the utility of the subject. Do not explain the algorithm

___

## 9. Go-Specific Code Smells

| Smell | Fix |
|---|---|
| Function takes a `bool` to switch behaviour | Split into two named functions |
| Constructor has 5+ parameters | Introduce a `Config` or functional options pattern |
| `interface{}` / `any` used as a shortcut | Define a typed interface or use generics |
| `return err` with no wrapping | `return fmt.Errorf("context: %w", err)` |
| Logic duplicated in handler and test | Extract to a domain function; test the function |
| Package imported only for `init()` side effect | Make the dependency explicit in `main` |
| Goroutine launched with no lifecycle owner | Use `errgroup` or explicit `WaitGroup` |
| Global variable used as dependency | Inject via constructor |
| `time.Sleep` in tests | Use channels, `require.Eventually`, or a fake clock |
| Interface defined next to its implementation | Move to the consumer package |
| Error type compared with `==` instead of `errors.Is` | Use `errors.Is` / `errors.As` |
| `context.Background()` passed mid-call-chain | Propagate `ctx` from the caller |
