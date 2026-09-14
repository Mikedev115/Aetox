---
name: aetox-brainstorm
before: building a feature, a component or a change to how parts fit together, before the first line of code
description: ตอนผู้ใช้ขอให้สร้างฟีเจอร์ คอมโพเนนต์ หรือเปลี่ยนวิธีที่ส่วนต่าง ๆ ต่อกัน และยังไม่มีแบบที่ตกลงกัน
source: https://github.com/obra/superpowers (brainstorming), adapted
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Brainstorm

An idea becomes a design in conversation before it becomes code. The size of
the design follows the size of the request; the yes before building does not
shrink with it.

## Classify first, out loud

Say which of the three this is, so the user can overrule it:

- **Spike**: a feasibility question ("can we…", "is it possible…"). The
  output is an answer, not code anyone keeps. Say what you will try in two
  sentences, get a nod, find out as cheaply as correctness allows, and label
  anything built as throwaway.
- **Bounded**: a well-scoped change to a flow that already exists in this
  repository: a flag, a small endpoint, a one-file fix. Bounded measures the
  repository, not your familiarity with this kind of app. Ask the questions
  that matter, present a short design in chat, and stop until the user says
  yes. No document.
- **Architectural**: a new subsystem, a change that restructures how parts
  fit, an interface others depend on. Questions, then two or three
  approaches with a recommendation, then a sectioned design; a whole system
  with no code yet goes to `aetox-idea-to-architecture`, a plan to be
  stress-tested to `aetox-grill`, a spec to slice to `aetox-slice`.

In doubt between two, take the heavier. The ratchet is one way: complexity
found mid-task upgrades the path; say so and step up. Nothing downgrades.

A job that changes no flow (rename, typo, a value) is not on this ladder; it
just gets done.

## How the conversation goes

- Read what is already here before the first question: the code, the docs,
  the recent commits. Never ask what the repository answers.
- One question at a time, the one that changes the design most. Prefer a
  question with options to an open one.
- Present the design in pieces short enough to read, and ask after each
  whether it is right so far. Doubt in the answer is a question, not a
  detail to gloss.
- Say what you decided and what you assumed. An assumption the user has not
  seen is a defect waiting for its bug report.

## What is not an exit

"This is too simple to need a design" means a two-sentence design in chat,
then a yes; simple tasks are where an unexamined assumption costs the most.
"The design is obvious, I will start while they read" is starting without
the yes. "I understand this kind of app" is not the same as having read this
repository's flow.
