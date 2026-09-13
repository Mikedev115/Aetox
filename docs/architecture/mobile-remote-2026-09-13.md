# Remote, Take Two: The Phone Opens Work, the Machine Does It (2026-09-13)

> ## Status: DECIDED 2026-09-13. Supersedes [mobile-remote-2026-08-14.md](mobile-remote-2026-08-14.md).
>
> That document parked the feature and left three questions for whoever unparked
> it. The owner answered all three on 13 Sep, and two of the answers **overturn
> rules the parked document had written down as settled**. They are overturned
> here deliberately and in writing, which is what that document asked for:
> *"should overturn the rule deliberately if that is the answer, rather than
> drift past it."*
>
> **Nothing in this document is built yet. No code was touched to write it.**

---

## What changed, in the owner's words

| Parked document said | Owner, 13 Sep | Status |
|---|---|---|
| *"Never build a relay."* | *"A เต็มรูปแบบเลย"* | **Overturned — narrowly. See §1.** |
| *"The phone is a remote control, not a third desk."* | *"สั่งงานใหม่ตั้งแต่ต้นจากร้านกาแฟได้ ... เลือกโปรเจกต์ พิมพ์งาน ปล่อยมันทำ"* | **Overturned. See §2.** |
| *"LAN, zero-config, scan-and-go."* | *"ไม่จำกัดวงแลน"* | **Superseded. See §3.** |
| *"After 1.0.0."* | *"มันเลย 1.0.0 มานานแล้วครับ"* | **Correct — the tree is on 1.5.28.** |

Two further constraints the owner set the same day, and everything below obeys
them:

- **Free. Not cheap — free.** *"ไม่เอาเสียเงินเด็ดขาด"* No rented box, no paid
  tier, no domain purchase.
- **No account.** *"ระบบล็อคอินอาจจะไม่จำเป็น แค่สแกนคิวอาก็พอแล้ว"*
  [internal/account](../../internal/account) is **not touched**; `DefaultBaseURL`
  stays the empty string and `TestNoServerIsConfiguredInThisBuild` keeps passing.

And one the owner stated as a limit rather than a gap, so it is written here as
a property of the design and not as a defect to fix later: **the machine has to
be on.** *"ปิดคอมก็ใช้ไม่ได้นั่นแหละ"* Work happens on the owner's machine or it
does not happen.

---

## 1. "Never build a relay" — why the old refusal does not reach this

The parked document refused a relay in bold, and the refusal was right about the
thing it was aimed at. Its argument, in full:

> *"Aetox in the cloud has no hands — no files, no terminal, no browser already
> logged in, no project open. What is left is a chat window, which is a category
> where we are the worst entrant."*

**That argument is about where the work runs, and this design does not move it.**
The turn runs on the owner's machine, against the owner's files, in the owner's
shell, with the owner's browser already logged in. What crosses the network is
the conversation about that work.

What survives from the old refusal is the second half — cost, uptime, and attack
surface carried by one maintainer — and §4 answers it by making the thing in the
middle **too dumb to be worth attacking and too cheap to be worth paying for**.

**The distinction to hold, because it is what makes this reversal honest rather
than convenient:** a *rented server* runs your work; a *rendezvous* tells two
programs where to find each other. This document builds the second and still
refuses the first.

## 2. "Remote, not a third desk" — overturned, and the new line drawn precisely

The owner wants to open new work from a phone. That crosses the parked
document's line by name — it listed *"starting new work from the phone with a
fresh project"* under **Deliberately not in this**.

It is overturned. The replacement rule:

> ### The phone opens anything the desk can open, and mirrors none of the desk's rooms.

**Opens anything.** Pick a space, pick a desk, type the work, let it run. Watch
it live, answer its questions, approve its tool calls, type more into it. Stop
it.

**Mirrors nothing.** No file tree, no Monaco editor, no terminal emulator, no
browser panel, no deck editor. When the agent uses those rooms, the phone sees
exactly what the conversation already shows: a diff folded under a tool row
([§161](../DECISIONS.md)), a tool-call line, streamed text.

**Why the line sits there.** The desk's rooms exist so a human can *reach in and
work*. The phone exists so a human can *watch the agent work and steer it*.
A terminal on a six-inch screen is not a small terminal, it is a bad one — and
nobody reaches in from a coffee shop. They steer. The old rule was drawn at the
wrong joint: it split by *how much power*, and the honest split is by *who is
doing the work in that surface*.

This also keeps the parked document's best rule intact, unchanged and
load-bearing: **the phone has no settings page, not one field.** Model,
provider, keys, permissions, agents — all inherited. A phone that can open work
still configures nothing, because it is a second screen on this process and not
a second install.

## 3. Transport — the machine dials out, and nothing in the middle is trusted

The phone is on 4G; the machine is behind a home router. Nobody opens a port and
nobody installs a VPN. So the machine **dials out** and something with a public
address introduces the two.

```
   phone (PWA)                  public address              your PC (Aetox)
        |                             |                            |
        |                             |<--- tunnel, dialled out ----|   no port opened
        |                             |                            |   no router touched
        |--- https ------------------>|---------------------------->|
        |                             |                            |
        |<== E2E-sealed payload ======================================>|
                     (the middle carries it; it cannot read it)
```

**Four ways to be that middle. The design commits to none of them**, because §5
makes the choice reversible:

| | Default | For the careful | For the self-hosting | At home |
|---|---|---|---|---|
| | quick tunnel | user's own named tunnel | user's own relay | plain LAN |
| Setup by the user | **none** | a free account | runs a binary | none |
| Address is stable | no → needs the signpost below | yes | yes | yes |
| Cost | free | free | their own | free |
| Fits the owner's *"scan and go"* | ✅ | ✗ | ✗ | ✅ |

**The signpost.** A quick tunnel's address changes every time the machine
restarts, so the phone needs somewhere to look it up. That is the only service
this project runs: **one endpoint that stores a string against an unguessable
id**, written by the machine and read by the phone. Roughly `PUT /d/<id>` and
`GET /d/<id>`.

Its size is the point. A few requests per machine per day; a free serverless
tier covers thousands of installs without the heavy traffic ever touching it,
because the conversation itself goes through the tunnel. **If the signpost dies,
nothing is lost but discovery** — a paired phone with a stored address keeps
working until the address changes.

**Explicitly refused, and why:** our own box holding every user's stream. It
would put the traffic of everyone on infrastructure one person patches, and it
is the thing the parked document was right to refuse.

## 4. Security — the transport is not trusted, so it does not have to be chosen carefully

**The pairing.** One QR on the desktop, carrying four things: the signpost id
(128 bits of randomness — it is a capability, so it must be unguessable), a
device secret, the machine's ECDH public key, and its fingerprint.

Then the fix the parked document demanded and did not have:

> **A scan is not consent. A six-digit code appears on both screens and the
> desktop answers yes or no.** Pairing needs the screen *and* the machine.

**The wire.** Every payload is sealed end-to-end between the phone and the
machine. Keys are agreed from the QR, which never crosses the network.

What this buys, stated as what an attacker gets rather than as a promise:
whoever runs the middle — Cloudflare, a VPS, us, an attacker who takes any of
them — sees **that two endpoints are connected, when, and how many bytes**. Not
the code, not the file names, not the commands, not the replies.

This matters more than it did in August for a reason that is not technical: the
owner confirmed on 13 Sep that **other people will use this**. While it was one
person, *"the tunnel provider can read my traffic"* was a preference. With other
users it is a disclosure made on their behalf, against a README that sells the
sentence *"not a single byte goes anywhere."* **Sealing the payload is what keeps
that sentence true rather than nearly true.**

**Cost of the seal: no new dependency.** Browsers ship Web Crypto; Go 1.25 ships
`crypto/ecdh` and `crypto/cipher`; `gorilla/websocket` is already in `go.mod`.
Use ECDH **P-256** — supported in every browser Web Crypto, where X25519 is
still too new to rely on.

**The client-agnostic API rule from the parked document is kept**, and now for a
second reason: bearer token in a header, never `Set-Cookie`; pairing returns
JSON. It kept a native client possible; it is also what lets the page be served
over a tunnel that terminates TLS somewhere else.

**Gone: the same-subnet gate.** It was never authentication — the parked
document said so itself — and it cannot mean anything once the peer arrives via
a tunnel. Its replacements are the sealed payload and the device table.

**Kept from [`remote.go`](../../desktop/remote.go), unchanged in spirit:** device
rows store the *hash* of a secret and never the secret, so the table is not a
list of working keys; revoking stamps the row rather than deleting it, so
*"this phone was let in that day and cut off on this one"* stays answerable.

## 5. One rule that makes every choice above reversible

**Nothing above the seal knows what it is running on.**

```
                      ┌─ quick tunnel      (default · free · scan and go)
   sealed session ────┼─ named tunnel      (stable address)
   (never changes)    ├─ self-hosted relay (their box)
                      └─ plain LAN         (already works today)
```

Three problems collapse into this one decision. **Free** — the default costs
nothing. **Terms of service** — a quick tunnel is a testing facility, and if it
is ever throttled the transport is swapped without touching a line of security
code. **Privacy** — no transport is trusted, so none of them has to be audited.

This is the same lesson the parked document learned the expensive way, stated
forwards instead of backwards: *"designed for one reader now, it is a rewrite
later; designed for two now, it is free."*

## 6. What the tree already gives us — including one thing the parked document got wrong

| What the phone needs | Where it already is |
|---|---|
| Open work in a project | `Spaces()`, `RecentProjects()`, `NewSessionInSpace()`, `NewSessionAt(desk)` — [spaces.go](../../desktop/spaces.go), [sessions.go](../../desktop/sessions.go) |
| Send a turn | `SendMessage(text, to)` — one door, [app.go:2659](../../desktop/app.go) |
| Several chats at once, each with its own model and approval mode | `conversation.cfg`, per chat since 2026-08-20 — [conversation.go:49](../../desktop/conversation.go) |
| Live reply, status and preview | `RunOnceStreamWithAttachments` + `OnStatus` / `OnContentPreview` — [app.go:269](../../internal/app/app.go) |
| Tee every UI event to a second listener | `emitEvent` checks the `a.emit` hook first — [terminal.go:84](../../desktop/terminal.go) |
| HTTP server, pairing, device table, revoke, PWA manifest | [remote.go](../../desktop/remote.go), schema v14, 9 tests |
| WebSocket, ECDH, AES-GCM | `gorilla/websocket` in `go.mod`; the rest is Go 1.25 stdlib |

**The per-chat config row is worth reading twice.** It was written in August to
stop one chat's model picker from changing another chat's answers. The property
it bought is exactly the one a phone needs: **work opened from the phone is its
own conversation — it does not take over the screen at home, and the screen at
home cannot change its model or its approval mode mid-turn.** The hardest
prerequisite was met by work aimed somewhere else.

### The correction

The parked document named `Approve` as the hard part and called everything else
plumbing:

> *"A server is the host least likely to have a human watching. So the server's
> `Approve` must park and wait for a real answer ... This is the majority of the
> work."*

**It is already done, by other work.** `approveToolCall`
([ask_user.go:232](../../desktop/ask_user.go)) parks on a channel with no
timeout, and `AnswerUserQuestion(sessionID, answer)` is the single door that
releases it — addressed to a named session precisely because two chats can be
waiting at once. The phone is a **second caller of a function that already
exists**. No engine change, no new approver, no timeout that could fail open.

**So the hard part has moved, and it is no longer an engineering problem.** It
is what goes on the screen — the question that killed the August build, whose
verdict was *"ได้แค่เข้ามาดูประวัติแชท"*. Whoever builds this should spend their
design budget there and not on the transport.

### What is genuinely still missing

**`BackgroundTasks()` has no durable home.** It reads
`a.cur().delegations.Snapshot()` — in memory, bound to the session the window
currently has open ([background_tasks.go](../../desktop/background_tasks.go)).
*"What is it doing right now"* is the first question anyone asks from a phone,
and today it cannot be answered for anything but the chat on screen. This is
independent of every transport choice above, it can be built first, and
[ROADMAP.md](../../ROADMAP.md) already named the gap.

## 7. Slices

| # | Ships | Depends on |
|---|---|---|
| 0 | Machine dials out, signpost, pairing with QR + six digits, "connected" light | — |
| 1 | Read: spaces, sessions, the tail of a conversation | 0 |
| 2 | **Watch a running turn live** — streamed reply, status, tool rows | 1 |
| 3 | **Answer and approve** — `ask:user` teed out, `AnswerUserQuestion` back | 2 |
| 4 | **Open new work** — pick a space, pick a desk, type it, let it run | 3 |
| 5 | Push notification | 4 |
| 6 | Durable background-task state | independent; buildable first |

Slice 4 is the feature. 0–3 exist to make it worth having.

**Streaming rule, decided now rather than tuned later: coalesce chunks into
about 100 ms frames.** Not for bandwidth — for the phone's battery. A radio woken
every 20 ms costs more than the screen, and no eye can tell the two apart.

## 8. Deliberately not in this

- **The desk's rooms, drawn small.** §2.
- **An account.** The QR is the credential. If a later store needs identity,
  [internal/account](../../internal/account) is where that lives, and it is a
  different feature.
- **A second process.** One process owns the store, the MCP children and the
  browser. Two processes over one SQLite file is two versions of the truth.
- **Our own box carrying everyone's traffic.** §3.
- **Native iOS or Android.** The page is served by the same binary it talks to,
  so the two can never be different versions of each other. iOS degrades rather
  than being dropped: it gets everything except reliable push and long-lived
  storage unless the user adds the page to their home screen, which the pairing
  screen should say in one line.

## 9. Open

1. **The phone's screen.** The unanswered question from August, now the hardest
   thing left. What does a running turn look like on six inches?
2. **Which desk does a phone-opened chat default to**, and does the phone choose
   or inherit?
3. **Two chats, one machine, both approving.** The plumbing supports it
   (`AnswerUserQuestion` is addressed to a session). Whether the phone should
   *show* more than one at a time is a design question, not a plumbing one.
4. **What the desktop shows while the phone drives.** A chat opened from the
   phone must not steal the window — but silence is also wrong. Something
   belongs in the background-work tray.
