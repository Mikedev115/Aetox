// The skill shelf: which published skill packs Aetox puts in front of a person.
//
// The สกิล tab of ห้องความสามารถ drew an empty state for as long as it existed,
// and the comment above it in Capability.svelte said why: "Filling it is a
// curation job — pick the skills Aetox is good at and write a line each — not
// an engineering one, and inventing entries to make the tab look finished
// would be the shelf breaking its own promise." This file is that curation job
// done, on the same terms mcpShelf.ts sets for servers. Nothing here was
// invented to fill a card.
//
// **Where these came from.** The first two community packs were curated from
// the owner's requested topics (long-running engineering work and marketing).
// The vendor packs added on 2026-09-17 come from their publishers' own docs
// and repositories: Replicate, Prisma and Dodo Payments. An official name is
// not enough by itself; each pack still has to pass the same install and
// overlap checks below.
//
// **The bar, which is mcpShelf.ts's bar with one word changed.**
//
//   1. **It teaches Aetox a job its own 44 bundled skills do not cover.** A
//      pack that repeats aetox-code-review or aetox-design is a second copy of
//      an answer the machine already gives, and the user pays for the
//      confusion of two.
//   2. **It works on one click.** plugin_install copies files and nothing
//      else: it makes no PATH entry, installs no package, builds nothing. A
//      pack whose SKILL.md tells the model to run a command that only its own
//      installer creates is a card that leads to a broken skill, which is the
//      exact failure mcpShelf.ts's rule 2 was written about.
//   3. **What lands was measured, not read off the README.** Every count and
//      size below came from running this app's own findPlainSkills over the
//      repository's real git tree on the date in `verifiedAt` — see the
//      "measured" note under each entry. A repository's own description is
//      not evidence about what Aetox will write to disk.
//
// **Why a big pack is not a context bill.** The obvious objection to a card
// that installs fifty skills is that the model then carries fifty skills.
// It does not: internal/skill/progressive.go replaced per-skill tool
// definitions with skills_list and skill_view years of commits ago, so a
// library costs a flat price and a skill's body is paid for only when the
// model opens it. That is what makes a pack shelf-able at all.
//
// **The install granularity is the repository, and that is why `installs` is
// a list rather than a count.** InstallSkillFromGitHub takes a repo URL and
// writes every skill folder in it. There is no way to take one. So the card
// has to say plainly what arrives, and the room marks a pack installed by
// looking for these names among the user's skills — one list, no second
// number to drift out of step with it.

export interface SkillPreset {
  /** The name on the card, and the key everything else is looked up by. */
  name: string
  /** Passed to InstallSkillFromGitHub verbatim. */
  repo: string
  /** One line: what this pack is. */
  desc: string
  /** Rule 1 said out loud — what it reaches that Aetox has no skill for. */
  why: string
  /** Every skill folder the install writes, measured. Length is the count. */
  installs: string[]
  /** Total size of those folders, in KB, measured the same day. */
  kb: number
  /** The licence the repository publishes, as GitHub reports it. */
  licence: string
  /** When the tree was read and the install measured. */
  verifiedAt: string
}

export const SKILL_PRESETS: SkillPreset[] = [
  // Replicate explicitly publishes skills and MCP as complementary pieces:
  // MCP calls the API, while these teach model selection, trade-offs and
  // prompting. Measured 2026-09-17: 7 skills, 7 files, 62 KB. Every skill is
  // one markdown file, so there is no setup script or private launcher for the
  // installer to miss. build-models and publish-models naturally use Cog when
  // that is the user's task; the remaining five need no Cog installation.
  {
    name: 'replicate/skills',
    repo: 'https://github.com/replicate/skills',
    desc: 'Official model discovery, comparison, execution and image/video prompting skills from Replicate',
    why: 'The Replicate MCP gives Aetox hands on the API; it does not teach which model fits a job, how cost and speed compare, or how to prompt image and video models consistently. This pack supplies that missing judgement and the build/publish workflow for custom models.',
    installs: [
      'build-models', 'compare-models', 'find-models', 'prompt-images',
      'prompt-videos', 'publish-models', 'run-models',
    ],
    kb: 62,
    licence: 'Apache-2.0',
    verifiedAt: '2026-09-17',
  },

  // Prisma's own current reference, including the v7 break and the special
  // MongoDB path. Measured 2026-09-17: 9 skills, 71 files, 299 KB. The extra
  // files are references and examples inside their skill folders; no pack
  // installer or repository-local command is required.
  {
    name: 'prisma/skills',
    repo: 'https://github.com/prisma/skills',
    desc: 'Official Prisma ORM skills for schema work, Client APIs, database setup, adapters and upgrades',
    why: 'Aetox can design and tune a database, but its bundled skills do not carry Prisma\'s current CLI, Client API, driver-adapter contracts or the v6-to-v7 and MongoDB migration decisions. The Prisma MCP can operate on data; this pack teaches the code and migration work around it.',
    installs: [
      'prisma-cli', 'prisma-client-api', 'prisma-compute', 'prisma-database-setup',
      'prisma-driver-adapter-implementation', 'prisma-mongodb-upgrade', 'prisma-postgres',
      'prisma-postgres-setup', 'prisma-upgrade-v7',
    ],
    kb: 299,
    licence: 'MIT',
    verifiedAt: '2026-09-17',
  },

  // Dodo's examples are checked against its real TypeScript, Go and Python
  // SDKs in the publisher's CI. Measured 2026-09-17: 17 skills, 17 files,
  // 268 KB. They are prose and compile-checked examples only; the packages a
  // skill asks for are the application's Dodo SDKs, not a hidden skill runner.
  {
    name: 'dodopayments/skills',
    repo: 'https://github.com/dodopayments/skills',
    desc: 'Official payment, subscription, billing, checkout and webhook integration skills from Dodo Payments',
    why: 'Aetox already knows documents, Thai tax facts and accounting checks; it does not know how to implement checkout, subscriptions, usage or credit billing, proration, disputes and signed webhooks in an application. The Dodo MCP manages merchant data, while these skills cover the integration code and test-to-live path.',
    installs: [
      'better-auth-integration', 'billing-sdk', 'checkout-integration', 'credit-based-billing',
      'customer-management', 'discounts-and-promotions', 'dodo-best-practices',
      'framework-adapters', 'license-keys', 'localized-pricing', 'mobile-checkout',
      'product-catalog-management', 'refunds-and-disputes', 'subscription-integration',
      'testing-and-go-live', 'usage-based-billing', 'webhook-integration',
    ],
    kb: 268,
    licence: 'MIT',
    verifiedAt: '2026-09-17',
  },

  // The one the channel actually names. /wayfinder got its own video, so did
  // /grill-with-docs beside /plan, and the "AIHero" video is the same author's
  // material again — three videos, one repository.
  //
  // Measured 2026-09-17 over the real tree: 37 skills, 100 files, 245 KB. It
  // is prose and references only — no launcher — which is also why rule 2 is
  // not in question for it.
  //
  // Rule 1 is the interesting part because much of the pack now overlaps the
  // bundled forge/debug/review/grill/spec/slice workflow. It remains for the
  // part Aetox still does not have: `wayfinder` externalises an unresolved
  // decision tree as issue-tracker tickets so several sessions can advance it
  // independently. Aetox can run a long plan, but it does not create or work
  // that durable decision map.
  //
  // Worth knowing before pressing: 6 of the 37 come from the repository's own
  // `in-progress/` folder and are half-finished by their author's own
  // labelling, and `setup-matt-pocock-skills` exists because several of the
  // others expect an issue tracker to have been chosen first — wayfinder says
  // so in its own body.
  {
    name: 'mattpocock/skills',
    repo: 'https://github.com/mattpocock/skills',
    desc: 'Planning and interrogation skills for real engineering work, including /wayfinder and /grill-with-docs',
    why: 'Aetox already covers most of this pack\'s planning, review, debugging and implementation workflow. The reason to install it is wayfinder: it charts work too big for one session as a durable tree of decision tickets on the repository\'s issue tracker and resolves that map one decision at a time. Nothing bundled externalises long work that way.',
    installs: [
      'ask-matt', 'claude-handoff', 'code-review', 'codebase-design', 'diagnosing-bugs',
      'domain-modeling', 'git-guardrails-claude-code', 'grill-me', 'grill-with-docs', 'grilling',
      'handoff', 'implement', 'implement-spec', 'improve-codebase-architecture', 'loop-me',
      'migrate-to-shoehorn', 'prototype', 'research', 'resolving-merge-conflicts', 'retro',
      'scaffold-exercises', 'setup-matt-pocock-skills', 'setup-pre-commit', 'setup-ts-deep-modules',
      'tdd', 'teach', 'to-questionnaire', 'to-spec', 'to-tickets', 'triage', 'wait-what',
      'wayfinder', 'wizard', 'writing-beats', 'writing-for-agents', 'writing-fragments',
      'writing-shape',
    ],
    kb: 245,
    licence: 'MIT',
    verifiedAt: '2026-09-17',
  },

  // The channel's SEO video ("SEO Skill ของดี! ทำเว็บให้โหลดไว + SEO ดีขึ้น")
  // names no repository that could be read off the page, so this is the topic
  // answered rather than that video's exact pack, and it is here instead of
  // the obvious candidate for a reason worth writing down.
  //
  // AgriciDaniel/claude-seo is the popular dedicated one — 16k stars, MIT, 33
  // skills, no test fixtures — and it fails rule 2. Its main SKILL.md tells
  // the model to run its bundled Python through `claude-seo run <script>`, a
  // command its own plugin installer puts on PATH. plugin_install copies
  // files; it creates no command. Pressing เพิ่ม would install a skill whose
  // own instructions cannot be followed. It is a good pack and the wrong
  // shape for this button.
  //
  // This one is prose and templates only, so it installs and works. Measured
  // 2026-09-17: 50 skills, 279 files, 2.5 MB. Its `seo-audit`,
  // `programmatic-seo`, `schema`, `ai-seo` and `site-architecture` cover the
  // video's ground; the other 45 are the rest of a marketing department.
  {
    name: 'coreyhaines31/marketingskills',
    repo: 'https://github.com/coreyhaines31/marketingskills',
    desc: 'SEO, conversion, copywriting, pricing and growth — a marketing department as skills',
    why: 'Aetox builds the site and has never had a word to say about whether anyone finds it. There is no SEO, copywriting, pricing, conversion or analytics skill among the 44 bundled — the whole category is missing, and this is the one pack that fills it without needing a command Aetox cannot install.',
    installs: [
      'ab-testing', 'ad-creative', 'ads', 'ai-seo', 'analytics', 'aso', 'attribution',
      'churn-prevention', 'co-marketing', 'cold-email', 'community-marketing',
      'competitor-profiling', 'competitors', 'content-strategy', 'copy-editing', 'copywriting',
      'cro', 'customer-research', 'directory-submissions', 'emails', 'events', 'free-tools',
      'image', 'influencer-marketing', 'launch', 'lead-magnets', 'marketing-council',
      'marketing-ideas', 'marketing-loops', 'marketing-plan', 'marketing-psychology', 'offers',
      'onboarding', 'paywalls', 'popups', 'pricing', 'product-marketing', 'programmatic-seo',
      'prospecting', 'public-relations', 'referrals', 'revops', 'sales-enablement', 'schema',
      'seo-audit', 'signup', 'site-architecture', 'sms', 'social', 'video',
    ],
    kb: 2596,
    licence: 'MIT',
    verifiedAt: '2026-09-17',
  },
]

// Checked on 2026-09-05 and deliberately NOT on the shelf. Written down so the
// next person does not spend the afternoon rediscovering them, in the spirit of
// mcpCandidates.ts — except these are refusals with reasons, not a waiting room.
//
//   - AgriciDaniel/claude-seo — rule 2. Needs a `claude-seo` command on PATH
//     that only its own plugin installer creates. See the note above the
//     marketing entry.
//   - codexstar69/bug-hunter — rule 2, differently. It keeps a SKILL.md at the
//     repository root, which findPlainSkills reads as "the repository is the
//     skill", so the install is the whole repository as one skill: 194 files,
//     38 MB, fetched one raw file at a time inside InstallSkillFromGitHub's
//     60-second timeout. It would not finish.
//   - NVIDIA/SkillSpector — this one changed the installer rather than the
//     shelf. It is a scanner for malicious skills and keeps its samples at
//     tests/fixtures/, so findPlainSkills returned 25 skills: the scanner plus
//     24 traps named malicious_skill, mcp_poisoned_tool,
//     ssd1_semantic_injection and so on. Installing it wrote prompt-injection
//     samples into the skills directory as live skills. The test directories
//     are now in notSkillMaterial (internal/skill/github_tools.go) and it
//     installs one skill; it stays off the shelf only because a scanner that
//     wants a Python environment is not a one-click card.
//   - anthropics/skills — 20 skills but 414 files and 10.4 MB, the same
//     one-file-at-a-time problem as bug-hunter, and GitHub reports no licence
//     for the repository.
//   - obra/superpowers — was on the shelf from 2026-09-05 to 2026-09-14 for
//     the half about running an agent (worktrees, parallel subagents, plans,
//     verifying before claiming done). Rule 1 retired it: those cores are
//     bundled now as aetox-brainstorm, aetox-run-plan, aetox-parallel,
//     aetox-worktree, aetox-verify, aetox-review-feedback,
//     aetox-finish-branch and aetox-skill-writing (DECISIONS §274), beside
//     the aetox-debug and aetox-forge that already covered its debugging and
//     TDD. Installing the pack on top would be fourteen second voices, and
//     its using-superpowers entry skill is the shout-and-announce mechanism
//     internal/prompt/prompt.go's reads() deliberately does not use.
//   - Quality-Max/free-qa-skills — clean, small, and 9 stars on 2026-09-05.
//     Not a reason to refuse it, only a reason not to be the one to put it in
//     front of everybody yet.
//   - cloudflare/skills — official and useful, but the measured 2026-09-17
//     tree is 14 skills / 346 files. The current installer fetches every raw
//     file sequentially under one 60-second request; putting this on the shelf
//     would advertise a button likely to time out. Reconsider after archive or
//     concurrent fetching lands.
//   - microsoft/skills — official, but its 175 skills live under
//     `.github/skills`. Aetox deliberately ignores dot-directories as repo
//     metadata, so the current installer correctly finds no installable skill.
//   - openai/skills — deprecated by its owner in favour of OpenAI Plugins, and
//     its curated/system folders are dot-directories the Aetox installer does
//     not treat as published skill material.
