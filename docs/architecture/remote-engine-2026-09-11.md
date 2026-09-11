# จอกับเครื่องยนต์ — The Desktop as a Screen on an Engine Process (2026-09-11)

Direction document for coding on a machine that is not the one the window is
on: the desktop becomes a **screen**, the chat loop and its tools become an
**engine process**, and the engine runs either as a child on this machine or
on a Linux host at the far end of an ssh tunnel. The VS Code Remote-SSH shape,
not the "teach every tool to cross the network" shape.

Written before the code, deliberately. Status labels follow ARCHITECTURE.md:
`Direct` = confirmed by reading the file, `Proposed` = design intent, not
built. Everything in §3–§7 is `Proposed` until its phase lands; §10 is where
each phase records that it did.

Live code this touches (all `Direct`):
[`desktop/app.go`](../../desktop/app.go) ·
[`desktop/conversation.go`](../../desktop/conversation.go) ·
[`desktop/ask_user.go`](../../desktop/ask_user.go) ·
[`desktop/terminal.go`](../../desktop/terminal.go) ·
[`desktop/filehost.go`](../../desktop/filehost.go) ·
[`desktop/remote.go`](../../desktop/remote.go) ·
[`internal/bootstrap/bootstrap.go`](../../internal/bootstrap/bootstrap.go) ·
[`internal/model/httpclient.go`](../../internal/model/httpclient.go) ·
[`internal/model/factory.go`](../../internal/model/factory.go) ·
[`internal/lsp/lsp.go`](../../internal/lsp/lsp.go)

---

## 0. Why

The owner asked whether the system could ssh to another host and code there,
was shown four levels — run `ssh` from the shell tool (works today, shell only);
an ssh `proc.Backend` beside the WSL one (shell only, still); a remote
filesystem behind every file tool (87 `os.*` call sites in 34 files, a rewrite
of `internal/skill`); or move the engine to the host — and answered:
*"ที่ผมอยากได้คือระดับ 3 ครับ"*. Then, once the shape was on the table:
*"เห็นด้วยครับ แต่เรื่องนี้ลึกมากและกระทบหลายส่วน ร่างแผนมาก่อนลงมือจริง"*.

What the survey found is why the plan is a document first. `desktop.App` is one
Wails-bound struct with **389 exported bindings in 71 files** (`app.go` alone is
5,447 lines). About 290 of them are the engine — chat, turn, tools, the
session store, pending changes, plans, MCP; about 45 are the window — WebView2
tabs, computer-use, voice; about 55 are both, mostly a dialog followed by
engine work, or `revealInFileManager`. Nothing stops one half reaching into the
other, which ARCHITECTURE.md's Reader's Map has said since July. The remote
feature is the first thing that *needs* the boundary, so the boundary is most
of the work and the ssh part is the last few weeks.

## 1. The rule — six decisions, settled 11 ก.ย. 2026

1. **The screen and the engine are two processes.** The engine owns
   `aetox.db`, the MCP children, every file/shell/git tool, sessions, pending,
   plans, memory — and the **terminal pane**, because both PTY backends are
   pure Go and the only window dependency in `openDeskTerminal` is an
   "is the UI up" guard on `a.ctx`. The screen owns the Wails window, the
   WebView2 tabs, dialogs, computer-use, voice/TTS/STT, self-update, and the
   Aetox account.
2. **Local mode goes through the socket too.** One code path. There is no
   in-process fast path to drift away from the wire while nobody is using
   remote; every day of ordinary use is a test of the remote path. (In Go tests
   the socket may be an in-memory listener — it is still the RPC path.)
3. **The provider API key never leaves the screen machine.** The engine cannot
   hold a key: `internal/engine` may not import `internal/credentials` or
   `internal/oauth`, and a test enforces it. A model call is an HTTP request
   the engine hands *back* to the screen, which attaches the key or the OAuth
   token and forwards it. The proxy is at the **HTTP level** — the body is
   already provider JSON and a stream is already SSE or NDJSON bytes — not at
   the `model.Provider` interface, whose three callbacks, `[]byte` images
   tagged `json:"-"` and `errors.Is` sentinels would all have to be reinvented
   on the wire.
4. **The Linux engine binary is fetched on demand**, through the same
   `internal/capability` installer that fetches ffmpeg — manifest, sha256,
   GitHub Releases assets — and never bundled in the Store package.
5. **Every other secret belongs to the host it is used on.** GitHub PAT,
   n8n/windmill keys, cloud image/speech keys, MCP headers, `.env`: per host in
   v1. This is not a shortcut. A credential has an owner, and the owner is
   wherever the thing is *used*: `git push` already uses the host's own ssh key
   or credential helper; an n8n on a LAN is reachable from one machine; an MCP
   server started on the host reads its key from that host's environment. The
   model key is the exception because the model is on no host at all — only
   the screen should know it. `provider.open` names its auth header in a field
   so that widening it into a general credential proxy later is one more
   method, not a redesign.
6. **How the big move lands is chosen when it is reached** — see §9 Stage B.

### What this overturns, and why the objection does not apply

[mobile-remote-2026-08-14.md](mobile-remote-2026-08-14.md) refused *"a separate
daemon binary. One process owns the store, the MCP children, and the browser.
Two processes over one SQLite file is two versions of the truth."* The
objection was about **two writers**. Here exactly one process links sqlite and
opens `aetox.db` — the engine — and the screen never does, in local mode or
remote. There is one truth; it moved hosts. That sentence is struck in the
mobile doc and points here.

### What this keeps

From the same doc: *the server host goes through `bootstrap.Engine` or it does
not get built* (the engine is built by `bootstrap.Engine` and nothing else);
*Approve parks and waits for a real human, a timeout is a no*; *do not shape the
API around a browser* — bearer token in a header, JSON on the wire, no served
HTML. And the phone remote itself stays parked and untouched: once the screen
is one client of the engine, a phone is another, through the same door, which
is the "no second gate" rule that file wrote for itself.

## 2. Ownership

```
screen machine (Windows)                          host (this machine, or Linux over ssh)
┌───────────────────────┐  one WebSocket, JSON-RPC 2.0  ┌──────────────────────────┐
│ desktop/  App = screen│ ── bindings (call) ─────────▶ │ cmd/aetox-engine          │
│ Wails, WebView2, dialog│ ◀─ 37 events (notify) ─────── │ internal/engine.Engine    │
│ computer, voice, update│ ◀─ provider.open (call) ───── │ bootstrap.Engine, tools,  │
│ credentials, oauth    │ ── provider.chunk (notify) ──▶ │ aetox.db, MCP, PTY        │
│ screen.tool handler   │ ◀─ screen.tool (call) ─────── │ /file/ on the same socket │
└───────────────────────┘                               └──────────────────────────┘
```

### 2.1 Packages

| Where | What |
|---|---|
| `internal/engine/` (package `engine`, type `Engine`) | everything engine-side above, plus the terminal and `conversation`. **May not import** `wailsapp`, `internal/credentials`, `internal/oauth` — `deps_test.go` runs `go list -deps` and fails the build if they appear. |
| `internal/engine/api_gen.go` | `type API interface` — every exported method of `*Engine`, generated. |
| `internal/engine/screen.go` | `Screen` and `Session`: the engine → screen surface (§3). |
| `internal/engine/rpc/` | JSON-RPC 2.0 over WebSocket: `Conn` (the `internal/lsp` pattern with two id spaces and a goroutine per request), `Server`, `Client` (implements `engine.API`), `ScreenPeer`, the provider-proxy `RoundTripper`, the screen-tool stub, the `/file/` handler; `client_gen.go`, `server_gen.go`. |
| `internal/engine/rpc/gen/` | the generator — `go/ast` only, no new dependency. Reads the exported methods of `*Engine` (minus `Close`, `Attach`) and emits the interface, the screen's forwarders, the client stubs and the server dispatch table. `TestGeneratedFilesAreCurrent` regenerates into a temp dir and diffs. |
| `internal/credentials/` | `credentials.json` — load, `KeyFor`, save, forget — moved out of `internal/config`, still wrapped by `atrest`. |
| `cmd/aetox-engine/` | ~150 lines: `serve --socket <path> | --tcp 127.0.0.1:0`, `--root`, `--token-stdin | --token-file`, `--idle-exit 30m`. Console is `DiscardConsole`; log is `<DataRoot>/logs/engine.log`, a different file from the desktop's so local mode never has two writers on one log. **A binary of its own, not `aetox serve`:** `release.yml` deliberately does not ship the CLI (§30); a fixed file name is what `ps`, `pkill` and `~/.aetox/server/<version>/` need; and the CLI's own future is to become a second screen on this engine, in a later phase. |
| `desktop/` (package `main`, `App` = the screen) | `main.go`; a new `app.go` of ~250 lines (`ctx`, `api engine.API`, `browsers`, `speakJobs`, the computer-use lock, `openDir`, `emit`, `remoteSrv`, `staged`, `exported`); `browser*.go`, `computer_*.go`, `uia_windows.go`, `speak/speech/voice*.go`, `ttshost.go`, `update_notify.go`, `account.go`, `oauth.go`, `mcp_oauth.go`, `deck_render/reveal/pick.go`, `export.go`, `remote*.go`; new: `engine_forwarders_gen.go`, `engine_local.go`, `engine_remote.go`, `provider_forward.go`, `screen_tools.go`, `engine_status.go`. |

Files that move to `internal/engine` (renamed `App`→`Engine`, receiver
`a`→`e`): `app.go`→`engine.go` minus ~400 lines of window (`startup`,
`beforeClose`, `shutdown`, `fitToScreen`, dialogs, `Quit`); `conversation.go`
whole — its `openTabs`, `video`, `taskChips`, `askCh` are engine data that
bindings and tools push in, none of it touches Wails; `db.go` and the
migrations; `sessions*.go`, `pending.go`, `plan*.go`, `goal_run.go`, `usage.go`,
`jobs.go`, `spaces.go`, `subagents.go`, `office.go`, `mcp.go`, `connections.go`,
`skills.go`, `skilltune*.go`, `habits*.go`, `presets.go` (presets expand inside
`runTurn`, so they live where the turn runs), `sources.go`, `stance.go`,
`delegate_switch.go`, `background_tasks.go`, `task_chips.go`, `ask_user.go`,
`regenerate.go`, `review.go`, `prepared_reply.go`, `workspace.go`,
`browse_root.go`, `shell_backend.go`, `git_*.go`, `pr_room.go`,
`engine_server.go`, `run_block.go`, `run_script.go`, `repomap_view.go`,
`artifact*.go`, `model_load.go`, `image.go`, `video_*.go`, `videotooling.go`,
`decks.go` with `deck_flatten/draw/image/pdf` (building a deck; rendering one
through a hidden WebView2 stays on the screen), `capabilities.go` (tools are
installed where tools run), `filehost.go` (becomes the engine's `/file/`
handler), `terminal*.go`, `workbench_desk.go`, `busy_signal.go`, `address.go`,
`launch_log.go`. `workbench.go` splits: the `ExtraSkills` assembly goes to the
engine, the browser implementation stays.

### 2.2 The "both" methods — one rule

**Dialog → a path on the host → an engine binding.** Every dialog site becomes
`pick*()` on the screen (a Wails dialog in local mode, an in-webview picker in
remote mode) plus an exported `*Path(path)` or `*Bytes(name, data)` on the
engine. `OpenProjectFolder`/`OpenProjectPath` already have this shape and are
the template. The fourteen `revealInFileManager` sites become `<Thing>Path()`
on the engine, which answers with the host path, and `revealInFileManager` on
the screen, which in remote mode returns a named error — *"ไฟล์อยู่บนเครื่อง
<host>"* — shown verbatim. Degraded, not hidden. `applyConfig` stays whole on
the engine.

### 2.3 `config.Config`

| engine | screen (v1: still stored in the engine-owned `model-preference.json`, read and written through the bindings Settings already calls) | removed |
|---|---|---|
| `SandboxRoot`, `ApprovalMode`, `AutoApprove`, `MaxRetries`, `ApprovalTimeoutSec`, `ThinkLevel`, `ModelProvider/Name/BaseURL/WireFormat/TimeoutSec/ContextTokens`, `ImageEngine/ModelName`, `UILocale`, `Delegate*`, `WorkersOff` | `Speech*`, `TTS*`, `Busy*` | **`ModelAPIKey`**, `ModelPreference.ModelAPIKeys` |

The wart this buys: in remote mode the TTS voice follows the host. Written
down; a `screen-preferences.json` split is a later phase if it bites.

### 2.4 DataRoot — disjoint writers

In local mode both processes share `<DataRoot>` and **no file has two writers**.
Engine: `aetox.db`, `model-preference.json`, `memory/`, `modes/`, `agents/`,
`subagents/`, `tools/`, `identity/`, `permissions.json`, `hooks.json`,
`mcp-servers.json`, `connections.json`, `snapshots/`, `project/`, `workspace`,
`shell-audit.log`, `models/`, `bin/`, `prompts/`, `logs/engine.log`. Screen:
`credentials.json`, `oauth.json`, `account.json`, `webview/`, `updates/`,
`update-check.json`, `screen.json` (new — remote hosts and their tokens,
`atrest`-wrapped), `logs/desktop.log`. This is an invariant, not a description.

## 3. The seam, in-process first

Before any socket exists the boundary is an interface the screen implements
and the engine calls. It is also, unchanged, what the wire carries in the
engine → screen direction.

```go
// internal/engine/screen.go
type Screen interface {
    Emit(event string, data any)                          // the 37 event names; data is what Wails would marshal
    ProviderTransport(provider string) http.RoundTripper  // phase 1: attaches the key; phase 2: proxies over RPC
    WindowTools(s Session) []skill.Skill                  // phase 1: the real browser/computer packs; phase 2: stubs
}
type Session interface { ID() string; Root() string; Emit(event string, data any) } // *conversation
```

Every engine → frontend event already leaves through one door,
`emitEvent` ([terminal.go:84](../../desktop/terminal.go)), which has a test
seam (`a.emit`) — seven call sites in `pending.go` and `task_chips.go` still
call `wailsruntime.EventsEmit` directly and are the first commit. Every
blocking question the engine asks — tool approval, `ask_user`, the workspace
widen — already uses one mechanism: emit `ask:user`, block on
`conv.askCh`, be answered by the `AnswerUserQuestion` binding
([ask_user.go:253-298](../../desktop/ask_user.go)). Event out, call in. That
shape crosses a wire without changing.

The provider seam: `newModelHTTPClient`
([httpclient.go:40-80](../../internal/model/httpclient.go)) builds every wire
client's `*http.Client` and today has no injection point. `ProviderOptions`
gains `Transport`, `TokenSource`, `Headers`; when `Transport` is set the four
wire clients attach no auth header and require no key — the transport
authenticates. `factory.go` stops importing `oauth` (its three lookups move to
the caller). `bootstrap.Options.ProviderTransport` threads it through — a
field, not a package-level setter, for the reason `Approve` is a field: tests
build several engines in one process and the CLI in the same binary must not
inherit it. `retryTransport` and `idleTimeoutBody` stay on the engine, wrapping
whatever transport it is handed exactly as they wrap `DefaultTransport` now.

## 4. The wire

**Transport.** One WebSocket per screen↔engine pair (`gorilla/websocket`
1.5.3, already a direct dependency), text frames, one JSON-RPC 2.0 object per
frame, no batching, both directions. Ids are strings with a side prefix —
`"s12"` from the screen, `"e7"` from the engine — so the two id spaces cannot
collide. The server runs each request on its own goroutine; each side has one
writer goroutine fed by an ordered queue, because events must keep their order
and gorilla requires a single writer. Ping every 15 s, pong deadline 30 s.

**Shapes.**

```jsonc
// a binding: method is the Go name, params positional to match the TS signature
{"jsonrpc":"2.0","id":"s12","method":"SendMessage","params":["เทสๆ",""]}
{"jsonrpc":"2.0","id":"s12","error":{"code":-32000,"message":"…","data":{"kind":"turn_busy"}}}

// an event, engine → screen; the screen hands data to EventsEmit as json.RawMessage
{"jsonrpc":"2.0","method":"event","params":{"name":"agent:chunk","data":{…}}}

// the provider proxy: engine → screen call, then screen → engine notifications
{"jsonrpc":"2.0","id":"e7","method":"provider.open","params":{"id":"p3","provider":"anthropic",
  "auth":"x-api-key","method":"POST","baseURL":"https://api.anthropic.com","path":"/v1/messages",
  "headers":{…},"body":"<base64>","headerTimeoutMs":300000}}
  → {"status":200,"headers":{…}}
{"jsonrpc":"2.0","method":"provider.chunk","params":{"id":"p3","data":"<base64, ≤32 KiB>"}}
{"jsonrpc":"2.0","method":"provider.close","params":{"id":"p3","error":""}}
{"jsonrpc":"2.0","method":"provider.cancel","params":{"id":"p3"}}      // engine → screen, on ctx done

// a window tool, engine → screen
{"jsonrpc":"2.0","id":"e8","method":"screen.tool","params":{"session":"…","root":"…","name":"browser","args":{…}}}

// the handshake, first message from the screen
{"jsonrpc":"2.0","id":"s1","method":"hello","params":{"protocol":1,"version":"1.5.x",
  "tools":[/* model.ToolDefinition… */],"features":["dialogs","fileManager"]}}
  → {"protocol":1,"version":"1.5.x","root":"/home/u/proj","dataRoot":"…","os":"linux","arch":"arm64","pid":4242}
```

Bodies are base64: JSON-safe, binary-safe, and SSE and NDJSON pass through
byte-exact. Binary frames are an optimisation for later, not a design change.
`SendMessage` holds `s12` for the whole turn; `CancelTurn`,
`AnswerUserQuestion` and every event flow past it because the pending map is
keyed by id. A Go `error` becomes `-32000` with `data.kind` from a small
registry (`turn_busy`, `no_screen`, `canceled`, …); the client returns an
`*rpc.Error` whose `Is()` matches the sentinel; the frontend only ever saw
`err.message` and still does.

**Auth and socket.** A 32-byte token generated by the screen, handed to the
engine on **stdin** (never argv), carried as `Authorization: Bearer` on the
handshake; one middleware guards `/rpc` and `/file/`. The socket is
`net.Listen("unix", <DataRoot>/engine-<pid>.sock)` on both OSes — Go supports
AF_UNIX on Windows 10 1803+, with the usual caveats: `sun_path` ≤ ~107 bytes,
remove a stale file before listening, no `SO_PEERCRED`, so the token stays.
The fallback is `--tcp 127.0.0.1:0` plus the token (the engine prints one
`{"addr":…}` line), which is also exactly where the ssh tunnel lands in §5 —
no named pipes, no `go-winio`. Proving AF_UNIX on Windows is the first test of
phase 2.

**Where the proxy plugs in.** `Screen.ProviderTransport(provider)` on the RPC
side returns a `RoundTripper` whose `RoundTrip` sends `provider.open` and
returns an `*http.Response` whose `Body` is the read end of an `io.Pipe` fed by
`provider.chunk` and closed by `provider.close`; a done context sends
`provider.cancel`. On the screen, `provider_forward.go` resolves auth —
`oauth.TokenSource` for a signed-in provider (Bearer plus `oauth.Headers`),
else `credentials.KeyFor` under the header the engine named in `auth` —
honours `oauth.Endpoint` when `baseURL` is the catalog default, sets
`ResponseHeaderTimeout` from `headerTimeoutMs`, and streams the body in ≤32 KiB
reads. `NoteQuotas` stays on the engine, unchanged: the response headers come
back in the `provider.open` result.

**No screen attached.** The proxy waits up to 60 s for a peer, then fails with
`engine.ErrNoScreen` and the turn is written down as failed. An approval or an
`ask_user` parks indefinitely and the pending question is re-emitted on
reconnect — *park and wait; a timeout is a no*, kept.

**`/aetox-file/`.** Files reach the panes as URLs, not binding values
([filehost.go](../../desktop/filehost.go)), and `SlidesPane` relies on
same-origin DOM access to its iframe. So the engine serves `GET
/file/<rel>?session=<id>` on the same listener — today's `resolveProduced`,
`http.ServeContent`, Range, `no-store`, the pinned content types — and the
screen's `fileHost` middleware becomes an `httputil.ReverseProxy` into the
socket or tunnel, adding the bearer and rewriting the prefix. The webview still
fetches the Wails origin.

**Window tools do not wait for phase 4.** Decision 2 means local mode goes
through the socket the moment phase 2 flips; the browser and computer tools
would be dead in daily use until phase 4 otherwise. So the mechanism lands in
phase 2: an engine-side `skill.Tool` stub per window tool whose `ExecuteTool`
calls `screen.tool`, with the definitions arriving in `hello` and registered
as `ExtraSkills` with the workbench source — `desk.DrivesMachine` and the
per-action permission keys never notice.

**The local child.** `NewApp` spawns `<exe dir>/aetox-engine serve --socket …
--token-stdin`, writes the token and keeps stdin open. `proc.KillTreeOnExit`
(the Job Object) covers Windows; on Linux and macOS the child is `Setpgid` and
exits on stdin EOF. A crash restarts it with backoff (three a minute, then
`failed`), reconnects, and re-syncs: the screen replays the last
session-selecting call it saw, then re-emits `model:switched`,
`learning:changed`, `tasks:changed`, `workspace:changed`, `plan:update`,
`stance:update` so the existing listeners refetch. Live agent context is
rebuilt from the transcript by `LoadSession`, as after an app restart today.
One new event, `engine:status`, drives one new chip.

## 5. Over ssh

`desktop/engine_remote.go` drives the system `ssh` from `PATH`, so
`~/.ssh/config`, the agent and jump hosts come for free:

1. `ssh <host> 'uname -sm; test -x ~/.aetox/server/<ver>/aetox-engine && echo ok'`
   — architecture (`x86_64`→amd64, `aarch64`→arm64) and whether this version
   is already there.
2. If not: fetch `aetox-engine-linux-<arch>` through `internal/capability`
   and stream it up on stdin — `ssh <host> 'mkdir -p … && cat > …/aetox-engine.tmp
   && chmod +x … && mv …'`. No `scp`, so it works through a jump host.
3. Start it detached with the token on stdin and `--idle-exit 30m`; if an
   engine is already listening, try the token stored in `screen.json` first
   and reuse it.
4. Tunnel: pick a free local port, then `ssh -N -o ExitOnForwardFailure=yes
   -L 127.0.0.1:<port>:/home/<u>/.aetox/server/engine.sock <host>`. A
   unix-socket *target* is handled by the remote sshd; the client only
   requests the channel, and Win32-OpenSSH tracks upstream — but this is a
   spike to be proven on the owner's machine before anything is built on it.
   The fallback is designed in from the start: `--tcp 127.0.0.1:0` on the host
   and a plain port forward; the token keeps other users on the host out, the
   ssh channel is the encryption.
5. Supervise the `ssh -N`; on exit, back off and re-dial. The engine keeps
   running; a turn caught mid-request fails after 60 s with `ErrNoScreen`;
   parked questions come back on reconnect.

The screen's directory dialog cannot browse the host, so the engine gains
`ListDir(path)` and `HomeDir()` and the screen a modal picker that reuses the
Files pane's row rendering. Settings gains a "เครื่องระยะไกล" section: hosts,
paths, engine version, connect/disconnect, a log tail.

**Settings on the host are per host in v1.** `permissions.json`, `hooks.json`,
`mcp-servers.json`, `modes/`, `memory/`, skills, `prompts/` live in the
engine's DataRoot on that machine and say so in Settings. A "copy my settings
to this host" action is a later phase — a tar over the same ssh session.

**The terminal on the host is free.** Because the pane moved to the engine in
phase 1, `TerminalStart` on Linux spawns `creack/pty`, output leaves as
`terminal:data:<id>` through the same door, and `TerminalShells` lists the
host's shells. Nothing to build; something to prove.

## 6. Window tools across the wire

| tool | class | why |
|---|---|---|
| `ask_user`, approval, workspace-widen, `todo`, `plan`, `desk` open/list/close, `suggest_task`, `desk_terminal` | already wire-shaped | event out, binding in; the mirrors live in `conversation`; the PTY is on the host |
| `session_search`, the video pack, `image_make`, deck *build* | engine | reads the DB, or is a shell-type tool that runs where the files are |
| `browser` (every action), `computer` | screen-tool proxy (§4) | acts on the screen machine, which is right: the logged-in browser and the mouse are there |
| deck *render* (hidden WebView2), cutting-room preview | screen-tool proxy | the engine builds HTML, `screen.render.deck` returns PNG/PDF bytes |
| `revealInFileManager`, `OpenExport` | unavailable in remote v1 | a named error, shown as it is |
| a browser/computer action that **writes a file** into the project (a screenshot) | refused in remote v1 | *"บันทึกไฟล์ข้ามเครื่องยังไม่รองรับ"*; phase 4 routes it through `PUT /file/attach`, as it will attachments and drops |

The stub uses `skill.Tool` as it exists; `internal/skill` is not touched.

## 7. Tests

Phase 1: `internal/engine/deps_test.go` (the import ban); `api_gen_test.go`
(generated files current, `*Engine` satisfies `API`); the ~105 engine test
files move with a rename and keep their struct-literal seams (`dbDir`,
`emit`); the ~16 screen test files use a `screenSeed`; `stream_delivery_test.go`
— a whole turn with the built-in `aetox-render:test` model and no network —
moves unchanged in substance. `go build ./... && go test ./...` green at every
commit; the frontend sees one mechanical change (§9 B2) gated by `svelte-check`
and vitest.

Phase 2: `rpc/conn_test.go` (two concurrent calls answered out of order;
notification order preserved; a missed pong is an error);
`rpc/spike_unix_test.go` (AF_UNIX on Windows); `rpc/stream_delivery_test.go`
(the twin of the phase-1 test, the engine behind an `httptest` WebSocket
upgrade in the style of `remote_test.go`, driven by a real `*rpc.Client`);
`rpc/provider_proxy_test.go` (an `httptest` provider streaming SSE — the key
present at the provider and absent from the engine's request; cancel
mid-stream closes upstream; 429 then 200 replays through `retryTransport`;
bytes exact); `rpc/screen_tool_test.go`; `rpc/file_test.go` (Range through
the reverse proxy); `desktop/engine_local_test.go` (build `cmd/aetox-engine`
once in `TestMain`, spawn it, a full turn over the real socket, kill → restart
→ re-sync, no screen → `ErrNoScreen`). `live_all_providers_test.go` is run
with `ProviderTransport` set — every real provider through the proxy — before
the default flips. CI's Windows job adds `CGO_ENABLED=0 GOOS=linux
GOARCH=amd64|arm64 go build ./cmd/aetox-engine`, twenty seconds of proof that
the engine half is pure Go.

Phase 3: `scripts/remote-smoke.sh` builds the Linux engine, runs it in Docker
with `--tcp` and a token, and drives a full turn and a terminal session from
Windows; then the owner's real host.

Numbers to write into §248 as they are measured: RPC round-trip locally
(expected under a millisecond, against ~190 ms per git spawn that the Git pane
already pays every 2 s), the turn latency delta, and event volume
(`agent:chunk` at ~100/s).

## 8. Lines this feature does not cross

- **No relay, no cloud engine.** The tunnel is ssh and the host is the user's.
  Away-from-home is Tailscale, as the mobile doc already said.
- **No second gate.** The screen is a client of the engine's bindings; it
  computes nothing it could ask for.
- **No key on the host.** Not as a convenience, not as a fallback, not in
  `.env`. The import ban is the enforcement.
- **No frontend re-architecture.** The 389 bindings and 37 events are what
  the frontend sees before and after; the one change is a generated
  namespace.
- **No in-process fast path**, ever, for the reason in decision 2.
- **`bootstrap.Engine` is the only door.** The engine binary builds its engine
  the way the desktop does; the CLI's hand-wiring is debt this does not add to.

## 9. Order of work

**Phase 0 — this document, DECISIONS §248, the ARCHITECTURE rows.** One day.

**Phase 1 — the carve, in-process, compiler-enforced.** Stage A prepares
every seam where the code is, in small commits on `main`: the seven direct
`EventsEmit` calls through `emitEvent`; the turn's lifetime off `a.ctx` onto
an engine context; `Transport` through `ProviderOptions` and `bootstrap`;
`internal/credentials` and the deletion of `ModelAPIKey`; dialogs and reveals
split by the §2.2 rule; the `Screen` interface and `workbenchSkills` split;
test helpers for the screen half. Stage B moves everything engine-side in one
landing — `git mv` of ~200 files, `App`→`Engine`, the generator, the ban test —
followed in the same push by the regenerated `wailsjs/` (Wails groups TS
types by Go package, so `namespace main` becomes `namespace engine` in ~20
frontend files; this is the one unavoidable frontend touch, and `wails dev`
would regenerate it anyway). Stage B is built on a worktree branch rebased
daily and landed in an announced window, or as one commit on `main` — the
owner's call when it is reached. About three weeks.

**Phase 2 — the wire and the local child.** The `rpc` package and its spike;
the generator's client and server outputs; the provider proxy on both sides;
the screen-tool stub and `hello`; `/file/` and its reverse proxy;
`cmd/aetox-engine`; `engine_local.go` and the flip of `NewApp` onto the RPC
client, with `aetox-engine.exe` in the build; reconnect, no-screen and
crash-restart tests, and the numbers. Decision 2 becomes true at the flip and
there is no in-process path left. About three weeks.

**Phase 3 — over ssh.** The spike on unix-socket forwarding; install, start,
tunnel, supervise; the remote picker and the Settings section; the Docker
smoke; the owner's host. Two to three weeks.

**Phase 4 — remote behaviour of window tools.** Uploads through
`/file/attach`, deck render over the wire, the refusals in §6 made polite.
One to two weeks.

**Later, not decided here:** the CLI as a second screen; the phone remote
unparked on the same door; `screen-preferences.json`; a general credential
proxy; MCP OAuth for a server configured on a host that has no browser.

## 10. Record — what each phase actually did

*(empty until phase 1 lands; every phase appends its measured numbers and the
tests that pin it, here and in §248.)*
