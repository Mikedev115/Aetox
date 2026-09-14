---
name: aetox-skill-writing
before: writing a new skill, or changing what an existing skill says
description: ตอนจะเขียนสกิลใหม่ หรือแก้เนื้อหาของสกิลที่มีอยู่ รวมถึงร่างแก้ที่ระบบเสนอให้
source: https://github.com/obra/superpowers (writing-skills), adapted; the shelf mechanics are this app's (DECISIONS §221)
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Skill Writing

A skill is a document a model reads at a moment. Two things decide whether
it works: whether the model finds it at that moment, and whether the model
does what it says under pressure. Both are tested, neither is assumed.

## The two lines the model sees first

Every prompt carries each skill as its name and the first 100 characters of
its `description:`, and a `before:` line when the skill declares one. The
body is never sent until the model opens it. So:

- `description:` says **when**, not what. Only the situation that means
  "read me now": the request, the symptom, the file type. A description
  that summarises the skill's steps is followed instead of the skill, and
  the body becomes documentation nobody opens. Lead with the moment; the
  first 100 characters are all that reach the prompt.
- `before:` names the work the skill is read before, in one clause. The
  prompt draws "before …: skill_view" itself; nothing is added to a desk
  file, and the skill is never named in another skill's list of moments.
- Name it by the job, `aetox-<job>`, in the shelf's style.

## The body

- Short. Every skill is a lookup the model pays for; a skill that is opened
  often is paid for often. Say the rule, the moment, and the smallest set of
  steps; leave the essay out.
- One idea per line; the model follows short lines more reliably than a
  paragraph carrying three.
- Rules, not narration: nothing that makes the model announce it is using
  the skill. A model told it `MUST` tells the user that it must, instead of
  doing the work.
- Name the moment to stop, and the moment to hand to another skill, by that
  skill's name.
- Reference material (a checklist, a template, a long table) goes in a file
  beside SKILL.md and is named from it, not inlined.

## Test before it ships

First the shape, by machine: `aetox skill lint <folder>` reads the rules
above that a program can read (a description whose first hundred characters
name no moment, a hedge inside a rule line, a shouted `MUST`, a file the body
names and does not ship, a `$ARGUMENTS` door this app does not have, CRLF,
an unclosed fence). An error is a door the model will be refused at and
does not ship; a warning is read and either fixed or kept on purpose with
`<!-- lint-allow rule -->`. The app runs the same check on every skill it
drafts itself and writes what it found on the proposal card.

Then the behaviour, by running a model, which no linter replaces:

- A rule skill: give a model the situation with the pressure that makes
  skipping tempting (time, sunk cost, "this one is simple") and see whether
  it follows the rule. Then read what it said instead, and add that
  rationalisation to the skill as a named non-exit.
- A technique skill: give a model a case and a variation, and check that it
  applies the technique, not the description of it.
- A reach test: give a model the moment without naming the skill, and check
  that it opened it before starting.

An edit to a skill is tested the same way as a new one. Untested changes are
not kept "for reference"; they are removed.

## The one-line check

Could a model that has never seen this codebase pick this skill from a
shelf of fifty by its first line, open it, and do the job without asking you
what it meant? If not, it is not finished.
