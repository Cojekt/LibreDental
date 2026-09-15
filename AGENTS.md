# Agent Instructions for LibreDental

This file orients coding agents (Claude Code, etc.) working in this repo. It's a
supplement to `README.md`, not a replacement — read that too for project philosophy.

## This is medical software — act accordingly

LibreDental stores real patient health and financial data (clinical history,
insurance, billing/ledger). That changes how you should work here, not just what:

- **Prefer small, careful, reviewable changes over sweeping ones.** Don't refactor
  or "clean up" code adjacent to what you were asked to touch.
- **Never weaken or bypass the audit trail.** `internal/domain/audit.go`, the
  `audit_service`, and the separate `audit_schema`/`audit_migrations` tree exist to
  record who did what to patient/billing/clinical data and when. Any change that
  touches patient records, billing, charting, or documents should be checked against
  whether it needs a corresponding audit log entry — don't add a new mutation path
  that silently skips auditing.
- **Don't silently change data semantics.** Money is stored as integer cents; dates,
  timezones, and unit conventions matter (see `docs/release-checklist.md` for known
  footguns). If a change affects how existing stored data is interpreted, call that
  out explicitly rather than "fixing it forward" quietly.
- **Correctness and data integrity take priority over speed.** When in doubt about
  a schema change, a data migration, or anything touching stored patient/financial
  records, stop and ask rather than guessing.

## What this project is

LibreDental is an open-source, local-first dental practice management app:
patient records, scheduling, odontogram/charting, billing & insurance claims.

- **Backend:** Go 1.26, wrapped as a native desktop app via Wails v3. Can also run
  headless as an HTTP server (`task server`, port 4242) for LAN multi-client setups.
- **Frontend:** Svelte 5 + TypeScript + Vite + Tailwind v4, i18n via Paraglide.
- **Storage:** Embedded SQLite (modernc.org/sqlite, pure Go, no cgo). Migrations run
  via goose, schema/migrations authored with Atlas.

## Repo layout

```
main.go                   Entry point, Wails app wiring
internal/app/              Build-mode switching (desktop/server, dev/prod), server config
internal/domain/           Core types: patient, appointment, billing, chart, document,
                            timecard, audit, config, integrations, country
internal/services/         Business logic, one service per domain area (+ _test.go each)
internal/storage/          Repository interfaces
internal/storage/sqlite/   SQLite repo implementations + migrations/schema (main + audit)
internal/demo/             Demo data seeder (task demo) incl. sample DICOM/PDF/image fixtures
frontend/src/              Svelte app: components/, components/ui/ (design system), lib/
frontend/src/paraglide/    Generated i18n messages — do not hand-edit, regenerate instead
build/                     Wails build config, icons, platform packaging
docs/                      LAN server setup, release checklist
```

Two separate SQLite schemas/migration trees exist: the main app schema
(`internal/storage/sqlite/schema` + `migrations/`) and a separate audit-log schema
(`audit_schema` + `audit_migrations/`). Don't mix them up when adding migrations.

## Build / test / format commands

**Always prefer the `task` commands below over calling the underlying tools
yourself.** They're the project's single source of truth for how these steps are
run (flags, working directories, ordering with codegen steps like bindings/i18n
compilation) and are what CI mirrors. Don't shell out to `gofmt`, `go test`,
`npm run ...`, `wails3`, or `atlas` directly unless there's no `task` for it, or
you're deliberately running a narrower slice (e.g. `go test ./internal/services/...`
while iterating on one package).

All via [Task](https://taskfile.dev) (`Taskfile.yml` at repo root):

| Command | What it does |
| --- | --- |
| `task desktop` | Live-reload dev mode (Wails) |
| `task server` | Build + run server binary |
| `task test` | `go test ./...` + `npm run check` (frontend type/svelte check) |
| `task format` | `gofmt -w -s .` + prettier on frontend — **use this to format, always** |
| `task artifact` | Build desktop binary, server binary, and demo data archive |
| `task demo` | Generate `dist/libredental-demo-data.zip` |

**Before considering a change done: run `task format` then `task test`.** Both
must be clean. CI (`.github/workflows/ci.yml`) separately enforces `gofmt -l`
emptiness, `go vet ./...`, `go test -v ./internal/...`, frontend `format:check`,
and `svelte-check` — if you need to check CI's exact invocations, read that
workflow file rather than assuming, since it can drift from `Taskfile.yml`.

## Conventions

- Go: standard `gofmt -s` formatting; gopls is configured for `gofumpt` locally, so
  lean toward gofumpt-compatible style (no unnecessary blank lines, simplified code).
- Every service/repo file has a matching `_test.go` — keep that pattern when adding
  new services or repositories.
- Frontend: Prettier config is 100-char print width, double quotes off (`singleQuote:
  false` — i.e. double quotes), semicolons on, es5 trailing commas. Svelte files use
  the svelte parser.
- **No raw/hardcoded text in the frontend.** Every user-visible string (labels,
  buttons, placeholders, alerts, confirm dialogs, validation and error messages,
  aria-labels, etc.) must go through Paraglide — never inline English literals in
  `.svelte`/`.ts` files. To add copy: add a key to `frontend/messages/en.json`, run
  `npm run build:i18n` to regenerate `frontend/src/paraglide/`, then reference it via
  `m.<key>()`. Don't hand-edit the generated `frontend/src/paraglide/messages/`
  files directly — they're regenerated from `en.json`. This applies to new code and
  to any existing code you touch; if you notice raw strings in a file you're already
  editing, localize them as part of that change.
- No emojis in code, commit messages, or UI copy (recent commit history explicitly
  removed emojis project-wide — keep it that way).
- Comments should be sparse and explain *why*, not *what* — match the existing style
  of terse, low-comment Go and Svelte code.

## Database migrations

This is the highest-care area of the codebase: it's a medical records schema, and
real installs have real patient/billing data already applied against the existing
migration chain. Never hand-roll a migration file or SQL that you didn't verify
against Atlas — use its tooling so the generated migration matches what the
declared schema actually implies.

- Two independently versioned trees, both goose-formatted, both managed by Atlas
  (`internal/storage/sqlite/atlas.hcl` defines the `main` and `audit` environments):
  - `internal/storage/sqlite/schema/` + `migrations/` — main app schema
  - `internal/storage/sqlite/audit_schema/` + `audit_migrations/` — audit log schema
- **Normal workflow:** edit the declarative schema file(s) in `schema/` (or
  `audit_schema/`), then let Atlas generate the migration from the diff rather than
  writing the `ALTER TABLE`/`CREATE TABLE` SQL by hand:
  ```bash
  cd internal/storage/sqlite
  atlas migrate diff <description> --env main    # or --env audit
  atlas migrate hash --env main                  # re-sum the directory after
  ```
- **Prefer additive, backward-compatible changes:** new nullable columns or columns
  with sensible defaults, new tables, new indexes. Avoid `DROP COLUMN`, `DROP TABLE`,
  destructive renames, or tightening constraints (`NOT NULL` on existing data, new
  `UNIQUE`/`CHECK` constraints) unless you've confirmed how existing installed
  databases with real data will be migrated — these can silently break upgrades or
  destroy data for existing installs. If a destructive change seems necessary, flag
  it explicitly and prefer an additive/deprecate-then-remove path across releases
  instead.
- Migration filenames follow `YYYYMMDDHHMMSS_description.sql` — Atlas stamps this
  for you when you use `migrate diff`.
- `atlas.sum` files are checksums Atlas manages — don't hand-edit; `atlas migrate
  hash` regenerates them.
- Baseline consolidation (squashing the whole migration chain into one file) is a
  deliberate, rare, release-time operation — see `docs/release-checklist.md` §2. It
  is destructive for any existing installed database and is **not** something to do
  as a side effect of a routine schema change.

## Gotchas

- `frontend/bindings` and `frontend/dist` are generated by Wails/Vite — don't hand-edit.
- `dist/` at repo root holds build artifacts (binaries, demo zip) — not source.
- Server mode and desktop mode share the same Go backend but differ via build tags
  (`server`, `dev`) — check `internal/app/build_dev.go` / `build_server.go` /
  `build_mode.go` before assuming code paths are shared.
- The demo seeder (`internal/demo/`) ships real sample binary fixtures (DICOM, PDF,
  images) — treat these as test fixtures, not something to regenerate casually.
