// รู้จักกับ Aetox — whether the tour is on screen outside the wizard.
//
// The wizard plays it in its own flow (Onboarding.svelte, after the language
// is picked). This flag is the other door: ตั้งค่า › เกี่ยวกับ › ดูการแนะนำอีกครั้ง,
// which App.svelte answers by drawing the tour over everything. A module of
// its own rather than a cockpit field because the cockpit is one chat's state
// (§150) and the tour belongs to the app.
export const tourState = $state({ open: false })

export function openTour(): void {
  tourState.open = true
}
