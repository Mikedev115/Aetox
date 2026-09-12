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
`shell-audit.log`, `models/`, `bin/`, `prompts/`, `logs/engine.log`,
`engine-<pid>.sock` (the local socket, named for the screen's pid). Screen:
`credentials.json`, `oauth.json`, `account.json`, `webview/`, `updates/`,
`update-check.json`, `screen.json` (new — remote hosts and their tokens,
`atrest`-wrapped), `logs/desktop-<time>.log` (debuglog, one file per launch;
the engine's is `logs/aetox-<time>.log`, and the two are named for the
process because they open within the same second). This is an invariant,
not a description.

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

**As built (2026-09-12, `internal/engine/remote` + `desktop/engine_remote.go`).**
The road is the five steps above with these differences. *TCP on the host,
not a unix socket:* the streamlocal spike could not be run from the machine
this was built on (no Linux host, no WSL distribution, no Docker), so the
engine listens on `--tcp 127.0.0.1:0` there and the tunnel is a plain
`-L 127.0.0.1:<port>:127.0.0.1:<rport>` — the fallback step 4 named, with
the token as the guard against other users on the host. *The engine is not
`~/.aetox/server/<ver>/aetox-engine` but `~/.aetox/server/bin/<ver>/aetox-engine`*,
beside `token`, `engine.pid`, `engine.addr`, `engine.version`, `engine.out`,
`engine.err` — one directory the screen owns on the host, and the engine's
own DataRoot (`~/.config/aetox`) untouched by it. *Every script is
`sh -c '<script>'`* so the host's login shell (fish, csh) is not parsing it,
and every script begins with `: aetox-<step>` so `ps` on the host says what
the screen is doing. *The ssh options are fixed:* `BatchMode=yes` (no
terminal, so no prompt — a key or an agent is the way in),
`StrictHostKeyChecking=accept-new` (remembered on first sight, refused when
it changes; a fingerprint dialog is a ceremony nobody performs),
`ConnectTimeout=15`, and for the tunnel `ExitOnForwardFailure=yes` with
`ServerAliveInterval=15`/`CountMax=3` so a dead network is noticed in under a
minute. *The token is minted per start* (`rpc.NewToken`) and lives in
`screen.json` (atrest-wrapped, the screen's file under DataRoot); a running
engine is reused only when it is this version AND the screen holds its
token — otherwise it is stopped and replaced. *The Linux engine comes from
`internal/update`, not `internal/capability`:* the capability manifest pins
a sha256 in the source, and the engine of THIS version is built after the
source is frozen; so `update.FetchEngine` takes it from the release of the
running version by tag, verifies `checksums.txt` against the release key
first and the file against the checksums second — the same chain as an
update — and caches it under `<DataRoot>/updates/engine/<ver>/`, re-hashed
on every use. A file beside the program or in `AETOX_ENGINE_LINUX_DIR` wins
over the download (a development tree, a build of one's own).
*Switching engines reloads the frontend* (`WindowReload`, a plain
`location.reload()` — not `WindowReloadApp`, which navigates to the start
URL and left the native window black on the first real switch): another
engine is another database, and every store the window holds is about the
one before. The top bar wears the host's name as a badge while the engine is
there, because the window otherwise looks exactly as it does at home. A switch is not a restart — it does not count toward the three
a minute — but a tunnel that drops does, and is redialed without touching
the engine (the next spawn's probe finds it running). *The picker* is
`RemoteDirPicker.svelte` on `ListDir`/`HomeDir`; only the project door
(`openFolder`) uses it in v1 — the other native dialogs (add a workspace
folder, browse a folder, import a file) still open this machine's disks
when the engine is on a host, and hand it a path it cannot read; phase 4's
`/file/attach` is the door for those. The manual road stays:
`AETOX_ENGINE_ADDR` + `AETOX_ENGINE_TOKEN[_FILE]` attach the window to an
engine somebody else started (`engine_local.go`'s `attach`), for a host the
Settings page cannot describe. The fake host that pins all of this on
Windows is `internal/engine/remote/testdata/fakessh`; the real scripts on a
real sshd are `TestRemoteSmoke` (`scripts/remote-smoke.sh`), which the CI
Linux job runs with the runner as its own host.

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

**Numbers not measured in phase 1.** Nothing crossed a socket yet; they are
below, under phase 2.

### Phase 2 — 2026-09-12, eight commits on `claude/engine-carve`

The wire, and the flip. Decision 2 is true as of P2-7: `desktop.App` holds no
`*engine.Engine` any more, `App.api` is an `*rpc.Client` on one socket to
`cmd/aetox-engine`, and there is no in-process path to drift. Every commit
green on `go build ./... && go test ./desktop/ ./internal/engine/rpc/
./cmd/aetox-engine/ ./internal/update/` and the frontend's `svelte-check` +
vitest; the engine's own suite once at the end.

| # | commit | what it is | pinned by |
|---|---|---|---|
| P2-1 | `6b0715ce` | `internal/engine/rpc`: `Conn` — one writer, one reader, a goroutine per request, side-prefixed ids, ping 15 s / pong 30 s, a 512-frame outbound queue as back-pressure; `Error` carrying the Go error's sentence and a `kind` from `Kinds` (`canceled`, `deadline`, `no_screen`, `disconnected`, `method_not_found`); a panicking handler answers an internal error, not a crash; `Listen("unix"|"tcp")` removes a stale socket first; `Accept`/`Dial` check the bearer token in constant time | `conn_test.go` (calls answered out of order, 500 notifications in order, a call back while a call is open, errors keep sentence and kind, void calls answer, a cancelled call frees the wire, a dead peer fails every open call at once, the token); **`spike_unix_test.go` — AF_UNIX on this Windows 11 listens, dials, round-trips** |
| P2-2 | `7f21d68c` | the generator's two new outputs — `rpc/client_gen.go` (`var _ engine.API = (*Client)(nil)`: `(T, error)` returns the error, bare `T` answers the zero value and tells the failure hook, several values through `callN` as one array) and `rpc/server_gen.go` (positional params: fewer → zero values, more → ignored, wrong → invalid-params naming the position); `Client`, `Server` (`NewServer(token, engine.NewEngine)` — the engine is built with the peer as its screen, since each needs the other first; one screen at a time, the newer replaces the older), `ScreenPeer` = `engine.Screen` over the wire | `gen_test.go` diffs all four files; **`turn_test.go` — a whole turn driven with nothing but the bindings** (`OpenProjectPath` → the three switches → `NewSession` → `SendMessage`), `agent:chunk` crossing with `Replace:true` exactly once and equal to the `TurnReply` |
| P2-3 | `663ed3e5` | the provider proxy: `providerProxy` is the `model.Transport` under `retryTransport`/`idleTimeoutBody`, one `RoundTrip` = one `provider.open` carrying method, URL, headers, body and the first-byte budget read off the transport it replaced; chunks queue per stream (256 × ≤32 KiB) and a goroutine feeds the pipe so the read loop never blocks on it; cancel — the Stop button, a timeout, a body closed early — withdraws the request on the screen; `ServeProvider(client, Signer)` on the screen, a transport per first-byte budget so connections are reused | `provider_proxy_test.go`: the key at the provider and nowhere in the engine's request, SSE byte-exact with the quota header, cancel reaching the provider, **429 → 200 through `model.NewProvider` on the proxy** (replay and re-sign), no screen fails after the wait |
| P2-4 | `ad653e4b` | window tools across the wire: `hello` announces each pack's shape (`Announce`: name, description, definition, actions, guidance per action and `steps`), `ScreenPeer.WindowTools` lends a `screenTool` per announcement that is `skill.Tool` + `skill.Packed` + `skill.Guided`; a call crosses as `screen.tool` and comes back as the whole `skill.Output` with the error's mark (`callfault`/`statereport`) intact; `Narrow` asks the screen for the narrowed shape once (`screen.toolCut`, cached); no screen answers as a state report | `screen_tool_test.go`: lent and run with session/root/args and the picture back, marks kept, the narrowed enum and description are the screen's, `ListTools` of a real session carries `browser` as workbench after hello, no screen does not hang |
| P2-5 | `01c93097` | `/file/` on the engine's listener (`engine.FileHandler`, one resolver behind two doors) behind the token; `rpc.FileProxy` is the screen's `/aetox-file/` — a `ReverseProxy` into the socket, token added, flushed as it streams, Range through both, 502 with a sentence when the engine is gone. Found on the way: the desk questions must not wait for a screen (`ScreenPeer.ask` answers `ErrNoScreen` at once; only a turn's `provider.open` and `screen.tool` wait `screenWait`) | `file_test.go` |
| P2-6 | `202ff12f` | `cmd/aetox-engine serve (--socket | --tcp) [--root] [--token-stdin | --token-file] [--idle-exit]`; the token on stdin (EOF = the screen is gone, exit) or a file, never argv; one stdout line `{network, address, pid, version}`; `<DataRoot>/logs/engine.log`; `engine.Startup` in the child as before; `aetoxapp.Discard` console + `engine.UseConsole`; cross-compiles `linux/amd64` and `linux/arm64` with `CGO_ENABLED=0` | `main_test.go`: built, served, spoken to, refused a wrong token, gone on stdin close, refuses to start with no token |
| P2-7 | `beacf7b6` | **the flip.** `desktop/engine_local.go`: the supervisor — unix socket under DataRoot first, `--tcp 127.0.0.1:0` when it cannot listen; a dropped wire is redialed before the process is replaced; a crash restarts with 1/2/4 s backoff, three in a minute → `failed` until `RestartEngine`; `rpc.Client.await` makes every binding wait up to 30 s for a wire, so a restart is a pause and not a screen of errors; `PrepareToClose` across the wire before the window goes; the binary is `AETOX_ENGINE`, else `aetox-engine.exe` beside the app, else `go run ./cmd/aetox-engine` in a development tree. `rpc.ServeScreen(client, appScreen{a})` — the in-process `Screen` served whole. `EngineStatus`/`RestartEngine` bindings, `engine:status`, `EngineStatus.svelte` in the update card's shell. Packaging: `release.yml` builds the engine first, checks it, ships it in the installer (`project.nsi`), the msix stage and the portable zip; `internal/update` swaps both exes by name; `wails-dev.bat` builds it beside the dev binary | **the whole desktop suite over the wire** (`newTestApp(t)` = server + loopback listener + client, the engine shut down with the test); `engine_local_test.go` — the real binary: a turn through the child with events crossing as JSON, a killed child started again with a waiting binding answered, the window's close taking the child with it; `engineStatus.test.ts`; `TestSwapPortableSwapsTheEngineBesideTheAppByName` |
| P2-8 | this commit | after a restart the frontend puts the chat on screen back in front of the new engine (`resyncAfterEngineRestart`: `LoadSessionAnyProject` + `loadRealState`, once per restart count); the numbers; this record | `engineResync.test.ts`, `measure_test.go` |

**The numbers** (owner's machine, Windows 11, 2026-09-12, `measure_test.go`
and `engine_local_test.go`, logged not asserted):

| what | measured |
|---|---|
| RPC round trip, in-process engine behind a loopback TCP listener, `AppVersion` × 2000 | mean **58 µs**, p95 520 µs, max 660 µs |
| RPC round trip to the real child over AF_UNIX, `AppVersion` × 500 | mean **64 µs** |
| a turn on `aetox-render:test`, in-process vs over the wire, p50 of 5 | 1.204 s vs 1.219 s — **+15 ms**, inside the fixture's own noise |
| events per such turn | **62 frames, 9.2 KB** |

For scale: the plan's comparison point was a `git` spawn at ~190 ms, which
the Git pane polls every two seconds. A binding across the socket costs a
three-thousandth of that.

**As built, against §4.** `provider.open` carries the full URL and the wire
format, not `auth`/`baseURL`/`path`: the screen's own signer
(`providerTransport`, unchanged) knows the header, and the endpoint a sign-in
pinned is a separate question the engine already asks (`ProviderEndpoint`),
so the URL it builds is the one to send. `hello` carries a
`ToolAnnouncement` per tool (definition, actions, guidance) rather than bare
definitions, and a narrowed shape is fetched from the screen's own `Narrow`
(`screen.toolCut`) rather than reconstructed. `/file/<rel>` serves the
current session's root, as the in-process host did; `?session=` is not read.
The stdout line is `{network, address, pid, version}`. No-screen waits only
for what a turn cannot do without (a provider request, a window tool); the
desk questions answer at once with the catalog's answer. The re-sync after a
restart is the frontend's (`resyncAfterEngineRestart`), not a replay of calls
by the Go screen: the frontend is the only side that knows which chat is on
screen, and reopening it through `LoadSessionAnyProject` is what a launch
does. `turn_busy` is not a registered kind — there is no such sentinel. The
client waits for the wire (`connectWait` 30 s) instead of failing while the
engine restarts, which the design did not say and the first crash test
showed was needed.

**Left for phase 3.** `Setpgid` for a Linux child is not done (the local
child is Windows today; stdin-EOF exit covers the rest). The unix socket path
is `<DataRoot>/engine-<pid>.sock` and `sun_path`'s limit is real: a test's
long temp root made the child fall back to TCP, which is exactly the fallback
working, and a production DataRoot is fifty characters. `--idle-exit` and
`--token-file` are built and untested against a real ssh host.

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

**After the live run (`93c76b32`).** The first real turns through the child
on `wails dev` (a codex sign-in on the screen, `browser open`/`read` on the
screen through `screen.tool`, the reply streamed back through the provider
proxy) found two things: the screen had lost its own log file when
`debuglog.Init` moved into the engine's startup, and the same-second name
would have had both processes open one file with `O_TRUNC` —
`debuglog.InitAs(dir, prefix)`, `logs/desktop-<time>.log` beside
`logs/aetox-<time>.log` (§2.4 updated); and an attach mode —
`AETOX_ENGINE_ADDR=tcp:host:port|unix:/path` + `AETOX_ENGINE_TOKEN` or
`AETOX_ENGINE_TOKEN_FILE` — so a window can reach an engine it did not
start, which was the manual road to a host until phase 3 and remains the
road to a host the Settings page cannot describe.

### Phase 3 — 2026-09-12, on `claude/engine-carve`

Over ssh. One commit, green on `go build ./... && go test ./desktop/
./internal/engine/ ./internal/engine/rpc/ ./internal/engine/remote/
./internal/update/ ./cmd/aetox-engine/`, `svelte-check` (0 errors) and
vitest (1601). §5's "as built" paragraph has every place the build differs
from the design; this is what it is.

| piece | what it is | pinned by |
|---|---|---|
| `internal/engine/remote` | `Host` (name, target, root, token, version, arch, last use), `CheckTarget` (a target that starts with a dash is an ssh option, so the first character is pinned), `Driver` (`SSH` path or `AETOX_SSH` or PATH; `Binary` — the engine to send for an arch; `IdleExit`), the five scripts as `sh -c` one-liners each opening with `: aetox-<step>`, `Probe`/`Install`/`Start`/`Stop`/`Tail`/`Tunnel`, and `Connect` = probe → (download → install if this version is not there) → (start with a fresh token unless this version is running and the token is held) → tunnel, reporting a `Step` per stage with download/upload progress | `remote_test.go` against `testdata/fakessh` — a program that parses ssh's argv, dispatches on the step marker, keeps a home directory, stores the bytes that arrive on stdin and runs them (the engine built for this OS), and forwards `-L` with a real TCP listener: a fresh host installs+starts+tunnels and a `Hello` crosses; a second `Connect` reuses (probe, tunnel only); a lost token or another version replaces the engine; ssh's own words (`Permission denied (publickey)`) come back; a non-Linux host is refused after the probe alone; a killed tunnel closes `Exited`; `Stop` ends the process and leaves the binary; the quoting; `parseProbe`. `smoke_test.go` — `TestRemoteSmoke` against a real host (`AETOX_REMOTE_SMOKE=user@host`): the scripts on a real sh, `Hello`, `HomeDir`/`ListDir`/`TerminalShells` there, the engine surviving the tunnel's close, `Stop` |
| `internal/update/engine.go` | `FetchEngine(ctx, version, goos, goarch)` — the release's `aetox-engine-linux-<arch>` by tag, `checksums.txt` + `.sig` from the same release, signature before hash, cached under `updates/engine/<ver>/` and re-hashed on each use; `EngineAssetName` | `engine_test.go`: downloads once and keeps, refetches a tampered copy, refuses a release without the asset, refuses a wrong signature |
| `desktop/engine_local.go` | the supervisor learns a target (`engineTarget{mode, host}`; `retarget` kicks it and marks a switch), `engineProcess.leave` (a tunnel is closed, not sent EOF), `Mode`/`Host` on `EngineStatus`, a kick-driven exit is not a restart (no count, no backoff), a tunnel's death is `reconnecting` with the host's name, and the window reloads once after a switch lands; the target at launch is `screen.json`'s `active_host` | `engine_remote_test.go` (below) |
| `desktop/engine_remote.go`, `screen_config.go` | `spawnRemote` (Connect with each step on the chip in Thai, the token written to `screen.json` before the wire is dialed), `engineBinaryFor` (`AETOX_ENGINE_LINUX_DIR` → beside the exe → `update.FetchEngine`), `reloadWindow`; `screen.json` (atrest, 0600, the screen's own; tokens registered with `debuglog.Redact`); bindings `RemoteHosts` (no tokens in the view; which ssh, where the Linux engine comes from), `SaveRemoteHost`, `ForgetRemoteHost` (refused for the active host), `ConnectRemote`, `DisconnectRemote`, `StopRemoteEngine`, `RemoteEngineLog` — all the screen's own, so they answer while the engine is unreachable | `engine_remote_test.go` with the fake host and the real engine: a window goes to a host (steps, `Mode`/`Host`, no restart counted, one reload, the project the row named opened there, the version on the row, the token on disk) and comes back (a child of its own again, a second reload, the host's engine left running); a dropped tunnel is opened again with the same engine behind it; a second window under the same DataRoot starts on the host it was left on; an unreachable host is `failed` with ssh's words and the way back needs no host; the active host cannot be forgotten, an option-shaped target cannot be saved |
| `internal/engine/listdir.go` | `HomeDir()`, `ListDir(path)` — folders only (symlinks to folders count), hidden last, capped at 2000, the parent to go up to | `listdir_test.go` |
| frontend | `stores/engine.svelte.ts` (the status as last heard, `pickerOpen`), `EngineStatus.svelte` (the road's steps shown at once with the detail line, `ใช้เครื่องนี้แทน` while on the road or failed, the host named in a failure), `RemoteDirPicker.svelte` (path box + up + list + hidden toggle, in the confirm dialog's shell), `RemoteEngine.svelte` = Settings › เครื่องระยะไกล (where the engine is, the hosts with connect/back/edit/log/stop/remove, the add form, the per-host note), `openFolder` raising the picker when the engine is remote | `remoteEngine.test.ts` (10 tests) |
| packaging, CI | `release.yml` builds `aetox-engine-linux-{amd64,arm64}` (`CGO_ENABLED=0`, static), lists them in the signed `checksums.txt`, attaches them to the release; `ci.yml` cross-builds both on Windows and runs `scripts/remote-smoke.sh` on the Linux job — the runner as its own host: a throwaway key, `authorized_keys`, sshd started, `TestRemoteSmoke` | the CI run |

**What a host can and cannot do to the screen (2026-09-13).** Asked
"ความปลอดภัยล่ะ" after the first real host, the answer was written down as
the threat model this build actually meets. The host is the user's own
machine, reached with their key; the engine there has the user's whole
shell, on purpose, and the token — 32 random bytes per start, on stdin here
and in a 0600 file there, never argv, DPAPI-wrapped in `screen.json` — is
what keeps other users of that host out of it (loopback only; the ssh
channel is the only door in). The one thing a host must never get is the
model credential (decision 3), and the phase-2 wire had a way to get it
anyway: `provider.open` carries the full URL, and the screen signed
whatever it was handed — a taken-over host, or one impersonated at first
contact under `accept-new`, could have asked for a request to a server of
its own and received the key on it. Closed in `credentialMayRide`: when the
engine is on a host, a credential rides only to where the screen itself
knows the provider lives — the catalog's endpoints, the sign-in's endpoint
(`oauth.Endpoint`), a custom row's endpoint as recorded in `screen.json`
when it was added here, and loopback (a runtime on this machine, which is
what a host's "localhost" means once the request is sent from here).
Anything else goes out bare and fails at the far end with the provider's
words. At home nothing changes: the engine is a child of this process on
this machine's ground. Two consequences to know: a custom base URL set on
the host for a catalog provider is not signed from here (set it on the
screen's own row instead), and provider HTTP always leaves from the screen
machine — a runtime on the host itself is not reachable through the proxy,
while one on the screen machine is. Still open, and named: `accept-new` is
trust on first sight (connect once by hand or seed `known_hosts` for a host
that matters); safety settings are per host and a fresh host starts at
defaults; whoever holds the token holds the shell, as with the local engine
today. Pinned by
`TestOnAHostACredentialRidesOnlyToWhereTheScreenKnowsTheProviderLives`.

**Landing, 2026-09-12.** `main` had moved twice under the branch: the Git
room and chat attachments (`c1c74d79`, merged as `7be94d44` — rename
detection carried `git_commit.go`/`git_worktree.go`'s hunks to their new
home), and the studio shelf (`fea7ebd6`, 138 files written against the
pre-split `desktop.App`). The shelf is engine work — a catalogue on disk,
ffmpeg probes, a tool for the video agents — so `studio_library.go`,
`studio_thumbs.go` and `studio_browse.go` moved to `internal/engine` with
`App`→`Engine`; its two reveals became the usual pair (`StudioAssetPath`/
`StudioLibraryPath` on the engine, `RevealStudio*` on the screen); the
folder dialog `AddStudioLibrary` was already the screen's half of
`AddStudioLibraryAt`. Its `/aetox-shelf/` host became the engine's
`/shelf/` (`engine.ShelfHandler`, mounted by `rpc.Server` behind the token
beside `/file/`) and the screen's `rpc.FileProxy` now carries both spaces —
so a shelf on a host over ssh is browsed and played through the same
tunnel as the project's files, with no wart to record. Pinned by
`TestTheScreenProxiesTheShelfOntoTheEngine`.

**Not done, on purpose.** The streamlocal spike (§5 step 4) — the TCP road is
built and the socket road is an improvement to make on a host that can prove
it. `Setpgid` for a local Linux child — still no local child on Linux. A
window on a host still opens this machine's dialogs for every door but the
project's (§5 as built); `revealInFileManager` and `OpenExport` still answer
with a host path. The three-a-minute rule counts tunnel drops: a flaky
network reaches `failed` after three and asks for a press. `--idle-exit` on
the host is 30 m fixed.

**The first real host — 2026-09-12, later the same day.** The owner has no
Linux machine, so WSL (Ubuntu 26.04, systemd, `openssh-server` on port 2222,
key-only) on the same PC became the host, reached as `ssh wsl` through
`~/.ssh/config` — a real sshd, a real `sh`, a real `nohup`, only the network
is loopback. `TestRemoteSmoke` passed in 7.3 s: probe, the 32 MB binary up
on stdin, start, tunnel, `Hello`, `HomeDir`/`ListDir`/`TerminalShells` on
the host, the engine outliving the tunnel, `Stop`. Then the dev app: a row
`wsl` with root `/home/mikedev`, one press, 1.2 s from "กำลังติดต่อ wsl" to
connected (probe 0.3 s, start 0.5 s, tunnel 0.3 s), the window reloaded on
`/home/mikedev` with `bash (default)` in the shell list and the host's log
under ดู log. Two things the setup taught, recorded for the next host:
Windows' `localhost` resolves to `::1` first and WSL forwards only IPv4, so
the config names `127.0.0.1`; and a key made by Windows `ssh-keygen -N ""`
from PowerShell carries a literal `""` passphrase, which BatchMode then
fails on in silence — the key was made in WSL. Neither is Aetox's to fix,
but the Settings note should say "key-based, tested with `ssh -o
BatchMode=yes <target>`" — it does not yet. One thing was: the owner's
window went black after the switch — `WindowReloadApp` (navigate to the
start URL) in the native webview; `WindowReload` (reload in place) is what
a switch does now, and the top bar names the host so "where am I" has an
answer on screen.
