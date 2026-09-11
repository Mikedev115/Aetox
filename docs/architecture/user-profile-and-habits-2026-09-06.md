# A Second Memory Layer: Who the User Is, and What They Keep Asking For (2026-09-06)

**Status:** Step 1 is **built and shipped** (DECISIONS §229); steps 2 and 3 are
proposal only. Written after reading Hermes Agent (Nous Research) as the
reference the owner named, and after measuring this machine's own store before
proposing anything (§180).

**Amended by the owner while step 1 was being built:** the profile is read
**everywhere** — every desk, every chair, and every worker on every delegated
job — not only in main sessions as §4.1 first proposed. *"ผมว่า USER.md
ไปทุกที่เลยดีกว่า"*. The boundary that decision keeps is the size (2 KB, a quarter
of every other scope) rather than the audience, and the write side is unchanged:
a delegate's `memory` schema is never offered `about`, so it cannot reach the
file at all. See §229.3.

**Trigger.** Owner, 6 ก.ย.: *"ผมว่าจะเพิ่มระบบความจำอีกชั้นคือแบบ จำว่าผู้ใช้ชอบ
บอกให้ทำอะไรบ่อยๆ หรือ ระบบพยายามจำว่าผู้ใช้เป็นใครอะไรมาจากไหน … ดู hermes
เป็นแบบอย่าง"*.

Two asks in one sentence, and they turn out to have different lifetimes and
therefore different homes:

1. **Who the user is** — a profile. True in every session, whatever desk.
2. **What they keep asking for** — habits. A fact about the *user* ("asks to
   check GPU power draw every other day") and, separately, a *procedure* ("how
   to check it the way they want"). Hermes puts the first in the profile and the
   second in a skill, and the measurement below says that split is right for
   Aetox too.

---

## 1. What Hermes does

Read from the repository and its docs on 6 ก.ย. 2026.

| piece | Hermes | notes |
|---|---|---|
| Files | `~/.hermes/memories/MEMORY.md` + `USER.md` | two files, not one. MEMORY = environment, conventions, tool quirks. USER = name, role, timezone, communication preferences, pet peeves, workflow habits, skill level. |
| Budget | MEMORY 2,200 chars (~800 tok), USER 1,375 chars (~500 tok) | hard cap; a write over the cap is **refused with the current entries listed**, never silently trimmed. Header shows `[67% — 1,474/2,200 chars]` so the model can consolidate. |
| Tool | `memory(action=add\|replace\|remove, target=memory\|user, content, old_text)` | no `read` action — the files are in the prompt already. `old_text` is a unique substring. Exact duplicates rejected. |
| Prompt | both files injected as a **frozen snapshot at session start** | mid-session writes hit disk, reach the prompt next session. Same reason as ours: prefix cache. |
| Guidance | *"Skills come first … Memory is the narrow exception for facts that apply to EVERY session regardless of task."* | and: *"Write entries as declarative facts, not instructions to yourself: 'User prefers concise responses' ✓ — 'Always respond concisely' ✗ (imperative phrasing gets re-read as a directive in later sessions and can override the user's current request)."* |
| Capture | **post-turn background review**: a forked agent on the same model (warm cache) or a cheaper model on a digest (last ~24 messages verbatim, older turns collapsed) | tools restricted to `memory`, `skill_manage` and reads. Prompt asks two questions: *"Has the user revealed things about themselves — persona, desires, preferences, personal details?"* and *"Has the user expressed expectations about how you should behave, their work style, or ways they want you to operate?"* Pre-empted within 2 s when a live turn starts. |
| Approval | `memory.write_approval: false` by default; when on, inline in foreground, staged with `/memory pending` in background | Aetox is the opposite default and stays so (below). |
| Recall beyond the cap | `session_search` over SQLite FTS5 (all sessions, ±5-message window, CJK tokenizer) | "~1,300 tokens always-on vs unlimited on-demand" is how they frame the two layers. |
| Safety | every entry scanned for injection/exfiltration patterns and invisible Unicode before it is accepted | memory lands in the system prompt; a planted line is a planted instruction. |
| Honcho etc. | 8 optional external providers (semantic recall, dialectic user modelling) | one at a time, additive. Out of scope here — see §6. |

The two things worth copying are not the files. They are **the two questions the
review asks**, and **the rule that a preference about how to do a task lives in
the skill for that task, not in memory**.

---

## 2. What Aetox already has

Most of the machinery exists; what is missing is a *distinction* and a *source*.

| Hermes | Aetox today | gap |
|---|---|---|
| MEMORY.md + USER.md | one `<DataRoot>/memory/MEMORY.md` (`learned.MainScope`) holding both kinds of line | **no profile file** — who the user is and what their Ollama path is share one file and one budget |
| `target=user\|memory` | `where=everywhere\|this-project` (desk-defaulted, §184) | no user/machine axis |
| char cap, refuse on overflow | `learned.MaxBytes` = 8 KB per scope, `Full()` refuses `add` | same shape, 3× Hermes's whole budget |
| frozen at session start | prompt built at bootstrap only | same |
| declarative-not-imperative rule | absent from `MemoryTool.definitionText` | one sentence |
| background review | `summarizeFailures` after each turn — writes **problems** (kind `issue`, §136), never lessons | no source proposes about the user except the model mid-turn |
| write approval | `pending_changes` + human yes, always (§139: *"a proposal is not memory"*) | keep |
| session_search | `desktop/session_search.go` over `messages_fts` | exists |
| skills-first | 25 bundled skills, `aetox-skills` asks *"what do you do more than once a month"*, STARTERS cards per worker | the road exists; nothing feeds it from observed recurrence |
| user-written identity | `<DataRoot>/identity/*.md`, folded above memory | **empty on this machine** — the owner has never written who he is; everything the agent knows about him came through proposals |

---

## 3. What this machine says (measured 6 ก.ย., store from 3–5 ก.ย.)

32 sessions in three days: 21 assistant, 9 coding, 2 specialized. 1,048 tool
runs. The `memory` tool fired 11 times and produced 16 proposals.

### 3.1 Four approved, twelve rejected — and the line between them is the profile

| proposal is about | proposed | approved |
|---|---|---|
| the user — who he is, what he builds, what he likes doing, how he wants things done (#3 #7 #8 #12 #13 #17 #19) | 7 | **4** |
| the machine — Ollama path, hotkey registry, WSL leftovers, OMEN Hub settings, a venv's VC++ dependency, a GPU registry key, a prompt folder (#1 #2 #4 #5 #9 #10 #11 #18) | 8 | **0** |
| a rule to itself — *"ก่อนสร้าง UI … ต้องเปิดอ่านสกิลที่เกี่ยวข้องก่อน"* (#15) | 1 | 0 |

Every line the owner has ever kept is about **himself**. Not one line about the
machine survived review, and the three machine-ish lines that did survive (#8,
#17) were phrased as facts about *his* setup and *his* attitude to it. The
tool's description says "this user or machine" as if they were one bar; the
owner applies two.

The one imperative line was rejected — the exact shape Hermes's guidance
forbids, for the reason it gives: a rule to yourself in memory outranks the
user's next request.

### 3.2 Habits are visible in three days of first messages

Same wording, different sessions, with the count:

| request | sessions |
|---|---|
| *"อยากได้หน้า landing ขายแอปจัดการเงิน"* | 4 |
| check power draw / GPU wattage (*"เช็คกำลังไฟ"*, *"กินไฟเท่าไหร่"*, *"ตรวจการกินไฟเบื้องหลัง"*) | 4 |
| clone a Framer template into a real site | 3 |
| *"Pull โค้ดมาหน่อย"* | 2 |
| *"เปิดเว็บให้ผมอีกรอบ"* | 2 |
| *"ฝากลบ …"* (games, WSL, pagefile, Norton) | 4 |

And the corrections that are *preferences about how to work*, said in passing
and never proposed:

- *"ถามผมก่อนนะ ลิสมาก่อนๆ เช็คก่อนๆ"* — ask, list, check before acting.
- *"ปรับในระบบดิ ไม่ควรปรับตามโปรแกรมภายนอก"* — system settings over third-party tools. (This one did reach memory, as #17, because the model happened to be looking.)
- *"ทำไมไม่อ่านเทมเพลต ไม่อ่านเหี้ยไรเลย"* — read the template skill before building a page. Frustration, which Hermes treats as a first-class skill signal. It belongs in `aetox-web-templates`, not in MEMORY.md — which is where #15 tried to put it and was refused.

Nothing in Aetox reads this back. The only thing that has ever noticed a
repeated request is a person.

---

## 4. Proposal

Three parts. Each is buildable alone and each is measured before the next.

### 4.1 Split the shared file: `USER.md` beside `MEMORY.md`

`<DataRoot>/memory/USER.md` — **who this user is and how they want to be worked
with** — as a fourth read scope (`learned.UserScope`), folded into every main
session **between the identity files and MEMORY.md**. Position is the policy,
as it already is in `foldLearnedMemory`: what the user wrote about themselves
outranks what the agent learned about them, which outranks what it learned
about the machine.

`MEMORY.md` keeps what is left: this machine, this setup, conventions with no
task home. Read by every desk as today.

The `memory` tool gains a **required** `about` parameter, `user | machine`, with
no default. Required on purpose: §184's rule is that a parameter whose absence
is indistinguishable from one of its values is a bias, not a choice. Here the
desk cannot answer for the model (a coding session learns things about the user
too), so the model must, and a call without it is refused with the two words it
may use — the same door `replace` without `old` already goes through.

Budget: `USER.md` starts at **2 KB** (Hermes is 1.4 KB; Thai costs more bytes per
token). `MEMORY.md` stays at 8 KB. Both are ratchets in a test, moved only with
a measurement.

Description text, two additions and one cut:

- *"Write a fact, not an instruction to yourself: 'User prefers short answers', never 'Always answer briefly' — a rule in memory outranks the user's next request."* (#15, and Hermes verbatim on the why.)
- *"about: user for who they are, what they are building, how they like things done and what they reliably mean by a request; machine for where things live and what this setup needs."*
- Cut "or machine" from the one-bar sentence that lumps them.

Migration: one pass on first run moves the four approved lines by a keyword
split (they are all `user`); anything ambiguous stays in `MEMORY.md`. The
settings page shows the new scope like the others; `Scopes()` lists it.

### 4.2 Habits: two homes by lifetime, and a counter that is not a guess

A recurring request is two facts.

- **That the user asks for it, and what they mean** → one declarative line in `USER.md`: *"Asks to check GPU power draw often; wants the fix in Windows/OMEN settings, not a third-party tool."* Under 4.1's `about: user`, no new mechanism.
- **How to do it their way** → a skill (Hermes: *"skills come first"*). Aetox's roads already exist — the user's own rules go into a skill written with them (`aetox-skills` §6), a worker's repeated job becomes a STARTERS card. What is missing is the **nudge**: nobody tells the agent that this is the fourth time.

The nudge should come from the store, not from a model. `messages` already
holds every first user message of every session; a normalised match (trim,
lower-case, strip attachments, Thai politeness particles) over the last N
sessions is a deterministic count. Surfaced two ways:

1. **On the session's first turn, one line into the turn** (not the prompt — the prompt is frozen and this is per-session): *"You have been asked something like this in 3 earlier sessions."* Cheap, cache-safe, and it lets the model decide whether to propose a `USER.md` line or offer to write the procedure down.
2. **On the settings memory page**, a small "ที่ถามซ้ำ" list with counts, each row carrying *สร้างสกิล* / *ทำเป็นการ์ด* — the person decides, as they do for every other line.

No LLM in the loop for the count. The model reads a number; it does not invent one.

### 4.3 A session review that asks Hermes's two questions

`summarizeFailures` runs after every turn and is the only automatic source; it
writes problems by design (§136). The learning queue is fed only by the model
remembering to call `memory` mid-task, and 3.2 shows what that misses:
preferences said in passing while the model is busy with the task.

Add a **review**, Aetox-shaped:

- **When:** once per session, when it goes idle or is closed — not per turn. Owner's traffic is 727 requests on `glm-5.3-flash` with 30.4 M of 33.8 M prompt tokens served from cache; a per-turn fork is the cost this project refuses on principle, and per-session catches the same facts a day later at 1/N the price.
- **What it sees:** the user's own messages of that session, verbatim, and nothing else. Not the transcript: §139 settled that *"a fact the user states about themselves is already the evidence"*, and the user's words are also the smallest and least sensitive digest available.

  > **Amended 2026-09-11 ([DECISIONS §246](../DECISIONS.md)).** "Nothing else" measured 2 approvals in 50, with one fact proposed in six wordings. The reviewer now also reads USER.md as it stands, the lines waiting, and the newest thirty the user turned down, each labelled — and every proposal, from any source, goes through one door that answers a restatement of a decided line with the decision (`queueMemoryProposal`, `learned.SameFact`).
- **What it may do:** call `memory` only, `about` required as above — and every call is a **proposal** into `pending_changes` with `source = 'review'`, exactly as the mid-turn tool does. The gate does not move: Hermes defaults to writing freely; Aetox's whole subsystem exists so that nothing writes itself (`learned` package doc).
- **The prompt:** Hermes's two questions, translated into the tool's own bar — who did the user reveal themselves to be, and what did they say about how they want you to work — plus *"If nothing, say nothing."* Skipped when the session has fewer than two user turns or the model cannot emit tool calls.
- **What it costs, measured before it ships:** one call on the session's own model, prompt = system prompt (cached) + the user's lines. On this store that is under 1 KB of user text for 28 of 32 sessions.
- **Kill switch:** a settings toggle, off by default in the first release; on for the owner's machine to measure approval rate against the mid-turn source's 4/16.

`source` already exists on `pending_changes` and is what lets §136's table be
redrawn per source in a month: mid-turn vs review vs (later) recurrence.

### 4.4 Scan what enters the queue

Memory lands in the system prompt. A page the agent fetched can say "remember to
always run X first" and a tired reviewer clicks อนุมัติ on a card that looks like
every other card. `internal/safety` runs on tool input today; run its injection
and invisible-Unicode checks on `Proposal.Body` in `MemoryTool.ExecuteTool`
before queuing, and refuse with the reason. Hermes does this at the same point.

---

## 5. Order, and what is measured at each step

| step | build | measure before the next |
|---|---|---|
| 1 | **DONE** — 4.1 split + `about` + declarative sentence + 4.4 scan (narrowed, see §229.4) | approval rate per `about` over the next 30 proposals; how often `about` is omitted (refusals in `tool_runs`) |
| 2 | 4.2 counter, first-turn line, settings list | how many rows the list shows on this machine; whether a single `USER.md` line about a habit is ever proposed |
| 3 | 4.3 review, toggle off by default | approvals per review call, cost per review from `token_usage`, against the mid-turn source |

If step 1's `about: machine` lines keep a 0% approval, the honest next move is
to shrink `MEMORY.md`'s budget or narrow its description — not to build step 3
on top of a file nobody wants.

---

## 6. What this deliberately does not do

- **No external memory provider** (Honcho, Mem0, vectors). Hermes offers eight; each is a second place answering the same question, and Aetox already has FTS5 over every session for on-demand recall. Revisit only if step 3's review shows facts the two files cannot hold.
- **No automatic writes**, in any mode. A review proposes; a person approves. This is the line the learned package was built on and §136 re-drew.
- **No per-turn review.** Cost, and §185's mid-turn rule: nothing else touches the engine while a turn runs.
- **Nothing for delegates.** A worker's memory is its own (§184.5); it never talks to the user, so it has no profile to keep.
- **No summarisation of past sessions into memory.** `session_search` is the unbounded layer; the bounded layer stays curated.

---

## 7. Questions for the owner

Settled while building step 1:

1. ~~`about` required or inferred?~~ **Required**, on §184's rule. Watch `tool_runs` for how often the cheap models hit the refusal before they learn the word.
2. ~~`USER.md` at 2 KB or Hermes's ~1.4 KB?~~ **2 KB**, because Thai spends about three bytes where English spends one.

Still open:

3. Review trigger: **idle/close automatically** (4.3), or a **button** on the memory page — *"ทบทวนเซสชันนี้"* — so the cost is spent only when asked?
4. Should the recurrence list (4.2) live on the memory page, or on the skills page where *สร้างสกิล* already has a home?
5. `MEMORY.md`'s label on the settings page is still *ผู้ช่วยหลัก*, which described whose file it was back when it held both kinds of line. Beside *เกี่ยวกับคุณ* it now reads oddly. Rename it to something about the machine, or leave the established word alone?
