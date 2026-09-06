// The address of a file in the open project, as the webview sees it.
//
// Served by the Go side's file host (desktop/filehost.go). A pane that points
// an <img>, <video> or <iframe> here gets the file streamed the way a browser
// streams any URL — only what is on screen is in memory — instead of carrying
// the whole thing across a binding as a base64 string. That is the difference
// between a picture with a 20MB ceiling and a video with none.

/** URL for a project-relative path, safe for any filename.
 *
 * Each segment is encoded on its own: encodeURIComponent would turn the
 * separators into %2F and the path would arrive at the server as one long
 * filename. Both separators are accepted because the paths reaching here come
 * from two places — the file tree and RelativizePath, which on Windows answers
 * with backslashes. */
export function fileURL(path: string): string {
  const parts = path.split(/[\\/]/).filter(Boolean).map(encodeURIComponent)
  return '/aetox-file/' + parts.join('/')
}

/** The project-relative path behind a file-host URL, or '' for an address that
 * is not one.
 *
 * The inverse of fileURL, and it lives beside it so the two cannot drift — the
 * prefix and the per-segment encoding are one decision, written once. What
 * needs it is the gallery in chat (markdown.ts): an <img> in an answer carries
 * an address, and whether that address is a file on the desk or a picture
 * somewhere on the web decides which pane is able to open it.
 *
 * A query or a fragment is cut before decoding. The file host answers neither,
 * so anything after them was never part of the name. */
export function filePath(url: string): string {
  const prefix = '/aetox-file/'
  if (!url.startsWith(prefix)) return ''
  const bare = url.slice(prefix.length).split(/[?#]/)[0]
  return bare.split('/').filter(Boolean).map(decodeURIComponent).join('/')
}
