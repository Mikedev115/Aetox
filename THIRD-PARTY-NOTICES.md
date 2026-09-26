# Third-party notices

Aetox itself is proprietary (see [LICENSE](LICENSE)). The components listed
here are **not**: each one is the work of its own authors, licensed under its
own terms, and nothing in the Aetox licence restricts your rights in any of
them. They are listed so that the notices travel with the software, which is
what those licences ask for.

Every component below is under a permissive licence (MIT, BSD, Apache-2.0,
SIL OFL, or MPL-2.0 with an Apache-2.0 alternative). There is no GPL or AGPL
code in Aetox.

The list is what is actually linked into the shipped binary and bundled into
the shipped frontend — not everything `go.mod` or `package.json` mentions.
The Go table is `go list -deps -f '{{if .Module}}{{.Module.Path}} {{.Module.Version}}{{end}}' ./desktop`;
it was checked against that command on 2026-08-25 (v1.5.7) and every row matched.
The console (`cmd/aetox`, shipped as its own `aetox-cli` zip since 25 ก.ย. 2026) adds its screen —
the Charm libraries (Bubble Tea, Lip Gloss, Bubbles, Glamour) and what they
link — checked against the same command on `./cmd/aetox` that day.

---

## Go modules

Full licence texts are in each module's own repository, and in the local Go
module cache under `$(go env GOMODCACHE)`.

| Module | Version | Licence |
|:---|:---|:---|
| charm.land/bubbles/v2 | v2.2.1 | MIT |
| charm.land/bubbletea/v2 | v2.0.9 | MIT |
| charm.land/glamour/v2 | v2.0.1 | MIT |
| charm.land/lipgloss/v2 | v2.0.6 | MIT |
| github.com/alecthomas/chroma/v2 | v2.14.0 | MIT |
| github.com/atotto/clipboard | v0.1.4 | BSD-3-Clause |
| github.com/aymerick/douceur | v0.2.0 | MIT |
| github.com/charmbracelet/colorprofile | v0.4.3 | MIT |
| github.com/charmbracelet/ultraviolet | v0.0.0-20260811164956 | MIT |
| github.com/charmbracelet/x/ansi | v0.11.8 | MIT |
| github.com/charmbracelet/x/exp/slice | v0.0.0-20250327172914 | MIT |
| github.com/charmbracelet/x/term | v0.2.2 | MIT |
| github.com/charmbracelet/x/windows | v0.2.2 | MIT |
| github.com/clipperhouse/displaywidth | v0.11.0 | MIT |
| github.com/clipperhouse/uax29/v2 | v2.7.0 | MIT |
| github.com/dlclark/regexp2 | v1.11.0 | MIT |
| github.com/dlclark/regexp2/v2 | v2.5.2 | MIT |
| github.com/dop251/goja | v0.0.0-20260806115107 | MIT |
| github.com/dustin/go-humanize | v1.0.1 | MIT |
| github.com/go-sourcemap/sourcemap | v2.1.3+incompatible | BSD-2-Clause |
| github.com/google/jsonschema-go | v0.4.3 | MIT |
| github.com/google/pprof | v0.0.0-20250317173921 | Apache-2.0 |
| github.com/gorilla/css | v1.0.1 | BSD-3-Clause |
| github.com/gorilla/websocket | v1.5.3 | BSD-2-Clause |
| github.com/leaanthony/go-ansi-parser | v1.6.1 | MIT |
| github.com/leaanthony/slicer | v1.6.0 | MIT |
| github.com/leaanthony/u | v1.1.1 | MIT |
| github.com/lucasb-eyer/go-colorful | v1.4.1 | MIT |
| github.com/mattn/go-isatty | v0.0.20 | MIT |
| github.com/mattn/go-runewidth | v0.0.27 | MIT |
| github.com/microcosm-cc/bluemonday | v1.0.27 | BSD-3-Clause |
| github.com/modelcontextprotocol/go-sdk | v1.6.1 | Apache-2.0 |
| github.com/muesli/cancelreader | v0.2.2 | MIT |
| github.com/ncruces/go-strftime | v1.0.0 | MIT |
| github.com/pkg/errors | v0.9.1 | BSD-2-Clause |
| github.com/remyoudompheng/bigfft | v0.0.0-20230129092748 | BSD-3-Clause |
| github.com/rivo/uniseg | v0.4.7 | MIT |
| github.com/segmentio/asm | v1.1.3 | MIT |
| github.com/segmentio/encoding | v0.5.4 | MIT |
| github.com/skip2/go-qrcode | v0.0.0-20200617195104 | MIT |
| github.com/UserExistsError/conpty | v0.1.4 | MIT |
| github.com/wailsapp/go-webview2 | v1.0.22 | MIT |
| github.com/wailsapp/wails/v2 | v2.13.0 | MIT |
| github.com/xo/terminfo | v0.0.0-20220910002029 | MIT |
| github.com/yosida95/uritemplate/v3 | v3.0.2 | BSD-3-Clause |
| github.com/yuin/goldmark | v1.7.8 | MIT |
| github.com/yuin/goldmark-emoji | v1.0.5 | MIT |
| golang.org/x/net | v0.54.0 | BSD-3-Clause |
| golang.org/x/oauth2 | v0.35.0 | BSD-3-Clause |
| golang.org/x/sync | v0.22.0 | BSD-3-Clause |
| golang.org/x/sys | v0.46.0 | BSD-3-Clause |
| golang.org/x/term | v0.43.0 | BSD-3-Clause |
| golang.org/x/text | v0.37.0 | BSD-3-Clause |
| modernc.org/libc | v1.74.1 | BSD-2-Clause |
| modernc.org/mathutil | v1.7.1 | BSD-2-Clause |
| modernc.org/memory | v1.11.0 | BSD-2-Clause |
| modernc.org/sqlite | v1.54.0 | BSD-3-Clause |

### Vendored copies

Two of the above are vendored into this repository under `replace` directives
in [go.mod](go.mod), because Aetox carries patches against them. Their
unmodified licence files ship with the source:

| Path | Upstream | Licence file | Patches |
|:---|:---|:---|:---|
| `third_party/conpty/` | github.com/UserExistsError/conpty | [LICENSE](third_party/conpty/LICENSE) (MIT, © 2020 UserExistsError) | — |
| `third_party/go-webview2/` | github.com/wailsapp/go-webview2 | [LICENSE](third_party/go-webview2/LICENSE) (MIT, © 2020 John Chadwick) | [AETOX-PATCH.md](third_party/go-webview2/AETOX-PATCH.md) |

The MIT licence permits these modifications and requires the copyright notice
and permission notice to be kept, which they are. The patches are documented
rather than silent so that the difference from upstream can be read.

### Ported rule sets

Logic written again in Go from another project's source, rather than a copy
of its files:

| Path | Upstream | Licence | What was taken, what was changed |
|:---|:---|:---|:---|
| `internal/designlint/` | [impeccable](https://github.com/pbakaus/impeccable) v4.1.0, the detector's regex matchers and CSS scans (`crates/detect`, `crates/core`) | Apache-2.0, © Paul Bakaus and contributors | Ten of its rules — `gradient-text`, `dark-glow`, `side-tab`, `border-accent-on-rounded`, `overused-font`, `bounce-easing`, `layout-transition`, `broken-image`, `repeating-stripes-gradient`, `codex-grid-background` — ported by hand with their ids and thresholds, reduced to what a source file proves without a browser. Changed: `ai-color-palette` reads hues from CSS colours where the original reads Tailwind classes; `tiny-text` reads the declaration where the original reads the computed size; `glyph-icon` is the craft floor's ban made mechanical and is not in the original detector; `design-allow` replaces `impeccable-disable`. The package comment names the same. |

Apache-2.0 asks that changes be marked and the attribution kept; both are in
the package's own comments as well as here.

---

## Frontend packages

Bundled into the application by Vite and shipped inside the binary.

| Package | Version | Licence |
|:---|:---|:---|
| @xterm/xterm | 6.0.0 | MIT |
| @xterm/addon-fit | 0.11.0 | MIT |
| @xterm/addon-search | 0.16.0 | MIT |
| monaco-editor | 0.56.0 | MIT |
| marked | 14.0.0 | MIT |
| katex | 0.18.4 | MIT |
| highlight.js | 11.11.1 | BSD-3-Clause |
| dompurify | 3.4.8 | MPL-2.0 OR Apache-2.0 |
| svelte (runtime) | 5.x | MIT |
| vscode-textmate | 9.3.2 | MIT |
| vscode-oniguruma | 2.0.1 | MIT |
| tm-grammars | 1.32.20 | MIT (the collection); each grammar keeps its upstream licence — svelte, vue, astro, zig, graphql and the VS Code html/css/js/ts/json/markdown grammars they embed: MIT; prisma: Apache-2.0; toml and yaml: the TextMate bundle licence (permissive) |

`dompurify` is dual-licensed; Aetox includes it under the **Apache-2.0**
alternative, which carries no file-level copyleft obligation.

### Fonts

| Family | Package | Licence |
|:---|:---|:---|
| Inter | @fontsource-variable/inter 5.3.0 | SIL Open Font License 1.1 |
| Noto Sans Thai | @fontsource-variable/noto-sans-thai 5.3.0 | SIL Open Font License 1.1 |
| Noto Sans SC | @fontsource-variable/noto-sans-sc 5.3.0 | SIL Open Font License 1.1 |

The OFL permits bundling these fonts in software of any licence, including
proprietary software. It requires that the fonts themselves stay under the
OFL and not be sold on their own, neither of which Aetox does.

---

## Bundled skills

Skill documents compiled into the binary — the shared shelf at
`internal/skill/skills/`, and the private shelf an agent carries in its own
folder at `internal/subagent/profiles/agents/<name>/skills/`. These are prose
and data — instructions the assistant reads, not code that runs — and each was
written by someone else under a permissive licence. They were adapted for
Aetox: renamed to the `aetox-` prefix, given Thai descriptions, and had the
parts that assume a runtime or a host Aetox does not have rewritten. The
originals are unmodified in their own repositories.

| Skill | Adapted from | Author | Licence |
|:---|:---|:---|:---|
| `aetox-design` | design | claudekit | MIT |
| `aetox-ui-design` (tokens, specs, slide tables) | design-system | claudekit | MIT |
| `aetox-audit-xls` (agent `sheet`) | audit-xls | Anthropic, PBC | Apache-2.0 |
| `aetox-clean-data-xls` (agent `sheet`) | clean-data-xls | Anthropic, PBC | Apache-2.0 |
| `aetox-roll-forward` (agent `sheet`) | roll-forward | Anthropic, PBC | Apache-2.0 |
| `aetox-variance-commentary` (agent `sheet`) | variance-commentary | Anthropic, PBC | Apache-2.0 |
| `aetox-gl-recon` (agent `sheet`) | gl-recon | Anthropic, PBC | Apache-2.0 |
| `aetox-break-trace` (agent `sheet`) | break-trace | Anthropic, PBC | Apache-2.0 |
| `aetox-contract-standards` (agent `doc`) | Standard Agreements | Common Paper | CC BY 4.0 |

`aetox-contract-standards` is different in kind from the rest of this table: it
carries **the licensed work itself**, not instructions adapted from one. The
five agreement texts in its `templates/` folder are Common Paper's standard
agreements, unmodified, each keeping the attribution line CC BY 4.0 requires at
the foot of its own file. Three of the five did not carry that line upstream and
it was added rather than assumed.

The six on the `sheet` agent's shelf all come from
[claude-for-financial-services](https://github.com/anthropics/financial-services-plugins).
Each was rewritten around what Aetox actually has: the originals reach a ledger
through an internal MCP server and edit a live workbook through Office JS, and
Aetox has neither, so every one of them now works from the files the user hands
over and writes a new workbook rather than editing theirs.

The `aetox`, `aetox-mcp`, `aetox-skills` and `aetox-slides`
skills are Aetox's own and are covered by [LICENSE](LICENSE), not by this
section.

Each adapted skill also carries its own `source`, `license` and `copyright`
lines in its `SKILL.md` frontmatter, so the notice travels inside the file
itself rather than only in this list.

---

## External programs Aetox can use but does not ship

These are separate programs on your machine. Aetox runs them if they are
present and tells you how to install them if they are not. They are not
distributed with Aetox and are not covered by anything here — they carry
their own licences from their own authors.

| Program | Used by | Licence |
|:---|:---|:---|
| Tesseract OCR | `image_ocr`, `video_ocr` | Apache-2.0 |
| poppler (`pdftotext`, `pdftoppm`) | `pdf_read` | GPL-2.0-or-later |
| ffmpeg | `video_ocr`, `audio_transcribe` | LGPL-2.1-or-later / GPL-2.0-or-later |
| whisper.cpp | `audio_transcribe` | MIT |
| git | `git` | GPL-2.0 |

Calling a separate program is not linking it, so their terms — including the
copyleft ones — do not reach Aetox's own code. The Windows installer offers
to fetch some of them from their official sources; each arrives under its own
licence, and nothing is downloaded without asking.

---

## Design owed, code not

No code from this section is in the binary, and none of it is a licence
obligation.

Aetox's agent loop and tool surface follow conventions that open-source coding
agents converged on; the one read closest while building them was
[OpenCode](https://github.com/sst/opencode) (Apache-2.0). No OpenCode source was
copied, translated, or vendored — the implementations below are original Go.

| What follows their shape | Where it lives |
|:---|:---|
| Tool names and parameter conventions — `read` `write` `edit` `grep` `glob` | [internal/skill/](internal/skill) |
| A tool loop that runs until the model stops calling tools, rather than to a fixed cap | [internal/cognitive/agent.go](internal/cognitive/agent.go) |
| The doom-loop guard: warn at three identical calls, stop after more | same file |
| One global output-token ceiling per turn (`OUTPUT_TOKEN_MAX`) | same file |
| Per-tool permission rules, glob-matched, last match wins | [internal/safety/safety.go](internal/safety/safety.go) |
| Scanning `~/.agents/skills/` then `~/.claude/skills/` for skills somebody else wrote | [internal/skill/discovery.go](internal/skill/discovery.go) |

`agent.go` names the upstream constant or issue in a comment beside each one.
