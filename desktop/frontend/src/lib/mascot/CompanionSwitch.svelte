<script lang="ts">
  // The account menu's row for the assistant on the screen: on or off.
  //
  // One row in the menu the owner pointed at (12 ก.ย.: "หน้าตั้งค่า เพิ่มในส่วนนี้
  // ด้วยนะ จะได้กดได้และเลือกได้"), drawn with the app's standard switch
  // (.mswitch — a real checkbox in a label, the same one ทีมเอเจน uses).
  //
  // The label is spelled here rather than in locales/*.ts because those files
  // are mid-change in another session today; the three strings move there the
  // moment they are free (see companionSetting.svelte.ts for the same reason on the
  // setting's storage).
  import Icon from '../Icon.svelte'
  import { i18n } from '../i18n.svelte'
  import { companion, setCompanionOn } from './companionSetting.svelte'

  const LABEL: Record<string, string> = { th: 'ผู้ช่วยบนจอ', en: 'Assistant on screen', zh: '桌面助手' }
  const label = $derived(LABEL[i18n.locale] ?? LABEL.en)
</script>

<div class="plus-menu-item companion-row">
  <span class="ic"><Icon name="bot" size={14} /></span>
  {label}
  <label class="mswitch">
    <input type="checkbox" checked={companion.on} aria-label={label} onchange={(e) => setCompanionOn(e.currentTarget.checked)} />
    <span></span>
  </label>
</div>

<style>
  .companion-row { display: flex; align-items: center; gap: 8px; }
  .companion-row .mswitch { margin-left: auto; }
</style>
