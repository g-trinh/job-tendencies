---
name: react-guidelines
description: >
  DEPENDENCY — never triggered directly. This skill is imported exclusively by
  frontend-development. It contains React-specific implementation rules: functional
  component model, hook design, React state APIs (useState, useReducer, useRef,
  Context), data fetching with React Query/SWR, React naming conventions, and
  React-specific code smells. Do NOT use standalone. Trigger only when
  frontend-development explicitly says "read react-guidelines/SKILL.md".
---

# React Implementation Guidelines

> **This file is a dependency of `frontend-development`.** It is never used standalone.
> After reading this file, return to `frontend-development` and continue from where you left off.

---

## 1. Component Model

- **Functional components only** — class components are legacy. Never introduce them.
- Avoid default exports for components — named exports make refactoring and importing explicit.
- A component file should read like a template, not a logic dump.
  Extract all non-trivial logic into custom hooks.

```tsx
// Good — focused, named export, logic in hook
export function OrderForm({ orderId }: OrderFormProps) {
  const { order, isLoading, submit } = useOrderForm(orderId)
  if (isLoading) return <Spinner />
  return <form onSubmit={submit}>...</form>
}

// Bad — logic and template mixed, default export hides the name at import
export default function Thing({ data, isLoading, onSave, onCancel, compact }: any) {
  const [order, setOrder] = useState(null)
  useEffect(() => { fetch(`/orders/${orderId}`).then(r => r.json()).then(setOrder) }, [orderId])
  return <form>...</form>
}
```

---

## 2. Hook Design

- Extract logic into **custom hooks** (`useX`). Custom hooks must start with `use` and
  follow the rules of hooks. If a function calls hooks internally, it must be a hook —
  name it accordingly.
- **Co-locate state** at the lowest common ancestor that needs it.
  Lift only when multiple siblings genuinely share it.
- Avoid `useEffect` for data derivation — use `useMemo` or compute inline.
  `useEffect` is for synchronisation with external systems, not for reacting to state.
- Never use `useEffect` with an empty dependency array as a constructor substitute
  unless the intent is truly "run once on mount" and that is documented.
- Always return a cleanup function from `useEffect` when the effect sets up a
  subscription, timer, or any resource that needs disposal.
- Memoize (`useMemo`, `useCallback`, `memo`) only when you have measured a performance
  problem. Premature memoisation adds complexity without benefit.

```tsx
// Good — hook exposes behaviour, not implementation details
export function useOrderForm(orderId: string) {
  const { data: order, isLoading } = useQuery(['order', orderId], () => fetchOrder(orderId))
  const submit = useCallback(async (values: OrderFormValues) => { ... }, [orderId])
  return { order, isLoading, submit }
}
```

---

## 3. State APIs

- **`useState`** for simple, independent values.
- **`useReducer`** for state with multiple sub-values or complex transitions.
  Model state machines explicitly for multi-step flows.
  A boolean `isLoading` + `isError` + `isSuccess` is a broken state machine — use a discriminated union.
- **`useRef`** for mutable values that do not trigger re-renders, and for
  referencing DOM nodes. Never use `document.querySelector` — use refs.
- **Context** for state shared across a subtree that would require prop-drilling
  more than two levels. Context is not a global state replacement — it adds
  re-render coupling. Keep context values stable (memoize the value object).

```tsx
// Good — context with stable value
const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const value = useMemo(() => ({ user, setUser }), [user])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
```

---

## 4. Data Fetching

- Do not fetch data in a component with raw `useEffect` + `fetch`.
  Use **React Query** (`@tanstack/react-query`) or **SWR** for server state —
  they handle caching, deduplication, background refresh, and error/loading states.
- Do not duplicate server state in client state.
  If it comes from an API, it lives in the query cache, not in `useState`.
- Use **global client state** (Zustand, Jotai) only for state that is genuinely
  global and not server-derived: auth session, theme, UI shell state.

```tsx
// Good — React Query manages server state
export function useOrder(orderId: string) {
  return useQuery({
    queryKey: ['order', orderId],
    queryFn: () => fetchOrder(orderId),
  })
}

// Bad — manual fetch with useEffect reinvents what React Query solves
function useOrder(orderId: string) {
  const [data, setData] = useState(null)
  useEffect(() => { fetch(`/orders/${orderId}`).then(r => r.json()).then(setData) }, [orderId])
  return data
}
```

---

## 5. React Naming Conventions

| Thing | Convention | Example |
|---|---|---|
| Context object | PascalCase + `Context` | `AuthContext` |
| Context provider component | PascalCase + `Provider` | `AuthProvider` |
| Custom context hook | `use` + PascalCase | `useAuth` |

These extend the framework-agnostic naming conventions in `frontend-guidelines`.

---

## 6. React Code Smells

| Smell | Fix |
|---|---|
| `useEffect` used to sync derived state | Compute inline or with `useMemo` |
| `useEffect` with no cleanup for subscriptions/timers | Always return a cleanup function |
| Fetching data in a component with raw `useEffect` | Use React Query / SWR |
| Direct DOM manipulation with `document.querySelector` | Use refs (`useRef`) |
| Context value object recreated on every render | Memoize the value with `useMemo` |
| Hook called conditionally or inside a loop | Move to the top level of the component/hook |
| Business logic inside a component body | Extract to a custom hook |
| Premature `useCallback`/`useMemo`/`memo` without a measured perf issue | Remove until a profiler shows a problem |
