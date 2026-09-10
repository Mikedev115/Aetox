import { describe, it, expect } from 'vitest'
import { assessDangerousFile } from '../lib/workbench/gitSecurity'

describe('gitSecurity: assessDangerousFile', () => {
  it('detects .env and env variants as secrets', () => {
    expect(assessDangerousFile('.env')).toEqual({
      path: '.env',
      reason: 'Environment / Secrets (.env)',
      category: 'secret',
    })
    expect(assessDangerousFile('config/.env.local')).toEqual({
      path: 'config/.env.local',
      reason: 'Environment / Secrets (.env)',
      category: 'secret',
    })
    expect(assessDangerousFile('backend/.env.production')).toEqual({
      path: 'backend/.env.production',
      reason: 'Environment / Secrets (.env)',
      category: 'secret',
    })
  })

  it('allows safe template files like .env.example, .env.sample, .env.template', () => {
    expect(assessDangerousFile('.env.example')).toBeNull()
    expect(assessDangerousFile('server/.env.sample')).toBeNull()
    expect(assessDangerousFile('.env.template')).toBeNull()
  })

  it('detects private keys and certificates', () => {
    expect(assessDangerousFile('id_rsa')).toEqual({
      path: 'id_rsa',
      reason: 'Private Key / Certificate',
      category: 'key',
    })
    expect(assessDangerousFile('keys/server.key')).toEqual({
      path: 'keys/server.key',
      reason: 'Private Key / Certificate',
      category: 'key',
    })
    expect(assessDangerousFile('certs/cert.pem')).toEqual({
      path: 'certs/cert.pem',
      reason: 'Private Key / Certificate',
      category: 'key',
    })
    expect(assessDangerousFile('client.p12')).toEqual({
      path: 'client.p12',
      reason: 'Private Key / Certificate',
      category: 'key',
    })
  })

  it('detects credential and token files', () => {
    expect(assessDangerousFile('credentials.json')).toEqual({
      path: 'credentials.json',
      reason: 'Credentials / Access Tokens',
      category: 'credentials',
    })
    expect(assessDangerousFile('config/service-account.json')).toEqual({
      path: 'config/service-account.json',
      reason: 'Credentials / Access Tokens',
      category: 'credentials',
    })
    expect(assessDangerousFile('.npmrc')).toEqual({
      path: '.npmrc',
      reason: 'Credentials / Access Tokens',
      category: 'credentials',
    })
    expect(assessDangerousFile('home/.aws/credentials')).toEqual({
      path: 'home/.aws/credentials',
      reason: 'Credentials / Access Tokens',
      category: 'credentials',
    })
  })

  it('detects database files and dumps', () => {
    expect(assessDangerousFile('data/app.db')).toEqual({
      path: 'data/app.db',
      reason: 'Database Storage / Dump',
      category: 'database',
    })
    expect(assessDangerousFile('local.sqlite3')).toEqual({
      path: 'local.sqlite3',
      reason: 'Database Storage / Dump',
      category: 'database',
    })
    expect(assessDangerousFile('backup.dump')).toEqual({
      path: 'backup.dump',
      reason: 'Database Storage / Dump',
      category: 'database',
    })
  })

  it('detects executable binary files', () => {
    expect(assessDangerousFile('bin/tool.exe')).toEqual({
      path: 'bin/tool.exe',
      reason: 'Executable Binary',
      category: 'binary',
    })
    expect(assessDangerousFile('lib/native.dll')).toEqual({
      path: 'lib/native.dll',
      reason: 'Executable Binary',
      category: 'binary',
    })
    expect(assessDangerousFile('libplugin.so')).toEqual({
      path: 'libplugin.so',
      reason: 'Executable Binary',
      category: 'binary',
    })
  })

  it('returns null for normal source and doc files', () => {
    expect(assessDangerousFile('src/main.ts')).toBeNull()
    expect(assessDangerousFile('README.md')).toBeNull()
    expect(assessDangerousFile('package.json')).toBeNull()
    expect(assessDangerousFile('cmd/server/main.go')).toBeNull()
  })
})
