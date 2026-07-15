#!/usr/bin/env node
/*
 * Scans lockfiles for npm packages known to have been hijacked.
 * Exits non-zero when any compromised name@version is present.
 *
 * Run locally:   node scripts/check-poisoned-deps.mjs
 * Run in CI:     same command; fail the job on non-zero exit.
 *
 * Update KNOWN_POISONED when new supply-chain incidents are disclosed.
 */
import fs from 'node:fs'
import path from 'node:path'

// name -> Set<string> of poisoned exact versions.
// Sources: GitHub advisories, Socket, Snyk, npm security working group.
const KNOWN_POISONED = {
  // 2025-09-08  chalk/debug maintainer phish
  chalk:                  new Set(['5.6.1', '5.6.2']),
  debug:                  new Set(['4.4.2']),
  'ansi-styles':          new Set(['6.2.2']),
  'strip-ansi':           new Set(['7.1.1']),
  'supports-color':       new Set(['10.2.1']),
  'color-convert':        new Set(['3.1.1']),
  'color-name':           new Set(['2.0.1']),
  'is-arrayish':          new Set(['0.3.3']),
  'error-ex':             new Set(['1.3.3']),
  'simple-swizzle':       new Set(['0.2.3']),
  'has-ansi':             new Set(['6.0.1']),
  'chalk-template':       new Set(['1.1.1']),
  'ansi-regex':           new Set(['6.2.1']),
  'wrap-ansi':            new Set(['9.0.1']),
  backslash:              new Set(['0.2.1']),
  color:                  new Set(['5.0.1']),
  'color-string':         new Set(['2.1.1']),
  'supports-hyperlinks':  new Set(['4.1.1']),

  // 2025  Shai-Hulud / s1ngularity worm
  '@ctrl/tinycolor':      new Set(['4.1.1', '4.1.2']),
  'rxnt-authentication':  new Set(['0.0.6']),
  'rxnt-healthchecks-nestjs': new Set(['1.0.5']),

  // 2023  Lottie player
  '@lottiefiles/lottie-player': new Set(['2.0.4','2.0.5','2.0.6','2.0.7']),

  // 2021  ua-parser-js
  'ua-parser-js':         new Set(['0.7.29','0.8.0','1.0.0']),

  // 2018  event-stream  (kept for posterity)
  'event-stream':         new Set(['3.3.6']),
  'flatmap-stream':       new Set(['0.1.1','0.1.2']),
}

const REPO_ROOT = path.resolve(path.dirname(new URL(import.meta.url).pathname), '..')

const LOCKFILES = [
  'web/package-lock.json',
  'web/classic/bun.lock',
  'web/default/bun.lock',
  'web/hai/bun.lock',
  'web/bun.lock',
  'web/classic/package-lock.json',
  'web/default/package-lock.json',
  'web/hai/package-lock.json',
  'package-lock.json',
  'bun.lock',
]

function scanPackageLock(file, body) {
  const data = JSON.parse(body)
  const hits = []
  const pkgs = data.packages || {}
  for (const [p, info] of Object.entries(pkgs)) {
    if (!info || typeof info !== 'object') continue
    const name = info.name || (p.includes('node_modules/') ? p.split('node_modules/').pop() : p)
    if (!name) continue
    const ver = info.version
    if (KNOWN_POISONED[name]?.has(ver)) hits.push({ name, ver, where: p })
  }
  // npm v6 layout fallback
  function walkDeps(deps, prefix) {
    for (const [n, info] of Object.entries(deps || {})) {
      if (KNOWN_POISONED[n]?.has(info.version)) {
        hits.push({ name: n, ver: info.version, where: `${prefix}/${n}` })
      }
      if (info.dependencies) walkDeps(info.dependencies, `${prefix}/${n}`)
    }
  }
  walkDeps(data.dependencies, file)
  return hits
}

function scanBunLock(file, body) {
  const hits = []

  function addHit(name, ver, where) {
    if (KNOWN_POISONED[name]?.has(ver)) hits.push({ name, ver, where })
  }

  // Fast regex fallback works even when bun.lock contains unescaped control chars.
  for (const [name, versions] of Object.entries(KNOWN_POISONED)) {
    const escaped = name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    for (const ver of versions) {
      const re = new RegExp(`(?:^|["\\s])${escaped}@${ver.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}(?:["\\s,]|$)`, 'gm')
      if (re.test(body)) addHit(name, ver, file)
    }
  }

  // bun.lock is JSONC-ish in simple cases; parse it for more precise locations.
  const cleaned = body
    .replace(/\/\/.*$/gm, '')
    .replace(/,(\s*[}\]])/g, '$1')
  let data
  try {
    data = JSON.parse(cleaned)
  } catch (err) {
    if (hits.length === 0) console.warn(`[warn] could not parse ${file}: ${err.message}`)
    return dedupeHits(hits)
  }

  // bun.lock packages section: { "name": [ "name@ver", "", {...meta...}, "sha512-..." ] }
  for (const [key, arr] of Object.entries(data.packages || {})) {
    if (!Array.isArray(arr) || arr.length === 0) continue
    const head = arr[0]
    if (typeof head !== 'string') continue
    const at = head.lastIndexOf('@')
    if (at <= 0) continue
    const name = head.slice(0, at)
    const ver = head.slice(at + 1)
    if (KNOWN_POISONED[name]?.has(ver)) hits.push({ name, ver, where: `${file}::${key}` })
  }
  return dedupeHits(hits)
}

function dedupeHits(hits) {
  const seen = new Set()
  return hits.filter((hit) => {
    const key = `${hit.name}@${hit.ver}@${hit.where}`
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
}

let totalHits = 0
let scannedAny = false

for (const rel of LOCKFILES) {
  const abs = path.join(REPO_ROOT, rel)
  if (!fs.existsSync(abs)) continue
  scannedAny = true
  const body = fs.readFileSync(abs, 'utf8')
  const hits = rel.endsWith('package-lock.json')
    ? scanPackageLock(abs, body)
    : scanBunLock(abs, body)
  if (hits.length === 0) {
    console.log(`[ok]   ${rel}`)
  } else {
    totalHits += hits.length
    console.log(`[FAIL] ${rel}  (${hits.length} hit${hits.length > 1 ? 's' : ''})`)
    for (const h of hits) console.log(`         ${h.name}@${h.ver}  at  ${h.where}`)
  }
}

if (!scannedAny) {
  console.error('No lockfiles found to scan.')
  process.exit(2)
}

if (totalHits > 0) {
  console.error(`\n${totalHits} poisoned package version${totalHits > 1 ? 's' : ''} detected.`)
  console.error('Run a clean reinstall and verify upstream packages have been updated to safe versions.')
  process.exit(1)
}

console.log('\nNo known-poisoned package versions detected.')
