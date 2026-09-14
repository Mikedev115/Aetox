// The template gallery for a new agent (§284) — the sheet the editor opens ON
// when somebody presses "+ เอเจน", before the empty form.
//
// Two kinds of card, one gallery. The four SHAPES (agentTemplates.ts) are
// briefs with blanks — a form the person fills. The ROLES below are whole
// briefs, taken from msitarzewski/agency-agents (MIT, licence beside the
// files) and trimmed on the way in (see agentGallery/README.md): a finished
// colleague that works as pasted and is edited, not filled.
//
// Forty-two roles, not the repo's 230 (owner asked "สัก 60 ตัวดีไหมหรือน้อยกว่า
// นี้ดี"). Fewer, for three reasons that are all costs the person pays, not
// the shelf: a gallery is scanned, not searched, and seven groups of six is
// one screen; each role is 1–3.5k words of SYSTEM PROMPT that agent re-reads
// on every turn, so a role shipped is a cost recommended; and the other 190
// are one paste away in the link box under the role field — nothing is lost
// by not carrying them. Region-specific briefs (China hiring platforms,
// Spanish↔English, Chinese students abroad) were passed over.
//
// The bodies are English, and so are the cards (owner, 14 ก.ย.: "เนื้อหาควร
// จะเป็นภาษาอังกฤษ"): a role is known by its trade name — Backend Architect,
// SEO Specialist — and a Thai rendering of that is a translation the person
// then has to translate back. The group headings are the app's own words and
// stay in the app's languages.
import type { TKey } from './i18n.svelte'

export type GalleryGroup = 'code' | 'marketing' | 'sales' | 'ops' | 'product' | 'quality' | 'misc'

export type GalleryRole = {
  id: string
  group: GalleryGroup
  /** The role's name, as the trade knows it. */
  title: string
  /** One line on the card, and the agent's description field once picked. */
  desc: string
  /** Words in the body — on the card, because it is what the role costs per turn. */
  words: number
}

/** Locale keys of the group headings, in the order the gallery draws them. */
export const GALLERY_GROUPS: { id: GalleryGroup; title: TKey }[] = [
  { id: 'code', title: 'settings.galleryGroupCode' },
  { id: 'marketing', title: 'settings.galleryGroupMarketing' },
  { id: 'sales', title: 'settings.galleryGroupSales' },
  { id: 'ops', title: 'settings.galleryGroupOps' },
  { id: 'product', title: 'settings.galleryGroupProduct' },
  { id: 'quality', title: 'settings.galleryGroupQuality' },
  { id: 'misc', title: 'settings.galleryGroupMisc' },
]

export const GALLERY_ROLES: GalleryRole[] = [
  // ── โค้ดและระบบ ──
  { id: 'backend-architect', group: 'code', words: 1241,
    title: 'Backend Architect',
    desc: 'Designs server-side systems, databases, APIs and cloud infrastructure that scale and stay secure' },
  { id: 'frontend-developer', group: 'code', words: 1049,
    title: 'Frontend Developer',
    desc: 'Builds responsive, accessible web UI with React/Vue/Angular, pixel-accurate and fast' },
  { id: 'mobile-app-builder', group: 'code', words: 1658,
    title: 'Mobile App Builder',
    desc: 'Native iOS/Android and cross-platform mobile apps' },
  { id: 'devops-automator', group: 'code', words: 1315,
    title: 'DevOps Automator',
    desc: 'CI/CD pipelines, infrastructure as code and cloud operations' },
  { id: 'code-reviewer', group: 'code', words: 409,
    title: 'Code Reviewer',
    desc: 'Reviews for correctness, security and performance — not style preferences' },
  { id: 'technical-writer', group: 'code', words: 1778,
    title: 'Technical Writer',
    desc: 'READMEs, API references and tutorials developers actually read' },

  // ── การตลาดและคอนเทนต์ ──
  { id: 'social-media-strategist', group: 'marketing', words: 820,
    title: 'Social Media Strategist',
    desc: 'Cross-platform campaigns, community building and engagement' },
  { id: 'seo-specialist', group: 'marketing', words: 2872,
    title: 'SEO Specialist',
    desc: 'Technical SEO, content optimisation and link authority for organic growth' },
  { id: 'email-strategist', group: 'marketing', words: 2267,
    title: 'Email Marketing Strategist',
    desc: 'Lifecycle sequences, segmentation and deliverability' },
  { id: 'tiktok-strategist', group: 'marketing', words: 841,
    title: 'TikTok Strategist',
    desc: 'Content that rides the algorithm and builds a TikTok audience' },
  { id: 'livestream-commerce-coach', group: 'marketing', words: 2683,
    title: 'Livestream Commerce Coach',
    desc: 'Live scripts, product sequencing, closing and reading the numbers mid-stream (written for Chinese platforms — rename yours)' },
  { id: 'instagram-curator', group: 'marketing', words: 723,
    title: 'Instagram Curator',
    desc: 'Visual storytelling, a consistent feed and community' },

  // ── ขายและดูแลลูกค้า ──
  { id: 'sales-outbound-strategist', group: 'sales', words: 1618,
    title: 'Outbound Strategist',
    desc: 'Defines the ideal customer and designs researched multi-channel outreach' },
  { id: 'sales-proposal-strategist', group: 'sales', words: 1862,
    title: 'Proposal Strategist',
    desc: 'Turns RFPs and opportunities into proposals that persuade, not merely comply' },
  { id: 'sales-deal-strategist', group: 'sales', words: 1993,
    title: 'Deal Strategist',
    desc: 'MEDDPICC qualification, pipeline risk and win plans for complex B2B deals' },
  { id: 'customer-service', group: 'sales', words: 2528,
    title: 'Customer Service',
    desc: 'Inquiries, complaints, account support and clean escalation, warm and quick' },
  { id: 'support-responder', group: 'sales', words: 2179,
    title: 'Support Responder',
    desc: 'Multi-channel issue resolution that turns support into a good brand moment' },
  { id: 'customer-success-manager', group: 'sales', words: 3136,
    title: 'Customer Success Manager',
    desc: 'Onboarding, health scoring, churn prevention and renewals' },

  // ── การเงินและบริหาร ──
  { id: 'bookkeeper-controller', group: 'ops', words: 2005,
    title: 'Bookkeeper & Controller',
    desc: 'Day-to-day books, reconciliations, month-end close and internal controls' },
  { id: 'financial-analyst', group: 'ops', words: 1856,
    title: 'Financial Analyst',
    desc: 'Financial modelling, forecasting and scenario analysis for decisions' },
  { id: 'pricing-analyst', group: 'ops', words: 1633,
    title: 'Pricing Analyst',
    desc: 'Pricing from cost, competitors and market — not guesswork' },
  { id: 'operations-manager', group: 'ops', words: 2818,
    title: 'Operations Manager',
    desc: 'Process mapping, capacity planning, KPIs and vendor management' },
  { id: 'chief-of-staff', group: 'ops', words: 2822,
    title: 'Chief of Staff',
    desc: 'Filters noise, owns process and routes decisions so the boss can think' },
  { id: 'hr-onboarding', group: 'ops', words: 3086,
    title: 'HR Onboarding',
    desc: 'Orientation, paperwork, benefits and first-year support for new hires' },

  // ── ผลิตภัณฑ์และดีไซน์ ──
  { id: 'product-manager', group: 'product', words: 3605,
    title: 'Product Manager',
    desc: 'Owns the product lifecycle from discovery and roadmap to outcomes' },
  { id: 'feedback-synthesizer', group: 'product', words: 849,
    title: 'Feedback Synthesizer',
    desc: 'Turns feedback from every channel into ranked, actionable priorities' },
  { id: 'sprint-prioritizer', group: 'product', words: 1102,
    title: 'Sprint Prioritizer',
    desc: 'Feature prioritisation and resource allocation for each sprint' },
  { id: 'ui-designer', group: 'product', words: 1513,
    title: 'UI Designer',
    desc: 'Design systems, component libraries and consistent, accessible screens' },
  { id: 'ux-researcher', group: 'product', words: 1519,
    title: 'UX Researcher',
    desc: 'Usability testing and behaviour analysis that lead to concrete fixes' },
  { id: 'brand-guardian', group: 'product', words: 1346,
    title: 'Brand Guardian',
    desc: 'Builds and keeps brand identity consistent at every touchpoint' },

  // ── ตรวจงานและความปลอดภัย ──
  { id: 'api-tester', group: 'quality', words: 1391,
    title: 'API Tester',
    desc: 'Validation, performance and integration testing across every API' },
  { id: 'reality-checker', group: 'quality', words: 1238,
    title: 'Reality Checker',
    desc: 'Defaults to "needs work"; demands overwhelming proof before calling anything production-ready' },
  { id: 'accessibility-auditor', group: 'quality', words: 1950,
    title: 'Accessibility Auditor',
    desc: 'Audits against WCAG and tests with real assistive technology' },
  { id: 'appsec-engineer', group: 'quality', words: 3127,
    title: 'Application Security Engineer',
    desc: 'Threat modelling, secure code review and SAST/DAST in the pipeline' },
  { id: 'compliance-auditor', group: 'quality', words: 1021,
    title: 'Compliance Auditor',
    desc: 'SOC 2, ISO 27001, HIPAA and PCI-DSS from readiness to evidence' },
  { id: 'secrets-credential-engineer', group: 'quality', words: 1909,
    title: 'Secrets & Credential Engineer',
    desc: 'Detection, vaulting, rotation and leak response — no secrets in code' },

  // ── เฉพาะทาง ──
  { id: 'legal-document-review', group: 'misc', words: 2791,
    title: 'Legal Document Review',
    desc: 'Summarises contracts, flags risk clauses and compares versions (not a lawyer)' },
  { id: 'meeting-notes-specialist', group: 'misc', words: 825,
    title: 'Meeting Notes Specialist',
    desc: 'Decisions, action items and open questions from notes or a transcript, in four sections' },
  { id: 'project-shepherd', group: 'misc', words: 1098,
    title: 'Project Shepherd',
    desc: 'Cross-team coordination, timelines, risks and communication to the finish' },
  { id: 'resume-tailor', group: 'misc', words: 1478,
    title: 'Resume Tailor',
    desc: 'Maps real experience to the job post and rewrites for ATS without inventing' },
  { id: 'grant-writer', group: 'misc', words: 3417,
    title: 'Grant Writer',
    desc: 'Prospecting, letters of inquiry, full proposals, budgets and post-award reporting' },
  { id: 'personal-growth-mentor', group: 'misc', words: 1007,
    title: 'Personal Growth Mentor',
    desc: 'Goal clarity, habit design, decisions and accountability without motivational fluff' },
]

// Bodies load on pick, not with the page: forty-two briefs are ~600 KB of
// text, and a person opening the editor to change a model should not
// download a shelf they will not read. Vite splits each .md into its own
// chunk; the map is path → loader.
const bodies = Object.fromEntries(
  Object.entries(import.meta.glob('./agentGallery/*.md', { query: '?raw', import: 'default' }))
    .filter(([p]) => !p.endsWith('/README.md')), // the folder's own note, not a role
) as Record<string, () => Promise<string>>

/** The role's brief. Rejects for an id that is not on the shelf. */
export async function galleryBody(id: string): Promise<string> {
  const load = bodies[`./agentGallery/${id}.md`]
  if (!load) throw new Error(`no gallery role: ${id}`)
  return load()
}

/** The gallery's ids, for the shelf and the tests to agree on. */
export const galleryIds = () => Object.keys(bodies).map((p) => p.slice('./agentGallery/'.length, -'.md'.length))

/** Case-insensitive match of a query against a role's id, title and line. */
export function galleryMatches(r: GalleryRole, q: string): boolean {
  const needle = q.trim().toLowerCase()
  if (!needle) return true
  return [r.id, r.title, r.desc].some((s) => s.toLowerCase().includes(needle))
}

/** localStorage flag: the person ticked "do not open this on a new agent". */
export const GALLERY_SKIP_KEY = 'agentGallerySkip'
