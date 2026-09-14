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
// **Where these came from.** The owner pointed at a YouTube channel
// (youtube.com/@MilerDev) and asked for the skills it covers. Three of its
// videos are about skills from one repository — /wayfinder, /grill-with-docs
// with /plan, and "ของดีจาก AIHero" (AIHero is Matt Pocock's) — and that
// repository is the first entry below. The channel's SEO video and its
// agentic-engineering videos name no repository on screen that could be read
// from the page, so the other two entries answer the topic rather than claim
// to be the exact thing shown; each says so in its own note. The video
// descriptions carry only sponsor and merchandise links, and YouTube refused
// the caption tracks, so there was no way to confirm further without watching.
//
// **The bar, which is mcpShelf.ts's bar with one word changed.**
//
//   1. **It teaches Aetox a job its own 25 bundled skills do not cover.** A
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
  // The one the channel actually names. /wayfinder got its own video, so did
  // /grill-with-docs beside /plan, and the "AIHero" video is the same author's
  // material again — three videos, one repository.
  //
  // Measured 2026-09-05 over the real tree: 37 skills, 100 files, 90 KB. It is
  // the smallest pack here by a wide margin because it is prose and nothing
  // else — no scripts, no assets, no launcher — which is also why rule 2 is
  // not in question for it.
  //
  // Rule 1 is the interesting part, because two of the 37 do overlap:
  // `code-review` and `diagnosing-bugs` sit next to aetox-code-review and
  // aetox-debug. They are not why this is here. `wayfinder` plans work too big
  // for one session as a map of decision tickets on the repo's issue tracker,
  // and `grilling` / `grill-with-docs` interrogate a plan until it is sharp
  // and leave ADRs behind. Aetox has aetox-architect, which is knowledge about
  // architecture, and nothing at all that runs a planning process across
  // sessions. That is the gap.
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
    why: 'Aetox can design and review, and it has no process for work too big to hold in one session. wayfinder charts that work as decision tickets on the repo\'s issue tracker and resolves them one at a time; grilling and grill-with-docs interrogate a plan until nothing is left to decide, writing the ADRs as they go. Nothing bundled with Aetox does either.',
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
    kb: 90,
    licence: 'MIT',
    verifiedAt: '2026-09-05',
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
  // 2026-09-05: 50 skills, 279 files, 1.9 MB. Its `seo-audit`,
  // `programmatic-seo`, `schema`, `ai-seo` and `site-architecture` cover the
  // video's ground; the other 45 are the rest of a marketing department.
  {
    name: 'coreyhaines31/marketingskills',
    repo: 'https://github.com/coreyhaines31/marketingskills',
    desc: 'SEO, conversion, copywriting, pricing and growth — a marketing department as skills',
    why: 'Aetox builds the site and has never had a word to say about whether anyone finds it. There is no SEO, copywriting, pricing, conversion or analytics skill among the 25 bundled — the whole category is missing, and this is the one pack that fills it without needing a command Aetox cannot install.',
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
    kb: 1931,
    licence: 'MIT',
    verifiedAt: '2026-09-05',
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
