# ApplianceHub POS: Nuxt 2 → Vue 3 strangler-fig PoC

Runs the architecture from [`MIGRATION_PLAN.md`](./MIGRATION_PLAN.md) locally with Docker: an **nginx gateway** in front of a
**legacy Nuxt 2 (SSR)** app, a **new Vue 3** app and a **Go API with JWT auth**, all on the *same origin and URLs*.

```text
                    ┌────────────── gateway (nginx) :8080 ──────────────┐
 browser ──────────▶│  /api/*              → backend (Go, JWT)          │
                    │  default             → nuxt2  (Nuxt 2, SSR)       │
                    │  migrated-paths.conf → vue3   (Vite / Vue 3 SPA)  │
                    │  /assets/*           → vue3                       │
                    └───────────────────────────────────────────────────┘
        nuxt2 SSR ──(forwards the user's cookie)──▶ backend :8080  (internal)
```

## Run

```bash
docker compose up --build        # first build takes a few minutes (Nuxt 2 + Go modules)
open http://localhost:8080       # redirects to /login
```

That is the only command: nothing to install or generate first. The gateway starts last, once the API, legacy and Vue 3
containers are healthy. If port 8080 is taken: `GATEWAY_PORT=8090 docker compose up --build`.

### Demo login

| Username | Password | Role | Can do |
|---|---|---|---|
| `cashier` | `cashier-demo-pass` | cashier | POS, products, orders; settings are read-only |
| `manager` | `manager-demo-pass` | manager | everything, including editing settings |

These are demo accounts seeded from [`backend/seed/users.json`](./backend/seed/users.json). Never reuse them anywhere real.

| URL | Purpose |
|---|---|
| http://localhost:8080 | **The only published port.** Everything (both apps and the API) goes through the gateway |

Requirements: Docker with Compose v2. Node 18+ on the host is only needed for the route generator.

## What to try

1. Open `/pos` → redirected to `/login?redirect=%2Fpos` (legacy SSR checked your cookie). Sign in as **cashier**; you return to `/pos`.
2. Add appliances, then click **Add protection plans →**. The URL changes to `/settings/protection-plan`, served by **Vue 3**
   (green badge). It was a full page load; your session (cookie) and cart (server) came with you. Pick a plan: the **API** prices it.
3. **← Back to POS**: legacy shows the plan and the new total. **Pay**: receipt, stock drops, order appears in **Orders**, dashboard updates.
4. **Roles:** as cashier, `/settings` (Vue 3) is read-only; a forced `PUT /api/settings` returns 403. Sign in as **manager** to edit.
5. **Sign out** in either app: both are signed out (cookies cleared, refresh token revoked server-side); other open tabs go to `/login`.
6. **Rollback:** `document.cookie = "ui=legacy; path=/"`, reload `/settings`: same URL, legacy page, same API, same session.
7. **Token expiry:** `ACCESS_TTL=10s docker compose up -d backend`, sign in, wait 10 s, then act: the call gets 401, the client silently
   refreshes and replays it. A hard reload of an SSR page after expiry bounces through `/login`, which silently refreshes and returns you.

## Auth design (the PoC of plan §6.1)

| Piece | Implementation |
|---|---|
| Access token | JWT **HS256**, 15 min (`ACCESS_TTL`), in an `HttpOnly; SameSite=Lax` cookie `pos_session`. Also accepted as `Authorization: Bearer` (curl/scripts) |
| Refresh | Opaque random token, stored **hashed**, **rotated on every use**, cookie `pos_refresh` scoped to `Path=/api/auth`. 30 s **grace window** so two tabs/apps refreshing at once don't kill the session; replay after the window = theft → all sessions of that user revoked |
| CSRF | Double-submit: JS-readable `pos_csrf` cookie echoed in `X-CSRF-Token` on non-GET requests (exempt when a Bearer header is used) |
| Authorization | Role → permissions resolved **server-side per request** (not from the token); `GET /api/me` bootstraps both apps |
| Hardening | `alg` pinned to HS256, issuer + expiry required, bcrypt, dummy-hash compare for unknown users, 5-failures/min login throttle, `nosniff`, `Cache-Control: no-store`, 1 MB body limit, unknown JSON fields rejected |
| Legacy SSR | `auth` middleware calls `/api/me` + `/api/cart` on the server with the **forwarded cookie** and redirects to `/login` on 401 |
| Vue 3 | `router.beforeEach` awaits `session.init()` (`/api/me`); fetch client does single-flight refresh + replay |
| Login / logout | Login page stays in legacy (plan §6.1.4); both apps share one `/login?redirect=` contract (relative paths only, open-redirect safe) |

## API

| Method & path | Permission | Notes |
|---|---|---|
| `POST /api/auth/login` | – | Sets the 3 cookies |
| `POST /api/auth/refresh`, `POST /api/auth/logout` | cookies + CSRF | Rotate / revoke |
| `GET /api/me` | any user | user, permissions, outlet |
| `GET /api/products` | `products.read` | |
| `GET/DELETE /api/cart`, `POST /api/cart/items`, `PATCH /api/cart/items/{sku}`, `PUT /api/cart/items/{sku}/plan` | `cart.use` | Server-side cart; plan prices computed by the server |
| `POST /api/orders`, `GET /api/orders` | `orders.create` / `orders.read` | Checkout re-validates stock, decrements it |
| `GET /api/dashboard` | `orders.read` | |
| `GET /api/settings`, `PUT /api/settings` | `settings.read` / `settings.write` (manager) | |

## Tests

```bash
# Backend unit/integration tests (12): cookies, CSRF, forged/expired JWT, roles, cart math, checkout,
# refresh rotation + grace + reuse detection, logout revocation, login throttle
docker run --rm -v "$PWD/backend:/src" -w /src golang:1.23-alpine go test -count=1 ./...

node shared/scripts/generate.mjs --check                 # routing files are in sync
(cd app-vue3 && npm install && npm run typecheck)         # Vue 3 type check
```

## Migrate another route (the real workflow)

```bash
# 1. build the page in app-vue3/ and add it to its router
# 2. register it in the single source of truth
$EDITOR app-vue3/migrated-routes.json        # e.g. { "path": "/products", "match": "exact" }
# 3. regenerate the gateway conf + served route list (Vue 3's own copy is made by `npm run build` automatically)
node shared/scripts/generate.mjs
# 4. rebuild and restart (the Vue 3 image runs the generator in its prebuild step)
docker compose up -d --build
```

Routing-only change (no app code), e.g. a rollback: edit `app-vue3/migrated-routes.json`, run the generator, then
`docker compose exec gateway nginx -s reload` (the generated file is bind-mounted into the gateway).
The apps also embed the manifest for link handling, so rebuild them to keep link behaviour consistent.

## Layout

```text
backend/                      Go 1.23 API (stdlib net/http + golang-jwt + bcrypt), in-memory data
  seed/users.json             demo users (embedded at build time)
  auth.go middleware.go       JWT, refresh rotation, cookies, CSRF, throttle
  store.go server.go          products, cart, orders, settings, routes
  main_test.go                12 tests
shared/
  pos.css                     design tokens (synced into both apps by the generator)
  scripts/generate.mjs        reads app-vue3/migrated-routes.json → gateway/migrated-paths.conf + gateway/migrated-routes.json (+ --check)
gateway/                      nginx: nginx.conf, proxy-common.conf, migrated-paths.conf + migrated-routes.json (generated)
legacy-nuxt2/                 Nuxt 2.18, SSR started with `npm start`, Vuex, Node 16
  plugins/api.js              axios: SSR cookie forwarding, CSRF header, silent refresh
  middleware/auth.js          session bootstrap + redirect to /login
  plugins/hard-navigation.js  full page load when navigating INTO a Vue 3 route (list fetched at runtime)
  utils/migrated.js           fetches GET /migrated-routes.json from the gateway; nothing is bundled
  pages/login.vue             login + silent sign-in (stays in legacy until late)
app-vue3/                     Vue 3 + Vite + Pinia + Vue Router 4, TypeScript
  migrated-routes.json        SINGLE SOURCE OF TRUTH (path + exact|prefix)
  scripts/generate-routes.mjs runs as pre{dev,build,typecheck} -> src/generated/migrated-routes.json (git-ignored)
  src/api/http.ts             fetch client: CSRF, single-flight refresh + replay
  src/stores/session.ts       /api/me bootstrap, permissions, logout
  src/components/AppLink.vue  RouterLink for owned routes, plain <a> otherwise
  vite.config.ts              default base "/" (build files in /assets/; page URLs unchanged)
```

## Notes and known simplifications

- **Not production-ready by design:** data is in memory (restart resets orders, stock and settings), users are seeded, the JWT secret in
  `docker-compose.yml` is a dev placeholder, `COOKIE_SECURE=false` (set `true` behind HTTPS). Real deployments inject secrets from
  AWS Secrets Manager / EB environment properties and use a database.
- HS256 with a shared secret is the simplest example. A multi-service setup would use RS256/EdDSA with a JWKS endpoint.
- Cross-tab logout (`BroadcastChannel`) and the two-app concurrent-refresh race are implemented; the race is covered by a backend test,
  the cross-tab broadcast was not exercised in a browser.
- Both apps duplicate small helpers (`money`, `menu`, `isMigrated`, auth-sync). In the real repo these belong in shared packages
  (`packages/domain`, `nav-config`, `contracts`); see plan §8.
- No service worker / offline mode here; the plan covers it (§7).
- Production-style builds only (no HMR through the gateway). For Vue 3 work run `npm run dev` in `app-vue3/` or add a dev compose override.
- `webpack@^4.46` is pinned in `legacy-nuxt2/package.json`: without it npm hoists webpack 5 and the Nuxt 2 build fails.
- Only the gateway port is published (8080 by default, `GATEWAY_PORT` to change it). The apps and the API are reachable only through it, like in production.
