// The map of the app's own UI: every data-guide id, which page it lives on,
// whether the guide may press it, and which DECISIONS.md section justifies its
// design.
//
// Source of truth for words is docs/GUIDE-MAP.md, rendered through locales
// (lib/locales/{th,en,zh}.ts under guide.<id>.name, guide.<id>.what, guide.<id>.why).

export type GuidePage =
  | { view: 'chat' }
  | { view: 'settings'; rail: string; tab?: string; head?: 'assistant' | 'coding' }
  | { view: 'capability'; page: string }
  | { view: 'office' }
  | { view: 'artifacts' }

export type GuideEntry = {
  id: string
  page: GuidePage
  safe: boolean
  ref?: string
  synonyms?: string[]
}

export const GUIDE_MAP: GuideEntry[] = [
  // ── Area 1: Sidebar ────────────────────────────────────────────────────────
  { id: 'sidebar.projects', page: { view: 'chat' }, safe: true, ref: '§158', synonyms: ['โปรเจกต์', 'projects', 'folder', 'โฟลเดอร์'] },
  { id: 'sidebar.open_folder', page: { view: 'chat' }, safe: false, ref: '§19', synonyms: ['เปิดโฟลเดอร์', 'open folder', 'browse'] },
  { id: 'sidebar.create_project', page: { view: 'chat' }, safe: true, ref: '§158', synonyms: ['สร้างโปรเจกต์', 'new project', 'create project'] },
  { id: 'sidebar.create_project_name', page: { view: 'chat' }, safe: true, ref: '§158', synonyms: ['ชื่อโปรเจกต์', 'project name'] },
  { id: 'sidebar.create_project_move', page: { view: 'chat' }, safe: true, ref: '§14', synonyms: ['เปลี่ยนที่เก็บ', 'move project', 'project location'] },
  { id: 'sidebar.create_project_submit', page: { view: 'chat' }, safe: false, ref: '§158', synonyms: ['ยืนยันสร้างโปรเจกต์', 'create and chat'] },
  { id: 'sidebar.create_project_cancel', page: { view: 'chat' }, safe: true, ref: '§158', synonyms: ['ยกเลิกสร้างโปรเจกต์', 'cancel project'] },
  { id: 'sidebar.search', page: { view: 'chat' }, safe: true, ref: '§20', synonyms: ['ค้นหาแชท', 'search history', 'find chat'] },
  { id: 'sidebar.import_chat', page: { view: 'chat' }, safe: false, ref: '§27', synonyms: ['นำเข้าแชท', 'import session', 'load chat'] },
  { id: 'sidebar.new_session', page: { view: 'chat' }, safe: true, ref: '§19', synonyms: ['แชทใหม่', 'new chat', 'new session'] },
  { id: 'sidebar.history', page: { view: 'chat' }, safe: true, ref: '§20', synonyms: ['ประวัติ', 'history', 'past chats'] },
  { id: 'sidebar.desk.assistant', page: { view: 'chat' }, safe: true, ref: '§83', synonyms: ['โต๊ะผู้ช่วย', 'assistant desk', 'ผู้ช่วย'] },
  { id: 'sidebar.desk.coding', page: { view: 'chat' }, safe: true, ref: '§83', synonyms: ['โต๊ะโค้ด', 'coding desk', 'โค้ด'] },
  { id: 'sidebar.desk.capability', page: { view: 'capability', page: 'mcp' }, safe: true, ref: '§22', synonyms: ['ความสามารถ', 'capability', 'tools room'] },
  { id: 'sidebar.desk.office', page: { view: 'office' }, safe: true, ref: '§85', synonyms: ['ออฟฟิศ', 'office', 'ทีม', 'team'] },
  { id: 'sidebar.desk.artifacts', page: { view: 'artifacts' }, safe: true, ref: '§27', synonyms: ['คลังผลงาน', 'artifacts', 'ผลงาน'] },
  { id: 'sidebar.footer', page: { view: 'chat' }, safe: true, ref: '§270', synonyms: ['เมนูบัญชี', 'profile menu', 'account menu', 'footer'] },

  // ── Area 2: Account Menu ───────────────────────────────────────────────────
  { id: 'account.name', page: { view: 'chat' }, safe: true, ref: '§270', synonyms: ['ชื่อโปรไฟล์', 'your name', 'profile name'] },
  { id: 'account.theme', page: { view: 'chat' }, safe: true, ref: '§24', synonyms: ['ธีม', 'theme', 'color scheme'] },
  { id: 'account.language', page: { view: 'chat' }, safe: true, ref: '§33', synonyms: ['ภาษา', 'language', 'locale'] },
  { id: 'account.companion', page: { view: 'chat' }, safe: false, ref: '§292', synonyms: ['ตัวช่วยบนจอ', 'companion', 'mascot', 'มาสคอต'] },
  { id: 'account.tour', page: { view: 'chat' }, safe: true, ref: '§279', synonyms: ['รู้จักกับ Aetox', 'tour', 'ทัวร์'] },
  { id: 'account.update_check', page: { view: 'chat' }, safe: false, ref: '§23', synonyms: ['ตรวจอัปเดต', 'check update', 'version update'] },
  { id: 'account.settings', page: { view: 'settings', rail: 'general' }, safe: true, ref: '§24', synonyms: ['ตั้งค่า', 'settings', 'preferences'] },

  // ── Area 3: TopBar ─────────────────────────────────────────────────────────
  { id: 'topbar.door', page: { view: 'chat' }, safe: true, ref: '§279', synonyms: ['ประตูสลับหัว', 'door', 'switch head', 'หัว'] },
  { id: 'topbar.door.assistant', page: { view: 'chat' }, safe: true, ref: '§279', synonyms: ['เลือกหัวผู้ช่วย', 'assistant head'] },
  { id: 'topbar.door.coding', page: { view: 'chat' }, safe: true, ref: '§279', synonyms: ['เลือกหัวโค้ด', 'coding head'] },
  { id: 'topbar.sidebar_btn', page: { view: 'chat' }, safe: true, ref: '§19', synonyms: ['เปิดปิดแถบข้าง', 'toggle sidebar', 'hide sidebar'] },
  { id: 'topbar.inspector_btn', page: { view: 'chat' }, safe: true, ref: '§19', synonyms: ['เปิดปิดแผงตรวจ', 'toggle inspector', 'side panel'] },
  { id: 'topbar.space', page: { view: 'chat' }, safe: true, ref: '§158', synonyms: ['ป้ายโปรเจกต์', 'project badge', 'space name'] },
  { id: 'topbar.tabs', page: { view: 'chat' }, safe: true, ref: '§18', synonyms: ['แท็บห้องทำงาน', 'workbench tabs'] },
  { id: 'topbar.tab.chat', page: { view: 'chat' }, safe: true, ref: '§18', synonyms: ['แท็บแชท', 'chat tab'] },
  { id: 'topbar.tab.diff', page: { view: 'chat' }, safe: true, ref: '§15', synonyms: ['แท็บ diff', 'diff tab', 'เปรียบเทียบโค้ด'] },
  { id: 'topbar.tab.editor', page: { view: 'chat' }, safe: true, ref: '§15', synonyms: ['แท็บแก้ไขไฟล์', 'editor tab'] },
  { id: 'topbar.tab.browser', page: { view: 'chat' }, safe: true, ref: '§18', synonyms: ['แท็บเบราว์เซอร์', 'browser tab'] },
  { id: 'topbar.tab.terminal', page: { view: 'chat' }, safe: true, ref: '§15', synonyms: ['แท็บเทอร์มินัล', 'terminal tab'] },

  // ── Area 4: Composer & Chat ────────────────────────────────────────────────
  { id: 'composer.input', page: { view: 'chat' }, safe: true, ref: '§36', synonyms: ['ช่องพิมพ์', 'composer', 'message input', 'พิมพ์ข้อความ'] },
  { id: 'composer.attach', page: { view: 'chat' }, safe: false, ref: '§38', synonyms: ['แนบไฟล์', 'attach file', 'upload file'] },
  { id: 'composer.mic', page: { view: 'chat' }, safe: false, ref: '§31', synonyms: ['ไมค์', 'microphone', 'voice input', 'พูด'] },
  { id: 'chat.send', page: { view: 'chat' }, safe: false, ref: '§105', synonyms: ['ปุ่มส่ง', 'send button', 'stop', 'หยุด'] },
  { id: 'composer.stance', page: { view: 'chat' }, safe: true, ref: '§106', synonyms: ['ระดับการลงมือ', 'stance', 'approval mode', 'สแตนซ์'] },
  { id: 'composer.think', page: { view: 'chat' }, safe: true, ref: '§292', synonyms: ['ระดับความคิด', 'think level', 'reasoning'] },
  { id: 'composer.model', page: { view: 'chat' }, safe: true, ref: '§20', synonyms: ['สลับโมเดล', 'switch model', 'choose model'] },
  { id: 'chat.empty_headline', page: { view: 'chat' }, safe: true, ref: '§270', synonyms: ['ข้อความทักทาย', 'greeting', 'room headline'] },
  { id: 'chat.starter', page: { view: 'chat' }, safe: true, ref: '§35', synonyms: ['ตัวอย่างคำสั่ง', 'starters', 'prompt starter'] },
  { id: 'chat.starter_more', page: { view: 'chat' }, safe: true, ref: '§35', synonyms: ['ขอตัวอย่างเพิ่ม', 'more starters', 'reroll starters'] },
  { id: 'chat.tour', page: { view: 'chat' }, safe: true, ref: '§279', synonyms: ['ทัวร์หน้าแชท', 'tour link', 'chat tour'] },

  // ── Area 5: Memory Cards ───────────────────────────────────────────────────
  { id: 'memory.card', page: { view: 'chat' }, safe: true, ref: '§279', synonyms: ['การ์ดขอจำ', 'memory proposal', 'memory card'] },
  { id: 'memory.card.accept', page: { view: 'chat' }, safe: false, ref: '§279', synonyms: ['อนุมัติให้จำ', 'accept memory', 'approve memory'] },
  { id: 'memory.card.reject', page: { view: 'chat' }, safe: false, ref: '§279', synonyms: ['ไม่ให้จำ', 'reject memory', 'dismiss memory'] },
  { id: 'memory.card.edit', page: { view: 'chat' }, safe: true, ref: '§279', synonyms: ['แก้ไขความจำ', 'edit memory'] },
  { id: 'memory.card.scope', page: { view: 'chat' }, safe: true, ref: '§116', synonyms: ['ขอบเขตความจำ', 'memory scope'] },

  // ── Area 6: Settings Rail & General ────────────────────────────────────────
  { id: 'settings.back', page: { view: 'settings', rail: 'general' }, safe: true, ref: '§24', synonyms: ['กลับไปแชท', 'back to app', 'close settings'] },
  { id: 'settings.search', page: { view: 'settings', rail: 'general' }, safe: true, ref: '§24', synonyms: ['ค้นหาการตั้งค่า', 'search settings'] },
  { id: 'settings.rail.general', page: { view: 'settings', rail: 'general' }, safe: true, ref: '§24', synonyms: ['ตั้งค่าทั่วไป', 'general settings'] },
  { id: 'settings.rail.appearance', page: { view: 'settings', rail: 'appearance' }, safe: true, ref: '§24', synonyms: ['รูปลักษณ์', 'appearance', 'theme settings'] },
  { id: 'settings.rail.avatar', page: { view: 'settings', rail: 'avatar' }, safe: true, ref: '§279', synonyms: ['ตั้งค่าอวตาร', 'avatar settings', 'mascot avatar'] },
  { id: 'settings.rail.you', page: { view: 'settings', rail: 'you' }, safe: true, ref: '§270', synonyms: ['เกี่ยวกับคุณ', 'about you', 'user profile'] },
  { id: 'settings.rail.issues', page: { view: 'settings', rail: 'issues' }, safe: true, ref: '§292', synonyms: ['แจ้งปัญหา', 'system issues', 'diagnostics'] },
  { id: 'settings.rail.brain', page: { view: 'settings', rail: 'models' }, safe: true, ref: '§279', synonyms: ['ต่อสมอง', 'connect brain', 'models', 'providers'] },
  { id: 'settings.rail.heads', page: { view: 'settings', rail: 'main' }, safe: true, ref: '§266', synonyms: ['ตัวหลัก', 'heads settings', 'main assistant'] },
  { id: 'settings.rail.teams', page: { view: 'settings', rail: 'teams' }, safe: true, ref: '§256', synonyms: ['จัดการทีม', 'teams settings'] },
  { id: 'settings.rail.agents', page: { view: 'settings', rail: 'team' }, safe: true, ref: '§85', synonyms: ['พนักงาน', 'specialists', 'agents'] },
  { id: 'settings.rail.hands', page: { view: 'settings', rail: 'agents' }, safe: true, ref: '§25', synonyms: ['ลูกมือ', 'helpers', 'subagents'] },
  { id: 'settings.rail.voice', page: { view: 'settings', rail: 'voice' }, safe: true, ref: '§33', synonyms: ['ตั้งค่าเสียง', 'voice settings', 'speech'] },
  { id: 'settings.rail.image', page: { view: 'settings', rail: 'image' }, safe: true, ref: '§21', synonyms: ['ตั้งค่าภาพ', 'image settings', 'vision'] },
  { id: 'settings.rail.studio', page: { view: 'settings', rail: 'studio' }, safe: true, ref: '§32', synonyms: ['ตั้งค่าสตูดิโอ', 'studio settings'] },
  { id: 'settings.rail.remote', page: { view: 'settings', rail: 'remote' }, safe: true, ref: '§248', synonyms: ['การเชื่อมต่อทางไกล', 'remote engine'] },
  { id: 'settings.rail.account', page: { view: 'settings', rail: 'account' }, safe: true, ref: '§270', synonyms: ['บัญชี Aetox', 'aetox account'] },
  { id: 'settings.rail.usage', page: { view: 'settings', rail: 'usage' }, safe: true, ref: '§20', synonyms: ['สถิติการใช้งาน', 'token usage', 'metrics'] },
  { id: 'settings.rail.about', page: { view: 'settings', rail: 'about' }, safe: true, ref: '§23', synonyms: ['เกี่ยวกับ Aetox', 'about app', 'version info'] },
  { id: 'settings.rail.sponsor', page: { view: 'settings', rail: 'sponsor' }, safe: true, ref: '§23', synonyms: ['ผู้สนับสนุน', 'sponsor'] },
  { id: 'settings.general.theme', page: { view: 'settings', rail: 'general' }, safe: true, ref: '§24', synonyms: ['เลือกธีม', 'select theme'] },
  { id: 'settings.general.language', page: { view: 'settings', rail: 'general' }, safe: true, ref: '§33', synonyms: ['เลือกภาษา', 'select language'] },
  { id: 'settings.general.font_scale', page: { view: 'settings', rail: 'general' }, safe: true, ref: '§24', synonyms: ['ขนาดตัวอักษร', 'font scale', 'text size'] },

  // ── Area 7: Settings Head & Agents ─────────────────────────────────────────
  { id: 'settings.head.hero', page: { view: 'settings', rail: 'main' }, safe: true, ref: '§266', synonyms: ['ข้อมูลตัวหลัก', 'head hero'] },
  { id: 'settings.head.rank', page: { view: 'settings', rail: 'main' }, safe: false, ref: '§266', synonyms: ['ตรายศ', 'rank badge', 'rank emblem'] },
  { id: 'settings.head.switch_desk', page: { view: 'settings', rail: 'main' }, safe: true, ref: '§266', synonyms: ['สลับหัวตัวหลัก', 'switch head in settings'] },
  { id: 'settings.head.tab.identity', page: { view: 'settings', rail: 'main' }, safe: true, ref: '§270', synonyms: ['แท็บตัวตน', 'identity tab'] },
  { id: 'settings.head.tab.mcp', page: { view: 'settings', rail: 'main' }, safe: true, ref: '§22', synonyms: ['แท็บ MCP ประจำหัว', 'head mcp tab'] },
  { id: 'settings.head.tab.skills', page: { view: 'settings', rail: 'main' }, safe: true, ref: '§22', synonyms: ['แท็บสกิลประจำหัว', 'head skills tab'] },
  { id: 'settings.head.tab.dialogue', page: { view: 'settings', rail: 'main' }, safe: true, ref: '§266', synonyms: ['แท็บเปิดบทสนทนา', 'dialogue starters tab'] },
  { id: 'settings.head.tab.memory', page: { view: 'settings', rail: 'main' }, safe: true, ref: '§279', synonyms: ['แท็บความจำ', 'head memory tab'] },
  { id: 'settings.head.tab.tools', page: { view: 'settings', rail: 'main' }, safe: true, ref: '§27', synonyms: ['แท็บเครื่องมือ', 'head tools tab'] },
  { id: 'settings.head.save', page: { view: 'settings', rail: 'main' }, safe: false, ref: '§266', synonyms: ['บันทึกหัว', 'save head profile'] },
  { id: 'settings.head.reset', page: { view: 'settings', rail: 'main' }, safe: false, ref: '§266', synonyms: ['รีเซ็ตหัว', 'reset head defaults'] },
  { id: 'settings.head.delete', page: { view: 'settings', rail: 'main' }, safe: false, ref: '§85', synonyms: ['ลบเอเจน', 'delete agent'] },

  // ── Area 8: Settings Brain ─────────────────────────────────────────────────
  { id: 'settings.brain.hero', page: { view: 'settings', rail: 'models' }, safe: true, ref: '§279', synonyms: ['ข้อมูลต่อสมอง', 'brain hero'] },
  { id: 'settings.brain.provider.ollama', page: { view: 'settings', rail: 'models' }, safe: true, ref: '§279', synonyms: ['โมเดลในเครื่อง', 'ollama', 'lm studio', 'local model'] },
  { id: 'settings.brain.provider.openai', page: { view: 'settings', rail: 'models' }, safe: true, ref: '§20', synonyms: ['openai', 'chatgpt', 'gpt-4o'] },
  { id: 'settings.brain.provider.anthropic', page: { view: 'settings', rail: 'models' }, safe: true, ref: '§20', synonyms: ['anthropic', 'claude'] },
  { id: 'settings.brain.provider.deepseek', page: { view: 'settings', rail: 'models' }, safe: true, ref: '§20', synonyms: ['deepseek', 'deepseek v3'] },
  { id: 'settings.brain.provider.google', page: { view: 'settings', rail: 'models' }, safe: true, ref: '§20', synonyms: ['google', 'gemini'] },
  { id: 'settings.brain.provider.groq', page: { view: 'settings', rail: 'models' }, safe: true, ref: '§20', synonyms: ['groq'] },
  { id: 'settings.brain.provider.openrouter', page: { view: 'settings', rail: 'models' }, safe: true, ref: '§20', synonyms: ['openrouter'] },
  { id: 'settings.brain.test_connection', page: { view: 'settings', rail: 'models' }, safe: false, ref: '§20', synonyms: ['ทดสอบการเชื่อมต่อ', 'test connection'] },
  { id: 'settings.brain.add_provider', page: { view: 'settings', rail: 'models' }, safe: true, ref: '§20', synonyms: ['เพิ่มผู้ให้บริการ', 'add provider'] },

  // ── Area 9: You, Voice, Avatar, About ──────────────────────────────────────
  { id: 'settings.you.name_input', page: { view: 'settings', rail: 'you' }, safe: true, ref: '§270', synonyms: ['ช่องกรอกชื่อคุณ', 'user name input'] },
  { id: 'settings.you.about_input', page: { view: 'settings', rail: 'you' }, safe: true, ref: '§270', synonyms: ['ช่องกรอกข้อมูลคุณ', 'about you notes'] },
  { id: 'settings.you.save', page: { view: 'settings', rail: 'you' }, safe: false, ref: '§270', synonyms: ['บันทึกข้อมูลคุณ', 'save user profile'] },
  { id: 'settings.voice.switch', page: { view: 'settings', rail: 'voice' }, safe: false, ref: '§33', synonyms: ['สวิตช์เสียง', 'toggle voice'] },
  { id: 'settings.voice.device_select', page: { view: 'settings', rail: 'voice' }, safe: true, ref: '§33', synonyms: ['เลือกอุปกรณ์เสียง', 'audio device'] },
  { id: 'settings.voice.test_play', page: { view: 'settings', rail: 'voice' }, safe: false, ref: '§33', synonyms: ['ทดสอบเสียง', 'test audio'] },
  { id: 'settings.avatar.role_select', page: { view: 'settings', rail: 'avatar' }, safe: true, ref: '§279', synonyms: ['บทบาทอวตาร', 'avatar role'] },
  { id: 'settings.avatar.accent_select', page: { view: 'settings', rail: 'avatar' }, safe: true, ref: '§279', synonyms: ['สีเน้นอวตาร', 'avatar accent'] },
  { id: 'settings.avatar.shell_select', page: { view: 'settings', rail: 'avatar' }, safe: true, ref: '§279', synonyms: ['เปลือกอวตาร', 'avatar shell'] },
  { id: 'settings.avatar.top_select', page: { view: 'settings', rail: 'avatar' }, safe: true, ref: '§279', synonyms: ['หมวกอวตาร', 'avatar hat', 'antenna'] },
  { id: 'settings.avatar.face_select', page: { view: 'settings', rail: 'avatar' }, safe: true, ref: '§279', synonyms: ['หน้าตาอวตาร', 'avatar face'] },
  { id: 'settings.avatar.prop_select', page: { view: 'settings', rail: 'avatar' }, safe: true, ref: '§279', synonyms: ['ของถืออวตาร', 'avatar prop'] },
  { id: 'settings.avatar.save', page: { view: 'settings', rail: 'avatar' }, safe: false, ref: '§279', synonyms: ['บันทึกอวตาร', 'save avatar'] },
  { id: 'settings.about.version_info', page: { view: 'settings', rail: 'about' }, safe: true, ref: '§23', synonyms: ['เลขเวอร์ชัน', 'version info'] },
  { id: 'settings.about.update_btn', page: { view: 'settings', rail: 'about' }, safe: false, ref: '§23', synonyms: ['ตรวจอัปเดตเกี่ยวกับ', 'check update in about'] },
  { id: 'settings.about.tour_btn', page: { view: 'settings', rail: 'about' }, safe: true, ref: '§279', synonyms: ['ดูการแนะนำอีกครั้ง', 'replay tour'] },
  { id: 'settings.about.github_btn', page: { view: 'settings', rail: 'about' }, safe: false, ref: '§23', synonyms: ['github aetox', 'open github'] },
  { id: 'settings.about.license_btn', page: { view: 'settings', rail: 'about' }, safe: true, ref: '§23', synonyms: ['สัญญาอนุญาต', 'license'] },

  // ── Area 10: Capability Room ───────────────────────────────────────────────
  { id: 'capability.rail.mcp', page: { view: 'capability', page: 'mcp' }, safe: true, ref: '§22', synonyms: ['mcp server', 'หน้า mcp'] },
  { id: 'capability.rail.skills', page: { view: 'capability', page: 'skills' }, safe: true, ref: '§22', synonyms: ['สกิลของคุณ', 'skills list'] },
  { id: 'capability.rail.builtins', page: { view: 'capability', page: 'builtins' }, safe: true, ref: '§27', synonyms: ['เครื่องมือในตัว', 'builtin tools'] },
  { id: 'capability.rail.computer', page: { view: 'capability', page: 'computer' }, safe: true, ref: '§22', synonyms: ['การใช้คอมพิวเตอร์', 'computer use'] },
  { id: 'capability.rail.connections', page: { view: 'capability', page: 'connections' }, safe: true, ref: '§22', synonyms: ['การเชื่อมต่อ', 'external connections'] },
  { id: 'capability.rail.prompts', page: { view: 'capability', page: 'prompts' }, safe: true, ref: '§35', synonyms: ['ชุดคำสั่ง', 'prompt presets'] },
  { id: 'capability.mcp.search', page: { view: 'capability', page: 'mcp' }, safe: true, ref: '§22', synonyms: ['ค้นหา mcp', 'search mcp'] },
  { id: 'capability.mcp.add_btn', page: { view: 'capability', page: 'mcp' }, safe: true, ref: '§22', synonyms: ['เพิ่ม mcp', 'add mcp server'] },
  { id: 'capability.mcp.install_action', page: { view: 'capability', page: 'mcp' }, safe: false, ref: '§22', synonyms: ['ติดตั้ง mcp', 'install mcp'] },
  { id: 'capability.mcp.refresh_btn', page: { view: 'capability', page: 'mcp' }, safe: false, ref: '§22', synonyms: ['รีเฟรช mcp', 'refresh mcp'] },
  { id: 'capability.skills.search', page: { view: 'capability', page: 'skills' }, safe: true, ref: '§22', synonyms: ['ค้นหาสกิล', 'search skills'] },
  { id: 'capability.skills.add_btn', page: { view: 'capability', page: 'skills' }, safe: true, ref: '§22', synonyms: ['เพิ่มสกิล', 'add skill'] },
  { id: 'capability.skills.folder_btn', page: { view: 'capability', page: 'skills' }, safe: false, ref: '§14', synonyms: ['เปิดโฟลเดอร์สกิล', 'skills folder'] },
  { id: 'capability.builtins.filter', page: { view: 'capability', page: 'builtins' }, safe: true, ref: '§27', synonyms: ['กรองเครื่องมือ', 'filter builtin tools'] },
  { id: 'capability.computer.toggle', page: { view: 'capability', page: 'computer' }, safe: false, ref: '§22', synonyms: ['สวิตช์ใช้คอมพิวเตอร์', 'toggle computer use'] },
  { id: 'capability.computer.apps_list', page: { view: 'capability', page: 'computer' }, safe: true, ref: '§22', synonyms: ['รายการโปรแกรม', 'allowed apps'] },
  { id: 'capability.connections.add_btn', page: { view: 'capability', page: 'connections' }, safe: true, ref: '§22', synonyms: ['เพิ่มการเชื่อมต่อ', 'add connection'] },
  { id: 'capability.prompts.new_btn', page: { view: 'capability', page: 'prompts' }, safe: true, ref: '§35', synonyms: ['สร้างชุดคำสั่ง', 'new prompt preset'] },

  // ── Area 11: Office & Teams ────────────────────────────────────────────────
  { id: 'office.header', page: { view: 'office' }, safe: true, ref: '§85', synonyms: ['ผังองค์กร', 'office roster', 'office view'] },
  { id: 'office.team_tab', page: { view: 'office' }, safe: true, ref: '§256', synonyms: ['แท็บทีม', 'teams tab'] },
  { id: 'office.new_agent_btn', page: { view: 'office' }, safe: true, ref: '§85', synonyms: ['จ้างพนักงานใหม่', 'new agent', 'hire agent'] },
  { id: 'office.member_card', page: { view: 'office' }, safe: true, ref: '§85', synonyms: ['การ์ดพนักงาน', 'agent card', 'member card'] },
  { id: 'office.chat_btn', page: { view: 'office' }, safe: true, ref: '§85', synonyms: ['คุยกับพนักงาน', 'chat with agent'] },
  { id: 'team.name_input', page: { view: 'office' }, safe: true, ref: '§256', synonyms: ['ชื่อทีม', 'team name'] },
  { id: 'team.lead_select', page: { view: 'office' }, safe: true, ref: '§256', synonyms: ['หัวหน้าทีม', 'team lead'] },
  { id: 'team.add_member_btn', page: { view: 'office' }, safe: true, ref: '§256', synonyms: ['เพิ่มสมาชิกทีม', 'add team member'] },
  { id: 'team.save_btn', page: { view: 'office' }, safe: false, ref: '§256', synonyms: ['บันทึกทีม', 'save team'] },
  { id: 'team.delete_btn', page: { view: 'office' }, safe: false, ref: '§256', synonyms: ['ลบทีม', 'delete team'] },

  // ── Area 12: Background Work & Studio ──────────────────────────────────────
  { id: 'background.panel', page: { view: 'chat' }, safe: true, ref: '§25', synonyms: ['แผงงานเบื้องหลัง', 'background work panel'] },
  { id: 'background.filter_running', page: { view: 'chat' }, safe: true, ref: '§25', synonyms: ['งานที่กำลังวิ่ง', 'running tasks'] },
  { id: 'background.filter_done', page: { view: 'chat' }, safe: true, ref: '§25', synonyms: ['งานที่เสร็จแล้ว', 'completed tasks'] },
  { id: 'background.clear_btn', page: { view: 'chat' }, safe: false, ref: '§25', synonyms: ['ล้างงานเสร็จ', 'clear completed tasks'] },
  { id: 'background.task_item', page: { view: 'chat' }, safe: true, ref: '§25', synonyms: ['รายการงาน', 'task item'] },
  { id: 'background.task_stop', page: { view: 'chat' }, safe: false, ref: '§25', synonyms: ['หยุดงานเบื้องหลัง', 'stop background task'] },
  { id: 'studio.browser', page: { view: 'artifacts' }, safe: true, ref: '§32', synonyms: ['คลังสตูดิโอ', 'studio browser', 'assets'] },
  { id: 'studio.search', page: { view: 'artifacts' }, safe: true, ref: '§32', synonyms: ['ค้นหาสื่อ', 'search assets'] },
  { id: 'studio.filter_tab', page: { view: 'artifacts' }, safe: true, ref: '§32', synonyms: ['กรองสื่อ', 'filter media assets'] },
  { id: 'studio.import_btn', page: { view: 'artifacts' }, safe: false, ref: '§32', synonyms: ['นำเข้าสื่อ', 'import asset'] },
  { id: 'studio.preview_item', page: { view: 'artifacts' }, safe: true, ref: '§32', synonyms: ['ดูตัวอย่างสื่อ', 'preview asset'] },
  { id: 'studio.delete_btn', page: { view: 'artifacts' }, safe: false, ref: '§32', synonyms: ['ลบสื่อ', 'delete asset'] },
]
