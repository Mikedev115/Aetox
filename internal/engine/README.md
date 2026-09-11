# internal/engine — the engine, without a window

> Direction: [docs/architecture/remote-engine-2026-09-11.md](../../docs/architecture/remote-engine-2026-09-11.md) (§248) · The screen: [desktop/README.md](../../desktop/README.md) · Module map: [ARCHITECTURE.md §4.1](../../ARCHITECTURE.md)

**What it is:** everything the desktop app does that is not a window. The chat
loop and its turns, every tool the model is handed (through `bootstrap.Engine`
and nothing else), the session store (`aetox.db`), pending changes, plans,
memory, MCP children, the terminal pane's PTYs, decks as files, the git and
PR rooms, usage — the `~290` bindings that used to be methods on
`desktop.App`, now methods on `Engine` in a package the window imports. Since
§248 B1 (2026-09-12) the compiler holds the line: nothing here can reach the
Wails runtime, a WebView2 tab, a dialog, or a provider key.

Since phase 2 (later the same day) the engine is a process:
[cmd/aetox-engine](../../cmd/aetox-engine/main.go) builds one with
`NewEngine(screen)` where the screen is [rpc](rpc/)'s `ScreenPeer`, and the
window reaches it over one JSON-RPC socket — on this machine today, over an
ssh tunnel in phase 3. Nothing in this package changed for that, which was
the whole reason for the shape below.

## The two surfaces

**Engine → screen: `Screen`** ([screen.go](screen.go)). Everything the engine
ever needs from the window, as one interface the screen implements
(`desktop/screen.go`, `appScreen`): `Emit` an event; `ProviderTransport` — a
`model.Transport` that signs a provider request with a credential the engine
does not hold; `ProviderEndpoint`; `WindowTools` — the packs that act on the
window's machine (browser, computer), lent per `Session`; `AgentTab` — the
tab the busy signal stamps; `RenderDeck` — a deck drawn in the screen's hidden
webview, answered as bytes; and the three desk questions that need a key:
`DefaultModel`, `Probe`, `ModelResident`. `noScreen` is the engine with nobody
watching — events go nowhere, packs are none, questions get `errNoScreen`.
Tests install `testScreen` (`apptest_seed_test.go`) through `seed`, or a
`fakeScreen`/`tabScreen`/`renderScreen` when the test is about what the screen
answers.

**Screen → engine: `API`** ([api_gen.go](api_gen.go), generated). Every
exported method of `*Engine`, as one interface. `desktop/App` holds it as
`api engine.API` — an `*rpc.Client`, which implements the same interface
across the socket ([rpc/client_gen.go](rpc/client_gen.go)) — and forwards
every engine binding to it (`desktop/engine_forwarders_gen.go`, also
generated); the engine process answers through
[rpc/server_gen.go](rpc/server_gen.go). **Every exported method here is
therefore a frontend binding and a wire method** — a method the frontend
must not see is unexported, and the lifecycle hooks (`Startup`,
`BeforeClose`, `Shutdown`, `AssetMiddleware`, `FileHandler`, `UseConsole`,
[lifecycle.go](lifecycle.go)) are package functions for that reason. Keep a
method's arguments and results plain values: they are JSON on the wire.
After any change to an exported method:

```bash
go run ./internal/engine/gen -root .
```

`gen_test.go` regenerates into a temp dir and diffs all four files, so a
stale one fails the build.

**The wire itself** is [rpc/](rpc/): `Conn`, `Client`, `Server`,
`ScreenPeer`, the provider proxy (a model call leaves here unsigned and is
signed on the screen), the window-tool stubs (`hello` announces them,
`screen.tool` runs them), `/file/`. The design doc §4 has the message shapes
and §10 what was measured: about 60 µs a round trip, 62 frames of events per
test turn.

## The rule, and the test that holds it

[deps_test.go](deps_test.go): no non-test file here imports `wailsapp` or
`internal/credentials`; no file here calls `oauth.TokenSource`, `Endpoint`,
`Headers` or `Token`. The rule is on this package's *own files*, deliberately
not on `go list -deps`: `imagegen`, `stt`, `tts` and the automation clients
read their own per-host keys (§248 rule 5) and `bootstrap` reads `${connect:}`
for MCP headers, so the transitive closure legitimately links both stores.
What is held is that the engine's code never does. The provider key's whole
path is `desktop/provider_forward.go` → `Screen.ProviderTransport` →
`bootstrap.Options.ProviderTransport` → `model.Transport`, and
`config.Config` has no field for one (§248 A4).

## The twins — how a window job and an engine job were split

Every binding that needed both a window and the engine became a pair, by one
rule each way (design doc §2.2):

| the screen does | the engine does | pinned by |
|---|---|---|
| a dialog, then `*Path(path)` / `*From(path)` | the work on that path (`ImportSessionFrom`, `InstallSkillsFromZipAt`, `OpenProjectPath`, …) | `desktop/screen_doors_test.go` — every door has its twin, by reflection |
| a reveal in the file manager | `*Path()` answers with the host path (`MemoryFolderPath`, `ArtifactPath`, …) | same |
| a save dialog, Downloads | bytes as `ExportFile` (`SessionExportBytes`, `PictureBytes`, `AgentPackageBytes`) or `DeckExportFiles` | `decks_test.go` |
| the browser's picture | `SaveBrowserShot` puts it in the project at `output/<session>/work` | `window_twins_test.go` |
| the browser's address bar | `ResolveWorkbenchURL`, `SandboxFile` resolve against the project | `app_test.go` |
| the machine switch flipped | `ComputerControlChanged` rebuilds the tool block | `screen_test.go` |
| a key saved, a sign-in, a sign-out | `ProviderCredentialChanged` forgets the old quotas and rebuilds if it is the provider on screen | `providers_test.go` |
| a voice picked, checked on this machine | `RememberTTSVoice`; `VoiceSettings` reads the pick back | `voice_test.go` |
| the page `open` wrote a sentence about | `BrowserOpenedLine` / `ParseBrowserOpened` sit side by side in [agent_pages.go](agent_pages.go); `RecentAgentPages` reads `tool_runs` back | `workbench_agentpages_test.go` |

Two things stayed here that read as the screen's: [mcp_oauth.go](mcp_oauth.go)
— the credential it stores is read by `${connect:}` on the host the MCP server
runs on, so the store must be that host's (a remote engine has no browser for
the step; v1 does not support it there) — and [remote.go](remote.go), the parked
phone remote, whose device table is in `aetox.db` and whose adapters call
bindings: a second screen on the same engine, which is what phase 5 makes of
it.

## Files — where to look

| Cluster | Files |
|---|---|
| the turn and its chat | [app.go](app.go) (`Engine`, `applyConfig`, `runTurn`, `SendMessage`, provider/model switching, project focus), [conversation.go](conversation.go) (one chat: config, agent, transcript, desk state), [stream_delivery_test.go](stream_delivery_test.go) |
| what the model is handed | `workbenchSkills` = `sessionSkills` (desk pack [workbench_desk.go](workbench_desk.go), [ask_user.go](ask_user.go), todo, plan, session search, video, cutting room) + `Screen.WindowTools`; [stance.go](stance.go) narrows per action; [toolargs.go](toolargs.go) the shared argument readers |
| the store | [db.go](db.go) and its migrations, [sessions.go](sessions.go), [session_edits.go](session_edits.go), [query.go](query.go) (`eachRow`/`queryAll`, ARCHITECTURE §6.7) |
| what waits for a person | [pending.go](pending.go) (memory proposals), [task_chips.go](task_chips.go), [ask_user.go](ask_user.go), [workspace.go](workspace.go) (widen) — all emit-and-block through `Screen.Emit`, which is why they cross a socket unchanged |
| plans and goals | `plan*.go`, [goal_run.go](goal_run.go) |
| the desk | [terminal.go](terminal.go) (PTY sessions; also `shutdown`), [busy_signal.go](busy_signal.go), [desk_events.go](desk_events.go), [window_twins.go](window_twins.go) |
| files and decks | [filehost.go](filehost.go) (`/aetox-file/`), [artifacts.go](artifacts.go), [decks.go](decks.go) + [deck_draw.go](deck_draw.go) (a deck is a file here; its pixels are the screen's), [image.go](image.go), `video_*.go` |
| git and PRs | `git_*.go`, [pr_room.go](pr_room.go), [review.go](review.go) |
| the rest of Settings' engine half | [mcp.go](mcp.go), [connections.go](connections.go), [skills.go](skills.go), `skilltune*.go`, `habits*.go`, [presets.go](presets.go), [subagents.go](subagents.go), [office.go](office.go), [spaces.go](spaces.go), [usage.go](usage.go), [voice.go](voice.go) / [speech.go](speech.go) (the pickers and the mic; the reading is the screen's), [capabilities.go](capabilities.go) |

## Tests

`go test ./internal/engine/` — about six minutes on the owner's machine,
under a `-timeout` longer than the default when the machine is busy. The
helpers: `seed(&Engine{…}, conv)` installs a conversation and `testScreen`;
`bootDeskApp(t, desk)` / `bootDeskAppLending(t, desk, packs)` is a real
engine at a desk with a recorder on `a.emit`; `captureEvents(a)`;
`newJobApp`, `newMCPTestApp`, `newSpeechTestApp`; `packStub` is a packed tool
the shape of the window's for the tests about what the engine does to one.
[tool_coverage_test.go](tool_coverage_test.go) drives every engine tool
through the real dispatcher; the window's two are driven the same way in
`desktop/window_tools_test.go`.

## Rules of thumb

- Nothing here may need a window. If a change wants `a.ctx`, a dialog, a
  tab, a mouse, or a key, it belongs on the screen and the engine gets a twin
  that takes a path, answers with bytes, or is told a fact.
- A new exported method is a new binding: regenerate, and keep its arguments
  and results plain values — they cross a socket in phase 2.
- An event goes through `emitEvent` and nowhere else; `a.emit` is the test
  seam, `Screen.Emit` the real door.
- Slices a binding hands back are never nil (`jsonSlice`, ARCHITECTURE §34;
  `binding_slices_test.go`); queries go through `eachRow`/`queryAll`.
