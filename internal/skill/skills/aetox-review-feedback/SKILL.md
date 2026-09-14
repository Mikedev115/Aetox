---
name: aetox-review-feedback
before: acting on code review feedback, from the user or from an outside reviewer, especially when a point is unclear or looks wrong
description: ตอนได้รับฟีดแบ็กจากการรีวิวโค้ด ไม่ว่าจากผู้ใช้หรือคนนอก ก่อนจะลงมือแก้ตาม
source: https://github.com/obra/superpowers (receiving-code-review), adapted
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Review Feedback

Review feedback is evaluated, not performed. Each point is checked against
the code before it is implemented, and a point that is wrong for this
codebase is answered with the reason, not with the change.

## For every item

1. Read all of it before reacting to any of it. Items are often related.
2. Restate the requirement in your own words; if you cannot, ask.
3. Check it against the code as it is: is it correct here, does it break
   something that exists, is there a reason the current code is the way it
   is, does the reviewer have the whole context.
4. Then either do it, or push back with the technical reason, or say what
   you cannot verify and what you would need.
5. Implement one item at a time, each one tested, in the order that keeps
   the code working between items.

Unclear items stop the whole batch: "1, 2, 3 and 6 are clear; 4 and 5 I
need to understand before doing any of them." Half a batch implemented on a
guess is worse than none.

## Who it came from

- From the user: trusted. Understand it, then act; still ask when the scope
  is unclear.
- From an outside reviewer or a tool: skeptical and careful. Their
  suggestion may be right in general and wrong here. A conflict with what
  the user already decided goes to the user before anything is changed.

## What not to write

No "you are absolutely right", no "great point", no "let me implement that
now" ahead of the check. The acknowledgement is the restated requirement,
the question, or the change itself.

## The "do it properly" suggestion

When a reviewer asks for the complete version of something, look up whether
it is used. Unused means propose removing it, not completing it. Both of
you answer to the user, and the user is not asking for features nobody
calls.
