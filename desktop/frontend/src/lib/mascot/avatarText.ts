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
  },
}

export function avatarText(locale: string): AvatarText {
  return TEXT[locale] ?? TEXT.en
}
