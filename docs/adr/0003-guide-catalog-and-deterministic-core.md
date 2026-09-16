# ADR-0003: Guide Catalog Architecture, Deterministic Core, and Targeted Fact Pack Retrieval

- **Status**: accepted
- **Date**: 2026-09-16
- **Authors**: Antigravity Pair Programming Team

## Context

Aetox's UI companion ("Guide") assists users in navigating the desktop application, discovering features, understanding architectural principles, and operating complex workflows. Prior to this decision, the guide relied on an ad-hoc 157-element map (`map.ts`), static walk geometry with magic numbers, and an all-or-nothing LLM tool-calling pattern where the model received the full list of elements or risk hallucinating.

This introduced several architectural tensions:
1. **Token Bloat & Latency**: Sending 157 UI target descriptions to local models (e.g., `qwen3:8b` on Ollama) added high token overhead, increased latency (~10–15s warmup), and diluted the model's attention.
2. **System Architectural Blindspots**: The guide knew about individual buttons (e.g. `composer.stance`), but lacked first-class knowledge of high-level architectural concepts (e.g., Stance §106, Desk Separation §86, Memory System §279, Brain Providers §279, Capability Room §265).
3. **DOM Coupling & Fragility**: Navigation routing previously lacked precondition verification, leading to failed clicks or guides pointing at unmounted DOM nodes.
4. **Viewport Clipping**: Magic-number figure placement caused clipping and visual occlusion on narrow viewports (<768px).

## Decision

We establish a unified, 3-tier Guide Architecture governed by the following decisions:

1. **Catalog as Single Source of Truth**:
   - UI elements (`TargetCatalogEntry`) and system architectural concepts (`ConceptCatalogEntry`) are unified in `desktop/frontend/src/lib/guide/catalog/`.
   - Each entry defines its action safety policy, architecture decision reference (e.g. `§106`, `§279`), prerequisites, and multilingual names/explanations (`th`, `en`, `zh`).
   - The legacy `GUIDE_MAP` is dynamically derived 1:1 from `CATALOG_TARGETS` to preserve 100% backward compatibility with existing tests and Go IPC contracts.

2. **Deterministic Core Before Model Adapter**:
   - A deterministic runtime (`GuideCore`) handles 100% of navigation, element discovery (`where`), keyword intent matching (`find`), fact retrieval (`describe`), route progression, and safe clicking (`press`).
   - The system guarantees full functionality without any LLM configured.
   - When an LLM session is active, the guide constructs a **Targeted Fact Pack** (< 2KB, top 3 facts, up to 10 visible targets) rather than dumping all 157 catalog entries into the context.

3. **Public DOM Semantic Contract**:
   - The guide interacts with the application strictly through public DOM attributes (`data-guide`, `data-guide-state`, `disabled`, visibility).
   - Component internal stores and framework states are never directly inspected by the guide, preserving clean component encapsulation.

4. **Unified Geometry Single Source of Truth & Adaptive Docking**:
   - Geometry constants and placement algorithms live in `geometry.ts`.
   - Figure placement dynamically measures real bubble dimensions (`sayEl.getBoundingClientRect()`) and switches from `beside` mode to a docked callout banner (`docked`) on viewports under 640px or when target occlusion cannot be avoided.

5. **Strict Action Safety Gate**:
   - The frontend window enforces safety policies directly in `press()`. Action policies strictly bar destructive suffixes (`.send`, `.stop`, `.save`, `.delete`, `.toggle`) from automated clicks, returning a standardized refusal message.

## Options Considered

- **Option A (Pure LLM Agent with Full DOM Injection)**:
  - Inject the complete accessibility tree and full 157-item catalog into every model prompt.
  - *Rejected*: Caused token exhaustion, 10s+ latency per turn on local LLMs, and made the guide completely non-functional when no model provider was active or when network connection dropped.

- **Option B (Static Tooltip / Tour Script Only)**:
  - Hardcode fixed walkthrough steps with simple static tooltips without model integration or concept search.
  - *Rejected*: Unable to answer user queries, explain system rationale ("why was this designed this way?"), or adapt dynamically when users navigate away.

- **Option C (Unified Catalog, Deterministic Core & Targeted Fact Pack Adapter - Chosen)**:
  - Catalog-driven single source of truth, deterministic engine for zero-model guarantee, and compact targeted context packs for model turns.
  - *Chosen*: Combines sub-millisecond deterministic response times, zero token waste, offline resilience, and deep model-assisted conversational explanations when desired.

## Consequences

- **Positive**:
  - **Zero-Model Guarantee**: 100% of guide navigation, search, and explanation functions work offline without an AI model.
  - **Token Efficiency**: Model turns consume < 2KB context instead of large multi-kilobyte dumps, drastically improving inference speed on local Ollama models.
  - **Explainability**: Guide explains architectural decisions citing canonical spec references (§86, §106, §265, §279).
  - **Robust Layout**: Eliminates viewport clipping and target occlusion across wide (1920x1080), tablet (768x1024), and mobile/narrow (375x667) screens.
  - **Safety Guarantee**: Destructive UI actions are impossible for the model or automated scripts to trigger.

- **Negative / Costs**:
  - Catalog entries must be kept in sync when new features or buttons are added.
  - Geometry calculations require reading DOM layout rects via `requestAnimationFrame` / measurement hooks.

- **Mitigations**:
  - Automated tests (`guideMap.test.ts`, `localeGuard.test.ts`, `guideCatalog.test.ts`) enforce that every `data-guide` attribute in the codebase has a corresponding catalog entry and complete locale translations in Thai, English, and Chinese.
  - Unsafe suffixes are barred by automated test assertion in `guideMap.test.ts`.
