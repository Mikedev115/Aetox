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
// Every row was verified 2026-09-03 by firing a real MCP `initialize` (and,
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
    id: 'coingecko',
    category: 'crypto/market data',
    source: 'https://mcp.api.coingecko.com/mcp',
    auth: 'none',
    gap: 'Aetox has no market/financial-data tool — calc.go only evaluates arithmetic the model already has numbers for; nothing fetches live crypto prices, market caps, or historical series.',
    overlaps: [],
    verifiedAt: '2026-09-03',
    toolCount: 2,
    evidence: 'POST /mcp (no headers) -> 200 {"result":{"protocolVersion":"2025-06-18","capabilities":{"tools":{},"logging":{}},"serverInfo":{"name":"coingecko_coingecko_typescript_api","version":"7.1.0"}}} — tools/list: search_docs, execute (runs TS against a pre-authenticated SDK client).',
  },
  {
    id: 'wolfram-cloud',
    category: 'computation / scientific knowledge',
    source: 'https://agenttools.wolfram.com/mcp',
    auth: 'none',
    gap: 'calc.go is a JS sandbox with no filesystem/network/real-world knowledge — arithmetic only, no units, no entities, no science/geography/history data. This reaches the actual Wolfram|Alpha engine and a Wolfram Language kernel.',
    overlaps: ['calc (internal/skill/calc.go) — conceptually both "compute", but calc.go cannot touch real-world data/units/entities at all'],
    verifiedAt: '2026-09-03',
    toolCount: 3,
    evidence: 'POST /mcp (no headers) -> 200 {"result":{"protocolVersion":"2025-03-26","capabilities":{"tools":{"listChanged":true}},"serverInfo":{"name":"Wolfram","version":"2026.08.31"}}} — tools/list: WolframContext, WolframLanguageEvaluator, WolframAlpha.',
  },
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
  {
    id: 'stripe',
    category: 'payments',
    source: 'https://mcp.stripe.com/',
    auth: 'static-header',
    gap: 'Aetox has no payment/billing tool of any kind.',
    overlaps: [],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: 'POST / (no headers) -> 401, header Www-Authenticate: Bearer resource_metadata=https://mcp.stripe.com/.well-known/oauth-protected-resource, body {"error":"Unauthorized. See https://docs.stripe.com/mcp for usage instructions."} — Stripe docs confirm a restricted API key works as a Bearer header, no OAuth required.',
  },
  {
    id: 'cloudflare-api',
    category: 'cloud infrastructure management (Cloudflare account)',
    source: 'https://mcp.cloudflare.com/mcp',
    auth: 'static-header',
    gap: 'The cloudflare-docs preset already on the shelf only searches documentation — it cannot read or change anything in the user\'s own Cloudflare account. This is the full account API (DNS, Workers, R2, Zero Trust, ...).',
    overlaps: ['cloudflare-docs (already on shelf) — docs search only, never touches a real account'],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401, header WWW-Authenticate: Bearer realm="OAuth", resource_metadata="https://mcp.cloudflare.com/.well-known/oauth-protected-resource/mcp" — Cloudflare docs confirm a scoped API token works as a Bearer header alongside OAuth.',
  },
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
  //    someone has signed in for real and written it.
  //    The three below are the rest of that research pass: the servers
  //    mcpShelf.ts's own history names as "blocked by rule 2 until the
  //    client learns OAuth" (Linear, Sentry, Atlassian) plus three more
  //    checked the same way (Figma, PagerDuty, Slack) — verified
  //    2026-09-03, after the DCR flow existed, not before. ─────────────────
  {
    id: 'linear',
    category: 'project management / issue tracking',
    source: 'https://mcp.linear.app/mcp',
    auth: 'oauth-dcr',
    gap: 'Aetox has no project-management/issue-tracker tool at all — nothing reads or writes issues, projects, or comments in a live workspace.',
    overlaps: [],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401, header WWW-Authenticate: Bearer realm="OAuth", resource_metadata="https://mcp.linear.app/.well-known/oauth-protected-resource/mcp". Discovery: AS metadata at https://mcp.linear.app/.well-known/oauth-authorization-server has registration_endpoint="https://mcp.linear.app/register". DCR supported — the second of the four named in the shelf history.',
  },
  {
    id: 'sentry',
    category: 'error tracking / observability',
    source: 'https://mcp.sentry.dev/mcp',
    auth: 'oauth-dcr',
    gap: 'Aetox has no tool that reads a live error-tracking project — nothing pulls an actual event, stack trace, or issue trend from a running Sentry account.',
    overlaps: [],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401, header WWW-Authenticate: Bearer realm="OAuth", resource_metadata="https://mcp.sentry.dev/.well-known/oauth-protected-resource/mcp". Discovery: AS metadata at https://mcp.sentry.dev/.well-known/oauth-authorization-server has registration_endpoint="https://mcp.sentry.dev/oauth/register". DCR supported — the third of the four named in the shelf history.',
  },
  {
    id: 'figma',
    category: 'design files',
    source: 'https://mcp.figma.com/mcp',
    auth: 'oauth-dcr',
    gap: 'Aetox has no tool that opens an actual Figma file — every design skill on the shelf (aetox-design, aetox-frontend-design, aetox-ui-design, aetox-design-system) is knowledge about designing well, none of them can see what is really on a real canvas.',
    overlaps: [],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: 'POST /mcp (no headers) -> 401 (plain text body "Unauthorized"), header www-authenticate: Bearer resource_metadata="https://mcp.figma.com/.well-known/oauth-protected-resource", authorization_uri="https://api.figma.com/.well-known/oauth-authorization-server". Discovery: AS metadata at api.figma.com has registration_endpoint="https://api.figma.com/v1/oauth/mcp/register". DCR supported.',
  },

  // ── auth: oauth-manual — OAuth-only, and the authorization server has NO
  //    registration_endpoint, so Aetox cannot register a client on the fly.
  //    Connecting would require Aetox to register a fixed OAuth app with
  //    each vendor by hand (a business step, not a coding one) — out of
  //    scope for internal/oauth/mcpauth.go's generic flow. Listed so the
  //    finding isn't lost, not because a click connects them today. ───────
  {
    id: 'atlassian',
    category: 'issue tracking / wiki (Jira, Confluence)',
    source: 'https://mcp.atlassian.com/v1/mcp/authv2',
    auth: 'oauth-manual',
    gap: 'Aetox has no tool for Jira or Confluence — nothing reads or writes issues, pages, or spaces in a live Atlassian site. The fourth server the shelf history named — checked again now that DCR exists, and still blocked.',
    overlaps: [],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: 'POST /v1/mcp/authv2 (no headers) -> 401 {"error":"invalid_token"}, header Www-Authenticate naming resource_metadata="https://mcp.atlassian.com/.well-known/oauth-protected-resource/v1/mcp/authv2". Discovery: authorization_servers=["https://auth.atlassian.com/<tenant>"] -> base AS metadata at https://auth.atlassian.com/.well-known/oauth-authorization-server has NO registration_endpoint (only client_id_metadata_document_supported:true, a different mechanism mcpauth.go does not implement) — DCR not supported, needs a pre-registered client_id from Atlassian.',
  },
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
  {
    id: 'vercel',
    category: 'deployment / hosting management',
    source: 'https://mcp.vercel.com',
    auth: 'oauth-manual',
    gap: 'Aetox has no tool to deploy or manage a hosted project (same category as netlify above, different vendor).',
    overlaps: ['netlify (oauth-dcr, above) — same category, different vendor; netlify is the one that actually connects today'],
    verifiedAt: '2026-09-03',
    toolCount: null,
    evidence: '401 {"error":"invalid_token","error_description":"No authorization provided"}. Discovery: AS metadata at https://vercel.com/.well-known/oauth-authorization-server has no registration_endpoint — DCR not supported, needs a pre-registered client_id from Vercel.',
  },
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
]
