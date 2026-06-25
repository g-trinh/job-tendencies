---
name: go-testing
description: >
  Go-specific testing skill. Covers unit tests, integration tests, and contract
  tests for Go code using the table-driven test cases format. Trigger on phrases
  like: "write tests for this Go code", "add unit tests", "write integration
  tests", "test this use case", "cover this acceptance criterion". Loads
  testing-strategy as a dependency for AC-mapping and test type selection.
  Can be triggered independently to retrofit tests onto existing code, or
  automatically by software-development at step 8.
---

# Go Testing

> **STOP — load this dependency before reading further.**
>
> Invoke the Skill tool now: `Skill(skill: "testing-strategy")`
> Do not read past this line until the skill has been loaded.
> Then return here and continue with the next section.

---

## 1. The Test Cases Format

All Go tests use the **test cases table** as the standard structure, even for a single case.
This makes adding scenarios trivial and keeps assertion logic in one place.

### Canonical structure

```go
func TestSubject_Condition(t *testing.T) {
    t.Parallel()

    cases := []struct {
        name    string
        // inputs
        // expected outputs
        wantErr error
    }{
        {
            name: "returns error when cart is empty",
            // ...
            wantErr: order.ErrEmptyCart,
        },
        {
            name: "creates order when cart has items",
            // ...
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()

            // arrange
            // act
            // assert
        })
    }
}
```

### Rules for the test cases table

- Every row name follows the naming convention from `testing-strategy` §4.
- Inputs and expected outputs are explicit fields — no magic, no shared state between cases.
- Arrange, act, assert are separated by blank lines inside `t.Run`. No comments needed if the
  sections are visually distinct.
- `t.Parallel()` at both function and subtest level unless the test has a specific reason not to
  (e.g. modifies a shared resource, uses `t.Setenv`).
- Use `require` for fatal assertions (wrong type, nil pointer, unexpected error that makes
  further assertions meaningless). Use `assert` for non-fatal ones.

---

## 2. Unit Tests

Unit tests cover domain logic and application services with no real I/O.

### What to test
- All domain entities: constructors (valid and invalid inputs), state transitions, invariants.
- All application service use cases: each success path, each error path, each edge case.
- All pure functions with conditional logic.

### Test doubles in unit tests
Use fakes defined in the same `_test` file or in a `testutil` package within the domain:

```go
type fakeOrderRepo struct {
    orders map[order.ID]*order.Order
    err    error // inject error to simulate failure
}

func (f *fakeOrderRepo) Save(_ context.Context, o *order.Order) error {
    if f.err != nil {
        return f.err
    }
    f.orders[o.ID()] = o
    return nil
}

func (f *fakeOrderRepo) FindByID(_ context.Context, id order.ID) (*order.Order, error) {
    if f.err != nil {
        return nil, f.err
    }
    o, ok := f.orders[id]
    if !ok {
        return nil, order.ErrNotFound
    }
    return o, nil
}
```

### Full unit test example

```go
func TestOrderService_PlaceOrder(t *testing.T) {
    t.Parallel()

    cases := []struct {
        name    string
        cmd     order.PlaceOrderCommand
        repoErr error
        want    *order.Order
        wantErr error
    }{
        {
            name:    "creates order for valid command",
            cmd:     order.PlaceOrderCommand{CustomerID: "cust-1", Lines: validLines()},
            want:    &order.Order{/* expected fields */},
        },
        {
            name:    "returns ErrInvalidCustomer when customer ID is empty",
            cmd:     order.PlaceOrderCommand{CustomerID: "", Lines: validLines()},
            wantErr: order.ErrInvalidCustomer,
        },
        {
            name:    "returns ErrEmptyCart when no lines provided",
            cmd:     order.PlaceOrderCommand{CustomerID: "cust-1", Lines: nil},
            wantErr: order.ErrEmptyCart,
        },
        {
            name:    "propagates repository error",
            cmd:     order.PlaceOrderCommand{CustomerID: "cust-1", Lines: validLines()},
            repoErr: errors.New("db unavailable"),
            wantErr: order.ErrStorageFailure,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()

            repo := &fakeOrderRepo{orders: map[order.ID]*order.Order{}, err: tc.repoErr}
            svc  := order.NewService(repo, &fakeEventBus{})

            got, err := svc.PlaceOrder(context.Background(), tc.cmd)

            require.ErrorIs(t, err, tc.wantErr)
            assert.Equal(t, tc.want, got)
        })
    }
}
```

---

## 3. Integration Tests

Integration tests verify infra adapters against real dependencies.

### What to test
- Repository implementations against a real database.
- HTTP clients against a real or test server.
- Message consumers against a real broker.

### Setup

- Use `testcontainers-go` to start external dependencies in `TestMain`.
- Integration tests live in `_test` packages (e.g. `package order_test`) to enforce public API testing.
- Tag all integration tests: `//go:build integration`

```go
//go:build integration

package postgres_test

import (
    "testing"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestMain(m *testing.M) {
    container, _ := postgres.RunContainer(context.Background(), ...)
    defer container.Terminate(context.Background())
    os.Exit(m.Run())
}
```

### Integration test cases example

```go
//go:build integration

func TestOrderRepository_Save(t *testing.T) {
    cases := []struct {
        name    string
        order   *order.Order
        wantErr error
    }{
        {
            name:  "persists a valid order",
            order: fixtures.ValidOrder(),
        },
        {
            name:    "returns error on duplicate ID",
            order:   fixtures.ValidOrder(), // same ID as already saved
            wantErr: postgres.ErrDuplicateKey,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {

            repo := postgres.NewOrderRepository(testDB)

            err := repo.Save(context.Background(), tc.order)

            require.ErrorIs(t, err, tc.wantErr)
        })
    }
}
```

---

## 4. Contract Tests

Contract tests verify the boundary between two bounded contexts or services.

### Consumer-driven pattern
The consumer defines what it expects from the provider. The provider verifies it satisfies those expectations.

- Store contract definitions alongside the **consumer** code.
- For async messaging: assert the event schema produced by the publisher matches what the consumer parses.

```go
// In the consumer package (e.g. internal/app/shipment)
func TestOrderPlacedEventContract(t *testing.T) {
    cases := []struct {
        name    string
        payload []byte
        want    OrderPlacedEvent
        wantErr bool
    }{
        {
            name:    "parses minimal valid event",
            payload: []byte(`{"order_id":"o-1","customer_id":"c-1"}`),
            want:    OrderPlacedEvent{OrderID: "o-1", CustomerID: "c-1"},
        },
        {
            name:    "returns error when order_id is missing",
            payload: []byte(`{"customer_id":"c-1"}`),
            wantErr: true,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()

            got, err := ParseOrderPlacedEvent(tc.payload)

            if tc.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tc.want, got)
        })
    }
}
```

---

## 5. Acceptance Tests

Acceptance tests directly verify ACs. They sit at the integration or e2e level
and are the last line of defence before shipping.

### Pattern
- One test function per AC or tightly related group of ACs.
- Named precisely after the AC: `TestAC_UserCannotCheckoutWithEmptyCart`.
- Use the test cases table even here — different setups for the same AC scenario.

```go
// AC: a guest user cannot access the order history page
func TestAC_GuestCannotViewOrderHistory(t *testing.T) {
    cases := []struct {
        name       string
        authHeader string
        wantStatus int
    }{
        {
            name:       "unauthenticated request is rejected",
            authHeader: "",
            wantStatus: http.StatusUnauthorized,
        },
        {
            name:       "expired token is rejected",
            authHeader: "Bearer " + expiredToken,
            wantStatus: http.StatusUnauthorized,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()

            req := httptest.NewRequest(http.MethodGet, "/orders", nil)
            if tc.authHeader != "" {
                req.Header.Set("Authorization", tc.authHeader)
            }
            rec := httptest.NewRecorder()

            handler.ServeHTTP(rec, req)

            assert.Equal(t, tc.wantStatus, rec.Code)
        })
    }
}
```

---

## 6. Quality Gate

Before presenting any test code, run:

```
a. go test ./... -run <TestName>          → ensure tests compile and pass
b. Read go-vet/SKILL.md                  → static analysis
c. Read go-golangci-lint/SKILL.md        → linting
```

Fix all findings before presenting. A test that does not compile is not a test.

---

## 7. Common Go Testing Smells

| Smell | Fix |
|---|---|
| `time.Sleep` to wait for async behaviour | Use `require.Eventually` or a fake clock |
| Global test state mutated across cases | Move state into each `tc` struct; use `t.Cleanup` |
| Mock of an interface you own | Write a fake instead |
| Single monolithic test with many `if` branches | Split into test cases table rows |
| `t.Fatal` / `t.Error` called in a helper without `t.Helper()` | Add `t.Helper()` at top of helper |
| Test file in the same package as production code for infra adapters | Use `_test` package suffix |
| Assertions on internal struct fields | Test through the public API only |
