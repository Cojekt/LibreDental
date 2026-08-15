# LibreDental — Pre-Release Cleanup Guide

This document describes the **categories of cleanup** to perform before every release.
For each release, copy the task template at the bottom into a new `RELEASE_CLEANUP.md`
at the repo root, fill in the specifics for that release cycle, and track progress there.

---

## Categories

### 1. Internationalization (i18n) — Frontend String Audit

All user-visible strings must go through the Paraglide message system (`messages/en.json`).
Hardcoded text in `.svelte` templates or JS fallbacks must be extracted and keyed.

**How to find violations:**

```bash
# Files with no i18n import at all
for f in $(find frontend/src -name "*.svelte"); do
  if ! grep -q "from.*paraglide\|import \* as m" "$f"; then
    echo "$f"
  fi
done

# Hardcoded confirm() dialogs
grep -rn "confirm(" frontend/src --include="*.svelte" | grep -v "m\."
```

For each file found: extract every user-visible string literal into `en.json`, replace
with the `m.key()` call, then recompile the paraglide output.

After adding keys, recompile:

```bash
cd frontend && npx @inlang/paraglide-js compile
```

---

### 2. Database Schema — Migrations Baseline

Over time, incremental `ALTER TABLE` migration files accumulate. Before a major release,
consider consolidating all migrations into a single baseline schema to simplify new
developer onboarding and reduce migration chain length.

**⚠️ This is a destructive operation for existing databases.** It requires:

- A coordinated data-migration or backup plan for existing installs
- Agreement on a version boundary (the baseline replaces all migrations up to that point)
- Testing on a real data snapshot before shipping

If not doing the merge this release, mark the task as deferred in the tracking doc.

---

### 3. Bug Hunt — Common Merge-Accumulation Issues

Scan for patterns that slip through during rapid multi-feature merges.

**Async race conditions** — check that any view which fires async loads on user input or
prop changes uses a request-generation guard (counter or AbortController) to discard
stale responses.

**Null/undefined returns** — service calls can return `null` from Go. All list results
should be null-coalesced before use (e.g. `results?.filter(Boolean) ?? []`).

**Money arithmetic** — all monetary values are stored as integer cents. Confirm every
template that displays a fee or payment divides by 100 before rendering.

**Date/timezone edge cases** — when converting a date-only string to ISO for the backend,
use the shared `getLocalDateString()` utility from `src/lib/date.ts` or append `T12:00:00`
to avoid day-shift across timezone boundaries.

**SQL schema correctness** — after merges, spot-check migration files for syntax issues
and verify that `ON DELETE CASCADE` relationships are all intentional.

**`confirm()` / `alert()` usage** — browser dialogs are not accessible, are not
translatable, and are blocked in some environments. Any new usage should instead use a
proper modal component.

---

### 4. Dead Code & Deduplication

Look for logic duplicated across feature branches without a shared abstraction.

Common places to check:

- Inline date-to-ISO conversion (`new Date().toISOString().split("T")[0]`) — extract to `src/lib/date.ts`
- Client-side `created_at`/`updated_at` stamping — the backend should own these; audit whether the Go services overwrite them
- Status badge `<span>` markup duplicated across views — consolidate into `StatusBadge.svelte`
- `errorMsg = e.message || "..."` fallback pattern — consider a shared `handleError()` utility

---

### 5. Test Coverage Gaps

Before shipping, identify any services or repositories added during the cycle that have
no corresponding `_test.go` file.

```bash
# Services without tests
for f in internal/services/*.go; do
  base=$(basename "$f" .go)
  [[ "$base" == *_test ]] && continue
  [ ! -f "internal/services/${base}_test.go" ] && echo "MISSING: $f"
done

# Repos without tests
for f in internal/storage/sqlite/*.go; do
  base=$(basename "$f" .go)
  [[ "$base" == *_test ]] && continue
  [ ! -f "internal/storage/sqlite/${base}_test.go" ] && echo "MISSING: $f"
done
```

Run the full suite before marking this category done:

```bash
go test ./...
cd frontend && npx svelte-check
```

---

### 6. Build & Distribution Sanity

- Perform a clean production build and confirm zero errors or warnings
- Verify the Wails binary bundles the latest compiled frontend (`dist/`) and paraglide output
- Confirm `go.mod`/`go.sum` and `package-lock.json` are committed and in sync
- Check `.gitignore` is not accidentally excluding files that need to ship (e.g. new migration files, generated bindings)

---

### 7. Accessibility & UX Polish

Quick pass — easy to miss during feature-focused development:

- Every new `<input>` has a `<label for="…">` (not just an aria-label)
- All new interactive elements are keyboard-reachable with visible focus rings
- Modal close buttons have descriptive `aria-label` attributes
- New table columns have `scope="col"` on `<th>` elements

---

### 8. Documentation & Changelog

- Update `README.md` if any setup steps, dependencies, or env vars changed
- Add a `CHANGELOG.md` entry covering features, fixes, and breaking changes
- Tag the release commit: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`

---

## Task Template

Copy the block below into a new `RELEASE_CLEANUP.md` at the repo root at the start of
each release cleanup branch. Fill in any release-specific notes under each item.

```markdown
# Release Cleanup — vX.Y.Z

> See `docs/release-checklist.md` for guidance on each category.

## 1. i18n — Frontend String Audit
- [ ] ...

## 2. Database Schema
- [ ] ...

## 3. Bug Hunt
### 3a. Async race conditions
- [ ] ...
### 3b. Null/undefined checks
- [ ] ...
### 3c. Money arithmetic
- [ ] ...
### 3d. Date/timezone
- [ ] ...
### 3e. SQL schema
- [ ] ...
### 3f. confirm() / alert() usage
- [ ] ...

## 4. Dead Code & Deduplication
- [ ] ...

## 5. Test Coverage
- [ ] ...
- [ ] `go test ./...` passes
- [ ] `npx svelte-check` passes

## 6. Build & Distribution
- [ ] Clean build succeeds
- [ ] ...

## 7. Accessibility & UX
- [ ] ...

## 8. Documentation & Changelog
- [ ] CHANGELOG.md updated
- [ ] README.md updated if needed
- [ ] Release tagged
```
