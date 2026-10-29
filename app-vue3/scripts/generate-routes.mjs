#!/usr/bin/env node
// Runs automatically before `npm run dev|build|typecheck` (see package.json pre* scripts).
// migrated-routes.json  ->  src/generated/migrated-routes.json
import { mkdirSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { loadRoutes, toAppManifest } from './routes-lib.mjs'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const routes = loadRoutes(resolve(root, 'migrated-routes.json'))
const out = resolve(root, 'src/generated/migrated-routes.json')
mkdirSync(dirname(out), { recursive: true })
writeFileSync(out, toAppManifest(routes))
console.log(`generated src/generated/migrated-routes.json (${routes.length} routes)`)
