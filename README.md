<p align="center">
  <img src="docs/assets/logo.png" alt="Aetox" width="110">
</p>

<h1 align="center">Aetox</h1>

<p align="center">
  <strong>A Windows-native agent harness — Assistant · Code desk · Agent teams are three surfaces of one system that does the work on your machine</strong>
</p>

<h3 align="center">System over model: the environment turns an answer into verified work.</h3>

<p align="center">
  <a href="https://github.com/Mikedev115/Aetox/releases/latest"><img alt="Release" src="https://img.shields.io/github/v/release/Mikedev115/Aetox?color=2f81f7"></a>
  <a href="docs/reports/coding-harness-1.9.7/README.md"><img alt="Coding benchmark" src="https://img.shields.io/badge/coding%20benchmark-461%20runs%20%C2%B7%205%20harnesses-8250df"></a>
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/badge/license-proprietary%20%C2%B7%20source%20available-blue"></a>
  <img alt="Platform" src="https://img.shields.io/badge/platform-Windows%2010%2B-lightgrey">
</p>

<p align="center">
  <a href="README.th.md">ภาษาไทย</a> ·
  <a href="https://mikedev115.github.io/Aetox-landing/">Website</a> ·
  <a href="https://apps.microsoft.com/detail/9N4KKBRRSCZZ">Microsoft Store</a> ·
  <a href="https://github.com/Mikedev115/Aetox/releases/latest/download/aetox-amd64-installer.exe">Download</a> ·
  <a href="https://github.com/Mikedev115/Aetox/releases/latest/download/aetox-cli-setup.exe">Aetox CLI</a> ·
  <a href="docs/reports/README.md">Coding benchmark</a> ·
  <a href="https://www.facebook.com/share/g/1BnXC5EiWg/">Community</a>
</p>

<p align="center">
  <img src="docs/assets/hero-app.png" alt="Aetox desktop" width="90%">
</p>

> Free to use · **source-available, not open source** · the source in this repository stops at v1.7.0, while the
> program itself still ships every release here — [read more](#source-licence-and-who-makes-it)

**Get started** — Windows 10 or later, x64, no API key needed to try it:
[Microsoft Store](https://apps.microsoft.com/detail/9N4KKBRRSCZZ) (signed, updates itself) ·
[installer](https://github.com/Mikedev115/Aetox/releases/latest/download/aetox-amd64-installer.exe) ·
[Aetox CLI](https://github.com/Mikedev115/Aetox/releases/latest/download/aetox-cli-setup.exe) for the terminal ·
Scoop, portable zip and SmartScreen notes in [Install](#install) · every other document is listed in [docs/](docs/README.md)

**Jump to:** [Harness surfaces](#one-harness-three-surfaces) · [System over model](#system-over-model) ·
[Assistant](#the-assistant-door) · [Code](#the-code-door) · [Team](#the-team-door) · [Models](#which-models-it-works-with) ·
[Measured results](#measured-results) · [Install](#install) · [Safety](#safety-and-your-data) ·
[How the system works](#how-the-system-works) · [Licence](#source-licence-and-who-makes-it)

## One harness, three surfaces

**Aetox is a Windows-native agent harness:** the system around a model that supplies tools, a workspace,
permissions, memory, and a verification loop so it can do real work rather than only generate an answer.

| If you want | Open the surface | Aetox gives you |
|:---|:---|:---|
| AI that works with your files, the web and documents on your machine | **Assistant**<br>Use, remember, and create | Real file reads and writes, real commands, and a real browser you watch and can take over. It reads images, PDFs and audio, and hands back spreadsheets, documents and slides that actually open |
| To write, fix and test code | **Code — flagship use case**<br>Build, debug, and ship | A desk bound to your project. Agent mode lets the agent do the work; Editor mode lets you write while Aetox helps. Language servers, Git, a real terminal, and `aetox` in your terminal too |
| To split work across several roles at once | **Team**<br>Arrange, start, and watch | Talk to one secretary; the work goes to a team, a department or a company of agents with a head at every level, and comes back as real files |

Assistant and Team are surfaces of the same harness as Code, sharing one set of keys, memory, settings and
permissions; switch in the top bar at any time. It works with [28 external model providers](#which-models-it-works-with), cloud and local, plus a
built-in trial provider, so you can start without an API key. The interface ships in Thai and English, with Thai
as the default. **[Install](#install)** from the Microsoft Store, the installer, or as Aetox CLI.

## System over model

**Models provide intelligence; Aetox provides the system that lets them act.**

```
                     MODEL        ← the brain: knows what should be done
                       ↓
                  ┌─────────┐
                  │  AETOX  │     ← the body: eyes, ears, hands, tools, permission
                  └────┬────┘
          ┌────────────┼────────────┐
          ↓            ↓            ↓
      Assistant       Code         Team
   files web docs   project Git   secretary → head → agent
```

> **The north star of this project.** *Do not build an AI that answers more. Build an AI that does more.*

There is a word for this arrangement: a **harness**, the program around a model that gives it tools, a loop,
and somewhere to work. The model is the engine; the harness is the car. That is why the same model can be a
different product in two apps, and why a published score belongs to a model and a harness together, never to
the model alone.

Much of what people assume has to come from a bigger model is, in Aetox, the system's job:

1. **A blind model drives the browser** — the system numbers every element on the page, so a model with no
   vision clicks the right button without guessing coordinates → [Assistant door](#the-assistant-door)
2. **A blind model reads images, a model that cannot hear transcribes, a model that only writes text hands back
   real files** — Thai and English OCR, offline speech-to-text, and the builders for `.xlsx` `.docx` `.pptx`,
   decks and video belong to the app → [Assistant door](#the-assistant-door)
3. **A small local model does big work** — a 9B/35B model in LM Studio or Ollama gets the same tools as a
   frontier one, and not a byte leaves your machine → [Which models it works with](#which-models-it-works-with)
4. **Same model, different result** — on GPT-6 Luna (`gpt-6-luna`, through the same ChatGPT (Codex) account for
   every harness), code passing the security tests (CWEval func-sec@1): Aetox 91.1% · OpenCode 66.7% ·
   Codex CLI 66.7% · omp 68.9% · pi 64.4%, over 461 runs judged by hidden tests (1.9.7). The method and where Aetox
   loses are in [Measured results](#measured-results)
5. **Many tools do not mean a full context** — an MCP server with 212 tools costs a 135-token index, and the
   definitions load when they are actually used → [How the system works](#how-the-system-works)
6. **An organisation run by the system, not by pleading in a prompt** — your words reach the head at every
   level verbatim, send-back rounds are counted by the system, and heads hold no run slot, so work fans out
   across several levels at once without deadlocking → [Team door](#the-team-door)

The app is two self-contained executables — `aetox.exe`, the window, and `aetox-engine.exe`, the engine that
thinks and works. No runtime to install, no `node_modules`, no bundled copy of Chromium. Because the engine is
a process of its own, it can run on another Linux machine: add the host under ตั้งค่า › เครื่องระยะไกล and
connect, and the app ships the engine over your own `ssh`; the chat, files, terminal and Git you see are that
machine's. Your provider keys never leave the machine the window runs on — the engine asks the window to sign
each request.

## The Assistant door

*Use, remember, and create* — the assistant works across the whole machine when no project is focused, or
inside a project folder plus the folders you add. Everything happens on the workbench beside the chat, which
has four rooms: slides, browser, files and terminal. The agent does not work behind a curtain and hand you a
file at the end; it opens a room, works in it, and you can reach in and change something yourself without
waiting for the turn to finish. The assistant has files and a shell, so it does software work too, and it
never hands a request back because it involves code.

> **What the system does for the model at this door**
> - Numbers every element on a web page, so a model with no vision clicks the right button
> - Turns images into text with Thai and English OCR, and transcribes audio offline on your machine
> - Builds `.xlsx` files with working formulas, `.docx`, `.pptx`, slide decks and video
> - Does arithmetic in a JavaScript interpreter compiled into the app, and shows you the script to check
> - Searches past conversations with SQLite on your machine, spending no tokens
> - Asks you before anything is posted, sent or shared outside, at every permission level

**A real browser you can watch.** Not a headless scrape: a WebView2 window composited into the app, with an
address bar, back and forward, DevTools, and eight device presets that resize the native window so CSS media
queries genuinely fire. The layer that drives it is ours, designed and built for Aetox; click and type aim at an
element's number, not at x,y coordinates. Every tab has a **Let the agent use this tab** button: lit means the agent
may touch that tab, and switching it off makes the tab yours.

<img src="docs/assets/cap-browser.png" alt="The agent driving a page in the workbench browser" width="100%">

**Images, PDFs and audio.** `image_ocr` runs Tesseract in Thai and English, so a screenshot, a scan or a
photographed form becomes text a 9B/35B model can reason about; a model that *can* see gets the image itself.
PDFs are read with the layout intact and audio is transcribed offline. Hand over a folder and ask for what you
actually want — *"go through this folder of receipts and give me one spreadsheet"*: it OCRs each image, works
out the totals in the JavaScript interpreter, puts the script beside the answer, and writes a real `.xlsx`
with live formulas.

<img src="docs/assets/cap-image-ocr.png" alt="OCR pulling Thai text out of an image" width="100%">

**Real files back.** A deck comes back as one self-contained `.html` file, editable by hand and openable on any
machine with a browser. Page through it, present it full screen, and export `.pdf`, `.png` or `.jpg` from the
slides room; the PDF comes from the same renderer that drew it on screen, and the 1280x720 slide is exactly
PowerPoint's widescreen page. `.xlsx`, `.docx` and `.pptx` are the `sheet` and `doc` agents' work, video is
`video`'s, and voice-over comes from `voice_make`.

**Hand work to a specialist.** Pick someone off the `+` menu — `@doc`, `@sheet`, `@deepresearch`, `@video` —
and your sentence reaches that agent word for word, not a paraphrase. (Typing `@` yourself does nothing, on
purpose: a pasted draft that merely quoted an agent's name once sent a whole brief to the wrong worker.) Or let
the assistant delegate through `task`: up to four specialists run at once, so three jobs cost the time of the
slowest rather than the sum. One that reaches a decision it should not make alone comes back as a question
under its name in the chat, and one still working when your answer arrives keeps going — the end of a turn is
not a deadline.

<details>
<summary>A real run: find 20 CRMs and compare them in a spreadsheet</summary>

Run on 2026-08-15, from one sentence — *"find 20 CRMs a Thai SME could actually pick and give me a spreadsheet
comparing them"*: **6m 51s, two agents, 42 tool calls between them** — 8 by the assistant, 27 by
`deepresearch` reading pricing pages, 7 by `sheet` — and one tool failure it worked around. `deepresearch` left
a report in the session's output folder and `sheet` was given the *path*, not the contents.

Twenty rows came back, fourteen with a real numeric price sorted low to high, and **six deliberately left blank**
with the reason beside them — quoted-only pricing, or a page that would not state a figure. Every row carries
the date the page was read and the link the number came from. The blanks are the part worth trusting: a table
with no gaps in it is a table that guessed.

</details>

**Give it a job, not a step.** Long work is planned before it is worked, and the plan sits on screen as one box
ticked off as it goes, so what you watch is the order it chose rather than a spinner. For thinking before acting
there is a planning mode that can read anything and change nothing; that button is yours alone, and the model
cannot switch itself into it. Two chats can sit side by side in one window (`Ctrl+\`).

**Chats can write to each other.** Copy a chat's ID beside its title or from its menu, then ask another chat to
send it a question or an update. The receiving chat handles the message while working or opens a turn for it
when idle; a question accepts one reply. A closed chat in another project must be opened first.

**It remembers, and you can undo.** The assistant writes down what is worth keeping on its own — when you
correct it, confirm an unusual approach, or make a decision the code does not show — and the card under the
answer shows the kept line with an **Undo** button. A fact you declined or undid is never kept again, in any
wording. Lessons from a mistake repeated three times still wait in the review queue for you to accept or
discard. Everything kept is plain markdown you can open, edit or forget. **Standing instructions** are your
own always-on files that ride into every desk, every project and every agent. Every conversation and tool run
lives in local SQLite with FTS5, so `session_search` is a query on your machine rather than a question to the
model, in Thai and English alike.

**Connect the services you already use.** Paste a link, an install command, or just the name of an MCP server
or skill into the chat; the system finds the official one, checks whether it needs a key or OAuth, and puts up
a card for you to press. Every install needs a person to press it. The MCP library in the app shows each
server's measured tool count and tokens before you install. Built-in account connections are n8n and Windmill
(the `automation` agent builds and edits workflows — read the limits in
[How the system works](#how-the-system-works) before you rely on it), Meta (the `ads` agent reads ad accounts,
`content` posts to Pages) and YouTube (the `youtube` agent uploads clips).

**A companion that lives on your desktop.** The robot mascot is drawn from code, has poses for what the
assistant is doing, greets you by name and reads its finished answer aloud. Keep it in the window, or let it
out onto the desktop as a real Win32 window you can drag to any monitor.

<details>
<summary>Use the assistant from Telegram or Discord</summary>

Open **Yours → Connections**, then choose Telegram or Discord:

1. Telegram: create a bot with `@BotFather` and connect its bot token.
2. Discord: create an app in the Discord Developer Portal, add a bot, and invite it with View
   Channels, Send Messages, and Read Message History. Message Content Intent is not required.
3. Aetox shows a one-time `/pair 123456` command after connecting. Send it from the Telegram chat
   or Discord channel you want to authorize (mention the bot when pairing in a Discord server).

This is Aetox's real main assistant, not a separate bot: it uses the Assistant desk's identity and tools. Each
paired chat or channel keeps its own conversation history. The bot accepts messages only from the paired
conversation; mention it in Discord server channels, while DMs work directly. Use `/new` to start with fresh
context. Tokens and pairing data are encrypted at rest, and listeners run only while Aetox is running.

</details>

## The Code door

*Build, debug, and ship* — the second door is a workshop, not a chat with a coding mode switched on. It is
bound to the project folder you open and has working rules of its own: a turn ends when the work is done, not
when a plan for it is written; done means proven by execution; new work has a test in the project's own suite;
and after the narrow test comes the project's full check.

> **What the system does for the model at this door**
> - Asks the language server where an identifier is declared, who calls it, and what a change would affect
> - Loads an `.html` page it just wrote at desktop and phone width, light and dark, and reports what to fix
> - Hands the model the result of a command that has been still for 2 minutes, instead of waiting out the timeout
> - Sends back an answer that says itself the work is unfinished, to be finished
> - Keeps history already sent unchanged, so the provider's cache holds every round
> - Snapshots files before every turn edits them with shadow-git, so any turn can be undone

### Two modes: Agent · Editor

The switch in the top bar changes mode at once. Each mode is a desk of its own with its own chat list, bound to
the same project.

The mode switch can be folded into a small tab that still names the mode and shows whether the chat is working
or waiting on you. Git now has fetch, pull and push, ahead/behind counts, a branch graph, and a menu for opening
or creating worktrees. A pull starts with fast-forward; a diverged branch offers rebase or merge.

**Agent — the agent works, you watch the workbench.** The chat sits on the left and the workbench on the right:
a real ConPTY terminal with unlimited tabs, the browser the agent drives, the file tree, Git with the commit
timeline on the same page, and a code map.

- **Find out why a test is flaky** — it greps the repo, runs the suite in a terminal tab you are watching,
  reads the failure, and edits the file.
- **Know what a name is before you trust it** — `codebase` asks the language server instead of a text search
  that has to guess.
- **See the shape of a repository you did not write** — a code map of the whole tree, with `grep` and `glob`
  across all of it.
- **Write a page and know at once where it breaks** — the check comes back with the write: a broken script,
  text under AA contrast, a page wider than the phone, a dark mode still light underneath, Thai marks that float
  or collide, and controls too small to tap.
- **Read a diff without leaving the conversation** — expand a tool row and the change shows as git hunks.

**Editor — you write, Aetox helps.** The file sits in the centre in Monaco, the right panel holds the agent,
Explorer, Git and deliverables, and terminals dock below. There is no extension market, so everything is built in.

- **Ctrl+K to edit in place** — a few words under the selected lines; the answer comes back as hunks you keep or
  revert one at a time.
- **Completions from any model** — DeepSeek, OpenAI-style and Ollama FIM endpoints, or any chat model, set
  separately from the chat's model.
- **Language servers behind the editor** — completion, hover, go-to-definition, a problems list and symbol
  search. The languages page is a catalogue of 44 servers, installed through a toolchain or downloaded from the
  server's own release with its SHA-256 checked.
- **Git in the editor** — changed lines in the gutter, blame on the cursor line, a side-by-side diff against
  HEAD, and committing hunk by hunk without touching the real index a shell may be using.
- Split view · run the test under the cursor · replace across the project · `.editorconfig` · format on save ·
  SQLite files open as read-only tables

### Long runs that actually finish

Real coding work runs for tens of minutes, and what brings it down is mostly not the model but what surrounds it.

<details>
<summary>How the system keeps long runs from hanging or failing, and why GitHub work lives with the github agent</summary>

- **A still command does not stall the turn** — "still" means no output, no CPU and no change in the process
  set, so a quiet compile that is still working is not counted as hung. Shell output is capped at 30 KB,
  keeping both the head and the tail, so one huge log does not make every later request dearer.
- **Dropped connections and 5xx really retry** — a failed DNS lookup or a cut connection retries at the
  connection layer and again at the turn layer; a 5xx backs off and retries at 2, 5 and 10 seconds instead of
  ending the turn in a fraction of a second.
- **Long sessions cost less** — tool results are plain text, re-reading lines still in the conversation says
  where they are instead of sending them again, and grep cuts between results, never through one.

GitHub work — searching repositories, reading and opening pull requests, reading CI, commenting — goes to the
`github` agent, which holds those tools. The Code desk stopped carrying GitHub and connected-service tools on
every request: in 30 days they were never called, yet they cost about 1.4k tokens every round. Merge and close
are deliberately absent: closing an argument is one click on a page you already have open. This desk also
deliberately has no document or spreadsheet writer, no OCR and no PDF or audio reader; a deck *about* code is
the Assistant door's job.

</details>

### Aetox CLI — the Code desk in your terminal

Type `aetox` in any folder to open the Code desk full screen: tools run above the prompt, permission cards are
answered in place, and you can type while it works.

```text
aetox                       full screen (--plain for line by line)
aetox chat "goal"           one turn, the answer on stdout, for scripts
aetox login codex           sign in to a provider (codex, copilot, ...)
aetox --whole-machine       tools reach the whole machine, credential stores stay closed — for disposable containers or VMs
aetox --report runs.jsonl   append one JSON line per turn: rounds, tokens, cost, seconds, tools
```

The CLI has [its own installer](#install), separate from the app. When the app is installed, the CLI uses the
app's engine, keys, chats and memory; without it, the CLI uses its own engine. It tells you when a new release
is out. This CLI is the build the coding benchmark measures.

### Coding results

On the same models against OpenCode · Codex CLI · omp · pi (1.9.7, 461 runs), Aetox leads on secure code on both
models and passes the most checkpoints in long multi-turn coding on GPT-5.6 Terra; on hard tasks with GPT-6 Sol it
ties Codex CLI. It trails on Web-Bench, and the price is more time and more tokens — see [Measured results](#measured-results).

## The Team door

*Arrange, start, and watch* — the third door is where Aetox stops being one assistant and becomes an
organisation. You talk to one **secretary**. The secretary has no hands and no shell: it takes the order, asks
when it must, decides which level the work should reach, hands it on and reports back. It never redoes the work
itself and never rewrites your words into a brief of its own.

> **What the system does for the model at this door**
> - Delivers your words to the head at every level verbatim, never rephrased
> - Gives heads no run slot, so a department hands work to three teams and a company to three departments at once
> - Counts how many times a head sends work back, instead of asking for a limit in the prompt
> - Carries company and department rules and context to the heads along the route the work took; members get only what they need
> - Lets every agent use its own model, so the cost follows each agent's work
> - Checks the organisation's shape: departments with no head, teams never used, agents holding the same tool set

### Three levels, each with a head that thinks

| Level | Made of | What the head does |
|:---|:---|:---|
| **Team** | a team head + agents | Takes the whole job, splits it into plain sentences, one non-overlapping deliverable per member, and judges what comes back |
| **Department** | a department head + teams | Broad work within one domain, handed to several teams at once; a team may belong to several departments |
| **Company** | a company head + departments | The whole picture, handed to departments or straight to a team; founded once three departments have heads |

The secretary picks **the lowest level that can own the job**: clear work for one team goes straight to that
team, broader work goes up to a department or the company. No level is a forced pass-through, because going
higher than the job needs costs a round and gains nothing. Members receive only their head's order and their
own seat in the team, and the team's rules reach everyone in it.

### Watch the work at every level

- **The card in the left bar sets where you stand** — team, department or company. The secretary is told on
  every message; changing level starts a new chat. History stays one list, each row labelled with where it started.
- **Delegated work is one tree in the chat** — the head on top, members below, each row saying what it is doing
  with a stop button of its own. A question appears under whoever asked it and is answered right there; click a
  row to open the full work in the right panel.
- **Deliverables are real files** — in the team's folder or the focused project, and they show up as cards in
  the secretary's answer.
- **The company room** — the right panel is a 3D office with a campus or tower map (a 2.5D room on machines
  without WebGL). Each agent sits at a desk and moves with the tool it is using, heads have their own desks and
  ranks, finished work is carried off to delivery, time of day follows your clock, and a board shows done today
  · working · waiting on you.
- **The workroom** — save a pipeline where each step names the team that takes which piece; start it and it
  walks step by step, and a step that stops to ask resumes once you answer.
- **A web console** — the `:8317` page on your machine configures teams, departments and companies, and has a
  simulated phone that chats with the secretary through the same chat as the window.

### The organisation is plain files

Every level is markdown on disk, managed under **ตั้งค่า › องค์กร** (Settings › Organisation), which draws the
whole organisation as a tree, checks its shape, and summarises what each team has spent.

```text
<DataRoot>/teams/<name>/TEAM.md                head · members · each member's seat · ## team rules
<DataRoot>/departments/<name>/DEPARTMENT.md    department head · teams · ## department context · ## department rules
<DataRoot>/companies/<name>/COMPANY.md         company head · departments · ## company context · ## company rules
```

The installer seeds one team, **ทีมคอนเทนต์** (the content team). Everything else you create from Settings or
from template sheets that fill in identity, way of thinking and duties, with recommended MCP servers per role.
Twenty-one people ship with the app to choose from:

| Role | Ships with the app |
|:---|:---|
| Team heads | `lead` `producer` `planner` `headwriter` `director` `publisher` |
| Department heads | `chief` `showrunner` |
| Company head | `boss` |
| Working agents | `ads` `automation` `content` `deepresearch` `doc` `editor` `github` `scriptwriter` `sheet` `video` `web` `youtube` |

### One agent is one folder

Hiring someone new is dropping a folder into `<DataRoot>/agents/` — no release, no plugin API, no restart. The
folder is the agent's whole identity: `AGENT.md` (who it is, which tools it narrows itself to, which model it
pins, and `role: head` for a head), `MEMORY.md` (what it has learned), `STARTERS.md` (how an empty chat with it
opens) and a private `skills/` folder no other agent can see.

That folder is where a clever assistant and an organisation part ways. Each agent pins **its own model, at its
own provider**, so the one that opens twenty pricing pages can run on something cheap while the one that weighs
what was found runs on something strong, and the bill follows the work instead of the hardest task in it. Each
keeps **its own memory**, so what the research agent learned about a source does not leak into the document
agent's judgement about a contract.

Agents work for you outside the Team door as well. At the Assistant and Code doors, the assistant hands work to
the agents that chat may delegate to (set under ตั้งค่า › สิทธิ์มอบงาน, Settings › Delegation), or you can talk
to an agent directly in the specialist room. Separately there are two **sub-agents** (`explore`, `general`):
internal helpers, a fixed set that cannot be extended by design, though you may tune each one's model,
provider, prompt, step ceiling and look.

## Which models it works with

**28 external providers, and the window shows every one** — OpenAI · OpenAI-compatible (your own endpoint) ·
Anthropic · Gemini · DeepSeek · Qwen · Z.ai · OpenRouter · Codex · Groq · Mistral · Kimi · MiniMax ·
Xiaomi MiMo · Xiaomi MiMo Token Plan · xAI · Meta · ThaiLLM · ModelScope · NVIDIA · GitHub Copilot · Kilo ·
Ollama Cloud · OpenCode Zen · OpenCode Go · LM Studio · Ollama · 9Router (Xiaomi MiMo and its Token Plan count
separately because they use different endpoints and keys). On top of those, the built-in `aetox` trial provider
is not counted among the 28. ChatGPT (Codex), GitHub Copilot and OpenRouter sign in; the rest take an API key or
a local server address. An endpoint you add yourself chooses its API shape (openai · anthropic · responses).

Local models are first-class: Aetox asks LM Studio and Ollama which model is *loaded*, streams both the answer
and the reasoning, makes real tool calls, and counts tokens into the same statistics. You can switch provider or
model mid-conversation and the whole context comes along — tool calls, results and compacted summaries, not just
the visible messages. Aetox never quietly reroutes your turn to another provider you happen to pay for. Model
lists, API versions and reasoning levels are asked of the provider at run time rather than written into the
code, so a new model shows up without waiting for a new Aetox.

Settings has an independent on/off switch per model: turning one off hides it from pickers without changing a
chat already using it. Stars are favourites. Search covers the full list while only a window of rows is drawn.
Under **Personnel**, employees and helpers share a searchable directory with rank and source filters;
their overview shows separate model choices. Each chat keeps its own selected model and reasoning dial.
9Router, LM Studio and Ollama have setup cards showing installation, server and model/key status.

## Measured results

The rules are in [BENCHMARK.md](docs/reports/BENCHMARK.md), and the one rule it enforces is that a number that
has not passed them does not appear here or on the website.

**Coding, 1.9.7 (4 October 2026).** Same models and byte-identical tasks as OpenCode · Codex CLI · omp · pi:
11 tasks × 2 models (`gpt-6-luna`, `gpt-5.6-terra`, medium reasoning) × 3 rounds, every task judged by hidden
tests, no user skills or settings anywhere. The other harnesses were measured on 27–28 September with the same
account. What held in every set of rounds:

- **Secure code** — Aetox leads on both models: CWEval func-sec@1 91.1% against 68.9% for the next harness on
  Luna, 93.3% against 77.8% on Terra
- **Long multi-turn coding** — second in % tests passed on both models (Luna 90.1%, Terra 92.0%; OpenCode 90.8%
  and 92.8%) and the most checkpoints fully passed on Terra (7.0 of 14)
- **Hard tasks on GPT-6 Sol** — DeepSWE 1.1 and Terminal-Bench 2.1 hard, 12 tasks at high reasoning: Aetox 7 ·
  Codex CLI 7 · OpenCode 6
- **The price** — more time and more tokens than the others (Terra 76 min against 46–71; about 90% of the tokens
  are cached input). Against 1.9.3, Terra is faster and cheaper (81 → 76 min, cost −11%). The benchmark is
  one-shot, so the main agent does nearly all of it (helpers called in 1 of 78 runs); in a real session helpers
  kept the main context at 107k instead of 190k and cut cost by about 6% on OpenAI prices, 18–30% on other providers
  ([session report](docs/reports/LIVING-REALM-DELEGATION-TOKENS-20261004.md) ·
  [cost report](docs/reports/MODEL-COST-SIMULATION-LIVING-REALM-20261004.md))
- **Where it trails** — Web-Bench, where pi passes more on both models and omp on Terra · the single SWE-bench
  bug on Luna

Rounds swing by 1.5–2.5 points on SlopCodeBench, so a gap of a few points reads as a tie. The tables, every
round, the method and where Aetox loses are in the reports — start at [the reports page](docs/reports/README.md).

| Other reports | What they measure |
|:---|:---|
| [Reports page](docs/reports/README.md) | Every report, and the latest edition of each subject |
| [coding-harness-1.9.7](docs/reports/coding-harness-1.9.7/README.md) | The 1.9.7 edition against the same four harnesses, with two sets of rounds and the swing between them |
| [HARD-SOL-1.9.7](docs/reports/HARD-SOL-1.9.7-20261004.md) | Hard tasks on GPT-6 Sol: DeepSWE 1.1 and Terminal-Bench 2.1 hard, Aetox against Codex CLI and OpenCode |
| [HARD5-1.9.7](docs/reports/HARD5-1.9.7-20261004.md) | Five mid-level tasks across releases 1.9.0 → 1.9.7 and three model sizes |
| [MODEL-COST-SIMULATION](docs/reports/MODEL-COST-SIMULATION-LIVING-REALM-20261004.md) | One real session re-priced on eight other models, and what helpers save |
| [coding-harness-2026-09-27](docs/reports/coding-harness-2026-09-27) | The 1.9.0 edition against OpenCode · Codex CLI on the same tasks, with each task's prompt, source and licence |
| [SKILL-BENCH.md](docs/reports/SKILL-BENCH.md) | How much Aetox's skills actually help, skills on against off on the same model, including what did not move |
| [TOKEN-AUDIT.md](docs/reports/TOKEN-AUDIT.md) | Where the tokens go in each request |
| [TEST-REPORT.md](docs/reports/TEST-REPORT.md) | Tests by module |
| [aetox-research-reasoning-evaluation.md](docs/reports/aetox-research-reasoning-evaluation.md) | Research and reasoning evaluation |
| [PUBLISHED-NUMBERS.md](docs/reports/PUBLISHED-NUMBERS.md) | Where every published number appears, and where it was measured |

<details>
<summary>The app's own numbers — 1.9.5 package sizes and dated runtime figures</summary>

**1.9.5 packages, measured 2026-10-02.** SHA-256 and the release signature were checked; the Store package's
executables match the portable package. MiB is bytes / 1,048,576. See the [package report](docs/reports/release-1.9.5-verification.md).

| Package | Size |
|:---|---:|
| App on disk, two executables | **91.6 MiB** (window 55.8 + engine 35.8) |
| App installer | **37.1 MiB** |
| Portable app | 36.1 MiB |
| CLI installer / portable | 22.7 / 31.2 MiB |
| Microsoft Store package | 37.4 MiB · version 1.9.5.0 |

**Historical measurements below.**

The two size rows and the two test-count rows were measured on 2026-09-17 on v1.7.2; turn assembly on
2026-08-13; the rows marked ⁽ᵈ⁾ on 2026-07-27 on v0.9.2, before the engine became a process of its own, so
the process count is one lower than today. Every row is a dated figure, not a current one.

| | |
|:---|---:|
| What you download | Installer, 34.6 MB |
| What lands on disk | **83.4 MB** in two files — `aetox.exe` 50.9 MB + `aetox-engine.exe` 32.5 MB |
| Assembling one turn | 0.32 ms · 174.9 KB allocated |
| Go tests | 3,648 across 62 packages, none failing |
| UI tests | 2,106 across 205 files, none failing |
| First start (cold) | 1.77 s ⁽ᵈ⁾ |
| Later starts | 0.53 s ⁽ᵈ⁾ |
| RAM committed | 252 MB ⁽ᵈ⁾ |
| Processes | 7 ⁽ᵈ⁾ |

The old figures stay rather than being deleted, because on the day they were taken they passed the rules —
which is the whole difference between an old number and an unusable one.

Two things said plainly. Turn assembly was 0.12 ms and 96.2 KB when the block held 27 tools; it is 0.32 ms and
174.9 KB now, because the block holds more. That is a real regression, and it is still three ten-thousandths of
a second — the time you wait is the model thinking. And the size went from 48.5 MB in one file to 83.4 MB in two:
the engine is its own file now, but the window still links the engine packages for types and forwarders, so the
split added a binary without the first one shrinking. That is the real cost of splitting the engine out.

**Against Zed**, the harsher yardstick — native Rust, and known for being light.

| | Aetox | Zed |
|:---|:---|:---|
| First start (cold) | 1.77 s | 2.12 s |
| Later starts | 0.53 s | 0.53 s |
| RAM committed | 252 MB | 471 MB |
| Size on disk | **83.4 MB** | 419 MB |

Every cell except Aetox's size was measured on 2026-07-27 on the same machine under the same rules, and neither
has been re-measured. Electron apps in this category ship 240 MB to 1 GB because each carries its own Chromium.
Aetox uses the WebView2 Windows already has — and WebView2 *is* Chromium, so memory is not where it beats
Electron; the win is that you do not keep a second browser.

**How it was measured.** Size on disk: unpack the
[portable zip](https://github.com/Mikedev115/Aetox/releases/latest/download/aetox-windows-amd64-portable.zip) and
add the two files; other apps are measured from their install folder after installing, never from a download
page · start-up, RAM and processes: `bench.ps1 -Start`, an empty project, median of 5 runs after discarding the
first, read after 60 seconds idle; a true cold start needs a reboot first · turn assembly: `bench.ps1 -Engine`,
median of 3 runs.

**What was removed.** An earlier README published "97% of input tokens came from cache across six consecutive
messages" and local time-to-first-token figures of 1.42 s and 1.75 s. Neither had a source in this repository —
no test, no log, no BENCHMARK entry — so they were removed rather than dated.

</details>

## Install

Windows 10 or later, x64. **You do not need an API key to start**: the built-in `aetox` provider ships trial
models that exercise the real machinery — real tool calls, real delegation, long streamed reasoning — so you see
what the app does before signing up for anything.

| Channel | Install | Best for |
|:---|:---|:---|
| **Microsoft Store** | `winget install --id=9N4KKBRRSCZZ --source=msstore`<br>or the [Store page](https://apps.microsoft.com/detail/9N4KKBRRSCZZ) | Signed by Microsoft: no SmartScreen prompt, and Windows keeps it updated |
| **Installer** | [aetox-amd64-installer.exe](https://github.com/Mikedev115/Aetox/releases/latest/download/aetox-amd64-installer.exe) | Program Files with a Start-menu shortcut |
| **Aetox CLI** | [aetox-cli-setup.exe](https://github.com/Mikedev115/Aetox/releases/latest/download/aetox-cli-setup.exe) | The Code desk in your terminal; installs per user, no admin rights, adds itself to PATH, not in the Store |

Other ways: Scoop for the app and the CLI, or a portable zip
([app](https://github.com/Mikedev115/Aetox/releases/latest/download/aetox-windows-amd64-portable.zip) — keep both
files together ·
[CLI](https://github.com/Mikedev115/Aetox/releases/latest/download/aetox-cli-windows-amd64.zip) — run
`.\aetox.exe path add` once to put it on PATH).

```powershell
scoop install https://raw.githubusercontent.com/Mikedev115/Aetox/main/scoop/aetox.json
scoop install https://raw.githubusercontent.com/Mikedev115/Aetox/main/scoop/aetox-cli.json
```

**Updating.** The app checks for a new release shortly after it opens and once a day, and shows a card in the corner.
Step by step for every channel, and what to do when an update does not take: [the update guide](docs/UPDATING.md).

| Channel | How it updates |
|:---|:---|
| Installer | **Download the update**, then **Restart to update**: the app closes, runs the new installer (Windows asks for admin rights, press Yes) and opens again |
| Portable zip | **Download the update**: the app swaps both files in its folder and is the new build the next time it opens |
| Microsoft Store | Windows updates it |
| Scoop | `scoop update aetox` |

If a restart brings back the same build: close Aetox completely and run the
[latest installer](https://github.com/Mikedev115/Aetox/releases/latest/download/aetox-amd64-installer.exe)
over it yourself. Your data, chats and keys stay. (Installers up to 1.9.8 could skip the window's exe when
another program held it open; fixed in [1.9.9](docs/release-notes/v1.9.9.md).)

The installer carries only Aetox's own files. Tesseract, poppler, ffmpeg and the speech model are downloaded by
the app later, only for the capabilities you tick. When the app is installed, the CLI uses the same engine, keys,
chats and memory.

> **Pick one channel and stay with it.** Windows gives packaged apps their own data folder, so the Store build
> and the installer build are two different Aetoxes on the same machine — different settings, history, memory
> and keys.

<details>
<summary>If SmartScreen or antivirus warns you</summary>

None of this happens on the Store build — Microsoft signs that one. What follows is about the installer and the
zip.

**"Windows protected your PC", unknown publisher.** The installer is not code-signed yet, so Windows has no
publisher name to show for it — **More info → Run anyway**.

**"Virus detected", or a name ending in `!ml` such as `Program:Win32/Wacapew.C!ml`.** A cloud machine-learning
verdict, not a signature: nobody analysed this file and judged it dangerous. It fires on what the file *is*
rather than what it does — an unsigned binary whose hash the world has never seen, which every release is by
definition. Desktop apps built with Go and Wails hit this across the ecosystem; an empty Wails app with no code
in it at all is [reported as the same detection](https://github.com/wailsapp/wails/issues/3308). Everything the
app downloads later is pinned to an immutable release tag and verified against a SHA256 compiled into the binary
before use, and a mismatch skips that component. Until code signing exists, the Store build or the zip is the way
past it.

**The app opens but every provider list is empty, and the engine card says `ไม่พบ aetox-engine.exe`.** The same
verdict, aimed at the second file (`Trojan:Script/…` is the family Defender uses for an unsigned executable that
starts shells — the engine does, on your behalf). Open **Windows Security → Protection history**, find the
entry, **Restore** and then **Allow on device** — Restore alone puts the file back for the next scan to take
again — then press *เริ่มใหม่* on the engine card, or reinstall. `checksums.txt` lists each exe's hash on its
own line, so a restored file can be checked.

Releases *are* signed: an ed25519 public key is compiled into the binary and the updater verifies the signature
over `checksums.txt` before it trusts a single hash. An empty or wrong key refuses the update rather than falling
back.

</details>

<details>
<summary>winget search finds nothing</summary>

The msstore source only answers an `--id` search with `--exact`, so
`winget search --id 9N4KKBRRSCZZ --source msstore` on its own returns nothing — a winget habit, not a missing
listing. Use `winget search aetox`, or paste `ms-windows-store://pdp/?productid=9N4KKBRRSCZZ` into Run (Win+R)
to open the Store app directly.

</details>

<details>
<summary>Linux and macOS</summary>

Not shipped as an app. The engine and the desktop package both compile on Linux and macOS; the browser pane is
stubbed and packaging is not done. What ships for Linux on every release is the engine alone
(`aetox-engine-linux-amd64` and `-arm64`), to run on a remote host driven by the Windows window over `ssh`.
**1.0.0 is the Windows release**: the old criterion that 1.0.0 had to ship on all three platforms was **changed by
the owner, not met**. See [PLATFORM-SUPPORT.md](PLATFORM-SUPPORT.md) for where the port actually stands.

</details>

<details>
<summary>Build it yourself from the source in this repository (v1.7.0)</summary>

```powershell
go build -o desktop/build/bin/aetox-engine.exe ./cmd/aetox-engine   # the engine, beside the window
go build ./cmd/aetox                                                 # the terminal console (runs the engine in-process when none sits beside it)
cd desktop
wails build          # → desktop/build/bin/aetox.exe
wails build -nsis    # with the installer
```

The window looks for `aetox-engine.exe` beside itself first, then falls back to `go run ./cmd/aetox-engine`
inside a dev tree — `wails-dev.bat` builds it for you.

</details>

## Safety and your data

- **Your data stays on your machine** — chat history, deliverables and browser data live on your disk. There is
  no server of ours in between and no analytics; a cloud provider sees what its API normally sees.
- **Keys and secrets** — keys are DPAPI-wrapped against your Windows account, and secrets are stripped before
  they reach a log or before a tool result reaches the model.
- **Three permission levels** — Ask · Unsafe only · Full access, and two things asked about at every level:
  publishing anything outside, and stopping programs by name.
- **Secret files are refused on every desk** — `.ssh`, `.aws`, the Windows credential stores, browser profiles
  and Aetox's own key files cannot be opened by any tool.

<details>
<summary>In detail: desks, the workspace, permissions, the shell scanner, and what is stored</summary>

**A desk** is the tool ceiling of a session. There are five — `assistant`, `coding` (Code, Agent mode),
`editor` (Code, Editor mode), `secretary` and `specialized` (a specialist agent). A chat's desk is fixed when it
opens. MCP servers and external connections are placed per desk and per agent, so a tool installed for one kind
of work does not show up in another — not hidden from the model, simply absent from that chat.

**The workspace.** With a project focused, the workspace is that folder plus any folder you add, which gets the
same read and write rights as the root. With no project focused, the workspace is the machine and writes land
under `output/<session>`. One function resolves every path, symlinks and all, and there is deliberately no
second check anywhere else.

**Approval.** One gate that every tool call goes through — built-in tools, shell and MCP alike.

| Level | What it asks about |
|:---|:---|
| **Ask** | Anything that is not a plain read inside the workspace |
| **Unsafe only** | Deletes, `git` changes, shell, anything touching a path outside the workspace, deleting things on an outside service, and tools that do not declare what they do |
| **Full access** | Nothing, except the two below |

Asked at every level, full access included: **publishing outside** (posting, sharing, sending, replying — from
the browser or a connected account) and **stopping programs by name**, which would stop other programs with the
same name too; a session stops what it started through its own handle. MCP tools are judged by what their server
declares (read-only · adds · may delete), not by the name the server's author chose, and a tool that declares
nothing counts as unknown, not as harmless.

**The shell scanner** checks real paths rather than matching patterns, and reads each shell by its own grammar. A
path hidden in quotes, behind a flag, behind a redirect, or behind `%VAR%` / `$VAR` / `~` is still resolved and
checked. The PowerShell backtick is an escape, and a heredoc fed to a program is data. What it cannot read —
`$(...)`, `${...}`, POSIX backticks, `-EncodedCommand`, `FromBase64String`, `Invoke-Expression` — is refused
rather than guessed at. Every command run is appended to a 0600 audit log.

**Refused to every file tool, on every desk:** `.ssh` `.aws` `.gnupg` `.azure` `.kube` `.netrc`
`.git-credentials` `.config/gh` `.aetox`, the Windows Credentials and Protect stores, Chrome / Edge / Firefox /
Brave profiles, and Aetox's own `credentials.json`, `oauth.json`, `account.json`, `mcp-servers.json`,
`screen.json` and browser profile. Folder-picking refuses them too, so it fails at the door rather than as a
confusing error later. The CLI's `--whole-machine` lifts the project wall but not this list. One exception,
for deploys: a key inside `.ssh` handed to `ssh` / `scp` / `sftp` with `-i` or `-o IdentityFile=`, to
`ssh-add`, or the same inside `rsync -e`, `GIT_SSH_COMMAND` and `core.sshCommand`, reaches the program — ssh
uses the key and never prints it. Reading it, copying it, `-F` on it, or naming it anywhere else in the same
line is still refused, and the local commands those settings and ssh's own options run (`ProxyCommand`,
`LocalCommand`, `KnownHostsCommand`, with `%d` read as your home) are checked like any other command. A
command that logs in to another machine — ssh, scp, sftp, rsync to a remote, autossh, mosh, ssh-copy-id,
sshfs, and a `git push` over ssh — asks first in every mode, full
access included; the card can trust that machine for good, and the trusted list lives under Capabilities ›
Connections, where each one can be removed.

|  | Where it stands |
|:---|:---|
| Chat history, tool runs, produced files | On your disk, in local SQLite and plain folders |
| Browser data (history, cookies, session) | On your machine only |
| Cutting the cloud off entirely | Run through LM Studio or Ollama and not a single byte leaves your machine |
| API keys | Their own file, 0600, DPAPI-wrapped. Off Windows there is no encryption at rest — stated rather than implied |
| Secrets in logs and tool results | Stripped through one registry: debug log, shell audit log, the buffer the bug-report form reads, and tool results before they reach the model |
| MCP secrets | `${env:VAR}` indirection, so a key never lands in the settings file |
| Taking it with you | Export any chat to `.md` or `.json`, and import a `.json` back into any Aetox |
| Bug reports | The app transmits nothing. It prefills a GitHub issue, already scrubbed, and you read every line before sending it from your own account |

</details>

## How the system works

In-depth detail and limits, for anyone who wants to check the system.

<details>
<summary>Every tool the model receives</summary>

The engine hands the model 33 tools on a fresh install, about 10,000 tokens on every request, against a ceiling
of 10,500 tokens and 48 tools enforced by a test. Add `browser`, lent by the window, and `computer`, only when
you switch it on. Each desk takes fewer, because the desk narrows the set. Many are **packed** tools — one name
with several verbs inside. Measured 2026-09-29 on v1.9.3.

| Group | Tools |
|:---|:---|
| **Files** | `change` *(write · edit · append · batch · delete)* `read` `search` *(list · glob · grep)* |
| **Running commands** | `desk_terminal` `git` `shell` *(run · output · kill · list)* `computer` *(list_apps · read · capture · focus · click · click_at · scroll · drag · type · close — only when enabled under Settings › Computer use)* |
| **Handing back files** | `asset_find` `doc_write` `sheet_write` `video` *(new · check · render · record)* |
| **Reading and making media** | `image_make` `media_read` *(image · video · audio)* `pdf_read` `video_project` `voice_make` |
| **Web** | `browser` *(open · read · click · type · wait · back · scroll · capture · tabs · dialog · console · network · hover · drag · key · upload · eval)* `media_fetch` `web_fetch` `web_search` |
| **Code work** | `codebase` *(errors · symbol · impact · map · trace · design · page)* `rename` |
| **GitHub** | `github` *(search · repo_summary · list_files · read_file)* `pr` *(list · read · checks · create · comment)* |
| **How the assistant works** | `ask_user` `calc` `desk` *(open · list · close · focus)* `memory` `plan` *(write · amend · read · step · report)* `plugin_install` `session_search` `skill_view` `task` *(start · collect · answer · message · plan)* `time` `todo_write` |

The table is generated from the registry the model is actually handed (`go test ./internal/engine -run
TestPrintReadmeToolTable -v`, plus the two the window lends), because a hand-kept list of what a program contains
is a second source of truth for a question the program can answer. Connecting n8n or Windmill adds one packed
tool — `n8n` *(list · read · create · update · activate)* or `windmill` *(workspaces · list · read · create ·
update)* — and adds nothing before that.

</details>

<details>
<summary>Skills and the MCP shelf: growing without paying on every request</summary>

Skills are markdown documents, not tools. The skill index in the prompt holds only names grouped by area, and
`skill_view` returns the content when it is used. The shelf that ships with the app is down to 24 skills
[measured to help real work](docs/reports/SKILL-BENCH.md), and every skill that changes must pass a three-way gate
(off · natural · forced on) to stay.

MCP tools live on a **shelf**: the index carries only tool names or topic groups, one line per server, and the
full definitions are sent the first time a chat reaches for them. A server with 212 tools therefore costs 135
tokens instead of about 121k.

</details>

<details>
<summary>Automation: what it can and cannot do</summary>

Connect an n8n or Windmill instance you host, and the `automation` agent lists, reads, creates and updates
workflows in it, activates them on n8n, and can start the server for you from a command you saved.

**It cannot run a workflow and read the result.** There is no execution API call in the code at all. The
nearest thing is the agent pressing Execute in n8n's own UI through the browser, which is not a run it can
verify. Windmill has no activation call either, so a flow it builds stays saved until you run it yourself. The
agent says so plainly, and a test exists whose only job is to keep it saying so.

**The Mission Control page** on `:8317` now shows work across chats, deliverables, chat and scheduled jobs. Web
chat actions are off by default, and Wi-Fi access requires pairing. Scheduled jobs run while Aetox is open;
missed runs execute once when it reopens. This schedules Aetox conversations, not n8n or Windmill executions.

</details>

<details>
<summary>When a turn goes wrong</summary>

**An answer that hits the output-token ceiling is continued**, up to three times, appended to what is already on
screen.

**A tool call with a repeated parameter name is refused.** A model sometimes writes `{"query":"A","query":"B"}`,
which is valid JSON, so the parser keeps the last and silently drops the first — one job never runs while the
answer talks as if it did. Aetox refuses the call and tells the model the real cause.

**A provider that returns nothing is an error, not an empty answer.** A round that comes back with not a single
frame is replayed after withdrawing what was already streamed; what the turn had done stays in context, and every
silence a turn survives is a row on screen.

**Dropped connections, failed DNS and 5xx retry without ending the turn**; 401, 403, 400 and exhausted quota are
told to you plainly.

**Model capabilities only ever narrow the tool set, never widen it.** A model the registry has never described
keeps the full set; only one the registry says cannot call tools is narrowed, because withholding tools by
mistake turns an agent into a chat box.

</details>

## Status — v1.9.10

The core is in place. [This release](docs/release-notes/v1.9.10.md), 6 October 2026, brings:

- **An editable cut room**: the agent and the person work on the same `.cut` file, with a multi-track timeline,
  live playback, editable captions, clip properties, transitions, titles, keyframes and export settings.
- **Prompt presets for each desk**: the assistant, code and secretary desks have their own empty-chat starters,
  a shared slash menu and glass-style design recipes with a bundled component catalogue.
- **Clearer Thai speech and an easier PowerPoint connection**: reading aloud handles English words in Thai answers,
  local transcription falls back to CPU when CUDA fails, and the MCP shelf offers a PowerPoint preset for Windows
  with desktop PowerPoint installed. It adds the server only, not Office or the upstream skill.

**Next** — a provider chain that switches accounts when a plan window runs out · external agent programs as
engines · a personal secretary connected to Google through the user's own Apps Script bridge · RAM measured under
the same conditions for every app in the tour's last scene

Documents: [every release note](docs/release-notes/) · [roadmap](ROADMAP.md) · [platform support](PLATFORM-SUPPORT.md) ·
[all measurements](docs/reports/README.md) · [every document](docs/README.md).
Architecture notes, decision records and design standards are not in this repository.

## Source, licence and who makes it

**Source.** The source published in this repository stops at v1.7.0 (15 September 2026), a complete release that
can be read and studied under the [LICENSE](LICENSE); the code here is the real v1.7.0, unmodified. The source was
open through v1.8.0 for one day (20–21 September 2026) and was then rolled back to v1.7.0. Every release since is
still published here in full — tags, installers, the portable zip, the Linux engine, checksums, release notes and
measurement reports — and the in-app update check, scoop and the Microsoft Store work exactly as before. The
architectural layer that arrived after v1.7.0 — the MCP shelf, the editor with language servers behind it, the
session context handed back on reopen, and a window nearly half as heavy at rest — is the most valuable part of the
work, and the author, who builds it alone, has chosen to keep it in order to keep developing it.

**Licence.** Aetox is free to use and is not open source. From v1.3.0 it is under a [proprietary licence](LICENSE):
install it on as many machines as you like, use it for commercial work, and read and audit the source published in
this repository — but do not modify it, redistribute it, rebrand it, or sell it. The source is published to be
*read*, not to be built on.

**Your own extensions are yours.** Skills, agents, prompts, configuration and MCP servers you write are your
property, and selling them is expressly permitted ([LICENSE](LICENSE) §4). That is what the extension points are
for.

The name **"Aetox"** and the logo are trademarks and are not licensed to anyone else. Third-party components keep
their own terms and are listed in [THIRD-PARTY-NOTICES.md](THIRD-PARTY-NOTICES.md) — none of them is GPL or AGPL.
Earlier releases keep the licence they shipped under, permanently: v0.7.1 and earlier are MIT, v0.8.0 through
v1.2.4 are Apache-2.0.

**Community and bug reports.** The Facebook group [Aetox](https://www.facebook.com/share/g/1BnXC5EiWg/) is for
questions, ideas, and the kind of half-formed problem that does not fit in an issue yet. Bugs are better filed as
[issues](https://github.com/Mikedev115/Aetox/issues), because an issue carries the version and the log with it:
Settings in the app prefills one with your version, install channel, OS and the recent internal log with secrets
already stripped, and hands it to you to read before you send it from your own account.

**Who makes this.** Aetox is written by one person. It exists because a model that can only produce text is half a
tool, and the missing half — hands, permission, and a place to put the result — is an application problem rather
than a model problem.

> Aetox was not born to compete with anyone. It exists to stand where the market has a gap — not to be one more
> agent framework, and not to lock anyone into anything. The comparisons with other tools in our reports are
> there to measure our own system.

📧 [phrmsawanachyphl@gmail.com](mailto:phrmsawanachyphl@gmail.com) ·
❤️ [Support the project](SPONSOR.md)

---

<p align="center">
  © 2026 Chayaphon Phromsawana · All rights reserved · <a href="LICENSE">Licence</a> · <a href="THIRD-PARTY-NOTICES.md">Third-party notices</a>
</p>
