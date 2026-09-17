# การใช้คอมพิวเตอร์ — Reaching Apps That Are Not Ours (2026-09-07)

Direction document for the reach Aetox does not have yet: driving programs that
are **already running on the user's machine** — their own Chrome, their own
Excel, a window belonging to a process we did not start — and the register in
Settings that says which of them the agent may touch.

Written before the code, deliberately. Status labels follow ARCHITECTURE.md:
`Direct` = confirmed by reading the file, `Proposed` = design intent, not built.
Every row in §5 that has no code behind it says so; **code that lands without its
row is the error this file exists to stop.**

Live code this touches (all `Direct`):
[`internal/mode/mode.go`](../../internal/mode/mode.go) ·
[`internal/safety/safety.go`](../../internal/safety/safety.go) ·
[`internal/capability/capability.go`](../../internal/capability/capability.go) ·
[`desktop/capabilities.go`](../../desktop/capabilities.go) ·
[`desktop/connections.go`](../../desktop/connections.go) ·
[`desktop/browser_tool.go`](../../desktop/browser_tool.go) ·
[`desktop/launch_windows.go`](../../desktop/launch_windows.go) ·
[`Settings.svelte`](../../desktop/frontend/src/lib/Settings.svelte)

---

## 0. Why the document comes first

The prompt for this work was a screenshot of a rival's settings page and one
sentence from the owner: *"เตรียมเพิ่มฟีเจอร์นี้ในระบบ"*.

A screenshot is the easiest thing in the world to copy and the worst thing to
copy from, because a settings page is the **last** artifact of a feature — it
shows the switches and hides every decision that made them mean something. Build
the page first and you get a register of promises: rows with toggles and no hand
behind them. ARCHITECTURE.md's own rule forbids exactly that (`Proposed` is never
drawn as if it existed), and the desk has the same rule in stronger words
([desk-file-panes-2026-08-06.md](desk-file-panes-2026-08-06.md) §0).

So: the shape here, the rows in §5, and the lines in §6 are settled in this file.
Then code.

---

## 1. What that panel actually is — three layers in one list

The rival's page ("การใช้คอมพิวเตอร์ · จัดการวิธีที่ ChatGPT ใช้แอปอื่นๆ บนคอมพิวเตอร์ของคุณ")
reads as one list of apps with switches. It is not. It is three different things
stacked, and only one of them is a permission:

| Row in the picture | What it really is |
|---|---|
| **แอปใดก็ได้** — toggle | A **permission**: may the agent drive apps at all. |
| **Google Chrome / Microsoft Edge** — "ติดตั้ง" button, red dot, *ไม่ได้ติดตั้งส่วนขยายเบราว์เซอร์* | Not a permission at all — an **installer for a reach mechanism**. The switch does not exist until the extension does. |
| **Microsoft Excel** — toggle, *ให้ ChatGPT ใช้ Add-in ของ Microsoft Excel* | A mechanism already installed, so the row is back to being a permission. |

Reading the picture as "a list of apps" is the mistake that produces a dead
register. The list is really **reaches**, each with three possible states:
*built in* · *needs installing* · *not supported on this machine*. An app appears
because a reach reaches it, never the other way round.

---

## 2. What Aetox already has — all three layers, in three different places

None of this is new work. It is the reason the feature is smaller than it looks.

| Layer | Where it lives today | Status |
|---|---|---|
| **Permission, coarse** | `safety.ApprovalMode` — ask / unsafe-only / full-access | `Direct` |
| **Permission, per tool and per argument** | `safety.PermissionRule{Tool, Pattern, Action}` — glob on both, allow/ask/deny, with `Default` marking a rule the app wrote rather than the user | `Direct` |
| **Permission, per desk and per chair** | `mode.AllowsTool` / `AllowsAction` / `Carries` / `CarriesForChair` ([mode.go:285](../../internal/mode/mode.go)) — a profile's `tools:` list is the grant | `Direct` |
| **A register of rows with per-desk switches** | Connections: `connect.Status{ID, Label, Connected, For[]}` + [`desktop/connections.go`](../../desktop/connections.go), drawn in Settings → การเชื่อมต่อ. Row, label, a switch per target, and an honest *"ยังไม่ได้เชื่อม"* line with the way to fix it | `Direct` |
| **An installer whose button returns immediately** | `capability.Statuses()` / `InstallCapabilities()` → `capabilities:progress` / `capabilities:done` events and the background strip ([capability-install-2026-08-21.md](capability-install-2026-08-21.md)) | `Direct` |
| **One tool on the outside, many rights on the inside** | [`browser_tool.go`](../../desktop/browser_tool.go): the pack is named `browser`, the *vocabulary of permission* stays `browser_open` / `_read` / `_click` / `_type`, and the description shown to the model lists only the actions this caller may use. Generalized in [`internal/skill/packed.go`](../../internal/skill/packed.go) | `Direct` |

The register in the screenshot is, line for line, the connections register we
already draw. The "ติดตั้ง" button is the capability installer we already run.

---

## 3. What is genuinely missing

Aetox drives **its own** browser — a WebView2 it created, in the workbench. It has
no way to touch anything it did not create:

- **No window reach.** `EnumWindows` appears once, in `findOwnMainWindow`, and
  filters *out* every process that is not us
  ([browser_windows.go:413](../../desktop/browser_windows.go)) — the opposite of
  what this feature needs. `Direct`
- **No UI Automation.** No `IUIAutomation`, no element tree, no invoke/value
  patterns anywhere in the tree. `Direct`
- **No extension.** Nothing to install into the user's Chrome or Edge; the
  browser we drive is a separate engine with a separate profile. `Direct`
- **Excel is written, not driven.** `sheet_write` builds an `.xlsx` through
  `internal/ooxml` ([sheet_write.go:13](../../internal/skill/sheet_write.go)) — a
  file Excel can open, not the workbook that is open right now. `Direct`
- **Launching is one-way.** `launchDetached` starts a command in its own console
  and lets go ([launch_windows.go](../../desktop/launch_windows.go)). Start,
  never steer. `Direct`

**So the whole of the new work is the hand.** The permission model, the register
and the installer are already built and already tested; they need rows, not
rewrites.

---

## 4. The shape

### 4.1 One packed tool, `computer`

Following `browser` exactly, for the reason `browser_tool.go` records: every new
capability must not cost another entry in the tool block of every request that
carries it.

```
computer(action, …)
  action ∈ { list_apps, focus, read, click, type, close }
```

The pack is granted by the single name `computer` in a profile's `tools:`. The
**actions are the permission vocabulary**: a profile naming none of them gets all;
a profile naming some gets exactly those, and the description handed to the model
lists only those — a tool that advertises what it will refuse is a wasted turn.

`read` is the one that must exist before any of the others are worth having:
**a reach that cannot see is a reach that guesses**, and a guessing click on a
window we did not draw is the most expensive mistake available here.

### 4.2 The register: rows are reaches, not apps

Settings gets a section `computer` — **การใช้คอมพิวเตอร์** — modelled on
การเชื่อมต่อ, not invented:

- one row per reach, with the three states of §1: *พร้อมใช้* · *ต้องติดตั้งก่อน*
  (with the button, wired to `InstallCapabilities`) · *เครื่องนี้ยังไม่รองรับ*
- per-desk placement through `PlacementTargets`, the same ids as an MCP server's
  `for:` — so ผู้ช่วย may reach a spreadsheet while โค้ด does not, without a second
  idea of "which desk" existing anywhere
- a row that is not supported **stays visible and says why**. `connections.go`
  already carries this rule in a long comment, learned twice the hard way: a
  control that vanishes in the broken state is a dead end, not a tidy UI.

### 4.3 Where the permission is actually enforced

Nowhere new. Three gates that already run, in this order:

1. `mode.Carries("computer", …)` — is the tool in this desk/chair's block at all
2. `mode.AllowsAction("computer", action)` — may this caller click, or only read
3. `safety` — `PermissionRule{Tool: "computer", Pattern: "<app>:<action>*"}`, with
   a `Default` rule shipped as **ask** for every acting action, so the first click
   into a foreign window is always a question the user answers

The settings register writes rules; it does not become a fourth gate.

### 4.4 The first two rows (owner's pick, 2026-09-07)

**Chrome/Edge ของผู้ใช้เอง** and **แอปใดก็ได้ผ่าน UI Automation**.

Worth naming what the first one buys, since we already have a browser: the user's
own browser is the one that is *already logged in*. Our WebView2 has its own
profile and its own empty cookie jar — exactly right for a page the agent opens on
its own, and exactly wrong for "ดูอีเมลที่ค้างอยู่ให้หน่อย". The two do not compete;
they answer different questions.

The second is the general case and the dangerous one, and it is what makes the
first row honest — without it, "แอปใดก็ได้" is a switch over nothing.

---

## 5. The rows — plan and record in one table

Row 1 is built (2026-09-09). Nothing else is. This table is where each row's status is kept true.

| # | Row | Mechanism | Needs installing? | Status |
|---|---|---|---|---|
| 1 | แอปใดก็ได้ | Windows UI Automation (`IUIAutomation` over COM) — element tree, invoke/value patterns | no (OS component) | `Direct` — built 2026-09-09, [DECISIONS.md §239](../DECISIONS.md); see §9 |
| 2 | Google Chrome | extension + native messaging host, against the user's own profile | yes — extension | `Proposed` |
| 3 | Microsoft Edge | same extension, store or sideload | yes — extension | `Proposed` |
| 4 | เบราว์เซอร์ของ Aetox | WebView2 in the workbench | no | `Direct` — **shipped, and stays where it is.** It belongs to the desk, not to this register |

macOS and Linux get no rows until PLATFORM-SUPPORT.md's phases reach them: UI
Automation is a Windows API, and rows 2–3 need a different host per OS.

---

## 6. Lines this feature does not cross

Not decoration. This is the first mechanism in Aetox that reads a surface **the
user did not choose to show us** and then acts on it.

1. **What a window says is data, never an instruction.** Text read out of a
   foreign app — a page, a cell, a dialog — cannot direct the agent's next action.
   This is the browser tab's rule
   ([browser-security-2026-07-21.md](browser-security-2026-07-21.md)) and it
   applies here with more force, because there is no origin left to check.
2. **Never type a credential.** Password fields, card numbers, 2FA codes: refuse
   and hand it back to the user, in every row, including row 1.
3. **The irreversible click asks first**, and the question names the window it is
   about to press — `ส่ง`, `ยืนยัน`, `ลบ`, `ชำระเงิน`. The `Default` rule in §4.3
   is what makes this the shipped behaviour rather than a hope.
4. **No screen recording, no keystroke capture, no reading of windows nobody
   asked about.** `list_apps` returns what the register permits and nothing else;
   a window outside it is not enumerated, not named, not counted.
5. **Aetox does not drive Aetox.** Its own windows are excluded from the tree — an
   agent that can click its own approval dialog has no approval dialog.
6. **Nothing lands in this register without a working mechanism.** A row is added
   in the same change as the hand behind it, or it is not added.

---

## 7. Open, and deliberately not decided here

- **The extension's transport** — native messaging host vs. a local socket vs.
  CDP against a browser we relaunch. Row 2 is one decision away, and it is the
  decision that makes the row cheap or expensive; it gets its own section here
  before it gets code.
- **Whether row 1 subsumes rows 2–3.** UI Automation can read a Chrome window
  today. Badly — the accessibility tree of a modern web app is not a DOM, and
  "badly" is how a reach becomes something the owner has to apologise for. The bet
  is that they stay two rows; if row 1 turns out to be good enough at a browser,
  row 2 becomes a nicety and this table changes.
- **Excel is not a dedicated reach.** The owner removed its placeholder row on
  2026-09-17. `sheet_write` remains the file-producing path, while a running
  Excel window is treated like any other explicitly granted Windows program by
  row 1. There is no Excel add-in or COM-specific promise in this register.
- **The section's Thai name.** `การใช้คอมพิวเตอร์` is borrowed from the rival
  because it is what a user will look for. If the house voice wants its own word,
  it changes here first.

---

## 8. Order of work

1. `computer` pack with **`list_apps` + `read` only** — the reach that cannot
   break anything, behind the three gates of §4.3.
2. The Settings section, drawing rows from the shape `connect.Status` already
   uses — row 1 real, rows 2–3 drawn honestly as *ต้องติดตั้งก่อน*.
3. `focus` / `click` / `type` for row 1, each arriving with its `Default` ask rule
   and the refusals in §6.2–6.3.
4. Row 2's transport decision, then the extension.

Steps 1 and 2 are the ones that make the register true; nothing after them changes
its shape.

---

## 9. Appendix — what building it added (2026-09-09)

Appended rather than edited into the sections above, because §0 says the shape
here is settled and a settled decision that quietly changes is worse than one
that is amended in public. Everything in §1–§8 still stands. Implementation is
[DECISIONS.md §239](../DECISIONS.md).

**Two actions joined §4.1's six.**

| Change | Why |
|---|---|
| **`capture` added** | `browser` has one and half the models in the picker have eyes. It is a picture of ONE window, taken by asking that window to draw itself, so nothing behind it or on another monitor is in it — which is why it does not cross §6.4's line against screen recording. It is for SEEING, never for aiming: a picture has no refs, and `read` is what a click is aimed by. |
| **`keys` folded into `type`, not made an action** | Both are typing. A user deciding "Aetox may type in this program" has not thereby drawn a line between a word and ctrl+s, so a separate permission would be a right nobody asked for and nobody would understand. |

**Two things §4 did not decide, and the owner did after using it.**

**The switch removes the tool, it does not refuse its calls.** §4.3's three gates
all assume the tool is there and the gate says no. A fourth answer turned out to
matter more: with the setting off, `computer` is not registered at all. A model
that reads it in the block plans around a capability it will be denied, and pays
~280 tokens a request for the privilege.

**The programs are chosen in Settings, not asked for mid-turn.** §4.3's
`Default: ask` rule made the first click into a foreign window a question. It
worked, and it asked at the worst possible moment — a card raised while an agent
waits is answered in a hurry, and everything about that moment argues for yes.
The rule it becomes: the user picks programs on this page, from the windows open
or by browsing to an executable, and a reach at anything else is refused and
names what to add. Same `PermissionRule`, same file, same register; the deciding
moved to where there is nothing waiting.

**And the screen is taken.**

An acting call raises the window it works on, holds a machine-wide lock, and
lights the whole edge of the screen for as long as it is driving. That is the
Codex-on-Windows model rather than the macOS one, chosen deliberately: a change
made in a window the user never saw is a change they cannot check.

The light is a window of its own — layered, click-through, never activated,
always on top — because the strip inside Aetox's chat is *behind* the window
being driven at exactly the moment it matters. The owner found this by watching
it run: *"ขอแสงวิวับที่ทำไว้อ่ะมาครอบจอด้วย ไม่งั้นไม่รู้"*.

**§5's table, updated.** Row 1 is `Direct`. Rows 2–3 are unchanged and are drawn
in Settings as *ต้องติดตั้งก่อน*, which is §4.2's rule about a row that stays
visible and says why. The former Excel placeholder was removed on 2026-09-17.

**§6, in code.** Each line is now a place rather than a promise: 6.1 is the
untrusted-data preamble on every read and capture; 6.2 is `IsPassword`, refused
in two places on purpose; 6.4 is a window capture rather than a screen one, and
a `list_apps` that omits what it will not touch; 6.5 is a process-id check and a
name check, because a second copy of Aetox is still Aetox.

---

## 10. Extension study — Codex comparison and Aetox decision (2026-09-17)

The owner asked which extension would let Aetox control the computer and asked
to compare Codex. Two different reaches must not be collapsed into one word:

| Reach | What Codex demonstrates | What Aetox should ship |
|---|---|---|
| Windows applications | A Computer Use plugin exposes windows, accessibility text, bounded screenshots, clicks, keys, values, scrolling and dragging. It is a desktop runtime, **not a browser extension**. | Keep the shipped `IUIAutomation` reach. Add actions only where the existing permission and visible-driving rules can protect them. No extra extension is required. |
| The user's signed-in Chromium browser | A separate Browser plugin can discover an `extension` browser, claim a user tab, use accessibility indexes, and fall back to Playwright-style locators. Its Windows bridge is a browser extension plus a registered Native Messaging host. | Build one **Aetox Browser Bridge** WebExtension and one Aetox native host. Package the same Manifest V3 source for Chrome and Edge first; Brave, Opera and Vivaldi can use the Chromium build later. |
| A browser Aetox owns | Codex also distinguishes an in-app browser and CDP-backed browsers from extension-backed user tabs. | Keep the current WebView2 workbench browser. It is the no-install, isolated-profile path and does not replace the signed-in-browser bridge. |
| OpenAI `computer` tool | The Responses API returns structured computer actions, but the application still provides the environment, executes actions, preserves state and returns screenshots. | Treat this as a possible model-side protocol, not an extension or a local reach. It cannot replace UI Automation or the browser bridge. |
| MCP | MCP can expose UI actions as tools, but it is transport/tool vocabulary rather than screen or DOM access by itself. | Optional adapter later. Do not make MCP a prerequisite for local computer control. |

### 10.1 One extension, two store packages

Chrome and Edge do not need different implementations. They need separate store
listings/identities around the same extension source, plus a native-host manifest
that allow-lists those exact extension origins. The desktop installer owns the
native host and its per-user registration; the extension never opens an
unauthenticated localhost control port.

The minimum bridge is:

```
Chromium tab
  ↕ content script / accessibility snapshot / bounded screenshot
Aetox Browser Bridge (Manifest V3)
  ↕ nativeMessaging, explicit tab claim and short-lived session capability
Aetox native host
  ↕ existing computer/browser permission gates
agent tool loop
```

The extension should expose fresh element indexes or refs, not raw arbitrary
JavaScript as its public contract. A tab is claimed before reading or acting;
indexes expire after a page change; password fields and browser-internal pages
remain denied; every acting result returns a fresh observation. These are the
same invariants already used by Aetox's own browser and Windows reach.

### 10.2 Rejected substitutes

- **Playwright/CDP alone:** good for a browser Aetox launches, but attaching to
  the user's ordinary browser requires remote debugging at launch and changes
  the security/profile story. It is not the default route into an already-open,
  signed-in tab.
- **PyAutoGUI alone:** useful as a screenshot/coordinate fallback, but weaker
  than the shipped accessibility reach for aiming, permissions and recovery.
- **An Excel add-in:** removed from scope. It duplicates the general Windows
  reach, adds an Office-specific deployment surface, and promises more access to
  live unsaved workbooks than the current product needs.

So the installable work is deliberately small: **one Chromium browser bridge,
published for Chrome and Edge, plus its native host**. General Windows control
already exists without an extension.

### 10.3 Native reach hardening shipped on 2026-09-17

The Windows reach now has the three input primitives the comparison exposed as
missing: `click_at`, `scroll` and `drag`. They are not arbitrary screen-input
calls. A model must first call `capture`; that capture mints a chat-local
snapshot id and records the exact window dimensions. Coordinate input accepts
only that latest id, rejects points outside the captured image, re-checks the
window dimensions immediately before input, and expires both snapshot and UIA
refs after one action. Moving the window is safe because its current origin is
read again; resizing it makes the snapshot stale and is refused.

Program grants were hardened in the same pass. New grants are keyed to the
executable's normalized absolute-path fingerprint rather than only its basename,
so an unrelated executable named `notepad.exe` does not inherit the real
Notepad's permission. The settings page still shows and revokes the familiar
program name; the stored permission does not expose the installation path.

The Chrome/Edge Browser Bridge remains `Proposed`, not silently half-enabled.
It is complete only when the extension, signed store identities, per-user Native
Messaging host registration, explicit tab claim, and end-to-end installed-build
test ship together. Until that boundary is crossed, Settings continues to say
that the browser extension is unavailable and the existing workbench browser is
the supported web reach.
