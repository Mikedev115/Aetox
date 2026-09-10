# Good Tests vs Bad Tests Reference

## 1. Good Tests (Observable Behavior at Seams)

Tests verify external behavior through public interfaces, not internal mechanics.

```go
// GOOD: Tests observable behavior via public contract
func TestCustomerCanCheckoutWithValidCart(t *testing.T) {
    svc := NewOrderService(testRepo)
    order, err := svc.Checkout(context.Background(), validCart)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if order.Status != "confirmed" {
        t.Errorf("status = %q; want confirmed", order.Status)
    }
}
```

### Key Characteristics:
- Tests what the user or calling system actually cares about.
- Uses public APIs and exported contracts only.
- Survives internal refactoring without changes to the test.
- Asserts on outcome values, not function call counts.

---

## 2. Bad Tests (Implementation Coupling & Tautologies)

### 2.1 Implementation Coupling
```go
// BAD: Asserts internal private helper call order or state
func TestCheckoutCallsValidateCustomerInternalMethod(t *testing.T) {
    // If we refactor Checkout to validate customer via middleware or inline,
    // this test breaks even though checkout works perfectly!
}
```

### 2.2 Tautological Tests (Self-Fulfilling Assertions)
```go
// BAD: Recomputes the expected value using the code's own logic
func TestCalculateTotal(t *testing.T) {
    items := []Item{{Price: 10}, {Price: 5}}
    // Re-calculating sum using same loop
    expected := 0
    for _, it := range items { expected += it.Price }
    if CalculateTotal(items) != expected { // Always passes or fails identically!
        t.Fatal("math mismatch")
    }
}

// GOOD: Expected value is an independent, known ground truth
func TestCalculateTotal(t *testing.T) {
    items := []Item{{Price: 10}, {Price: 5}}
    if got := CalculateTotal(items); got != 15 {
        t.Errorf("got %d; want 15", got)
    }
}
```
