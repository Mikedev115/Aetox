// Assessment of dangerous or sensitive files before Git staging and commit.

export interface DangerousFileAssessment {
  path: string
  reason: string
  category: 'secret' | 'key' | 'credentials' | 'database' | 'binary'
}

export function assessDangerousFile(path: string): DangerousFileAssessment | null {
  const filename = path.split(/[/\\]/).pop()?.toLowerCase() ?? ''
  const lower = path.toLowerCase()

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
