# Domain Context & Glossary Format

A project's domain model defines its ubiquitous language. Ambiguous terms lead to buggy assumptions and conflicting code architectures.

---

## CONTEXT.md Template

Create `CONTEXT.md` at the repository root for single-context projects.
Create it lazily: as soon as the first domain term is clarified or contested.

```markdown
# [Context Name]

[One or two sentences describing what business domain or system boundary this context covers and why it exists.]

## Language

**[Canonical Term]**:
[1-2 sentences defining what the entity or concept IS in this domain. Do not describe implementation details or database columns.]
_Avoid_: [Synonyms, deprecated terms, or ambiguous aliases that should not be used]

**Order**:
A formal request by a customer to purchase one or more catalog items.
_Avoid_: Cart, CheckoutSession, PurchaseRequest

**Customer**:
An individual or business entity that holds an account and executes transactions.
_Avoid_: User, Client, Buyer, Account

**Invoice**:
An immutable financial record requesting payment for fulfilled orders.
_Avoid_: Bill, Statement, PaymentNotice
```

---

## Rules for Domain Glossary

1. **Be Opinionated**: When multiple synonyms exist for one concept, establish a single canonical term. Explicitly list confusing alternatives under `_Avoid_`.
2. **Define What It IS, Not What It Does**:
   - Good: "A Token represents an authenticated session grant."
   - Bad: "A Token is a JWT stored in localStorage and validated by AuthMiddleware via Bearer header."
3. **No Implementation Details**:
   - `CONTEXT.md` is NOT a technical spec, scratchpad, or database schema. It is a pure domain language dictionary.
4. **Project-Specific Terms Only**:
   - General software engineering patterns (e.g. "mutex", "debounce", "adapter", "cache") do not belong here. Only domain-specific concepts belong.

---

## Multi-Context Repositories (`CONTEXT-MAP.md`)

When a codebase spans multiple distinct bounded contexts (e.g. billing, logistics, identity), place a `CONTEXT-MAP.md` at the root and give each context its own `CONTEXT.md`.

```markdown
# Context Map

## Contexts

- [Identity](./services/identity/CONTEXT.md): Handles credential verification and organization memberships.
- [Billing](./services/billing/CONTEXT.md): Handles invoicing, tax calculation, and payment gateways.
- [Inventory](./services/inventory/CONTEXT.md): Tracks physical stock levels and warehouse allocations.

## Relationships & Contracts

- **Identity -> Billing**: Identity emits `OrganizationCreated` events; Billing provisions financial ledgers.
- **Billing -> Inventory**: Inventory reserves stock on `InvoicePaid` confirmation.
- **Shared Entities**: `OrganizationId` and `Money` are shared value objects across boundaries.
```
