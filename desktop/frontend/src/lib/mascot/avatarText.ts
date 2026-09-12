// The avatar page's words, in the three UI languages.
//
// TEMPORARY HOME. These belong in locales/th.ts, en.ts and zh.ts under
// `settings.avatar*`, beside every other settings string. They are here
// because those three files (and Settings.svelte's own nav) are mid-change in
// another session on 12 ก.ย. 2026, and a key added to a file somebody else is
// rewriting is a merge nobody asked for. Moving them is a paste and a
// find-replace of `avatarText(locale).x` → `t('settings.avatarX')`; the
// switch's label (CompanionSwitch.svelte) goes at the same time.

import type { PoseId } from './poses'

export type AvatarText = {
  title: string
  blurb: string
  onScreen: string
  onScreenDesc: string
  voice: string
  voiceDesc: string
  /** The voice row's notice: the engine cannot run (its own reason follows). */
  voiceNoEngine: string
  /** No installed voice speaks the UI language. */
  voiceNoLang: string
  voiceChecking: string
  voiceSettings: string
  voiceTry: string
  voiceTryText: string
  greet: string
  greetDesc: string
  preview: string
  shell: string
  hue: string
  top: string
  face: string
  reset: string
  agentsNote: string
  /** A pose appended to POSE without a word here shows its id until it gets one. */
  poses: Partial<Record<PoseId, string>>
  /** The stage's callouts: what each part is, and what its panel changes. */
  parts: { top: string; shell: string; hue: string; face: string }
  mainNote: string
  personas: string
  personasNote: string
  persona: string
  personaEmpty: string
  use: string
  save: string
  clear: string
  worn: string
}

const TEXT: Record<string, AvatarText> = {
  th: {
    title: 'อวตาร',
    blurb: 'ตัวผู้ช่วยที่นั่งอยู่บนจอ — เลือกสีตัว สี accent ไฟบนหัว และหน้าประจำตัว ทุกอย่างวาดจากโค้ด ไม่มีไฟล์ภาพ',
    onScreen: 'ผู้ช่วยบนจอ',
    onScreenDesc: 'ตัวลอยที่ลากไปวางตรงไหนก็ได้ พูดเฉพาะตอนรายงาน ปิดได้จากที่นี่หรือจากปุ่ม × บนตัว',
    voice: 'พูดออกเสียง',
    voiceDesc: 'อ่านคำตอบของห้องที่เปิดอยู่ และคำถามที่มันติดอยู่ ด้วยเสียงของเครื่อง (ตั้งค่า › เสียง) — ระหว่างทำงานยาวจะเงียบจนกว่าจะรายงาน กดที่ตัวมันเพื่อหยุดพูด',
    voiceNoEngine: 'ตอนนี้พูดไม่ได้ จะเงียบไว้ก่อน —',
    voiceNoLang: 'ในเครื่องยังไม่มีเสียงสำหรับภาษาที่ใช้อยู่ จะอ่านด้วยเสียงอื่นหรือเงียบไป — เพิ่มเสียงหรือเปลี่ยนเอนจินได้ที่',
    voiceChecking: 'กำลังตรวจเสียงในเครื่อง…',
    voiceSettings: 'ตั้งค่า › เสียง',
    voiceTry: 'ลองพูด',
    voiceTryText: 'สวัสดีครับ ผมคือผู้ช่วยของคุณ ถ้าได้ยินแบบนี้แปลว่าพูดได้แล้ว',
    greet: 'ทักทายด้วยเสียง',
    greetDesc: 'พูดคำทักของห้องตอนเปิดแชทใหม่หรือสลับโต๊ะ — ปิดแล้วยังทักในฟองข้อความเหมือนเดิม',
    preview: 'ตัวอย่าง',
    shell: 'สีตัว',
    hue: 'สี accent',
    top: 'ไฟบนหัว',
    face: 'หน้าประจำตัว',
    reset: 'ค่าเริ่มต้น',
    agentsNote: 'เอเจนและซับเอเจนยังใช้หน้าแบบเดิม — จะย้ายมาใช้ตัวมาสคอตในรอบถัดไป',
    poses: {
      idle: 'พัก', greeting: 'ทักทาย', thinking: 'คิด', typing: 'พิมพ์', reading: 'อ่าน', research: 'ค้นเว็บ', searchDocs: 'ค้นเอกสาร',
      searchData: 'ค้นข้อมูล', searchFiles: 'ค้นไฟล์', answering: 'ตอบ', asking: 'ถาม', planning: 'วางแผน', coding: 'เขียนโค้ด',
      debugging: 'ดีบัก', presenting: 'นำเสนอ', helping: 'ช่วย', success: 'เสร็จ', recharge: 'ชาร์จ', walk: 'เดิน', listening: 'ฟัง',
      cheer: 'เชียร์', wink: 'ขยิบตา', error: 'ผิดพลาด', wake: 'ตื่น', startled: 'ตกใจ',
    },
    parts: { top: 'ไฟบนหัว — สัญญาณบนยอด', shell: 'ตัว — วัสดุของหัว ลำตัว แขน ขา', hue: 'accent — หมวก หู พื้นรองเท้า และแสงบนจอ · ขาวดำคือค่าเริ่มต้น โทนเดียวกับโลโก้', face: 'หน้า — แสงบนจอตอนพัก' },
    mainNote: 'นี่คืออวตารหลักของ Aetox — ตัวเดียวกันทุกโต๊ะ ทุกหน้า สิ่งที่เลือกที่นี่คือผู้ช่วยของคุณ',
    personas: 'บุคลิก',
    personasNote: 'บันทึกชุดที่ชอบไว้ได้ 6 ชุด สลับใช้ได้ทันที — และเป็นชุดที่จะนำไปใส่ให้เอเจนที่คุณออกแบบเองในอนาคต',
    persona: 'บุคลิก',
    personaEmpty: 'ว่าง',
    use: 'ใช้',
    save: 'บันทึกชุดนี้',
    clear: 'ล้าง',
    worn: 'ใช้อยู่',
  },
  en: {
    title: 'Avatar',
    blurb: 'The assistant that sits on your screen — pick its finish, accent, top light and resting face. Drawn from code; no image files.',
    onScreen: 'Assistant on screen',
    onScreenDesc: 'A floating figure you can drag anywhere. It speaks only when it reports. Turn it off here or with the × on it.',
    voice: 'Speak aloud',
    voiceDesc: "Reads the open chat's answer, and a question it is stuck on, with this machine's voice (Settings › Voice). A long run stays quiet until it reports. Click the figure to stop it.",
    voiceNoEngine: "Can't speak right now, so it stays quiet —",
    voiceNoLang: 'No installed voice speaks the current language; it will read with another voice or stay quiet — add one or change engine under',
    voiceChecking: 'Checking the voices on this machine…',
    voiceSettings: 'Settings › Voice',
    voiceTry: 'Try it',
    voiceTryText: 'Hello, I am your assistant. If you can hear this, I can speak.',
    greet: 'Say hello aloud',
    greetDesc: "Speaks the room's greeting when a new chat opens or you switch desks — off, it still greets in the bubble.",
    preview: 'Preview',
    shell: 'Finish',
    hue: 'Accent',
    top: 'Top light',
    face: 'Resting face',
    reset: 'Defaults',
    agentsNote: 'Agents and sub-agents still wear the old faces — the mascot comes to them next.',
    poses: {
      idle: 'Rest', greeting: 'Greet', thinking: 'Think', typing: 'Type', reading: 'Read', research: 'Web', searchDocs: 'Docs',
      searchData: 'Data', searchFiles: 'Files', answering: 'Answer', asking: 'Ask', planning: 'Plan', coding: 'Code',
      debugging: 'Debug', presenting: 'Present', helping: 'Help', success: 'Done', recharge: 'Recharge', walk: 'Walk', listening: 'Listen',
      cheer: 'Cheer', wink: 'Wink', error: 'Error', wake: 'Wake', startled: 'Startled',
    },
    parts: { top: 'Top light — the signal on the crown', shell: 'Body — the material of head, torso, arms, legs', hue: 'Accent — cap, ears, soles and the screen light · black and white is the default, the two tones of the mark', face: 'Face — the screen light at rest' },
    mainNote: "This is Aetox's main avatar — the same one on every desk and page. What you choose here is your assistant.",
    personas: 'Personas',
    personasNote: 'Keep up to six looks and switch between them — the looks you will hand to agents you design later.',
    persona: 'Persona',
    personaEmpty: 'Empty',
    use: 'Use',
    save: 'Save this look',
    clear: 'Clear',
    worn: 'Wearing',
  },
  zh: {
    title: '头像',
    blurb: '坐在屏幕上的助手——选择机身颜色、点缀色、头顶灯和默认表情。全部由代码绘制，没有图片文件。',
    onScreen: '桌面助手',
    onScreenDesc: '可拖到任意位置的小助手，只在汇报时说话。可在此处或用它身上的 × 关闭。',
    voice: '朗读',
    voiceDesc: '用本机的声音（设置 › 语音）朗读当前会话的回答和它卡住的问题。长任务期间保持安静，直到汇报。点击它即可停止。',
    voiceNoEngine: '现在无法朗读，先保持安静 —',
    voiceNoLang: '本机没有当前语言的声音，将用其他声音朗读或保持安静 — 可在这里添加声音或更换引擎：',
    voiceChecking: '正在检查本机声音…',
    voiceSettings: '设置 › 语音',
    voiceTry: '试听',
    voiceTryText: '你好，我是你的助手。能听到就说明我可以说话了。',
    greet: '语音问候',
    greetDesc: '打开新会话或切换工作台时朗读问候语——关闭后仍会在气泡里问候。',
    preview: '预览',
    shell: '机身',
    hue: '点缀色',
    top: '头顶灯',
    face: '默认表情',
    reset: '恢复默认',
    agentsNote: '代理与子代理仍使用旧头像——下一轮再换成吉祥物。',
    poses: {
      idle: '休息', greeting: '打招呼', thinking: '思考', typing: '输入', reading: '阅读', research: '搜网页', searchDocs: '查文档',
      searchData: '查数据', searchFiles: '找文件', answering: '回答', asking: '提问', planning: '规划', coding: '编码',
      debugging: '调试', presenting: '演示', helping: '帮忙', success: '完成', recharge: '充电', walk: '行走', listening: '倾听',
      cheer: '欢呼', wink: '眨眼', error: '出错', wake: '醒来', startled: '惊醒',
    },
    parts: { top: '头顶灯——顶部的信号', shell: '机身——头、躯干、手臂、腿的材质', hue: '点缀色——帽子、耳朵、鞋底和屏幕光 · 黑白为默认，与标志同色调', face: '表情——休息时的屏幕光' },
    mainNote: '这是 Aetox 的主头像——每个工作台、每个页面都是同一个。在这里选择的就是你的助手。',
    personas: '角色',
    personasNote: '最多保存六套外观随时切换——将来也可以交给你自己设计的代理。',
    persona: '角色',
    personaEmpty: '空',
    use: '使用',
    save: '保存此外观',
    clear: '清除',
    worn: '使用中',
  },
}

export function avatarText(locale: string): AvatarText {
  return TEXT[locale] ?? TEXT.en
}
