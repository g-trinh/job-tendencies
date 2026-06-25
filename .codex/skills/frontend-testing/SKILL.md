---
name: frontend-testing
description: >
  Frontend-specific testing skill. Covers test-driven development, unit tests
  for hooks and utilities, component tests with Testing Library, app-level
  integration tests, and e2e tests with Playwright. Trigger on phrases like:
  "write tests for this component", "add unit tests for this hook", "test this
  acceptance criterion in the UI", "write a Playwright test", "TDD this
  feature" — AND on feature-implementation phrases like "implement this
  component", "build this screen", "add this store/hook" whenever the work
  touches renderer/UI code, since production code is not written without a
  failing test driving it. Loads testing-strategy as a dependency for
  AC-mapping and test type selection. Can be triggered independently to
  retrofit tests onto existing code, or automatically by frontend-development
  at step 9.
---

# Frontend Testing

> **STOP — before writing or editing ANY production file** (component, hook,
> store, util, screen) as part of a feature implementation: if no failing
> test exists yet for the behavior you're about to write, this skill applies
> — do not proceed with `Write`/`Edit` on production code first "to add
> tests after." Load it now, even if you arrived here via a different skill
> (frontend-development, software-development) that didn't explicitly hand
> off to it.
>
> **STOP — load this dependency before reading further.**
>
> Invoke the Skill tool now: `Skill(skill: "testing-strategy")`
> Do not read past this line until the skill has been loaded.
> Then return here and continue with the next section.

---

## 1. Test Type Map

| Layer | Tool | What it covers |
|---|---|---|
| Pure functions, custom hooks | Vitest + `renderHook` | Logic, transformations, hook state |
| Components | Vitest + Testing Library | Rendering, interactions, states |
| Cross-component journeys (no real backend/browser needed) | Vitest + Testing Library, full app tree | Multi-screen flows, store wiring |
| Critical user journeys that need a real browser | Playwright | AC-level end-to-end flows |

---

## 2. Test-Driven Workflow

Tests are written **before** the production code that satisfies them — red, green, refactor:

1. **Red** — pick the next uncovered AC (or edge case). Write a failing test named after it (testing-strategy §4 naming convention). Confirm it fails for the *expected* reason (missing behavior, not a typo).
2. **Green** — write the minimum production code to make that test pass. Resist adding behavior the current test doesn't demand.
3. **Refactor** — clean up implementation and test alike, keeping the suite green. No behavior change in this step.

Repeat per AC/scenario — small cycles, not one giant test file followed by one giant implementation.

### Rules
- No production code without a failing test driving it. If you catch yourself writing implementation first, stop, delete it, write the test, then restore the implementation to make it pass.
- Each cycle covers one AC or one edge case — not a batch. Batching defeats the point: you lose the "did this fail for the right reason" signal.
- Refactor on green only. A refactor that breaks a test is a behavior change wearing a refactor's name.
- Test doubles (MSW handlers, boundary stubs — §8/§9) are written alongside the failing test, not retrofitted after the implementation exists.

---

## 3. Unit Tests — Hooks and Utilities

### What to test
- Custom hooks: all return values, status transitions, error states.
- Pure utility functions: every branch, edge case, transformation.
- State reducers: each action type, each invalid input.

### Pattern — describe / it

```ts
describe('useOrder', () => {
  it('returns loading state while fetching', async () => {
    server.use(http.get('/orders/:id', () => new Promise(() => {}))) // never resolves
    const { result } = renderHook(() => useOrder('o-1'))
    expect(result.current.status).toBe('loading')
  })

  it('returns order data on success', async () => {
    server.use(http.get('/orders/:id', () => HttpResponse.json(orderFixture)))
    const { result } = renderHook(() => useOrder('o-1'))
    await waitFor(() => expect(result.current.status).toBe('success'))
    expect(result.current.data).toEqual(orderFixture)
  })

  it('returns error state when request fails', async () => {
    server.use(http.get('/orders/:id', () => HttpResponse.error()))
    const { result } = renderHook(() => useOrder('o-1'))
    await waitFor(() => expect(result.current.status).toBe('error'))
  })
})
```

### Rules
- Test the hook's public API (return values, callbacks) — never its internal state or `useRef` values.
- Mock network requests at the HTTP level with MSW (§8). Mock non-HTTP boundaries — IPC, native bridges — at the boundary (§9). Never mock `fetch`, `axios`, or module internals.
- Use `renderHook` for hooks that require a React context. Provide the minimum context needed.

---

## 4. Component Tests

### What to test
- Every state from the design spec: loading, empty, error, populated, disabled.
- Every user interaction: click, type, submit, keyboard navigation.
- Conditional rendering: what appears / disappears based on props or data.

### Pattern — one describe block per AC or state group

```tsx
// AC: order form validates required fields before submission
describe('OrderForm — validation', () => {
  it('shows error when customer name is empty on submit', async () => {
    render(<OrderForm onSuccess={vi.fn()} />)

    await userEvent.click(screen.getByRole('button', { name: 'Place order' }))

    expect(screen.getByText('Customer name is required')).toBeInTheDocument()
  })

  it('does not call onSuccess when form is invalid', async () => {
    const onSuccess = vi.fn()
    render(<OrderForm onSuccess={onSuccess} />)

    await userEvent.click(screen.getByRole('button', { name: 'Place order' }))

    expect(onSuccess).not.toHaveBeenCalled()
  })
})

// AC: user sees confirmation after successful order placement
describe('OrderForm — successful submission', () => {
  it('shows confirmation message after order is placed', async () => {
    server.use(http.post('/orders', () => HttpResponse.json({ id: 'o-1' })))
    render(<OrderForm onSuccess={vi.fn()} />)

    await userEvent.type(screen.getByLabelText('Customer name'), 'Alice')
    await userEvent.click(screen.getByRole('button', { name: 'Place order' }))

    await screen.findByText('Order placed successfully')
  })
})
```

### Query priority (highest to lowest)

1. `getByRole` — prefer always (accessible, matches what screen reader users experience)
2. `getByLabelText` — for form fields
3. `getByText` — for static content
4. `getByPlaceholderText` — last resort for inputs without labels
5. `getByTestId` — only when no semantic query is possible; flag it for design review

Never query by CSS class, element tag, or internal component name.

### Rules
- Use `userEvent` over `fireEvent` — it simulates real browser input sequences.
- Use `findBy*` (async) when waiting for elements that appear after async operations.
- Mock network requests with MSW (§8); mock non-HTTP boundaries at the boundary (§9). Never mock module imports of data-fetching hooks.
- Assert on content and roles, never on CSS styles.

---

## 5. Accessibility Assertions

Accessibility is part of correctness — include it in component tests, not as a separate suite.

```tsx
it('form submit button is accessible', () => {
  render(<OrderForm onSuccess={vi.fn()} />)

  const button = screen.getByRole('button', { name: 'Place order' })
  expect(button).toBeEnabled()
  expect(button).toBeVisible()
})

it('error message is associated with its input', async () => {
  render(<OrderForm onSuccess={vi.fn()} />)
  await userEvent.click(screen.getByRole('button', { name: 'Place order' }))

  const input = screen.getByLabelText('Customer name')
  expect(input).toHaveAccessibleDescription('Customer name is required')
})
```

---

## 6. App-Level Integration Tests

A lighter alternative to e2e for journeys that don't actually require a real browser — render the full app tree (or a major feature root) with Testing Library in jsdom and drive it like a user, store wiring and all.

### When to use this instead of e2e
- The AC spans multiple screens/components but its outcome is observable in the rendered DOM (no real navigation, cookies, storage, native dialogs, multi-tab behavior involved).
- The app has no real backend, or all backend access goes through a boundary you already mock for component tests (MSW / §9 stubs).
- You want the confidence of a full-flow test without the cost (browser binary, flakiness, runtime) of Playwright.

### Pattern — AC-named, full-tree render

```tsx
// AC: surviving a boss advances the gauntlet to the next encounter without re-drafting
describe('Gauntlet — progression', () => {
  it('advances from Moloch to Sythara on narrow victory, keeping the drafted roster', async () => {
    render(<App />)

    await draftRoster(['Tank A', 'Healer B', 'DPS C'])
    await userEvent.click(screen.getByRole('button', { name: 'Begin attempt' }))
    await forceOutcome('narrow-victory') // test-only seam — see §9 for boundary stubs

    await userEvent.click(screen.getByRole('button', { name: /Onward to Sythara/ }))

    expect(screen.queryByRole('heading', { name: 'Draft your roster' })).not.toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Sythara the Plaguebound' })).toBeInTheDocument()
  })

  it('returns to drafting on defeat', async () => {
    render(<App />)

    await draftRoster(['Tank A', 'Healer B', 'DPS C'])
    await userEvent.click(screen.getByRole('button', { name: 'Begin attempt' }))
    await forceOutcome('defeat')

    await userEvent.click(screen.getByRole('button', { name: 'Muster again' }))

    expect(screen.getByRole('heading', { name: 'Draft your roster' })).toBeInTheDocument()
  })
})
```

### Rules
- Render the real app tree with real stores — don't mock internal components or hooks. The point is to test the wiring between them.
- Mock only at the same external boundaries component tests use (HTTP via MSW, IPC/native via §9 stubs). Anything that would be mocked in a component test stays mocked here.
- Name tests after the cross-cutting AC, not after the screens involved — `"advances ... without re-drafting"`, not `"DraftScreen then OutcomeScreen then DraftScreen"`.
- If a journey needs real browser mechanics (URL changes, `localStorage`, focus across page loads) to be observable, it belongs in e2e (§7) — don't force it into jsdom.

---

## 7. E2E Tests — Playwright

E2E tests cover critical user journeys whose outcome is only observable in a real browser. Keep the suite small — prefer §6 when jsdom is sufficient.

### What belongs in e2e
- Login / authentication flows that depend on real cookies/redirects.
- Core conversion journeys (checkout, signup, onboarding) spanning real navigation.
- Any AC whose outcome is only observable end-to-end (URL redirect after login, browser storage persistence, multi-tab/multi-window behavior).

### What does NOT belong in e2e
- Validation errors — covered by component tests.
- Loading/error states — covered by component tests with MSW.
- Cross-screen journeys observable in jsdom — covered by app-level integration tests (§6).
- Edge cases — too slow and brittle at e2e level.

### Pattern — Page Object + AC-named test

```ts
// tests/e2e/checkout.spec.ts

// AC: authenticated user can complete checkout with a valid cart
test('authenticated user completes checkout', async ({ page }) => {
  const cartPage = new CartPage(page)
  const checkoutPage = new CheckoutPage(page)

  await cartPage.goto()
  await cartPage.addItem('SKU-001')
  await cartPage.proceedToCheckout()

  await checkoutPage.fillShipping(shippingFixture)
  await checkoutPage.submit()

  await expect(page.getByRole('heading', { name: 'Order confirmed' })).toBeVisible()
})

// AC: unauthenticated user is redirected to login before checkout
test('unauthenticated user is redirected to login', async ({ page }) => {
  await page.goto('/checkout')
  await expect(page).toHaveURL('/login?redirect=/checkout')
})
```

### Page Object rules
- One Page Object per page or major UI section.
- Methods are named after user actions: `addItem`, `proceedToCheckout`, `submit`.
- Never assert inside Page Object methods — assertions stay in the test.
- Store Page Objects in `tests/e2e/pages/`.

---

## 8. MSW Handler Organisation (HTTP boundary)

Mock handlers are a shared fixture — keep them organised.

```
tests/
├── msw/
│   ├── handlers/
│   │   ├── orders.ts      ← handlers for /orders/* routes
│   │   ├── auth.ts        ← handlers for /auth/* routes
│   │   └── index.ts       ← re-exports all handlers
│   └── server.ts          ← setupServer(...handlers)
```

- Default handlers return happy-path responses.
- Override in individual tests with `server.use(...)` for error or edge-case scenarios.
- Reset after each test with `afterEach(() => server.resetHandlers())`.

---

## 9. Non-HTTP Boundary Mocking (IPC, native bridges, WebSocket)

Not every app talks to the outside world over HTTP. Electron apps cross through `contextBridge`/`window.api`, some use WebSockets, native shells expose their own bridges. MSW can't intercept any of that — but the same principle applies: **stub at the boundary you don't own, never the module that wraps it.**

### Pattern — one fake bridge implementation, shared and overridable

```ts
// tests/setup.ts
import { fakeElectronApi } from './fakes/electronApi'

beforeEach(() => {
  vi.stubGlobal('api', fakeElectronApi.create())
})

afterEach(() => {
  vi.unstubAllGlobals()
})
```

```ts
// tests/fakes/electronApi.ts
// A realistic in-memory implementation of the preload-exposed `window.api` contract.
// Override specific methods per test the same way MSW handlers are overridden with `server.use`.
export const fakeElectronApi = {
  create: (overrides: Partial<ElectronApi> = {}): ElectronApi => ({
    saveRun: vi.fn().mockResolvedValue({ ok: true }),
    loadRoster: vi.fn().mockResolvedValue(rosterFixture),
    ...overrides,
  }),
}
```

```ts
// in a test
vi.stubGlobal('api', fakeElectronApi.create({
  loadRoster: vi.fn().mockRejectedValue(new Error('disk error')),
}))
render(<App />)
await screen.findByText('Could not load your roster')
```

### Rules
- Write **one** fake per bridge/contract, matching its real shape (same method names, same async behavior). Reuse it across every suite — don't hand-roll a different shape per test file.
- Override individual methods per test, the same way `server.use(...)` overrides MSW handlers for one scenario.
- Never `vi.mock` the module that *calls* `window.api` — that hides the integration between your code and the contract. Stub the global the contract is exposed on instead.
- If the bridge contract changes (preload adds/renames a method), the fake must change with it — keep it next to the contract's type definition, not buried in a test file.

---

## 10. Quality Gate

Before presenting any test code:

```
a. Read frontend-eslint/SKILL.md    → lint + format check
b. Read frontend-tsc/SKILL.md       → TypeScript strict check
```

Fix all findings. A test with type errors is not a test.

---

## 11. Common Frontend Testing Smells

| Smell | Fix |
|---|---|
| Feature implementation goes straight from architecture/plan to `Write`/`Edit` on stores/components/screens, never invoking this skill | Stop, load `frontend-testing` now (see entry gate at top) before writing any more production code — even mid-feature |
| Production code written before its test | Delete it, write the failing test first (§2), restore it to make the test pass |
| One giant test file after a finished implementation | Retrofit AC-by-AC in red/green/refactor cycles going forward |
| `getByTestId` used as the primary query | Replace with `getByRole` or `getByLabelText` |
| Asserting on CSS class or `style` attribute | Assert on visible content or ARIA roles |
| `jest.mock` / `vi.mock` on a data-fetching module or bridge module | Use MSW (§8) or a boundary stub (§9) instead |
| `waitFor` wrapping a synchronous assertion | Remove `waitFor`; use `findBy*` for async |
| Page Object method contains `expect` | Move assertions to the test; PO is actions only |
| E2E test covering a validation error or jsdom-observable journey | Move to a component test (§4) or app-level integration test (§6) |
| `fireEvent.click` instead of `userEvent.click` | Use `userEvent` for real interaction simulation |
| `screen.debug()` left in committed tests | Remove before committing |
