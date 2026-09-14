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
| [report.go](report.go) | `--report`: the per-turn tally (rounds, tokens, cost, tool calls, questions) and the JSON line it becomes. |
| [login.go](login.go) · [account.go](account.go) | `aetox login <provider>` — the sign-ins the signer then uses. |

## Behavior notes

- **stdout is the answer, stderr is the work.** A script gets the model's text and nothing else. The answer streams as it is written; the authoritative delivery at the end prints only what streaming missed.
- **Questions come from stdin, one reader.** `ask_user` and an approval under `--approval ask` are the same event; the console prints the question and numbered options and takes one line. A closed stdin answers with nothing — the model is told, and an approval so answered is a refusal.
- **Ctrl-C** during an answer stops the answer (`CancelTurn`); at the prompt it exits.
- **Same data as the window** — `aetox.db`, the key store, the preference file, `<DataRoot>/logs`. Running here is a screen standing in a folder, so the window's next launch opens there too. A benchmark or a test sets `AETOX_DATA_ROOT` to a folder of its own.
- **In-process, not over the socket.** `cliScreen` is the same interface the window serves on the wire (`rpc.ServeScreen`), so a console on a remote engine is `rpc.Dial` in place of `engine.NewEngine` — nothing in run.go would change. See §268.3 for why this is not the drift §248 warns about.
- Windows vs other OS terminal setup is split into `main_windows.go` / `main_other.go`.
- Not shipped in the installer (§30); built with `go build ./cmd/aetox/`.

## ใช้งาน

```bash
go install github.com/Mikedev115/Aetox/cmd/aetox@latest
```

ครั้งแรกบนคีย์บอร์ดที่ยังไม่เคยเลือกโมเดล จะมีเมนูให้เลือกผู้ให้บริการ · โมเดล · ระดับคิด (คีย์ที่พิมพ์ถูกเก็บใน `credentials.json` ของแอป) ผู้ให้บริการที่ล็อกอินแทนคีย์ — Codex, Copilot — ใช้ `aetox login codex` ก่อน

| ทำอะไร | คำสั่ง |
|---|---|
| คุยในโฟลเดอร์นี้ | `aetox` |
| สั่งครั้งเดียว คำตอบออก stdout | `aetox chat "แก้เทสที่แดงใน internal/foo"` |
| ส่งโจทย์ทางท่อ | `cat task.md \| aetox` |
| เลือกโมเดล/ระดับคิด/โหมดอนุมัติ | `aetox --model-provider deepseek --model-name deepseek-flash --think low --approval full-access` |
| จดผลต่อเทิร์นเป็น JSON | `aetox --report runs.jsonl chat "…"` |
| ชี้โปรเจกต์อื่น | `aetox --root D:\work\app` |
| แยกข้อมูลจากแอป (ทดสอบ/วัด) | `AETOX_DATA_ROOT=D:\bench\data aetox …` |

ในลูป: `/status` `/new` `/provider` `/model` `/think` `/approval` `/help` `/exit` · Ctrl-C ระหว่างตอบ = หยุดคำตอบ · ที่พรอมต์ = ออก

**รหัสออก:** 0 ตอบแล้ว · 1 เทิร์นล้ม (ข้อความบน stderr) · 2 ใช้ผิด หรือไม่มีโมเดลจริงตอบ (สั่งครั้งเดียวแล้ว engine ตกไปที่ provider ในตัวที่ตอบได้แค่ "ต่อโมเดลก่อน" — สคริปต์ควรรู้ ไม่ใช่ได้คำตอบปลอม)

**`--report` หนึ่งบรรทัดต่อเทิร์น:** `outcome` (done · cancelled · failed) · `seconds` · `rounds` (จำนวนครั้งที่เรียกโมเดล) · `tokens.in/out/cached` (+`cacheReported` — false = ไม่รู้ ไม่ใช่ศูนย์) · `cost.usd` (+`priced` — false = มีรอบที่ไม่มีราคา ผลรวมเป็นแค่พื้น) · `tools[]` ทุก call ตามลำดับ (`name` `act` `subject` และ `parent` เมื่ออยู่ใน sub-agent) · `questions` (กี่ครั้งที่รอคนตอบ) · provider/model/think/approval/root/session · ข้อความและคำตอบเต็ม
