// The avatar page's words, in the three UI languages.
//
// TEMPORARY HOME. These belong in locales/th.ts, en.ts and zh.ts under
// `settings.avatar*`, beside every other settings string. They are here
// because those three files (and Settings.svelte's own nav) are mid-change in
// another session on 12 ก.ย. 2026, and a key added to a file somebody else is
// rewriting is a merge nobody asked for. Moving them is a paste and a
// find-replace of `avatarText(locale).x` → `t('settings.avatarX')`; the
// switch's label (CompanionSwitch.svelte) goes at the same time.

export type AvatarText = {
  title: string
  blurb: string
  onScreen: string
  onScreenDesc: string
  preview: string
  shell: string
  hue: string
  hueBrand: string
  top: string
  face: string
  reset: string
  agentsNote: string
  poses: Record<'idle' | 'greeting' | 'typing' | 'answering' | 'success', string>
  /** The stage's callouts: what each numbered part is, and what its panel changes. */
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
    preview: 'ตัวอย่าง',
    shell: 'สีตัว',
    hue: 'สี accent',
    hueBrand: 'สีแบรนด์',
    top: 'ไฟบนหัว',
    face: 'หน้าประจำตัว',
    reset: 'ค่าเริ่มต้น',
    agentsNote: 'เอเจนและซับเอเจนยังใช้หน้าแบบเดิม — จะย้ายมาใช้ตัวมาสคอตในรอบถัดไป',
    poses: { idle: 'พัก', greeting: 'ทักทาย', typing: 'พิมพ์', answering: 'ตอบ', success: 'เสร็จ' },
    parts: { top: 'ไฟบนหัว — สัญญาณบนยอด', shell: 'ตัว — วัสดุของหัว ลำตัว แขน ขา', hue: 'accent — หมวก หู พื้นรองเท้า และแสงบนจอ', face: 'หน้า — แสงบนจอตอนพัก' },
    mainNote: 'นี่คืออวตารหลักของ Aetox — ตัวเดียวกันทุกโต๊ะ ทุกหน้า สิ่งที่เลือกที่นี่คือผู้ช่วยของคุณ',
    personas: 'บุคลิก',
    personasNote: 'บันทึกชุดที่ชอบไว้ 3 ชุด สลับใช้ได้ทันที — และเป็นชุดที่จะนำไปใส่ให้เอเจนที่คุณออกแบบเองในอนาคต',
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
    preview: 'Preview',
    shell: 'Finish',
    hue: 'Accent',
    hueBrand: 'Brand',
    top: 'Top light',
    face: 'Resting face',
    reset: 'Defaults',
    agentsNote: 'Agents and sub-agents still wear the old faces — the mascot comes to them next.',
    poses: { idle: 'Rest', greeting: 'Greet', typing: 'Type', answering: 'Answer', success: 'Done' },
    parts: { top: 'Top light — the signal on the crown', shell: 'Body — the material of head, torso, arms, legs', hue: 'Accent — cap, ears, soles and the screen light', face: 'Face — the screen light at rest' },
    mainNote: "This is Aetox's main avatar — the same one on every desk and page. What you choose here is your assistant.",
    personas: 'Personas',
    personasNote: 'Keep three looks and switch between them — the looks you will hand to agents you design later.',
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
    preview: '预览',
    shell: '机身',
    hue: '点缀色',
    hueBrand: '品牌色',
    top: '头顶灯',
    face: '默认表情',
    reset: '恢复默认',
    agentsNote: '代理与子代理仍使用旧头像——下一轮再换成吉祥物。',
    poses: { idle: '休息', greeting: '打招呼', typing: '输入', answering: '回答', success: '完成' },
    parts: { top: '头顶灯——顶部的信号', shell: '机身——头、躯干、手臂、腿的材质', hue: '点缀色——帽子、耳朵、鞋底和屏幕光', face: '表情——休息时的屏幕光' },
    mainNote: '这是 Aetox 的主头像——每个工作台、每个页面都是同一个。在这里选择的就是你的助手。',
    personas: '角色',
    personasNote: '保存三套外观随时切换——将来也可以交给你自己设计的代理。',
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
