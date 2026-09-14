# agentGallery — the role shelf for a new agent (§284)

Forty-two briefs from [msitarzewski/agency-agents](https://github.com/msitarzewski/agency-agents) (MIT — `LICENSE.agency-agents` beside them), one `.md` per role, loaded on pick by `../agentGallery.ts`. The catalogue (group, title and line — English, the trade's own names — and word count) is in that file; this folder is only the bodies.

## What changed on the way in

Each file is the source body with:

- the source frontmatter removed — Aetox writes its own from the editor's form, and `name:` is never read (the folder is the name)
- emoji stripped from headings — the brief lands in a textarea in the UI, and the UI carries no emoji
- the **Learning & Memory** section dropped — Aetox gives every agent a real `MEMORY.md`, so a paragraph asking the model to "remember successful patterns" is tokens paid every turn for nothing
- LF line endings

Everything else is the author's, verbatim. Body text keeps its inline marks (`✅` / `❌` tables carry meaning there).

## Adding or refreshing one

`scratch`-side script, not shipped: clone the repo, run `extract.py <repo> <this folder>` with the id added to `CHOSEN`, then add the card to `GALLERY_ROLES` with the printed word count. `agentGallery.test.ts` fails when a body on disk has no card or a card has no body, and when a card's `words` drifts from the file by more than a few percent.

## Why 42 and not the repo's 230

A gallery is scanned, not searched — seven groups of six is one screen. Each role is 1–3.5k words of system prompt the agent re-reads on every turn, so a role on the shelf is a cost recommended. The other 190 are one paste away in the link box under the role field. Region-specific briefs (China hiring platforms, Spanish↔English, Chinese students abroad) were passed over for that reason and no other.
