# cmd/aetox — the console screen

> Module map: [ARCHITECTURE.md §4](../../ARCHITECTURE.md) · The other screens are [desktop/](../../desktop/README.md) (the window) and, over ssh, the same window with a remote engine (§248). The record of this shape is [DECISIONS §268](../../docs/DECISIONS.md).

**What it is:** the terminal entry point, and since 2026-09-14 a *screen* of the engine rather than a loop of its own. `engine.NewEngine(cliScreen)` in this process, `Startup`, the project the terminal stands in, a session at **โต๊ะโค้ด** — the same engine, desk, tools, memory and skills the window's coding desk has. `aetox chat "goal"` is one turn with the answer on stdout; `aetox` alone is a line loop; `echo goal | aetox` reads the message from the pipe.

The desk is not a flag: the console is the coding desk and nothing else. The other desks are the window's.

## Files

| File | What it holds |
|---|---|
| [main.go](main.go) | The command line — flags, `aetox login`, the first-launch provider/model/depth menu when nothing is chosen anywhere yet — then a hand-off to run.go. |
| [run.go](run.go) | The session: start the engine, open the project, sit at the coding desk, apply the dials (`SwitchProvider` · `SetProviderBaseURL` · `SwitchModel` · `SwitchThinkLevel` · `SwitchApprovalMode`); `turn` runs one message and answers the engine's questions from the terminal; `interactive` is the loop and its slash commands. |
| [screen.go](screen.go) | `engine.Screen` for a terminal: events → stdout (the answer) and stderr (tool calls, status, questions); the credential signer through [internal/signer](../../internal/signer/signer.go) with this screen's key store; no window tools, no deck renderer, and it says so. |
| [menu.go](menu.go) | The first-launch menu (arrow keys on a tty, numbers otherwise). |
| [login.go](login.go) · [account.go](account.go) | `aetox login <provider>` — the sign-ins the signer then uses. |

## Behavior notes

- **stdout is the answer, stderr is the work.** A script gets the model's text and nothing else. The answer streams as it is written; the authoritative delivery at the end prints only what streaming missed.
- **Questions come from stdin, one reader.** `ask_user` and an approval under `--approval ask` are the same event; the console prints the question and numbered options and takes one line. A closed stdin answers with nothing — the model is told, and an approval so answered is a refusal.
- **Ctrl-C** during an answer stops the answer (`CancelTurn`); at the prompt it exits.
- **Same data as the window** — `aetox.db`, the key store, the preference file, `<DataRoot>/logs`. Running here is a screen standing in a folder, so the window's next launch opens there too. A benchmark or a test sets `AETOX_DATA_ROOT` to a folder of its own.
- **In-process, not over the socket.** `cliScreen` is the same interface the window serves on the wire (`rpc.ServeScreen`), so a console on a remote engine is `rpc.Dial` in place of `engine.NewEngine` — nothing in run.go would change. See §268.3 for why this is not the drift §248 warns about.
- Windows vs other OS terminal setup is split into `main_windows.go` / `main_other.go`.
- Not shipped in the installer (§30); built with `go build ./cmd/aetox/`.
