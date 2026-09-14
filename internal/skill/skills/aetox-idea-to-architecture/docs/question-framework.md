# Question Framework

Ask questions that change architecture decisions. Avoid discovery theater.

## Priority Rule

Ask boundary-changing questions first: ownership, tenancy, roles, data
sensitivity, payment flow, compliance, integration ownership, and deployment
boundaries.

Ask detail questions only after critical unknowns are resolved or explicitly
deferred with approval.

## Problem Framing Questions

Use these sharpening questions before architecture proposal work when target
user, problem, constraint, or success signal is unclear:

| Focus | Ask When | Example |
| --- | --- | --- |
| Target user | The audience is broad or generic. | "Who is this for first? Recommended default: pick the narrowest user with the strongest pain." |
| Problem | The request names a solution before a pain point. | "What problem should this solve if the proposed solution changes?" |
| Constraint | The idea has no tradeoff or boundary. | "What must this avoid: cost, time, complexity, compliance risk, or operational burden?" |
| Success signal | The desired outcome is subjective. | "What observable result would make this proposal successful?" |
| Prior attempt | Existing workaround or failure mode is unknown. | "What has been tried before, and why did it fall short?" |
| Why now | Urgency or timing may change scope. | "Why does this need to happen now rather than later?" |

## How Might We Rules

Use a How Might We frame as an intent check:

```txt
How might we [desired outcome] for [target user] without [constraint]?
```

A good frame is:

- Actionable enough to guide architecture boundaries.
- Broad enough to allow multiple product or system directions.
- Constrained by a real tension, limit, or success signal.
- Not solution-embedded.
- Not so broad that ownership, data, or workflow boundaries cannot be reasoned
  about.

## Question With Recommendation

Do not ask open-ended questions that simply transfer the decision burden to the
user. When safe, pair each architecture-impacting question with:

- Recommended default
- Why it is the default
- Alternative
- Impact
- Approval status

Recommended defaults are not approvals. Mark them as `Assumed`, `Proposed`, or
`Requires approval` until the user accepts them.

## Question Types

| Type | Ask When | Example |
| --- | --- | --- |
| Goal | The outcome is ambiguous. | "Is the priority faster booking, higher utilization, or lower admin effort? Recommended default: faster booking first for v1." |
| Actor | Users or permissions are unclear. | "Which actor can cancel a paid booking? Recommended default: member self-cancel before cutoff, admin override anytime." |
| Boundary | Ownership could split across modules or services. | "Should payments be owned inside this product or delegated to a provider? Recommended default: provider-owned payment processing." |
| Data | Persistence, privacy, or reporting needs are unclear. | "Which customer data must be retained after cancellation? Recommended default: retain booking audit data, minimize payment data." |
| Workflow | Order of operations affects risk. | "Should payment be authorized before or after court availability is held? Recommended default: short availability hold before payment." |
| Integration | External systems are likely. | "Does this need calendar, payment, email, or club integration? Recommended default: payment and email only for v1." |
| Compliance | Rules may constrain design. | "Are there tax, refund, age, or membership rules? Recommended default: mark policy as approval-required before build." |
| Scale | Capacity changes architecture. | "Is this for one venue, many venues, or a marketplace? Recommended default: one venue first unless multi-venue is stated." |

## Must-Answer Questions

Treat a question as must-answer when the wrong assumption could:

- Expose sensitive data.
- Break payment or refund behavior.
- Create legal or compliance risk.
- Change tenancy or ownership boundaries.
- Make the proposed architecture materially more expensive.
- Invert the user experience or operating model.

## Safe-to-Assume Questions

Assume and label when:

- The answer only changes naming.
- The answer affects a replaceable integration.
- The answer can be revised before implementation.
- The user requested a first-pass proposal.

## Question Budget

Use the grill depth selected in `docs/workflow.md`:

- Low: ask 2-3 questions maximum.
- Medium: ask 4-6 questions.
- High: keep questioning until all boundary-changing unknowns are resolved or
  explicitly deferred with approval.

If lower-priority detail questions remain after the grill depth budget, group
them under open questions and continue with labeled assumptions where safe.
