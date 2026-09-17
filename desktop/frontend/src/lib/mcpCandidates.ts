// MCP server candidates — verified reachable, not yet on the shelf.
//
// This is NOT mcpShelf.ts. Nothing here is imported by Settings.svelte or
// Capability.svelte, and nothing here can be added with one click. It exists
// because mcpShelf.ts's own charter draws a hard line this file is built
// around: `why` on a real preset has to be something the owner wrote after
// trying the thing themselves, and an agent writing it instead makes it a
// different thing immediately. So the research that produced this list — real
// `initialize` calls, real 401 bodies, real tool listings — stops here, one
// step short of the shelf, until a person tries each row and writes `why`.
//
// Promoting a row: read it, try it, then add it to MCP_PRESETS in
// mcpShelf.ts with a `why` you wrote yourself. Delete the row here once it's
// promoted — this file is a waiting room, not a second catalog.
//
// Every row was verified (2026-09-03 for the first pass, 2026-09-05 for the
// second — each row carries its own date) by firing a real MCP `initialize` (and,
// where reachable without a key, a real `tools/list`) at the URL in `source`.
// `evidence` is the literal response, trimmed. Nothing here was taken from a
// vendor's docs page or a directory site's claim — see the `semgrep` row
// below for why that distinction mattered in practice.

export type CandidateAuth = 'none' | 'static-header' | 'oauth-dcr' | 'oauth-manual'

export interface MCPCandidate {
  id: string
  category: string
  source: string
  auth: CandidateAuth
  // What Aetox has no tool for. Never `why` — this is the gap/overlap
  // research half, the part an agent is allowed to write.
  gap: string
  overlaps: string[]
  verifiedAt: string
  toolCount: number | null
  evidence: string
}

export const MCP_CANDIDATES: MCPCandidate[] = [
  // ── auth: none — real 200, no header of any kind ──────────────────────
  {
    id: 'twilio-docs',
    category: 'vendor API/docs search',
    source: 'https://mcp.twilio.com/docs',
    auth: 'none',
    gap: 'web_search is generic DuckDuckGo scraping and context7 (already on the shelf) covers library docs by version — neither indexes Twilio\'s own API operation schemas the way this does.',
    overlaps: ['web_search (internal/skill/web_search.go) — general, not API-spec-aware', 'context7 (already on shelf) — general library docs, not a vendor API reference'],
    verifiedAt: '2026-09-03',
    toolCount: 2,
    evidence: 'POST /docs (no headers) -> 200 {"id":1,"jsonrpc":"2.0","result":{"capabilities":{"tools":{}},"protocolVersion":"2025-06-18","serverInfo":{"name":"twilio-docs-mcp","version":"0.1.0"}}} — tools/list: twilio__search, twilio__retrieve.',
  },
  {
    id: 'google-maps',
    category: 'maps / places / routing / location weather',
    source: 'https://mapstools.googleapis.com/mcp',
    auth: 'none',
    gap: 'Aetox has no geocoding, routing, place-search, or location-based weather tool at all.',
    overlaps: [],
    verifiedAt: '2026-09-03',
    toolCount: 5,
    evidence: 'POST /mcp (no headers) -> 200 {"id":1,"jsonrpc":"2.0","result":{"capabilities":{"tools":{"listChanged":false}},"protocolVersion":"2025-06-18","serverInfo":{"name":"StatelessServer"}}} — tools/list: search_places, resolve_names, resolve_maps_urls, compute_routes, lookup_weather.',
  },
  {
    id: 'socket',
    category: 'supply-chain / dependency security',
    source: 'https://mcp.socket.dev/',
    auth: 'none',
    gap: 'Aetox has no dependency vulnerability / supply-chain scanner at all. Docs claimed OAuth-on-first-connect; the live probe shows the general-purpose tools work with zero auth (only the org-scoped alerts/threat_feed tools would need an account).',
    overlaps: [],
    verifiedAt: '2026-09-03',
    toolCount: 6,
    evidence: 'POST / (no headers) -> 200 {"result":{"protocolVersion":"2025-06-18","capabilities":{"tools":{}},"serverInfo":{"name":"socket","version":"0.0.20"}}} — tools/list: depscore, organizations, alerts, threat_feed, package_files, package_file_contents, package_file_grep (first 4 general-purpose, last 3 need org_slug).',
  },

  // ── auth: static-header — real 401 naming a Bearer/key header, vendor
  //    docs confirm a static header is a first-class alternative to OAuth ──
  // stripe was here (static-header, https://mcp.stripe.com/) — it was
  // already on MCP_PRESETS in mcpShelf.ts (the 14 ส.ค. static-key addition,
  // "a one-line addition whenever it is wanted") the whole time this row
  // sat unpromoted beside it. Deleted per the rule at the top of this file;
  // found during the 2026-09-13 backlog triage.
  {
    id: 'fal',
    category: 'image / video / audio generation',
    source: 'https://mcp.fal.ai/mcp',
    auth: 'static-header',
    gap: 'picture.go, image_ocr.go, video_ocr.go, and audio_transcribe.go all read media that already exists — nothing in Aetox generates new images, video, or audio.',
    overlaps: ['picture (internal/skill/picture.go) — reads local files only, never generates'],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401, header Www-Authenticate: Bearer resource_metadata="https://mcp.fal.ai/.well-known/oauth-protected-resource/mcp", body {"error":"Authentication required"} — fal.ai docs give the static header as the primary (not fallback) setup: --header "Authorization: Bearer $FAL_KEY".',
  },
  {
    id: 'resend',
    category: 'email sending',
    source: 'https://mcp.resend.com/mcp',
    auth: 'static-header',
    gap: 'Nothing in Aetox sends an email — doc_write.go produces a local file, nothing transmits it.',
    overlaps: [],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401 {"jsonrpc":"2.0","error":{"code":-32000,"message":"Unauthorized: provide credentials via Authorization: Bearer <token>"},"id":null} — header format stated directly in the error body.',
  },
  {
    id: 'posthog',
    category: 'product analytics',
    source: 'https://mcp.posthog.com/mcp',
    auth: 'static-header',
    gap: 'Aetox has no analytics/telemetry tool — nothing reads events, funnels, or session data from a running product.',
    overlaps: [],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: '401, header WWW-Authenticate: Bearer resource_metadata=..., body "No token provided, please provide a valid API token. ... https://posthog.com/docs/model-context-protocol" — docs confirm Authorization: Bearer <personal API key> as an OAuth alternative.',
  },
  {
    id: 'cloudinary',
    category: 'media hosting/CDN & transformation',
    source: 'https://asset-management.mcp.cloudinary.com/mcp',
    auth: 'static-header',
    gap: 'picture.go only reads a file already on disk — Aetox has no way to upload media to a CDN and get a hosted URL back, or transform it (resize/crop/convert) on the fly.',
    overlaps: ['picture (internal/skill/picture.go) — local read only, no upload/host/transform'],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: '401 {"error":"unauthorized","error_description":"Authentication required. Provide either OAuth Bearer token or API key headers.",...} — states the static-header alternative directly.',
  },
  {
    id: 'airtable',
    category: 'hosted spreadsheet-database',
    source: 'https://mcp.airtable.com/mcp',
    auth: 'static-header',
    gap: 'sheet_write.go produces a one-shot local .xlsx — nothing in Aetox talks to a live, shared, queryable database the way Airtable does.',
    overlaps: ['sheet_write (internal/skill/sheet_write.go) — writes a local file once, no sync to a shared base'],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: '401, header www-authenticate: Bearer error="invalid_token", error_description="Missing Authorization header", body {"error":"UNAUTHORIZED","message":"Unauthorized"} — names the missing header directly.',
  },
  {
    id: 'neon',
    category: 'serverless Postgres database',
    source: 'https://mcp.neon.tech/mcp',
    auth: 'static-header',
    gap: 'Aetox has no database provisioning or SQL-execution tool of any kind.',
    overlaps: [],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: '401, header Www-Authenticate: Bearer error="invalid_token", error_description="No authorization provided" — Neon docs confirm header auth is meant "where OAuth is not available": headers.Authorization="Bearer <NEON_API_KEY>".',
  },
  {
    id: 'currencyapi',
    category: 'currency exchange rates',
    source: 'https://api.currencyapi.com/mcp',
    auth: 'static-header',
    gap: 'calc.go computes only with numbers the model already has — nothing fetches a live FX rate.',
    overlaps: ['calc (internal/skill/calc.go) — arithmetic only, no external rate data'],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: '401 www-authenticate: Key, body {"message":"No API key found in request",...} — the provider\'s OpenAPI spec names the header "apikey" as the recommended (not fallback) auth method, a query param is the only alternative.',
  },

  // ── auth: oauth-dcr — OAuth-only, but the authorization server supports
  //    RFC 7591 Dynamic Client Registration, so Aetox can register itself
  //    on the fly and needs no pre-arranged client_id. This is what
  //    internal/oauth/mcpauth.go's generic flow was built for.
  //
  //    semgrep, grafana and netlify — the first three found this way — are
  //    already promoted to MCP_PRESETS in mcpShelf.ts and deleted from here.
  //    notion joined them the same day, moved over to actually be tried
  //    rather than left waiting — see mcpShelf.ts for its `why` once
  //    someone has signed in for real and written it. Sentry and figma,
  //    below on 2026-09-03, were promoted the same way on 2026-09-13 and
  //    are deleted from here too (found during that day's backlog triage).
  //    Linear joined the shelf on 2026-09-17 after a fresh DCR probe. The
  //    remaining rows below are the servers still blocked or awaiting a
  //    product decision. ─────────────────────────────────────────────────

  // ── auth: oauth-manual — OAuth-only, and the authorization server has NO
  //    registration_endpoint, so Aetox cannot register a client on the fly.
  //    Connecting would require Aetox to register a fixed OAuth app with
  //    each vendor by hand (a business step, not a coding one) — out of
  //    scope for internal/oauth/mcpauth.go's generic flow. Listed so the
  //    finding isn't lost, not because a click connects them today. ───────
  // atlassian was here (oauth-manual, no registration_endpoint found on
  // 2026-09-03) — the owner added it to MCP_PRESETS anyway on 2026-09-13
  // (mcp.atlassian.com/v1/register does answer with a registration_endpoint
  // as of that date; the manual finding above was simply overtaken by the
  // vendor shipping DCR since). Deleted from here per the rule at the top
  // of this file; found during the 2026-09-13 backlog triage.
  {
    id: 'pagerduty',
    category: 'incident management / on-call',
    source: 'https://mcp.pagerduty.com/mcp',
    auth: 'oauth-manual',
    gap: 'Aetox has no incident-management tool that touches a live on-call system — nothing reads or manages a real incident, schedule, or escalation. Pairs with the incident-response half of aetox-deploy, which is discipline text, not a live connection.',
    overlaps: [],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401 {"error":"invalid_token","error_description":"Authentication required: the token is invalid or has expired."}, header www-authenticate naming resource_metadata="https://mcp.pagerduty.com/.well-known/oauth-protected-resource/mcp". Discovery: authorization_servers=["https://mcp.pagerduty.com/"] -> AS metadata (issuer app.pagerduty.com) has NO registration_endpoint — DCR not supported, needs a pre-registered client_id from PagerDuty.',
  },
  {
    id: 'slack',
    category: 'team chat / messaging',
    source: 'https://mcp.slack.com/mcp',
    auth: 'oauth-manual',
    gap: 'Aetox has no tool that reaches a live Slack workspace — nothing reads or posts a message, searches a channel, or reads a file shared there.',
    overlaps: [],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401, header www-authenticate: Bearer resource_metadata="https://mcp.slack.com/.well-known/oauth-protected-resource". Discovery: AS metadata at https://mcp.slack.com/.well-known/oauth-authorization-server has NO registration_endpoint — DCR not supported, needs a pre-registered client_id from Slack.',
  },
  {
    id: 'elevenlabs',
    category: 'text-to-speech / voice generation',
    source: 'https://api.elevenlabs.io/v1/mcp',
    auth: 'oauth-manual',
    gap: 'audio_transcribe.go only goes speech-to-text — Aetox has no text-to-speech tool.',
    overlaps: ['audio_transcribe (internal/skill/audio_transcribe.go) — opposite direction (STT not TTS), not a real overlap'],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: '401, header www-authenticate: Bearer resource_metadata="https://api.us.elevenlabs.io/.well-known/oauth-protected-resource", body {"detail":"OAuth bearer token required for the hosted MCP."}. Discovery: AS metadata at https://api.us.elevenlabs.io/.well-known/oauth-authorization-server has authorization_endpoint and token_endpoint but NO registration_endpoint — DCR not supported, needs a pre-registered client_id from ElevenLabs.',
  },
  // vercel was here (oauth-manual, no registration_endpoint found on
  // 2026-09-03) — the shelf history for the 2026-09-13 sign-in rows says
  // "Vercel's at api.vercel.com/login/oauth/register (it had none on
  // 2026-09-03; it does now)", so it was promoted the same way once the
  // vendor shipped DCR. Deleted from here per the rule at the top of this
  // file; found during the 2026-09-13 backlog triage.
  {
    id: 'shopify',
    category: 'e-commerce / store management',
    source: 'https://setup.shopify.com/mcp',
    auth: 'oauth-manual',
    gap: 'Aetox has no e-commerce/store-management tool.',
    overlaps: [],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: 'The MCP endpoint itself answers 403 (ambiguous — could be a bot check, not OAuth, per the shelf charter\'s own Cloudflare lesson about reading a 403 twice). The resource-metadata endpoint answers cleanly though: https://setup.shopify.com/.well-known/oauth-protected-resource -> 200, authorization_servers=["https://setup.shopify.com/auth"]. Discovery: AS metadata at https://setup.shopify.com/.well-known/oauth-authorization-server/auth has token_endpoint_auth_methods_supported=["none"] but NO registration_endpoint — DCR not supported, needs a pre-registered client_id from Shopify.',
  },

  // ── Second pass, 2026-09-05. Same method as above — a real `initialize`
  //    from a browser user agent, then for every 401 the RFC 9728 → RFC 8414
  //    walk to see whether the authorization server has a
  //    registration_endpoint. 57 endpoints probed; the rows below are the
  //    ones that both answered and reach something Aetox has no tool for.
  //    Five of them (microsoft-learn, aws-knowledge, zapier, supabase, canva)
  //    went straight to the shelf the same day on the owner's word and are
  //    deleted from here, per the rule at the top of this file.
  //
  //    Left out on purpose, so nobody re-probes them:
  //    - overlap with a tool already here (rule 1): tavily, brightdata,
  //      jina (22 tools, but search/read/screenshot is web_search + web_fetch
  //      + the browser), browser.mcp.cloudflare.com (a hosted browser).
  //    - mcp.devin.ai/mcp answers as serverInfo.name "DeepWiki" 2.14.3 —
  //      the same server as the deepwiki preset already on the shelf.
  //    - too narrow for a shelf a general user reads: kite (Zerodha, India
  //      only), mercadolibre, workos, stytch, observability.mcp.cloudflare.com
  //      (one account's Workers logs), astro-docs / resend-docs / neon-docs
  //      (one vendor's Mintlify docs each).
  //    - not one click by construction: pipedream (400 "external user id is
  //      required"), openrouter (307 to a docs page, no server).
  //    - no DNS at all on 2026-09-05: mcp.perplexity.ai, mcp.readme.com,
  //      mcp.coinbase.com, mcp.zerion.io, api.hub.docker.com.
  //    - stackblitz answered 403 with an HTML bot page — the Cloudflare
  //      lesson from the shelf charter, unread rather than refused. ─────────

  // ── auth: none ───────────────────────────────────────────────────────────

  // ── auth: static-header — 401 whose own body names an API token, so the
  //    header form works without a sign-in ─────────────────────────────────
  {
    id: 'apify',
    category: 'web scraping actors / datasets',
    source: 'https://mcp.apify.com',
    auth: 'static-header',
    gap: 'Thousands of ready-made scrapers (Google Maps listings, Instagram, Amazon, job boards) run as actors and return datasets. firecrawl walks a site; this runs a purpose-built extractor for a site and hands back rows.',
    overlaps: ['firecrawl (already on shelf) — crawl/scrape generic pages, not per-site extractors', 'web_fetch (internal/skill/web_fetch.go) — one page'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST / (no headers) -> 401 {"error":"invalid_token","error_description":"Missing or invalid access token. Pass an Apify API token in the Authorization: Bearer <token> header. Manage tokens at https://console.apify.com/account/integrations"} — the header format is in the error body. registration_endpoint also present at console-backend.apify.com.',
  },
  {
    id: 'postman',
    category: 'API collections / requests',
    source: 'https://mcp.postman.com/mcp',
    auth: 'static-header',
    gap: 'Reads and runs the user\'s own Postman collections and environments — a saved, authenticated API surface Aetox otherwise cannot see.',
    overlaps: ['web_fetch (internal/skill/web_fetch.go) — raw HTTP, no collections, no saved auth'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401 {"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request: API key required"}}, header WWW-Authenticate: Bearer resource_metadata="https://mcp.postman.com/.well-known/oauth-protected-resource/mcp" — body names an API key; DCR also available.',
  },
  {
    id: 'alphavantage',
    category: 'stock / forex / commodity market data',
    source: 'https://mcp.alphavantage.co/mcp',
    auth: 'static-header',
    gap: 'Equities, FX and commodities time series — the same gap coingecko on the shelf fills for crypto, for everything else. calc.go has no numbers of its own.',
    overlaps: ['coingecko (on shelf) — crypto only', 'currencyapi (above) — FX rates only'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401 {"error":"invalid_request","error_description":"Missing access token"}, header WWW-Authenticate: Bearer resource_metadata="https://mcp.alphavantage.co/.well-known/oauth-protected-resource/mcp". Alpha Vantage issues a free API key on its site; AS metadata also has a registration_endpoint.',
  },

  // ── auth: oauth-dcr — 401, and the authorization server named by the
  //    resource metadata has a registration_endpoint. Every one of these is
  //    what semgrep / grafana / netlify / notion were on 3 ก.ย.: one click
  //    with the browser sign-in, once someone has done that sign-in and
  //    written `why`. ──────────────────────────────────────────────────────
  {
    id: 'asana',
    category: 'project management / tasks',
    source: 'https://mcp.asana.com/sse',
    auth: 'oauth-dcr',
    gap: 'Aetox has no task or project tool at all — nothing reads or writes tasks, projects, or comments in a live workspace.',
    overlaps: ['linear (on shelf) — same category, different vendor'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST (no headers) -> 401, header WWW-Authenticate: Bearer realm="OAuth", error="invalid_token", error_description="Missing or invalid access token", resource_metadata="https://mcp.asana.com/.well-known/oauth-protected-resource". AS metadata at https://mcp.asana.com/.well-known/oauth-authorization-server has registration_endpoint. Note the path is /sse — the older transport — so internal/mcp/client.go\'s streamable-HTTP path needs checking against it before promotion.',
  },
  {
    id: 'monday',
    category: 'work management / boards',
    source: 'https://mcp.monday.com/mcp',
    auth: 'oauth-dcr',
    gap: 'Same gap as asana and linear — boards, items, updates in a live monday.com account — for the third vendor in that category.',
    overlaps: ['asana (above)', 'linear (on shelf)'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST (no headers) -> 401, header WWW-Authenticate: Bearer realm="OAuth", error="invalid_token", error_description="Missing or invalid access token", resource_metadata="https://mcp.monday.com/.well-known/oauth-protected-resource/mcp". AS metadata at https://auth.monday.com/.well-known/oauth-authorization-server/mcp has registration_endpoint.',
  },
  {
    id: 'paypal',
    category: 'payments (PayPal)',
    source: 'https://mcp.paypal.com/mcp',
    auth: 'oauth-dcr',
    gap: 'Aetox has no payment tool. Invoices, orders, subscriptions, disputes in a live PayPal business account.',
    overlaps: ['stripe (above) — same category, static key rather than sign-in'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST (no headers) -> 401, header WWW-Authenticate: Bearer realm="OAuth", error="invalid_token", error_description="Missing or invalid access token", resource_metadata="https://mcp.paypal.com/.well-known/oauth-protected-resource/mcp". AS metadata at https://mcp.paypal.com/.well-known/oauth-authorization-server has registration_endpoint.',
  },
  {
    id: 'square',
    category: 'payments / point of sale (Square)',
    source: 'https://mcp.squareup.com/sse',
    auth: 'oauth-dcr',
    gap: 'Same gap as paypal and stripe, for a seller whose money runs through Square — catalog, orders, payments, customers.',
    overlaps: ['stripe (above)', 'paypal (above)'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST (no headers) -> 401, header WWW-Authenticate: Bearer realm="OAuth", error="invalid_token", error_description="Missing or invalid access token", resource_metadata="https://mcp.squareup.com/.well-known/oauth-protected-resource/sse". AS metadata at https://mcp.squareup.com/.well-known/oauth-authorization-server has registration_endpoint. Path is /sse — same transport caveat as asana.',
  },
  {
    id: 'plaid',
    category: 'bank accounts / transactions (Plaid)',
    source: 'https://api.dashboard.plaid.com/mcp/sse',
    auth: 'oauth-dcr',
    gap: 'Nothing in Aetox reads a bank account. This is the Plaid dashboard — items, institutions, usage — not a consumer\'s statements, which is worth knowing before promoting.',
    overlaps: [],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST /mcp/sse (no headers) -> 401 {"error":"Unauthorized"}, no WWW-Authenticate. Default resource metadata at https://api.dashboard.plaid.com/.well-known/oauth-protected-resource answered, and the AS metadata at https://api.dashboard.plaid.com/.well-known/oauth-authorization-server has registration_endpoint. /sse transport caveat as asana.',
  },
  {
    id: 'webflow',
    category: 'website builder / CMS (Webflow)',
    source: 'https://mcp.webflow.com/sse',
    auth: 'oauth-dcr',
    gap: 'Aetox writes local HTML files (the web-templates skill) and publishes nothing — this edits pages, CMS items and styles on a live Webflow site.',
    overlaps: ['netlify (on shelf) — deploys a built site; webflow edits the site itself'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST (no headers) -> 401, header WWW-Authenticate: Bearer realm="OAuth", error="invalid_token", error_description="Missing or invalid access token", resource_metadata="https://mcp.webflow.com/.well-known/oauth-protected-resource/sse". AS metadata at https://mcp.webflow.com/.well-known/oauth-authorization-server has registration_endpoint. /sse transport caveat as asana.',
  },
  {
    id: 'wix',
    category: 'website builder (Wix)',
    source: 'https://mcp.wix.com/mcp',
    auth: 'oauth-dcr',
    gap: 'Same gap as webflow for the other builder: reads and edits a live Wix site, its store and bookings.',
    overlaps: ['webflow (above)'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST (no headers) -> 401, header WWW-Authenticate: Bearer realm="OAuth", error="invalid_token", error_description="Missing or invalid access token", resource_metadata="https://mcp.wix.com/.well-known/oauth-protected-resource/mcp". AS metadata at https://mcp.wix.com/.well-known/oauth-authorization-server has registration_endpoint.',
  },
  {
    id: 'sanity',
    category: 'headless CMS (Sanity)',
    source: 'https://mcp.sanity.io',
    auth: 'oauth-dcr',
    gap: 'Aetox has no CMS tool — nothing queries or writes documents in a live content store.',
    overlaps: ['webflow (above) — CMS half only'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST / (no headers) -> 401 {"error":"invalid_token","error_description":"Missing Authorization header"}, header WWW-Authenticate naming resource_metadata="https://mcp.sanity.io/.well-known/oauth-protected-resource". AS metadata at https://mcp.sanity.io/.well-known/oauth-authorization-server has registration_endpoint.',
  },
  {
    id: 'intercom',
    category: 'customer support / conversations (Intercom)',
    source: 'https://mcp.intercom.com/mcp',
    auth: 'oauth-dcr',
    gap: 'Nothing in Aetox reads a support inbox — conversations, contacts, tickets in a live Intercom workspace.',
    overlaps: [],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST (no headers) -> 401, header WWW-Authenticate: Bearer realm="OAuth", error="invalid_token", error_description="Missing or invalid access token". Default resource metadata at https://mcp.intercom.com/.well-known/oauth-protected-resource answered; AS metadata at https://mcp.intercom.com/.well-known/oauth-authorization-server has registration_endpoint.',
  },
  {
    id: 'honeycomb',
    category: 'observability / traces (Honeycomb)',
    source: 'https://mcp.honeycomb.io/mcp',
    auth: 'oauth-dcr',
    gap: 'Same gap grafana fills on the shelf, for a team whose telemetry lives in Honeycomb — queries, triggers, SLOs against real production data.',
    overlaps: ['grafana (on shelf) — same category, different vendor', 'sentry (above) — errors rather than traces'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401, empty body, header WWW-Authenticate: Bearer resource_metadata="https://mcp.honeycomb.io/.well-known/oauth-protected-resource". AS metadata at https://ui.honeycomb.io/.well-known/oauth-authorization-server has registration_endpoint.',
  },
  {
    id: 'jam',
    category: 'bug reports with recordings (Jam)',
    source: 'https://mcp.jam.dev/mcp',
    auth: 'oauth-dcr',
    gap: 'A Jam is a screen recording plus console, network and device data — a bug report a person made. Aetox has no way to open one; this reads them.',
    overlaps: ['sentry (above) — errors the code raised, not what a person saw'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401 {"error":"invalid_token","error_description":"Access token is missing or invalid"}, header WWW-Authenticate: Bearer resource_metadata="https://mcp.jam.dev/.well-known/oauth-protected-resource", scope="mcp:read mcp:write". AS metadata at https://api.jam.dev/.well-known/oauth-authorization-server has registration_endpoint.',
  },
  {
    id: 'semrush',
    category: 'SEO / keyword and domain data (Semrush)',
    source: 'https://mcp.semrush.com/v1/mcp',
    auth: 'oauth-dcr',
    gap: 'Keyword volumes, backlinks, domain rankings — data web_search never returns as numbers. Aetox has nothing in this category.',
    overlaps: [],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST /v1/mcp (no headers) -> 401, empty body, header WWW-Authenticate: Bearer resource_metadata="https://mcp.semrush.com/.well-known/oauth-protected-resource". AS metadata at https://oauth.semrush.com/.well-known/oauth-authorization-server has registration_endpoint.',
  },
  {
    id: 'algolia',
    category: 'hosted search index management (Algolia)',
    source: 'https://mcp.algolia.com/mcp',
    auth: 'oauth-dcr',
    gap: 'Manages the user\'s own Algolia indices — records, settings, analytics. Nothing in Aetox touches a search index.',
    overlaps: [],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401 {"status":401,"detail":"Unauthorized: Missing or invalid Authorization header","code":"UNAUTHORIZED"}, header WWW-Authenticate: Bearer resource_metadata="https://mcp.algolia.com/.well-known/oauth-protected-resource", scope="public". AS metadata at https://dashboard.algolia.com/.well-known/oauth-authorization-server has registration_endpoint.',
  },
  {
    id: 'globalping',
    category: 'network measurements from many locations (Globalping)',
    source: 'https://mcp.globalping.dev/mcp',
    auth: 'oauth-dcr',
    gap: 'ping, traceroute, DNS and HTTP checks run from probes around the world. Aetox can fetch a URL from this machine only; it cannot tell how a site answers from Singapore.',
    overlaps: ['web_fetch (internal/skill/web_fetch.go) — one vantage point, this one'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401, empty body, header WWW-Authenticate: Bearer realm="OAuth", resource_metadata="https://mcp.globalping.dev/.well-known/oauth-protected-resource/mcp", scope="measurements". AS metadata at https://mcp.globalping.dev/.well-known/oauth-authorization-server has registration_endpoint.',
  },
  {
    id: 'cloudflare-radar',
    category: 'internet traffic / outage / ranking data (Cloudflare Radar)',
    source: 'https://radar.mcp.cloudflare.com/mcp',
    auth: 'oauth-dcr',
    gap: 'Radar is Cloudflare\'s public view of the internet — traffic shifts, outages, domain rankings, attack trends — a data set nothing on the shelf reaches. Distinct from cloudflare-docs (reads docs) and cloudflare-api (one account).',
    overlaps: ['cloudflare-docs (on shelf) — docs only', 'cloudflare-api (on shelf) — the user\'s own account'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401, empty body, header WWW-Authenticate: Bearer realm="OAuth", resource_metadata="https://radar.mcp.cloudflare.com/.well-known/oauth-protected-resource/mcp". AS metadata at https://radar.mcp.cloudflare.com/.well-known/oauth-authorization-server has registration_endpoint.',
  },
  // ── auth: oauth-manual — no registration_endpoint, so not one click ──────
  {
    id: 'hubspot',
    category: 'CRM (HubSpot)',
    source: 'https://mcp.hubspot.com/anthropic',
    auth: 'oauth-manual',
    gap: 'Aetox has no CRM tool — contacts, companies, deals, tickets in a live HubSpot portal.',
    overlaps: ['intercom (above) — support side, not sales'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST /anthropic (no headers) -> 401, empty body, header WWW-Authenticate: Bearer resource_metadata="https://mcp.hubspot.com/.well-known/oauth-protected-resource". AS metadata at https://mcp.hubspot.com/.well-known/oauth-authorization-server has NO registration_endpoint — needs a pre-registered client_id from HubSpot.',
  },
  {
    id: 'box',
    category: 'file storage (Box)',
    source: 'https://mcp.box.com',
    auth: 'oauth-manual',
    gap: 'Aetox reads files on this machine only. This reaches a Box account — search, read, and AI-ask over files that live there.',
    overlaps: ['filesystem tools — local disk only'],
    verifiedAt: '2026-09-05',
    toolCount: null,
    evidence: 'POST / (no headers) -> 401, empty body, header WWW-Authenticate: Bearer realm="Service", error="invalid_request", error_description="The access token was not found.", resource_metadata="https://mcp.box.com/.well-known/oauth-protected-resource". AS metadata at https://api.box.com/.well-known/oauth-authorization-server has NO registration_endpoint.',
  },

  // ── Third pass, 2026-09-13. Split by who Aetox is for rather than run as
  //    one general sweep — the owner asked for candidates aimed at ordinary
  //    users, creators who edit video, developers, and business users, so
  //    a next pass can pick up where this one was cut short rather than
  //    re-cover ground already probed. Same method: a real `initialize`,
  //    then for a 401 the RFC 9728 -> RFC 8414 walk for a
  //    registration_endpoint. Runway ML, DeepL and Mapbox from this pass
  //    all found one and went straight to MCP_PRESETS in mcpShelf.ts as
  //    2026-09-13 sign-in rows instead of sitting here — see that file. ──

  // ── auth: oauth-dcr, held back from the shelf for a reason other than
  //    the auth chain ─────────────────────────────────────────────────────
  {
    id: 'attio',
    category: 'CRM',
    source: 'https://mcp.attio.com/mcp',
    auth: 'oauth-dcr',
    gap: 'Aetox has no CRM tool at all — nothing reads or writes contacts, deals, or notes in a live workspace, for the business-user audience this pass targeted.',
    overlaps: [],
    verifiedAt: '2026-09-13',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401 {"statusCode":401,"error":"Unauthorized","message":"The requesting user did not present an authentication token","status_code":401}, header www-authenticate: Bearer resource_metadata="https://mcp.attio.com/.well-known/oauth-protected-resource", scope="openid offline_access mcp". Discovery: protected-resource doc names authorization_servers=["https://app.attio.com"]; AS metadata at https://app.attio.com/.well-known/oauth-authorization-server has registration_endpoint="https://app.attio.com/oauth/register". DCR supported — this one clears rule 2 same as runwayml/deepl/mapbox. Held out of mcpShelf.ts on rule 3 of mcpMarks.ts instead: no vector logo found anywhere checked (attio.com/favicon.svg, /icon.svg, /apple-icon.svg, /logo.svg all 404; simple-icons and @lobehub/icons have never heard of it). Promote once a real mark turns up, or once someone decides a lettered tile is fine for this one and adds it to noMark in desktop/mcp_marks_test.go.',
  },

  // ── Probed live, 401, but the OAuth discovery chain was not finished —
  //    the research pass was cut short mid-probe. Recorded so the next
  //    pass starts the discovery walk rather than re-sending the same
  //    initialize call. None of these have a `gap`/`overlaps` write-up yet
  //    on purpose: that comes after the chain is known, per this file's own
  //    method. ───────────────────────────────────────────────────────────
  // Replicate, Xero and Smartsheet were completed in the 2026-09-17 pass and
  // promoted to MCP_PRESETS. Replicate now exposes DCR; Xero's official stdio
  // package takes its documented client id/secret; Smartsheet documents its
  // personal API-token header for custom MCP clients.
]
