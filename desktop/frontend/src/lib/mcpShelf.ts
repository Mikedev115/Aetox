// The shelf: which MCP servers Aetox puts in front of a person, and what each
// one is for.
//
// It lived inside Settings.svelte until ห้องความสามารถ opened, and moving it out
// is what that room is built on. This list is not configuration — it is the
// product saying out loud that Aetox connects to things, which is the one job
// it could never do while it sat at the bottom of a settings page. Two screens
// read it now (the room's shelf and the register's แนะนำ strip), and a preset
// table with two copies goes stale on one of them.
//
// The charter below is the original, moved here whole rather than summarised:
// it is the reasoning that decides what may be added, so it belongs beside the
// list it governs.

import { VideoEditorCommand, VideoEditorEnvironment, VideoEditorTools } from '../../wailsjs/go/main/App'
import { config } from '../../wailsjs/go/models'

export interface MCPPreset {
  name: string
  desc: string
  why: string
  command?: string[]
  url?: string
  headers?: string[]
  tools?: string[]
  // oauth marks a preset whose header resolves ${connect:name} from a
  // credential only a browser sign-in can produce (StartMCPSignIn /
  // desktop/mcp_oauth.go), not a pasted one. The header syntax alone cannot
  // tell this apart from github's — both read `${connect:<id>}` — so
  // needsPaste would call an oauth preset one-click, and clicking Add would
  // save a server with nothing behind its header yet. Only this flag
  // separates the two paths.
  oauth?: boolean
}

// What the shelf is for. **The rule changed on 2026-08-14 and both halves of
// it are recorded here, because the older one is still good reasoning and the
// next person deserves to see why it was set aside rather than forgotten.**
//
// *Until 12 ส.ค.:* only the servers this product's own agents declare they
// need. It had carried five general-purpose picks (context7,
// sequential-thinking, memory, js-repl, exa) and none of the things a bundled
// agent asks for by name, which was backwards in both directions — the github
// agent ships `needs: mcp:github` in its own file and sent the user to a page
// with nothing on it, while recommending a server Aetox does not depend on is
// a recommendation it has no standing to make.
//
// *From 14 ส.ค. (owner: "เพิ่มมาเลยครับ พวกที่ต้อง OAuth ตัดออกก็ได้"):* also
// the hosted servers that add something Aetox genuinely cannot do itself and
// work on one click. The standing objection is answered by the second half of
// that sentence rather than by dropping it — what made the old five a bad
// shelf was not that they were popular, it was that a list is a promise, and
// an entry that cannot connect breaks it. So the bar is now:
//
//   1. **It reaches something Aetox has no tool for.** No preset for a
//      filesystem, a fetcher or a browser — those already exist here, and a
//      second one is a slower path to the same place plus a tool-block bill.
//   2. **It works with one click.** Static-header auth at worst. Anything
//      that wants OAuth is left off: internal/mcp/client.go carries only
//      static headers ("OAuth stays deferred until a real need appears"), so
//      a Notion or Linear entry would be a button leading to a form asking
//      for a token the user has no way to obtain — the exact failure the
//      paragraph below was written about.
//   3. **The endpoint was answered by the provider, not remembered.**
//
// *From 3 ก.ย.:* **rule 2 changed, and it changed because its reason expired
// rather than because anyone argued with it.** The rule never said OAuth was
// bad — it said the client could not do OAuth, so a button promising one
// click would have been a lie. internal/oauth/mcpauth.go now walks the whole
// discovery chain itself (RFC 9728 → RFC 8414 → RFC 7591 → PKCE) on the
// loopback listener and encrypted credential store that already existed, so
// for a server that supports dynamic client registration the promise is true:
// press เพิ่ม, a browser opens, come back and it is connected.
//
// So the bar is now *"one click, sign-in included"* — and the half of rule 2
// that survives is the sharper half. A server WITHOUT a
// `registration_endpoint` still cannot be one click, because the user would
// have to go and register an OAuth app with the provider first. Probed on
// 3 ก.ย.: of six OAuth-only servers, semgrep / grafana / netlify register
// dynamically and are on the shelf; elevenlabs / vercel / shopify do not and
// are held in mcpCandidates.ts. That is the same rule, applied to a client
// that can now do more — not a relaxation of it.
//
// Every URL below was verified on 2026-08-14 by sending a real MCP
// `initialize` and reading the reply: the unauthenticated ones returned a
// protocol handshake, and github returned 401 naming the header it wants.
// firecrawl was added the same day and probed the same way — it answered
// twice, once bare and once with a deliberately invalid bearer token, and
// served the same tool set both times, which is what established that the key
// is optional rather than merely unchecked on the handshake.
// Notion, Linear, Sentry and Atlassian all answered `invalid_token` — real
// servers, all four blocked by rule 2 until the client learns OAuth. Stripe
// takes a static key and is a one-line addition whenever it is wanted.
//
// A first pass had four of those reading 403 and nearly went in the notes as
// "needs auth". It was Cloudflare's bot check refusing the probe's own user
// agent. **A verification that can fail for its own reasons has to be read
// twice**, which is the whole argument for rule 3.
//
// `headers` names what the server cannot work without, and an entry may carry
// the value's prefix after a colon — GitHub wants `Authorization: Bearer
// <token>`, and a form pre-filled with only the header name is one a token
// gets pasted into raw. A preset that needs a key used to be saved straight
// to disk with none, so one click produced a server that could never connect
// and the page never said which header it wanted — it knew, and did not tell.
// `why` is rule 1 said out loud, per entry: what this reaches that Aetox has
// no tool for. It is on screen because the shelf never answered the question
// a user actually has in front of it — not "what is this" but "why is it
// being recommended to me". An entry whose `why` cannot be written without
// hedging is an entry that does not pass rule 1 and should not be here.
export const MCP_PRESETS: MCPPreset[] = [
  // The only local preset, and the only one that is a program rather than an
  // endpoint: `kino --mcp` is a subprocess, so unlike the seven below it can
  // be added while the thing it names is not installed. That is said on the
  // card rather than hidden, and ห้องงานวิดีโอ has the install button — which
  // writes this entry itself when the download lands (connectVideoEditor), so
  // this card is for the person who removed it, or who wants to see what was
  // written on their behalf.
  //
  // Its command, environment AND tool allowlist all come from Go
  // (presetCommand / presetEnvironment / presetTools): the first two are
  // absolute paths only Go knows, and the allowlist is the measured 54-tool
  // bill (desktop/videotooling.go videoEditorTools) that must not exist
  // twice.
  { name: 'kinocut', desc: 'Cut, subtitle and render video, on this machine', why: 'Aetox reads video and produces none. This is the half that cuts and renders. Install it from ห้องงานวิดีโอ, which fetches it the same way ffmpeg and Tesseract are fetched; this entry is the connection.', command: [] },
  { name: 'github', desc: 'Repos, pull requests, issues, CI', why: "Aetox's own github tool only reads. This is the half that acts — opening a pull request, commenting, moving an issue.", url: 'https://api.githubcopilot.com/mcp/', headers: ['Authorization: Bearer ${connect:github}'] },
  // Second because it is the other one a bundled agent asks for by name — the
  // deepresearch agent ships `needs: mcp:firecrawl`, and the 12 ส.ค. half of the
  // rule above is exactly this case.
  //
  // **No `headers` entry, and that is the finding rather than an omission.**
  // Probed 2026-08-14: this endpoint answers a full handshake with no
  // credential at all (firecrawl-fastmcp 3.24.0, protocol 2025-06-18) and
  // serves search, scrape and parse under a usage limit — so it clears rule 2
  // more cleanly than github, which cannot connect until a token exists. A
  // key raises the limits and unlocks the account tools, and it goes in as
  // `Authorization: Bearer ${env:FIRECRAWL_API_KEY}` through แก้ไข. Listing
  // the header here instead would make needsPaste open the form and demand a
  // key for a server that works without one.
  //
  // Rule 1 is the judgment call, and it is a split: scrape overlaps web_fetch
  // and is not why this is here. `firecrawl_map` (enumerate every URL under a
  // site) and `firecrawl_agent` (multi-source research, collected later via
  // firecrawl_agent_status) are both things Aetox has no tool for — web_fetch
  // reads one page and web_search returns eight results.
  { name: 'firecrawl', desc: 'Crawl a whole site, and multi-source research', why: 'Aetox reads one page at a time and gets eight search results. This walks a whole site and researches across many sources at once.', url: 'https://mcp.firecrawl.dev/v2/mcp' },
  { name: 'context7', desc: 'Up-to-date docs for a library, by version', why: 'Docs for the version actually installed. Fetching a documentation page cannot tell you which release it describes.', url: 'https://mcp.context7.com/mcp' },
  { name: 'deepwiki', desc: 'Ask questions about any public GitHub repository', why: 'Answers about a repository without cloning it first — reading one that size through file tools costs a whole context.', url: 'https://mcp.deepwiki.com/mcp' },
  { name: 'exa', desc: 'Web search built for models to read', why: 'Results returned as text to read rather than as pages to open, so an answer costs one call instead of a search and five fetches.', url: 'https://mcp.exa.ai/mcp' },
  { name: 'huggingface', desc: 'Search models, datasets and spaces', why: 'Aetox has no index of models, datasets or spaces, and a web search finds blog posts about them rather than the things.', url: 'https://huggingface.co/mcp' },
  { name: 'cloudflare-docs', desc: "Search Cloudflare's documentation", why: "Cloudflare's own index of its own docs, which is a different thing from a web search that happens to land there.", url: 'https://docs.mcp.cloudflare.com/mcp' },

  // Added 2026-09-03 — the first three presets that need a sign-in rather
  // than a static header. internal/mcp/client.go still connects with a
  // header only, unchanged; what changed is that `${connect:name}` can now
  // resolve from a credential a browser flow produced (mcpauth.go's RFC
  // 9728 → RFC 8414 → RFC 7591 discovery + dynamic client registration),
  // not only one the user pasted. That flow was verified against all six
  // MCP servers found OAuth-only during the same research pass this shelf's
  // other entries came from — three registered on the spot
  // (registration_endpoint present) and are here; three did not
  // (elevenlabs, vercel, shopify — a real authorization server, but no way
  // to register a client without Aetox getting a fixed client id from each
  // vendor by hand first) and are recorded in mcpCandidates.ts instead of
  // here, because a preset on this shelf has to actually connect.
  //
  // `why` on these three is a placeholder rather than a hedge or a guess:
  // the rule that it has to be written after trying the thing does not
  // bend for these, and trying one means completing a real browser sign-in
  // first, which is a step only the owner can take. Delete this note once
  // all three carry a real `why`.
  { name: 'semgrep', desc: 'Scan code for security vulnerabilities', why: '[รอเจ้าของลองเข้าสู่ระบบจริงแล้วเขียนใหม่ — ต้องผ่านหน้าจอ OAuth ก่อนถึงจะรู้ว่า "ใช้แล้วเป็นไง"]', url: 'https://mcp.semgrep.ai/mcp', headers: ['Authorization: Bearer ${connect:semgrep}'], oauth: true },
  { name: 'grafana', desc: 'Query dashboards, metrics and alerts', why: '[รอเจ้าของลองเข้าสู่ระบบจริงแล้วเขียนใหม่ — ต้องผ่านหน้าจอ OAuth ก่อนถึงจะรู้ว่า "ใช้แล้วเป็นไง"]', url: 'https://mcp.grafana.com/mcp', headers: ['Authorization: Bearer ${connect:grafana}'], oauth: true },
  { name: 'netlify', desc: 'Deploy and manage a hosted site', why: '[รอเจ้าของลองเข้าสู่ระบบจริงแล้วเขียนใหม่ — ต้องผ่านหน้าจอ OAuth ก่อนถึงจะรู้ว่า "ใช้แล้วเป็นไง"]', url: 'https://netlify-mcp.netlify.app/mcp', headers: ['Authorization: Bearer ${connect:netlify}'], oauth: true },

  // Added 2026-09-03, separately: notion was one of the four servers this
  // file's own history above names as "blocked by rule 2 until the client
  // learns OAuth" (14 ส.ค.) — checked again the same way as the three above
  // now that mcpauth.go exists, and it answers a real registration_endpoint
  // too (see mcpCandidates.ts for the discovery trail). Same placeholder
  // rule applies: delete this note and write a real `why` once someone has
  // actually signed in.
  { name: 'notion', desc: 'Search, read and write pages in a workspace', why: '[รอเจ้าของลองเข้าสู่ระบบจริงแล้วเขียนใหม่ — ต้องผ่านหน้าจอ OAuth ก่อนถึงจะรู้ว่า "ใช้แล้วเป็นไง"]', url: 'https://mcp.notion.com/mcp', headers: ['Authorization: Bearer ${connect:notion}'], oauth: true },
]

/** A stdio preset with no command written in the table is the one that has to
 *  ask Go for both. */
export const isLocalPreset = (p: MCPPreset): boolean =>
  (p.command?.length ?? 0) === 0 && !p.url

/** A header entry that already carries a ${...} reference needs nothing from
 *  the user: the value resolves at connect time from a secret the app already
 *  holds. Only a header still waiting for a paste opens the form. */
export const needsPaste = (headers?: string[]): boolean =>
  (headers ?? []).some((h) => !/\$\{(env|connect):[^}]+\}/.test(h))

// What to spawn, for the one preset that is a program rather than an endpoint.
// The table cannot spell it: it is an absolute path into this user's own data
// folder, which only Go knows. Every other preset keeps its literal, and an
// empty `command` on a stdio entry is what marks the one that has to ask
// (VideoEditorCommand answers where Aetox installs to even before the download
// has landed, so the order of installing and connecting does not matter).
export const presetCommand = async (p: MCPPreset): Promise<string[]> =>
  isLocalPreset(p) ? await VideoEditorCommand() : (p.command ?? [])

// The editor is told where its ffmpeg is, in its own vocabulary
// (KINOCUT_FFMPEG_EXECUTABLE), rather than by anything being put on the
// machine's PATH. Same reason the command is resolved in Go: these are absolute
// paths into this user's own data folder.
export const presetEnvironment = async (p: MCPPreset): Promise<Record<string, string>> =>
  isLocalPreset(p) ? await VideoEditorEnvironment() : {}

// And its allowlist, for the same reason: the measured bill lives in Go and
// nowhere else. Every other preset takes everything, and says so with [].
export const presetTools = async (p: MCPPreset): Promise<string[]> =>
  isLocalPreset(p) ? await VideoEditorTools() : (p.tools ?? [])

/** The saved entry a preset becomes.
 *
 *  One builder, because there were two: `addPreset` (the แนะนำ strip) and
 *  `installNeeded` (an agent's card saying it is missing a server) each spelled
 *  the same six fields out, including the header-splitting line — the kind of
 *  thing that gets fixed in one copy and stays broken in the other. The room's
 *  shelf would have been a third. */
export async function presetConfig(p: MCPPreset): Promise<config.MCPServerConfig> {
  return new config.MCPServerConfig({
    name: p.name,
    command: await presetCommand(p),
    url: p.url ?? '',
    environment: await presetEnvironment(p),
    headers: Object.fromEntries((p.headers ?? []).map((h) => {
      const at = h.indexOf(':')
      return [h.slice(0, at).trim(), h.slice(at + 1).trim()]
    })),
    tools: await presetTools(p),
  })
}

/** The preset for a server an agent named, when it can be installed in one
 *  press. One that wants a token pasted cannot be finished without the form,
 *  and one that wants a sign-in cannot be finished without a browser round
 *  trip — both are deliberately not found here for the same reason: an
 *  agent-declared need is met by one call, and neither of those is one. */
export const presetFor = (id: string): MCPPreset | undefined =>
  MCP_PRESETS.find((p) => p.name.toLowerCase() === id.toLowerCase() && !needsPaste(p.headers) && !p.oauth)
