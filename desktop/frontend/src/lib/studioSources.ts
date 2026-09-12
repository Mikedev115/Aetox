// Where a person can go and GET material for the studio's shelf.
//
// Links, not downloads. Aetox does not fetch any of these and does not host a
// copy: none of them carries a licence anyone here can vouch for, and one
// pack the owner brought in (12 ก.ย. 2569) held a commercial title-card
// product, memes and a folder of pirated music videos. internal/assetlib's
// package comment has the whole reasoning. So the button opens the page in
// the user's browser, they download what they have the right to, and then add
// the folder on the same Settings page. The rights stay theirs, as they do for
// every file they attach to a chat.
//
// The two Drive folders are the ones the ii23 Edit Kit manual
// (edit-kit.ii23.dev/manual) links under "where do I find sound effects",
// named as their own Drive pages name them. Editing this list is the way to
// change what the page recommends.
export type StudioRights = 'cc0' | 'check' | 'unknown'

export type StudioSource = {
  // A proper noun, shown as-is.
  name: string
  url: string
  // Where it lives, for the small tag beside the name — "Google Drive".
  where: string
  // One line under the name: what is inside. A locale key.
  desc: 'settings.studioSourcePeen' | 'settings.studioSourceSfx2' | 'settings.studioSourceMyinstants'
  // What the page can honestly say about the licence: cc0 is redistributable,
  // check means the site has terms the user must read, unknown means nobody
  // said — which is the case for a shared Drive folder.
  rights: StudioRights
  // Folder name a scanned shelf would carry if the user downloaded this and
  // added it, so the card can say "นำเข้าแล้ว". Empty when the download has
  // no fixed name (a site, not a pack).
  folder?: string
}

export const STUDIO_SOURCES: StudioSource[] = [
  {
    name: "peen's SFX",
    url: 'https://drive.google.com/drive/folders/1TlMwVTsEOXpCTYPuEF3PJaGupkLDzLMm',
    where: 'Google Drive',
    desc: 'settings.studioSourcePeen',
    rights: 'unknown',
    folder: "peen's SFX",
  },
  {
    name: 'SFX Sound Effects',
    url: 'https://drive.google.com/drive/folders/1oM8iV1-eOWRPM3UWzaFJ0jmBcy7dJ16v',
    where: 'Google Drive',
    desc: 'settings.studioSourceSfx2',
    rights: 'unknown',
    folder: 'SFX Sound Effects',
  },
  {
    name: 'myinstants',
    url: 'https://www.myinstants.com/en/index/us/',
    where: 'myinstants.com',
    desc: 'settings.studioSourceMyinstants',
    rights: 'check',
  },
]
