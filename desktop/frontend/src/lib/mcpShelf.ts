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
import type { TKey } from './i18n.svelte'

export interface MCPPreset {
  name: string
  desc: string
  why: string
  command?: string[]
  url?: string
  headers?: string[]
  // env is the stdio counterpart of headers: `KEY=value` lines the spawned
  // program reads. A line with nothing after its `=` is a blank the person
  // has to fill (needsPaste), the same way a bare `Authorization: Bearer`
  // is — and for the same reason, so that Add never saves a server whose
  // program would start and immediately refuse to sign in.
  env?: string[]
  // hint names a locale line the form shows under the blanks: where the
  // value comes from, when that is not obvious from the key's name. A
  // Google OAuth client is made in a console most people have never opened;
  // a Firecrawl key is on the page the person just signed up on.
  hint?: TKey
  tools?: string[]
  // oauth marks a preset whose header resolves ${connect:name} from a
  // credential only a browser sign-in can produce (StartMCPSignIn /
  // desktop/mcp_oauth.go), not a pasted one. The header syntax alone cannot
  // tell this apart from github's — both read `${connect:<id>}` — so
  // needsPaste would call an oauth preset one-click, and clicking Add would
  // save a server with nothing behind its header yet. Only this flag
  // separates the two paths.
  oauth?: boolean
  // Which shelf this sits on. Twenty entries is past the seven a person can
  // pick from without a question in between (DESIGN.md §1), and the question
  // a person has is "what kind of thing do I want it to reach", not "which
  // vendor". Keyed, not a Thai string: the label lives in the locale.
  group: ShelfGroup
  // Whether somebody on this team has connected it and used it, and then
  // written `why` from what happened. False is the honest state for an entry
  // that clears every rule on paper and has never been pressed — those used
  // to carry their placeholder INSIDE `why`, in square brackets, and the room
  // printed the brackets to the user. Since 2026-09-13 the room draws no
  // unproven entry at all: false is the owner's queue, not a shelf band.
  proven: boolean
  // What the server offered when somebody here last connected to it, and
  // what that tool block costs on every message (the context meter's own
  // measure: JSON of each definition, ~4 chars a token). A person deciding
  // whether to add a server is deciding what every later message will carry,
  // so the card says it before the press, not after. Absent when nobody has
  // measured it — never guessed. Re-measure with a tools/list when a `why` is
  // rewritten; the count on the room's card is live and will disagree the day
  // the vendor changes the list, which is the day to update this.
  toolCount?: number
  tokens?: number
  // When toolCount/tokens were measured, so a stale number can be seen to be
  // stale rather than trusted.
  measured?: string
}

/** The shelves, in the order the room draws them. */
export const SHELF_GROUPS = ['search', 'code', 'apps', 'local'] as const
export type ShelfGroup = (typeof SHELF_GROUPS)[number]

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
// *From 13 ก.ย.:* **one exception to the sharper half, and it is written as
// an exception rather than a new rule.** google-workspace (the last row)
// needs the person to make an OAuth client in Google Cloud Console before
// the sign-in can happen — exactly the "register an app with the provider
// first" step the paragraph above keeps off the shelf. It is on anyway,
// because the owner asked for Google Drive by name, Google's own server
// cannot organise a Drive (see the row), and the only program that can
// takes its client from the person. What keeps it honest is the form: its
// two blanks open before anything is saved (needsPaste, the same door
// github's pasted token goes through), and the hint under them says where
// the values come from. A second row of this shape should make somebody
// reread this paragraph, not lean on it.
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
  { name: 'kinocut', group: 'local', proven: true, desc: 'Cut, subtitle and render video, on this machine', why: 'Aetox reads video and produces none. This is the half that cuts and renders. Install it from ห้องงานวิดีโอ, which fetches it the same way ffmpeg and Tesseract are fetched; this entry is the connection.', command: [] },
  { name: 'github', group: 'code', proven: true, desc: 'Repos, pull requests, issues, CI', why: "Aetox's own github tool only reads. This is the half that acts — opening a pull request, commenting, moving an issue.", url: 'https://api.githubcopilot.com/mcp/', headers: ['Authorization: Bearer ${connect:github}'] },
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
  // 3 tools and ~2.6k tokens is what the endpoint serves with NO key (scrape,
  // search, parse). The owner's keyed connection reports 25, so the number on
  // the card is the floor a fresh add pays, not the ceiling.
  { name: 'firecrawl', group: 'search', proven: true, toolCount: 3, tokens: 2558, measured: '2026-09-12', desc: 'Crawl a whole site, and multi-source research', why: 'Aetox reads one page at a time and gets eight search results. This walks a whole site and researches across many sources at once.', url: 'https://mcp.firecrawl.dev/v2/mcp' },
  { name: 'context7', group: 'code', proven: true, toolCount: 2, tokens: 1184, measured: '2026-09-12', desc: 'Up-to-date docs for a library, by version', why: 'Docs for the version actually installed. Fetching a documentation page cannot tell you which release it describes.', url: 'https://mcp.context7.com/mcp' },
  { name: 'deepwiki', group: 'code', proven: true, toolCount: 3, tokens: 314, measured: '2026-09-12', desc: 'Ask questions about any public GitHub repository', why: 'Answers about a repository without cloning it first — reading one that size through file tools costs a whole context.', url: 'https://mcp.deepwiki.com/mcp' },
  { name: 'exa', group: 'search', proven: true, toolCount: 2, tokens: 553, measured: '2026-09-12', desc: 'Web search built for models to read', why: 'Results returned as text to read rather than as pages to open, so an answer costs one call instead of a search and five fetches.', url: 'https://mcp.exa.ai/mcp' },
  { name: 'huggingface', group: 'search', proven: true, toolCount: 4, tokens: 1929, measured: '2026-09-12', desc: 'Search models, datasets and spaces', why: 'Aetox has no index of models, datasets or spaces, and a web search finds blog posts about them rather than the things.', url: 'https://huggingface.co/mcp' },
  { name: 'cloudflare-docs', group: 'search', proven: true, toolCount: 2, tokens: 294, measured: '2026-09-12', desc: "Search Cloudflare's documentation", why: "Cloudflare's own index of its own docs, which is a different thing from a web search that happens to land there.", url: 'https://docs.mcp.cloudflare.com/mcp' },
  // Added 2026-09-05 (owner: "ใส่ ... ขึ้นชั้นเลยครับ"), from the second
  // research pass recorded in mcpCandidates.ts. These two are the
  // cloudflare-docs shape again — a vendor's own index of its own docs — and
  // both answered a bare `initialize` with no credential of any kind. The
  // `why` on each was written after real calls the same day, not from the
  // tool descriptions: microsoft_docs_search asked about hidden files in
  // Get-ChildItem came back with the about_FileSystem_Provider chunk that
  // names the -Hidden parameter; aws___search_documentation asked about
  // presigned URL expiry came back with the verbatim page section (1 minute
  // to 12 hours in the console, up to 7 days from the SDK).
  { name: 'microsoft-learn', group: 'search', proven: true, toolCount: 3, tokens: 1016, measured: '2026-09-12', desc: "Microsoft's own docs: Windows, Azure, .NET, PowerShell, Office", why: "Microsoft's own index of its own docs, which is a different thing from a web search that happens to land there — and a code-sample search web_search cannot do at all. Asked for Get-ChildItem hidden files it returned the exact provider page naming the parameter, not a forum thread about it.", url: 'https://learn.microsoft.com/api/mcp' },
  { name: 'aws-knowledge', group: 'search', proven: true, toolCount: 5, tokens: 1978, measured: '2026-09-12', desc: "AWS's own docs, plus regions and per-region service availability", why: "AWS's own docs index, returning the verbatim page section rather than a snippet, so the answer is usually already in the result. Two things no web search returns as data sit beside it: the region list and which services exist in which region.", url: 'https://knowledge-mcp.global.api.aws' },
  // zapier answers 401 "Expected Bearer token for MCP authentication", and
  // there are two ways to get one: paste the token from the user's own Zapier
  // MCP page, or the sign-in — its authorization server publishes a
  // registration_endpoint (probed 2026-09-05), so it is the semgrep shape and
  // takes the same `${connect:...}` path. It is written as the sign-in rather
  // than the paste for the reason capabilityRoom.test.ts pins: every
  // non-oauth preset must be one click, and a bare `Authorization: Bearer`
  // is a form. The paste still works through แก้ไข for someone who has the
  // token already. `why` is the same placeholder as the rows below it.
  { name: 'zapier', group: 'apps', proven: false, desc: 'Run actions in ~8000 apps connected through Zapier', why: '', url: 'https://mcp.zapier.com/api/mcp/mcp', headers: ['Authorization: Bearer ${connect:zapier}'], oauth: true },

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
  { name: 'semgrep', group: 'apps', proven: false, desc: 'Scan code for security vulnerabilities', why: '', url: 'https://mcp.semgrep.ai/mcp', headers: ['Authorization: Bearer ${connect:semgrep}'], oauth: true },
  { name: 'grafana', group: 'apps', proven: false, desc: 'Query dashboards, metrics and alerts', why: '', url: 'https://mcp.grafana.com/mcp', headers: ['Authorization: Bearer ${connect:grafana}'], oauth: true },
  { name: 'netlify', group: 'apps', proven: false, desc: 'Deploy and manage a hosted site', why: '', url: 'https://netlify-mcp.netlify.app/mcp', headers: ['Authorization: Bearer ${connect:netlify}'], oauth: true },

  // Added 2026-09-03, separately: notion was one of the four servers this
  // file's own history above names as "blocked by rule 2 until the client
  // learns OAuth" (14 ส.ค.) — checked again the same way as the three above
  // now that mcpauth.go exists, and it answers a real registration_endpoint
  // too (see mcpCandidates.ts for the discovery trail). Same placeholder
  // rule applies: delete this note and write a real `why` once someone has
  // actually signed in.
  { name: 'notion', group: 'apps', proven: false, desc: 'Search, read and write pages in a workspace', why: '', url: 'https://mcp.notion.com/mcp', headers: ['Authorization: Bearer ${connect:notion}'], oauth: true },
  // Added 2026-09-05 with the two docs servers above: the first two of the
  // twenty oauth-dcr rows from the second research pass in mcpCandidates.ts,
  // picked by the owner. Both answered 401 with a resource_metadata pointer
  // and both authorization servers publish a registration_endpoint, so the
  // sign-in path is the one semgrep already walks. Same placeholder rule.
  { name: 'supabase', group: 'apps', proven: false, desc: 'Run SQL and manage tables, functions and logs in your Supabase projects', why: '', url: 'https://mcp.supabase.com/mcp', headers: ['Authorization: Bearer ${connect:supabase}'], oauth: true },
  { name: 'canva', group: 'apps', proven: false, desc: 'Create and export designs in your Canva account', why: '', url: 'https://mcp.canva.com/mcp', headers: ['Authorization: Bearer ${connect:canva}'], oauth: true },

  // ---- Five more of the sign-in shape, 2026-09-13 ----
  //
  // The vendors' own remote servers, chosen because each reaches something
  // Aetox has no tool for and each signs in through the browser with no
  // setup (internal/oauth/mcpauth.go). Every one was probed the same day:
  // 401 with an OAuth challenge, a real authorization server, PKCE S256, and
  // a registration_endpoint — Figma's at api.figma.com/v1/oauth/mcp/register,
  // Sentry's at mcp.sentry.dev/oauth/register, Stripe's at
  // access.stripe.com/mcp/oauth2/register, Vercel's at
  // api.vercel.com/login/oauth/register (it had none on 2026-09-03; it does
  // now), Atlassian's at mcp.atlassian.com/v1/register. Three of the five
  // needed mcpauth.go taught a shape it had not met (POST-only challenge,
  // unquoted resource_metadata, no protected-resource document at all);
  // mcpauth_test.go carries each one. `proven` stays false and `why` empty
  // by the same rule as the rows above: written after a real sign-in.
  //
  // What the room says over these rows changed the same day. It used to say
  // "ยังไม่ได้ลองจริง" — true, and read by the owner as "may not work".
  // The fact that separates these from the rows above is that they sign in
  // through the user's own account, so that is the band's name now
  // (capability.bandSignIn); `proven` is still the data.
  { name: 'figma', group: 'apps', proven: false, desc: 'Read the layers, components and variables of a Figma file', why: '', url: 'https://mcp.figma.com/mcp', headers: ['Authorization: Bearer ${connect:figma}'], oauth: true },
  { name: 'sentry', group: 'code', proven: false, desc: 'Errors, stack traces and issues from your Sentry projects', why: '', url: 'https://mcp.sentry.dev/mcp', headers: ['Authorization: Bearer ${connect:sentry}'], oauth: true },
  { name: 'stripe', group: 'apps', proven: false, desc: 'Customers, payments and invoices in your Stripe account', why: '', url: 'https://mcp.stripe.com', headers: ['Authorization: Bearer ${connect:stripe}'], oauth: true },
  { name: 'vercel', group: 'code', proven: false, desc: 'Deployments, build logs and projects on Vercel', why: '', url: 'https://mcp.vercel.com', headers: ['Authorization: Bearer ${connect:vercel}'], oauth: true },
  { name: 'atlassian', group: 'apps', proven: false, desc: 'Jira issues and Confluence pages in your Atlassian site', why: '', url: 'https://mcp.atlassian.com/v1/mcp', headers: ['Authorization: Bearer ${connect:atlassian}'], oauth: true },

  // ---- Three more of the sign-in shape, 2026-09-13, second pass ----
  //
  // A targeted sweep this time rather than a general one: the owner asked
  // for candidates split by who Aetox is for (ordinary users, creators who
  // edit video, developers, business users) rather than whatever answered.
  // These three each fill a gap none of the rows above touch and each
  // registered a client on the spot when probed the same day — Runway's at
  // mcp.runwayml.com/register, DeepL's at mcp.deepl.com/.idp/register,
  // Mapbox's at api.mapbox.com/oauth/register. Same placeholder rule as
  // every row above: `proven` stays false and `why` stays empty until
  // someone has actually signed in and used it.
  //
  // Attio (CRM, same sweep, same DCR shape at app.attio.com/oauth/register)
  // was held back from this batch on purpose: it has no published vector
  // logo anywhere findable (favicon, apple-icon, brand page all 404), and
  // McpMark's rule for a mark with nothing to draw is to say so in noMark
  // rather than invent a monogram (see mcpMarks.ts's deepwiki note on the
  // same question). It sits in mcpCandidates.ts instead until a real mark
  // turns up. Replicate, Xero and Smartsheet all answered live and 401 the
  // same session but the OAuth discovery chain wasn't walked to the end
  // before the research pass was cut short — also in mcpCandidates.ts,
  // marked unfinished rather than guessed at.
  { name: 'runwayml', group: 'apps', proven: false, desc: 'Generate and edit video and images with AI models', why: '', url: 'https://mcp.runwayml.com/mcp', headers: ['Authorization: Bearer ${connect:runwayml}'], oauth: true },
  { name: 'deepl', group: 'apps', proven: false, desc: 'Translate text with DeepL', why: '', url: 'https://mcp.deepl.com/v1/mcp', headers: ['Authorization: Bearer ${connect:deepl}'], oauth: true },
  { name: 'mapbox', group: 'apps', proven: false, desc: 'Maps, geocoding, routing and map styles', why: '', url: 'https://mcp.mapbox.com/mcp', headers: ['Authorization: Bearer ${connect:mapbox}'], oauth: true },

  // ---- google-workspace, 2026-09-13 ----
  //
  // The owner's ask was "จัดระเบียบ Google Drive" and then "เชื่อมตัวอื่นๆ
  // ด้วย". Google's own remote Drive server
  // (drivemcp.googleapis.com/mcp/v1, checked the same day) cannot do the
  // first: it is in developer preview with search/read/create/copy and no
  // move, rename or delete, and it wants a client id Google issued to the
  // app — no registration_endpoint, the elevenlabs shape mcpauth.go says
  // it cannot reach. So this is taylorwilsdon/google_workspace_mcp,
  // spawned with uvx, which carries full Drive plus Gmail, Calendar, Docs
  // and Sheets in one process. It is one row and not five because it IS
  // one process: five rows would be the same program running five times,
  // each paying its own sign-in.
  //
  // The two blanks are the OAuth client the person makes in Google Cloud
  // Console (hint below says where); the program opens the browser for
  // the Google sign-in itself, on its own loopback port, so Aetox's OAuth
  // is not in the picture. USER_GOOGLE_EMAIL is the account the tools act
  // as; left blank, the model asks. OAUTHLIB_INSECURE_TRANSPORT is what
  // lets its http://localhost callback through the Python OAuth library —
  // it reaches nothing but that one process. The service list is the
  // command line, not an allowlist: fewer services is fewer tools on every
  // message, and the เครื่องมือ tab can trim further after a first connect.
  //
  // `proven` false, `why` empty, by the rule above every row: written after
  // a real sign-in against a real Drive, which needs a client only the owner
  // can make.
  { name: 'google-workspace', group: 'apps', proven: false, desc: 'Google Drive, Gmail, Calendar, Docs and Sheets: organise, search, write', why: '', command: ['uvx', 'workspace-mcp', '--tools', 'drive', 'gmail', 'calendar', 'gdocs', 'gsheets'], env: ['GOOGLE_OAUTH_CLIENT_ID=', 'GOOGLE_OAUTH_CLIENT_SECRET=', 'USER_GOOGLE_EMAIL=', 'OAUTHLIB_INSECURE_TRANSPORT=1'], hint: 'capability.hintGoogleWorkspace' },
]

/** A stdio preset with no command written in the table is the one that has to
 *  ask Go for both. */
export const isLocalPreset = (p: MCPPreset): boolean =>
  (p.command?.length ?? 0) === 0 && !p.url

/** A header entry that already carries a ${...} reference needs nothing from
 *  the user: the value resolves at connect time from a secret the app already
 *  holds. Only a header still waiting for a paste opens the form — or an env
 *  line with nothing after its `=`, which is the stdio spelling of the same
 *  blank. */
export const needsPaste = (headers?: string[], env?: string[]): boolean =>
  (headers ?? []).some((h) => !/\$\{(env|connect):[^}]+\}/.test(h)) ||
  (env ?? []).some((e) => e.slice(e.indexOf('=') + 1).trim() === '')

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
//
// Every other stdio preset spells its environment in the table, and only the
// lines that already carry a value are saved: a blank one is the form's job
// (needsPaste), and saving `KEY=` would hand the program an empty string
// where it expects nothing at all.
export const presetEnvironment = async (p: MCPPreset): Promise<Record<string, string>> =>
  isLocalPreset(p) ? await VideoEditorEnvironment() : envOf(p.env)

const envOf = (lines?: string[]): Record<string, string> =>
  Object.fromEntries(
    (lines ?? [])
      .map((e) => [e.slice(0, e.indexOf('=')).trim(), e.slice(e.indexOf('=') + 1).trim()])
      .filter(([k, v]) => k !== '' && v !== ''),
  )

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
  MCP_PRESETS.find((p) => p.name.toLowerCase() === id.toLowerCase() && !needsPaste(p.headers, p.env) && !p.oauth)
