<script lang="ts">
  // The account menu's row for the assistant on the screen: on or off.
  //
  // One row in the menu the owner pointed at (12 ก.ย.: "หน้าตั้งค่า เพิ่มในส่วนนี้
  // ด้วยนะ จะได้กดได้และเลือกได้"), drawn with the app's standard switch
  // (.mswitch — a real checkbox in a label, the same one ทีมเอเจน uses). The
  // short explanation is part of the row now: "ผู้ช่วยบนจอ" says what it is,
  // while the second line says what switching it on will actually put there.
  import Icon from '../Icon.svelte'
  import { t } from '../i18n.svelte'
  import { companion, setCompanionOn } from './companionSetting.svelte'
</script>

<div class="plus-menu-item profile-card-row companion-row" data-guide="account.companion">
  <span class="ic"><Icon name="bot" size={14} /></span>
  <span class="profile-item-copy">
    <span class="profile-item-label">{t('account.companion')}</span>
    <span class="profile-item-desc">{t('account.companionDesc')}</span>
  </span>
  <label class="mswitch">
    <input type="checkbox" checked={companion.on} aria-label={t('account.companion')} onchange={(e) => setCompanionOn(e.currentTarget.checked)} />
    <span></span>
  </label>
</div>

<style>
  .companion-row { display: flex; align-items: center; }
  .companion-row .mswitch { margin-left: auto; }
</style>
