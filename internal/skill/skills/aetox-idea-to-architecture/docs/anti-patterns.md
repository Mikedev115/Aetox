# Anti-Patterns

Avoid these patterns when using this skill.

## Existing-System Drift

Do not inspect or describe an existing codebase. If the user points to current
files, services, repositories, current architecture, boundary review, or mixed
existing/proposed context, use `aetox-architect`.

## Treating Proposals as Implementation

Do not write as though a proposed module, workflow, data model, or integration
exists. Prefer "Proposed" and "would" until the user approves the direction.

## Diagram-First Work

Do not make diagrams the main output. Mermaid is optional editable source for
review, not the product of the skill.

## Document Bloat

Do not produce exhaustive enterprise architecture documentation for a raw idea.
Keep the proposal reviewable.

## Premature Service Splitting

Do not split modules into services unless scale, ownership, deployment,
compliance, or integration boundaries justify it.

## Hidden Assumptions

Do not bury assumptions inside prose. Put them in an assumptions section or table.

## Ignoring Approval Points

Do not let major product, compliance, payment, tenancy, or security decisions
pass as default choices.

## Bad / Corrected Examples

### Premature Service Splitting

Bad:

> The system will use microservices: booking-service, payment-service, and
> notification-service.

Problem:

The raw idea did not justify separate deployable services. This turns a
proposal into an implementation shape too early.

Corrected:

> Proposed modules for v1:
>
> - Booking module
> - Payment integration boundary
> - Notification module
>
> Service split: Requires approval. Not recommended for v1 unless scale,
> deployment ownership, or team ownership requires it.

### Proposal Written as Implementation

Bad:

> Create `src/payments/stripe.ts`, add the webhook handler, and migrate the
> bookings table.

Problem:

Implementation planning started before the architecture proposal was approved.

Corrected:

> Proposed payment boundary:
>
> - Payment provider: Requires approval
> - Webhook handling: Proposed
> - Booking/payment reconciliation: Unknown
>
> Safe next step: approve the payment boundary before implementation planning.

### Hidden Assumption

Bad:

> Members receive discounts automatically.

Problem:

The discount rule is unlabeled and may affect pricing, roles, reporting, and
payment behavior.

Corrected:

> Assumed: members receive automatic discounts at checkout.
>
> Impact if wrong: pricing workflow and payment reconciliation may change.
>
> Approval status: Requires approval.
