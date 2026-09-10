# Test-Driven Implementation (TDD) Execution Reference

TDD is the core engine of resilient code. It forces observable behavior through public interfaces before any implementation exists.

---

## 1. What Makes a Test Worth Keeping

A good test reads like an executable specification. It verifies behavior through public interfaces, surviving complete internal rewrites.
- **Good**: `TestCustomerCanCheckoutWithValidCart` verifies that calling `Checkout()` produces an `Invoice` and reduces inventory.
- **Bad**: `TestCheckoutCallsValidateCustomerInternalMethod` tests whether an unexported private helper was invoked with certain arguments.

---

## 2. Seams: The Only Place Tests Live

A **seam** is the public boundary where external behavior is observed without reaching inside.
- **Rule**: Test only at pre-agreed seams. Never test at an unconfirmed seam.
- **Rule**: If the seam requires mocks, mock only at system boundaries (e.g. real external payment gateway, network socket), never internal module collaborators.

---

## 3. Strict Anti-Patterns to Prevent

| Anti-Pattern | Description | Danger / Tell | Required Rule |
|---|---|---|---|
| **Implementation-Coupled** | Mocks internal classes/functions or tests unexported methods. | Breaks during ordinary refactoring even when user behavior is unchanged. | Test strictly through public exported interfaces. |
| **Tautological Tests** | Assertion recomputes expected values using the same formula as the implementation (`expect(add(a,b)).toBe(a+b)`). | Passes by construction and can never catch real calculation bugs. | Expected values must originate from an independent source of truth (known literal, worked example, spec). |
| **Horizontal Test Dumping** | Writing 20 tests upfront before writing any implementation. | Tests imagined behavior rather than reality; commits to brittle test shapes. | Work in vertical cycles: One test ➡️ One minimal implementation ➡️ Repeat. |

---

## 4. The Execution Loop Rules

1. **Red Before Green**: Write the failing test first. Run it. Verify it fails specifically because the capability is missing, not due to compilation or syntax errors.
2. **Minimal Implementation**: Write only enough code to make the failing test green. Do not anticipate future tests or build speculative helpers.
3. **Run Cadence**:
   - Run the single focused test file repeatedly during the loop.
   - Run typechecking / linter periodically.
   - Run the full test suite once at the end of the ticket.
4. **Handoff to Code Review**: Once all acceptance criteria pass, route the diff to `aetox-code-review` before final commit.
