# LibreDental — Pre-Release Checklist

Run through this on a dedicated branch before every (weekly beta) release. Track progress
in the release PR description using the template at the bottom; GitHub generates the
release notes from merged PR titles, so the PR title and description are the changelog.

---

## Categories

### 0. Tooling

Everything below assumes the same toolchain CI uses:

```bash
go install github.com/go-task/task/v3/cmd/task@v3.41.0
task deps:sync-wails        # wails3 CLI + @wailsio/runtime pinned to go.mod's Wails version
cd frontend && npm ci && cd ..
wails3 generate bindings -ts -i
```

Atlas is needed for §2 (`curl -sSf https://atlasgo.sh | sh`). If that host is blocked, build
the community edition from source: `git clone https://github.com/ariga/atlas && cd
atlas/cmd/atlas && go build -o "$(go env GOPATH)/bin/atlas" .`. Do not use
`go install ariga.io/atlas/cmd/atlas@latest`: it resolves to a 2023 release that cannot
read the current migration directories.

---

### 1. Internationalization (i18n) — Frontend String Audit

All user-visible strings must go through Paraglide (`frontend/messages/en.json`), including
attribute values (`title`, `placeholder`, `aria-label`, `alt`, `subtitle`, `helpText`),
string literals inside template expressions (`{x ? "Edit Claim" : ...}`), component prop
defaults, and error messages assigned in `<script>`.

**How to find violations:**

```bash
cd frontend/src

# Files with no i18n import at all (components that only render props are fine)
for f in $(find . -name "*.svelte" -not -path "./paraglide/*"); do
  grep -q "paraglide/messages" "$f" || echo "$f"
done

# Literal UI attributes
grep -rnE '(title|placeholder|subtitle|label|aria-label|alt|helpText|message)="[^"{]*[A-Za-z]{2,}' \
  --include=*.svelte . | grep -v paraglide

# Capitalized string literals in scripts and template expressions
grep -rnE '(=|\|\||\?\?|:|\(|return)\s*["`'"'"'][A-Z][a-z]+[ ,.!?:][^"`'"'"']*["`'"'"']' \
  --include=*.svelte --include=*.ts . | grep -vE 'paraglide|console\.|import |e\.key|case "'
```

Also look at text nodes spanning several lines in templates (button labels, empty states),
which the greps above miss.

**Stored enum values are not display text.** Roles, payment methods, claim and condition
statuses must render through the helpers in `src/lib/labels.ts`, never via
`.replace("_", " ")` or CSS `capitalize`. Check `StatusBadge` usages pass a `label`.

**Unused keys:** after edits, remove keys nothing references:

```bash
cd frontend && python3 -c "
import json,re,subprocess
d=json.load(open('messages/en.json'))
src=subprocess.run(['grep','-rhoE',r'm\.[a-zA-Z0-9_]+','src','--include=*.svelte','--include=*.ts','--exclude-dir=paraglide'],capture_output=True,text=True).stdout
used={x[2:] for x in src.split()}
print([k for k in d if k not in used])"
```

Recompile after changing `en.json`: `cd frontend && npm run build:i18n` (`task test` does
this too).

---

### 2. Database Schema & Migrations

Two independent trees, both managed by Atlas (`internal/storage/sqlite/atlas.hcl`): `main`
(`schema/` + `migrations/`) and `audit` (`audit_schema/` + `audit_migrations/`).

**Every release: check for drift.** The declarative schema and the migration chain must
describe the same database:

```bash
cd internal/storage/sqlite
atlas migrate validate --env main && atlas migrate validate --env audit
atlas migrate diff drift_check --env main     # must print "...synced... no changes"
atlas migrate diff drift_check --env audit
```

If `diff` writes a file, the schema files and migrations disagree. Work out which side is
the intended state before acting. Delete the generated `drift_check` file and restore
`atlas.sum` (`git checkout -- */atlas.sum`) unless the drift really is a missing migration.
Known gotcha: in SQLite `id TEXT PRIMARY KEY` allows NULL. Write `TEXT NOT NULL PRIMARY KEY`.

**Spot-check foreign keys:** `grep -n "REFERENCES" schema/*.sql`. Patient-linked
clinical, billing and document tables must use `ON DELETE RESTRICT` (or `SET NULL` for
optional links). `CASCADE` is only acceptable for configuration (currently
`fee_schedules`).

#### Baseline consolidation (only for releases that break upgrade compatibility)

> [!WARNING]
> Squashing the chain breaks every existing install. `db.go` refuses to open a database
> whose applied version predates the oldest embedded migration (`ErrIncompatibleDatabase`),
> so users get a clear error instead of a half-applied schema. Only squash when the release
> is explicitly declared non-upgradable, and say so in the release PR and notes (users must
> start from a fresh data directory).

1. Run the drift check above first. The squash must not change the schema.
2. Keep a copy of the old chain: `cp -r migrations /tmp/old_migrations`.
3. Remove the old files and regenerate:

   ```bash
   git rm migrations/*.sql
   atlas migrate hash --env main
   atlas migrate diff initial_schema --env main
   atlas migrate hash --env main
   ```

   Same for `--env audit` / `initial_audit`, but only if that tree has more than one file.

4. Prove the new baseline is equivalent to the old chain:

   ```bash
   DEV="sqlite://file?mode=memory&_fk=1"
   atlas schema diff --dev-url "$DEV" --from "file:///tmp/old_migrations?format=goose" --to "file://migrations?format=goose"
   atlas schema diff --dev-url "$DEV" --from "file://migrations?format=goose" --to "file://schema"
   ```

   Both must report "Schemas are synced". Column order may differ from the old chain
   (columns added by `ALTER TABLE` move to where the schema file declares them), which is
   harmless as long as no SQL uses `SELECT *` or positional `INSERT ... VALUES`.

5. Seed data lives in `internal/storage/seed/seed.go`, not in migrations. The `INSERT`s in
   `schema/billing.sql` are ignored by Atlas, so the squash cannot lose them.
6. `task test`, then `task demo` (the demo generator applies the new baseline from scratch).

---

### 3. Bug Hunt — Common Merge-Accumulation Issues

**Async race conditions:** any view that fires async loads on user input or prop
changes uses a request-generation guard (counter or AbortController) to discard stale
responses. A `$effect` already runs on mount, so don't also call the loader from `onMount`.

**Null/undefined returns:** Go nil slices arrive as `null`. Null-coalesce list results:
`(res?.filter(Boolean) as T[]) || []`.

**Money arithmetic:** monetary values are integer cents. Display goes through
`formatCurrency`. Forms that edit in major units convert both ways (`/ 100` on load,
`Math.round(x * 100)` on save).

**Date/timezone edge cases:** date-only values use the helpers in `src/lib/date.ts`
(`getLocalDateString`, `getDateOnlyString`, `dateOnlyToISO`). Never use
`toISOString().split("T")[0]`.

**Native dialogs:** there must be no `alert()`, `confirm()` or `prompt()`. They aren't
reliably available in every Wails webview, can't be translated, and aren't accessible. Use
`ConfirmModal` (which accepts body content for pickers) and inline `role="alert"` messages.
This must print nothing:

```bash
grep -rnE "(^|[^._a-zA-Z0-9])(confirm|alert|prompt)\(" frontend/src --include=*.svelte --include=*.ts | grep -v paraglide
```

**Silent failures:** a `catch` that only does `console.error` leaves the user with no
feedback in a desktop app. Show the error via `handleError()` from `src/lib/error.ts`.

---

### 4. Security & Audit Trail

Server mode exposes every bound service method over HTTP to the LAN, so authorization is
enforced in Go, not in the UI.

- Every exported method on a bound service that reads patient data, reads or writes
  credentials, or mutates anything must take a session `token` and check
  `auditService.GetSessionUser(token)`. List exceptions with:

  ```bash
  # Exported service methods with no session check in their body
  cd internal/services && awk '
    FNR==1 { if (n && !ok) print f": "n; n="" }
    /^func \([a-z]+ \*[A-Za-z]+Service\) [A-Z]/ { if (n && !ok) print FILENAME": "n; n=$4; sub(/\(.*/, "", n); f=FILENAME; ok=0 }
    /GetSessionUser|requireSession/ { ok=1 }
    END { if (n && !ok) print f": "n }' *_service.go
  ```

  Known intentional exceptions: login/session creation (`CreateSession`, `VerifyProviderPin`,
  `ListProviders` for the login picker, with PINs masked), first-run `CreateInitialProvider`
  (gated atomically on "no active providers"), locale/theme/window settings, provider-name
  lists, and the procedure-code/bundle catalogs. `SecretsService` is not bound (check
  `main.go`), so its methods don't need a token. Known open item: `TimecardService`
  reads (staff payroll) are still ungated.
- Anything that writes to the audit log must take its user from the session, never from
  the caller. A bound method that accepts a caller-built `AuditLogEntry` lets anyone forge
  the audit trail.
- Every mutation of patient, clinical, billing, document or integration data writes an
  audit entry (`LogAction` / `LogPatientAction`). Private helpers may rely on their callers.
- Run `govulncheck ./...` (needs access to vuln.go.dev) and `cd frontend && npm audit --omit=dev`.

---

### 5. Dead Code & Deduplication

- `staticcheck ./...` (`go install honnef.co/go/tools/cmd/staticcheck@latest`) is clean.
- No inline date-to-ISO conversions (see §3). Shared formatting lives in `src/lib/`.
- The backend owns `created_at`/`updated_at`, so the frontend never stamps them.
- Error fallbacks use `handleError(e, m.some_key())`, not `e.message || "..."`.
- Unused components and `src/lib` exports are removed.

---

### 6. Test Coverage

Every service and repository has a matching `_test.go`:

```bash
for d in internal/services internal/storage/sqlite; do
  for f in $d/*.go; do
    b=$(basename "$f" .go); [[ $b == *_test ]] && continue
    [ -f "$d/${b}_test.go" ] || echo "MISSING: $f"
  done
done
```

Then run exactly what CI runs:

```bash
task format && task test
gofmt -l .                         # must be empty
go vet ./...
cd frontend && npm run format:check
```

---

### 7. Dependency Synchronization

- `go mod tidy` leaves `go.mod`/`go.sum` unchanged and `go mod verify` passes.
- `npm ci` succeeds (lockfile in sync with `package.json`).
- The Wails Go module, `wails3` CLI and `@wailsio/runtime` are the same version
  (`task deps:sync-wails`).

---

### 8. Accessibility & UX Polish

- Every form control has a `<label for>` or, when the visible label is a column header or
  inline text, an `aria-label`.
- Icon-only buttons (✕, trash, pencil) have an `aria-label`. Controls revealed on hover
  (`opacity-0 group-hover:opacity-100`) also get `focus-visible:opacity-100`.
- Table column headers have `scope="col"`.
- Modal close buttons have descriptive `aria-label`s (built into `Modal.svelte`).

---

### 9. Build & Smoke Test

Unit tests don't catch a broken bundle or a view that throws on real data.

1. `task artifact` builds the desktop binary, server binary and demo archive.
2. Unzip `dist/libredental-demo-data.zip` into a scratch config dir and run the server
   against it (`XDG_CONFIG_HOME=/tmp/ld/.config dist/libredental-server`, using an absolute
   path), then open `http://localhost:4242`.
3. Log in (demo PINs: Dr. Sarah Jenkins `1111`, Dr. Marcus Vance `2222`, Elena Rostova
   `3333`) and click through every top-level tab and sub-tab in **both dark and light
   themes**, with the browser console open. There should be no console errors and no
   unreadable text.
4. Exercise anything this release changed end-to-end (create, edit, delete) and check that
   the Audit tab recorded it.

---

### 10. Documentation & Release Notes

- Update `README.md` / `docs/` if setup steps, dependencies, env vars or deployment
  requirements changed.
- Release notes are generated from merged PR titles, so give the release PR a descriptive
  title and call out **breaking changes** (e.g. "requires a fresh data directory") in the
  PR description.
- Tag the release commit on `main`: `git tag -a vX.Y.Z -m "Release vX.Y.Z"` and push the
  tag. This triggers `.github/workflows/release.yml`.

---

## Release PR Template

```markdown
# Release Cleanup — vX.Y.Z

> See `docs/release-checklist.md` for guidance on each category.

**Breaking changes:** none / ...

- [ ] 1. i18n — string audit, enum labels, unused keys
- [ ] 2. Database — drift check clean, FK review, (baseline squash + equivalence proof if breaking)
- [ ] 3. Bug hunt — races, nulls, money, dates, native dialogs, silent failures
- [ ] 4. Security & audit — session checks on bound methods, audit coverage, vuln scans
- [ ] 5. Dead code — staticcheck, dedup
- [ ] 6. Tests — `task format`, `task test`, `gofmt -l`, `go vet`, `format:check`
- [ ] 7. Dependencies — tidy, `npm ci`, Wails versions in lockstep
- [ ] 8. Accessibility
- [ ] 9. Build & smoke test — `task artifact`, demo data, both themes
- [ ] 10. Docs updated; PR title/description ready for release notes
```
