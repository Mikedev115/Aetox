// Assessment of dangerous or sensitive files before Git staging and commit.

export interface DangerousFileAssessment {
  path: string
  reason: string
  // 'app': not dangerous, but not the project's either — what the chat saved
  // when the user attached a file. Never ticked, never proposed, drawn with
  // its own quiet badge rather than the warning one.
  category: 'secret' | 'key' | 'credentials' | 'database' | 'binary' | 'app'
}

/** The folder the chat writes attachments into, under the project root
 * (desktop/app.go attachmentsDir). */
export const APP_ATTACHMENTS_DIR = '.aetox-attachments'

export function assessDangerousFile(path: string): DangerousFileAssessment | null {
  const filename = path.split(/[/\\]/).pop()?.toLowerCase() ?? ''
  const lower = path.toLowerCase()

  // 0. The app's own attachments — a screenshot pasted into the chat lands
  //    here, and thirteen of them rode into commits before this rule (owner,
  //    12 ก.ย.: "พวกนี้คืออะไร").
  if (path.split(/[/\\]/).includes(APP_ATTACHMENTS_DIR)) {
    return { path, reason: 'Aetox chat attachment (the app\'s, not the project\'s)', category: 'app' }
  }

  // 1. Environment & Secret files
  if (
    filename === '.env' ||
    (filename.startsWith('.env.') &&
      !filename.endsWith('.example') &&
      !filename.endsWith('.sample') &&
      !filename.endsWith('.template')) ||
    filename.endsWith('.env')
  ) {
    return { path, reason: 'Environment / Secrets (.env)', category: 'secret' }
  }

  // 2. Private Keys & Certificates
  if (
    /\.(pem|key|pkcs12|pfx|p12)$/i.test(filename) ||
    /^(id_rsa|id_dsa|id_ecdsa|id_ed25519)$/i.test(filename)
  ) {
    return { path, reason: 'Private Key / Certificate', category: 'key' }
  }

  // 3. Credentials & Tokens
  if (
    /^(credentials|service-account|service_account|secret|secrets|token|auth)\.(json|yaml|yml|txt)$/i.test(filename) ||
    filename === '.npmrc' ||
    filename === '.pypirc' ||
    lower.includes('.aws/credentials')
  ) {
    return { path, reason: 'Credentials / Access Tokens', category: 'credentials' }
  }

  // 4. Database dumps & SQLite
  if (/\.(sqlite|sqlite3|db|dump|sql\.gz|sql\.bak)$/i.test(filename)) {
    return { path, reason: 'Database Storage / Dump', category: 'database' }
  }

  // 5. Executables & Binaries
  if (/\.(exe|dll|so|dylib|bin)$/i.test(filename)) {
    return { path, reason: 'Executable Binary', category: 'binary' }
  }

  return null
}
