// Deterministic answers for the guide's prepared command chips.
//
// These buttons are part of the product map, not free-form questions. Sending
// them to the model made the same button answer differently from one press to
// the next and let a small model invent details about a screen it could not
// see. The map already knows the target, its page, and its action policy, so it
// is the right (and instant) source of truth.

import { t } from '../i18n.svelte'
import { pageDetail } from './greeting'
import { CATALOG_TARGETS } from './catalog/data'
import { guideEntry, guideText } from './map'

export type GuidePresetAnswer = 'explain' | 'common' | 'recommend'

function join(parts: string[]): string {
  return parts.map((part) => part.trim()).filter(Boolean).join('\n\n')
}

export function presetAnswer(kind: GuidePresetAnswer, targetId: string): string {
  const fixed = CATALOG_TARGETS.find((candidate) => candidate.id === targetId)
  const dynamic = fixed ? null : guideEntry(targetId)
  const entry = fixed ?? dynamic
  if (!entry) return t('guide.command.answerUnavailable' as never)

  const name = guideText(targetId, 'name') || targetId
  const what = guideText(targetId, 'what')
  const common = guideText(targetId, 'common')
  const recommend = guideText(targetId, 'recommend')
  const page = pageDetail(entry.page)

  if (kind === 'explain') {
    return join([
      `**${name}**`,
      what,
    ]) || t('guide.command.answerUnavailable' as never)
  }

  if (kind === 'common') {
    return common || page.use || what || t('guide.command.commonFallback' as never, { name })
  }

  if (recommend) return recommend

  const policy = fixed?.policy ?? { actionType: dynamic!.actionType, safe: dynamic!.safe }
  const actionKey = policy.actionType === 'focus'
    ? 'guide.command.recommendFocus'
    : policy.actionType === 'none'
      ? 'guide.command.recommendRead'
      : policy.safe
        ? 'guide.command.recommendOpen'
        : 'guide.command.recommendConfirm'

  return join([
    t(actionKey as never, { name }),
    what,
    page.ask,
  ])
}
