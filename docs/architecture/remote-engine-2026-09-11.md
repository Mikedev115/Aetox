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
| `internal/engine/` (package `engine`, type `Engine`) | everything engine-side above, plus the terminal and `conversation`. **May not import** `wailsapp` or `internal/credentials`, and may not call `oauth.TokenSource`/`Endpoint`/`Headers`/`Token` — `deps_test.go` parses the package's own files and fails the build if they do (narrower than `go list -deps`, §10 says why). |
| `internal/engine/api_gen.go` | `type API interface` — every exported method of `*Engine`, generated. |
| `internal/engine/screen.go` | `Screen` and `Session`: the engine → screen surface (§3). |
| `internal/engine/rpc/` | JSON-RPC 2.0 over WebSocket: `Conn` (the `internal/lsp` pattern with two id spaces and a goroutine per request), `Server`, `Client` (implements `engine.API`), `ScreenPeer`, the provider-proxy `RoundTripper`, the screen-tool stub, the `/file/` handler; `client_gen.go`, `server_gen.go`. |
| `internal/engine/rpc/gen/` | the generator — `go/ast` only, no new dependency. Reads the exported methods of `*Engine` (minus `Close`, `Attach`) and emits the interface, the screen's forwarders, the client stubs and the server dispatch table. `TestGeneratedFilesAreCurrent` regenerates into a temp dir and diffs. |
| `internal/credentials/` | `credentials.json` — load, `KeyFor`, save, forget — moved out of `internal/config`, still wrapped by `atrest`. |
| `cmd/aetox-engine/` | ~150 lines: `serve --socket <path> | --tcp 127.0.0.1:0`, `--root`, `--token-stdin | --token-file`, `--idle-exit 30m`. Console is `DiscardConsole`; log is `<DataRoot>/logs/engine.log`, a different file from the desktop's so local mode never has two writers on one log. **A binary of its own, not `aetox serve`:** `release.yml` deliberately does not ship the CLI (§30); a fixed file name is what `ps`, `pkill` and `~/.aetox/server/<version>/` need; and the CLI's own future is to become a second screen on this engine, in a later phase. |
| `desktop/` (package `main`, `App` = the screen) | `main.go`; a new `app.go` (`ctx`, `api engine.API`, `browsers`, `speakJobs`, the computer-use lock, `openDir`, `emit`, `staged`, `exports`); `browser*.go`, `computer_*.go`, `uia_windows.go`, `speak.go`, the TTS half of `voice.go`, `ttshost.go`, `update*.go`, `account.go`, `oauth.go`, `deck_render/image/pdf/reveal/flatten/pick.go`, `exports.go`, `providers.go`, `screen_doors.go`, `screen.go`; generated: `engine_forwarders_gen.go`; phase 2: `engine_local.go`, `engine_remote.go`, `screen_tools.go`, `engine_status.go`. As built (§10): `mcp_oauth.go`, `remote*.go`, `speech.go` and the STT half of `voice.go` stayed with the engine. |

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

The wart this buys: in remote mode the TTS voice — and the ติดตั้ง button
for a TTS vendor, which runs on the engine's host (`InstallVoiceEngine`) —
follow the host. Written down; a `screen-preferences.json` split, with the
install run where the vendor is used, is a later phase if it bites. As built,
the screen reads the picks through `VoiceSettings` and writes the one it
checks on its own machine through `RememberTTSVoice`.

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

### Phase 0 — 2026-09-11

This document, DECISIONS §248, the ARCHITECTURE rows, the struck sentence in
the mobile-remote doc. Commit `d6c6e1d1`.

### Phase 1, Stage A — 2026-09-11/12, six commits on `main`

Every seam of §3, prepared where the code is. Nothing moved packages yet;
`go build ./... && go test ./...` green at each.

| # | commit | what changed | what pins it |
|---|---|---|---|
| A1 | `4e06a434` | the seven direct `wailsruntime.EventsEmit` calls go through `emitEvent`; every writer of a memory file sends `learning:changed` with the count (a nil payload used to blank the badge — `Number(nil) \|\| 0`) | `pending_test.go` recorder |
| A2 | `dc767f37` | `App.engineCtx()` — the engine's own lifetime; turns, `gitContext`, snapshots, the catalog refresh and the workspace-widen card hang off it, and `finishTurnsForClose` ends it. `a.ctx` is left to dialogs, `Quit`, `EventsEmit` and screen work | `TestEngineWorkOutsideATurnEndsWithTheClose` |
| A3 | `cd8ac04a` | `model.Transport` — `func(network http.RoundTripper) http.RoundTripper`, a *wrapper* rather than a bare RoundTripper so `retryTransport`/`idleTimeoutBody` stay above it and a retry is re-signed; a provider built with one attaches no credential and requires none; `factory.go` no longer imports `oauth` — `TokenSource`/`Headers`/`SignedInEndpoint` are caller-supplied; `model.AuthScheme` names the header a wire format carries a key in; `bootstrap.Options.ProviderTransport`/`ProviderEndpoint`; `desktop/provider_forward.go` signs from the screen's stores per request | `httpclient_transport_test.go` (client attaches nothing, signer re-entered on retry, key still required without a transport), `provider_forward_test.go` (store key on the wire, sign-in outranks it with its account header) |
| A4 | `658937a5` | `internal/credentials` owns `credentials.json` (`Load`/`Save`/`KeyFor`/`StoredKeyFor`/`Set`/`Forget`/`ProviderAPIKey`, legacy migration by raw JSON); `config.Config.ModelAPIKey` and `ModelPreference.ModelAPIKeys` deleted; `bootstrap.Engine` passes no key; the CLI resolves its key at the moment of use (`keyFor`) and is the one host that still hands one straight to a provider | `credentials_test.go`; `turn_guard_test.go`'s reopened-chat test now pins provider name and address, the key being structurally absent |
| A5 | `bbbe4384` | `desktop/screen_doors.go` holds every dialog and reveal under its old binding name; each calls an engine twin that takes a path (`InstallSkillsFromZipAt`, `AddWorkspaceFolderAt`, `BrowseFolderAt`, `ImportSessionFrom`, `SetPresetImageFrom`, `AddSpaceContextFiles`), hands back bytes as `ExportFile` (`SessionExportBytes`, `PictureBytes`, `AgentPackageBytes`) or answers with a host path (`*FolderPath`, `ProjectFilePath`, `ArtifactPath`); six files stop importing Wails | `screen_doors_test.go` — every door has its twin, by reflection |
| A6 | `95f928c4` | `desktop/screen.go`: `Screen` (`Emit`, `ProviderTransport`, `WindowTools`) and `Session` (`ID`, `Root`); `appScreen` adapts this window (not `App` — every exported `App` method is a binding); `workbenchSkills` = engine `sessionSkills` + what the screen lends, the computer switch still the engine's; `browserSkill`/`computerSkill` hold a `Session` | `screen_test.go` — a fake Screen lends the packs, the real ones are not built, the switch still cuts |

A7 (screen-side test helpers) is folded into Stage B, where the partition of
the ~120 test files is known rather than guessed.

### Phase 1, Stage B — 2026-09-12, six commits on `claude/engine-carve`, landed as one set

The move, on a worktree branch off `28109eea` (the owner's choice: *"worktree
branch แล้ว landing ทีเดียว"*), every commit green on `go build ./... && go test
./desktop/ ./internal/engine/` and the frontend's `svelte-check` + vitest, then
rebased onto the six commits `main` gained meanwhile (`9992cd46`…`1dbec0d4`)
before landing — which is where the move's rule was tried on somebody else's
new files for the first time: `git_log.go`, `plan_report.go` and the working
tree's failed-read error went to the engine untouched; `attention*.go` (the
taskbar flash) stayed on the screen; and `open`'s new hand-over flag, which
had read the sandbox root off the app, became the twin `HandedOverFile`.
Big-bang first, then evictions: B1a moved *all* 294 desktop Go files into
`internal/engine` and made the window compile against it, and the five
commits after it carried the window's clusters back out one at a time, each
leaving a twin behind. That order kept every step buildable and made the
question at each file "what does the engine still need to know" rather than
"what does the window need".

| # | commit | what moved back to the screen | what the engine gained |
|---|---|---|---|
| B1a | `131a0c6d` | nothing yet — 294 files `git mv`'d to `internal/engine`, `App`→`Engine`, `desktop/app.go` new with `eng *engine.Engine`; `internal/engine/gen` writes `api_gen.go` (`type API interface`, every exported method) and `desktop/engine_forwarders_gen.go`; `gen_test.go` diffs; `wailsjs/` regenerated, `namespace main`→`engine` in 22 frontend files | the four lifecycle hooks as package functions (`lifecycle.go`), so they are not bindings |
| B1b | `c551b8a3` | `screen_doors.go`, `window*.go`, `update.go`/`update_notify.go` | `ReadyToRestart`, `SpeechModelFolderPath`/`DirPath`; `desktop/events.go` with `emitEvent` and the `emit` seam |
| B1c+d | `689dd2a9` | the credential desk (`providers.go`: keys, custom rows, `TestProviderConnection`, the discovery that needs a key), `screen.go`'s desk questions (`DefaultModel`, `Probe`, `ModelResident`), `attach.go`, `OpenProjectFolder` | `NewEngine(screen)`, `noScreen`, `resolveConfig` as a method, `defaultModel` falling back to the catalog, `AddCustomProviderRow`/`RemoveCustomProviderRow`/`CatalogModelChoices`/`ProviderQuotas`/`NoteProviderQuotas`/`ActiveModelFor`, `model.CatalogDefaultModel`; `provider_catalog.go`'s discovery no longer called from engine files |
| B1e | `c6edd196` | `browser*.go`, `computer_*.go`, `uia_windows.go`, `deck_render/image/pdf/reveal/flatten/pick.go`, `workbench.go`'s browser half, `exports.go` (Downloads is the screen's) | `Screen.RenderDeck` and `Screen.AgentTab`; `DeckExportFiles` (bytes per format); `window_twins.go` — `AnyTurnRunning`, `PageMarksOn`, `ComputerControlChanged`, `ResolveWorkbenchURL`, `SandboxFile`, `SaveBrowserShot`, `HandedOverFile`; `agent_pages.go` — `RecentAgentPages` with `BrowserOpenedLine`/`BrowserPageRef` beside `ParseBrowserOpened`; `App.api engine.API` on the screen with every forwarder and door through it; `deps_test.go` |
| B1f | `7d839076` | `oauth.go`, `account.go`, `speak.go`, `ttshost.go`, the TTS half of `voice.go` | `ProviderCredentialChanged` (one door for a key, a sign-in, a sign-out — forgets the old quotas, rebuilds if on screen); `VoiceSettings`/`RememberTTSVoice`; `AssetMiddleware` is `/aetox-file/` only, the screen chains `/aetox-tts/` in front |
| B3 | this commit | the two READMEs, this record, §248's status, the ARCHITECTURE rows | — |

**Where it ended up.** `internal/engine`: 212 Go files, 334 exported methods
= 334 forwarders on the screen; `desktop/`: 112 Go files, 96 bindings of its
own — 430 bindings where there were 389, the difference being the twins. The
frontend sees every name it saw before plus the twins; its one change is the
generated namespace (`engine.*` for the engine's types, `main.*` for
`StagedInfo`, `DeviceProfile`, `ComputerAppRow`, `AccountState`,
`TTSVoiceInfo`) and two `import type` lines.

**What the tests say now.** `internal/engine/deps_test.go` holds the import
ban and the four `oauth` selectors on the package's own files.
`screen_doors_test.go` pins every door's twin by reflection. The engine's
`tool_coverage_test.go` drives every engine tool through the real dispatcher
and no longer sees the window's two; `desktop/window_tools_test.go` drives
those the same way (tabs and list_apps for real, the rest routed and refusing
in words). Where an engine test needed a pack of the window's shape — stance
narrowing the browser per action, the busy signal's tab stamp — it gets a
stand-in through `Screen` (`packStub`, `tabScreen`) rather than the real one,
and the real one is held to the same rules in `desktop/`. Deck exports that
needed a renderer are no longer skipped: `renderScreen` answers `RenderDeck`
with a solid PNG per slide, so pdf/png/pptx-img are written by the test and
read back (`needsEngine` is gone). `desktop/apptest_test.go`'s `engineWith`
is how a screen test states one fact about the engine — a turn is running,
the marks are off, this is what the export contains, this is the voice —
without reaching into a package it cannot see into.

**Three things stayed with the engine that §2.1 had put on the screen, and
one temporary field.** `mcp_oauth.go`: the credential it stores is read by
`${connect:}` on the host the MCP server runs on (rule 5), so the store must
be that host's; a remote engine has no browser for the step and v1 does not
support it there (§9 already said so). `remote.go`: the parked phone remote
keeps its device table in `aetox.db` and its adapters call bindings — it is a
second screen on the same engine, which is what phase 5 makes of it, and
moving it now would have meant twins for a feature nobody can reach.
`speech.go` and the STT half of `voice.go`: `audio_transcribe` runs on the
host, so the STT engine, its model files and the mic's transcription are the
host's (the mic recording crosses as a data URL, the way it already did).
And `App.eng *engine.Engine` beside `App.api engine.API`: the lifecycle hooks
still take the concrete engine, and it is the last thing on the screen that
knows the engine is in this process — phase 2 deletes it.

**Numbers not measured yet.** Nothing crosses a socket in phase 1, so the RPC
round trip, the turn latency delta and the event volume are phase 2's to
record here.

**Two things Stage A found that narrow §2.1's import ban.** First, the ban is
on the *engine's own files* and on `internal/model`, not on the transitive
dependency list: `imagegen`, `stt` and `tts` read a per-host key through
`credentials.ProviderAPIKey` (decision 5), and `bootstrap`, `github` and
`automation` read `oauth` for `${connect:}`, the PAT and n8n — all per host,
all legitimately linked into the engine. What the test in Stage B must
enforce is that no file under `internal/engine` imports `internal/credentials`
or calls `oauth.TokenSource`/`Endpoint`/`Headers`/`Token`, and that
`internal/model` depends on neither store. Second, model discovery
(`ResolveDefaultModel`, `ModelChoices*`, the discovery half of
`provider_catalog.go`) still takes a raw key and is still called from
engine-side files (`SwitchModel`, `SwitchProvider`, `RetryActiveProvider`,
`resolveConfig`, the skill drafter); Stage B has to move it to the screen —
the picker is the screen's — or route it through the signed transport.
