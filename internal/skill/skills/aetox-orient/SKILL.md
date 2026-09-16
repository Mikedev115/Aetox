---
name: aetox-orient
before: the first change in a project this conversation has not read yet, or picking a project up again after a break
description: ตอนเข้าโปรเจกต์ที่ยังไม่ได้อ่านในบทสนทนานี้ หรือกลับมาสานต่อหลังเว้นไป ก่อนแตะไฟล์แรก
license: MIT
copyright: Copyright (c) 2026 Aetox Skills
---

# Aetox Orient

A project is read from the cheapest layer outward, and each layer says
which part of the next one to open. Reading a whole file to find out what
it is, is the failure this discipline exists to stop: the week this app's
map was built against counted eighteen whole reads of 50 KB or more, and
the two largest were architecture documents read end to end for one
section.

## Layer 0: what the conversation already holds

Before any tool: the project's own file (`AETOX.md`, `AGENTS.md` or
`CLAUDE.md`, already in your context when it exists) and the project memory
lines that earlier sessions wrote. They are what this project settled:
the command that proves a change, the folder that is generated, the
convention nobody wrote down elsewhere. An instruction there outranks a
guess from the tree.

## Layer 1: the shape, once

`codebase` with `action: map`, the whole project, one call per
conversation. It costs about a thousand tokens and returns files ranked by
how many other files use them, with their symbols and line numbers:

- The first files are the ones the project leans on: its types, its
  config, its registry. A change that touches one of them reaches
  everywhere; a change that touches none is local.
- Markdown files appear as heading outlines, so an architecture document is
  opened at the section, never end to end.
- "N more files under the budget line" means the map was cut to fit. Map
  the folder the task lives in (`path`) to see that folder deeper; do not
  map the whole project again.
- The map refuses when no project is focused; that is the answer, not an
  error to route around.

## Layer 2: the name, exactly

Use `codebase` with `action: symbol` when an identifier is unfamiliar: what
it declares, its signature, documentation and exact references. Use `search`
for text: a string in a message, a config key, or a word in a comment, and
`symbol` for code names.

## Layer 3: the change radius

Before changing a function, method, type, field or variable whose consumers
matter, use `codebase` with `action: impact`. It separates production, test
and other references, identifies confirmed generated or RPC surfaces, and
names the narrow checks the evidence supports. Skip it for prose, literal
text and a local edit that cannot change a code contract.

## Layer 4: the path, when that is the question

When the question is how one place reaches another, use `codebase` with
`action: trace`. It is most useful across generated bindings, RPC, or a
frontend/backend boundary, where a reference list does not prove the path.
Do not run it automatically after the map or for a local edit whose target is
already known; its first index build is deliberate work.

## Layer 5: the lines, not the file

`read` with `offset` and `limit` at the line the map or the symbol named,
and enough around it to see the function whole. A whole file is read when
its size is known to be small or when the task is the file itself; a whole
read of something large is said out loud, with the reason.

## Layer 6: the code's own verdict

After editing, `codebase` with `action: errors` on the files touched. These are
the language server's compile and type errors, which a test run reports later
and less precisely. Then the proof `aetox-verify` asks for.

## Closing: one line for the next session

What this work taught that the code does not say, such as a trap, a convention,
or the command that proves this project, goes to `memory` as one line. It is
what Layer 0 will hold next time. Narrating what was read is not that; the
map and the files are there to be read again.

## Not this

- Mapping, then reading the top file whole anyway.
- Announcing the layers. Orientation is part of the work, not a report on
  it; the user asked for the change.
- Skipping to `search` because it is familiar. A search finds where text
  appears; the map says what the project is made of, symbol says what a name
  is, impact says what changing that name reaches, and trace proves how two
  places connect when that path is the question.
