# Mocking Boundaries & Guidelines

Mock at **system boundaries** only:

- External network APIs (Payment gateways, webhook receivers, AI model endpoints)
- Operating system clocks / randomness (when deterministic output is required)
- External third-party CLI binaries or hardware drivers

### What You Must NEVER Mock:
- Your own structs, classes, or domain modules
- Internal collaborators within the same subsystem
- Any code or state that you control directly

---

## Designing for Mockability

At true system boundaries, design interfaces that make testing simple and clean:

### 1. Dependency Injection via Interfaces (Go Standard)
Pass boundary dependencies into constructors or functions instead of instantiating them internally:

```go
// GOOD: Boundary interface passed in
type PaymentGateway interface {
    Charge(ctx context.Context, amount int) error
}

type OrderService struct {
    gateway PaymentGateway
}

// BAD: Hardcoded internal instantiation
type OrderService struct{}
func (s *OrderService) ChargeOrder() {
    client := stripe.NewClient("key") // Impossible to test in isolation
}
```

### 2. Prefer Specific SDK-Style Interfaces over Generic Callers
- **Good**: Specific methods (`GetUser(id)`, `CreateOrder(data)`)
  - Each mock return has a deterministic, strongly-typed signature.
  - No conditional logic or URL parsing needed inside test fixtures.
- **Bad**: Generic `ExecuteRequest(method, url, body)`
  - Requires complex URL and payload branching inside the mock.
