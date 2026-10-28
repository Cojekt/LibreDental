# Insurance fee overrides

Billing → Insurance Fee Overrides manages user-entered insurance providers and their
replacement procedure fees. Names are normalized for duplicate detection. Providers can
be searched, renamed, archived and restored. Archived names remain reserved.

Each provider has one fee per country/procedure. Fees are entered as monetary amounts and stored in integer cents;
an explicitly saved zero is valid. Use Remove override to clear a configured fee.
The original procedure catalog and its reference price remain unchanged.

New billable treatments and direct claim lines use a shared pricing review. Staff can
choose effective standard pricing, a provider's configured insurance price, or an
explicit manual fee. Standard pricing preserves clinician → practice → base precedence.
Chart pricing can optionally pick a clinician fee schedule; claim lines use the claim's
provider. A charted treatment priced for a clinician can only be billed on that clinician's
claim, and chart-to-claim takes its provider from the treatments' clinician when one is set.
Insurance prices replace that amount without combining discounts. Missing or archived
insurance fees cannot silently fall back to standard pricing. Confirmation shows patient,
procedure, original reference price, selected source and the exact amount to save.

Provider fees are stored in `insurance_fee_overrides`, referencing `insurance_providers`
and the existing country/procedure catalog. Existing treatment and claim fee fields are
snapshots. Provider IDs, historical names and confirmation metadata accompany the saved
amount. Catalog edits, renames and archiving do not recalculate previous charges.
Explicit pricing corrections replace the line's current source through the existing audited editing workflow;
there is no patient-recommendation lookup in this version.

Existing claim notes/status/insurance-text edits preserve untouched line amounts and
source metadata, including legacy lines without pricing metadata or a procedure code.
New lines and deliberate pricing changes require review. Chart-to-claim copies saved
amounts. Manual fees and bundles retain their existing explicit behavior.

Management and pricing controls require backend-verified sessions. As with existing fee
management, all authenticated staff currently have access; this feature does not create
a new role system. Mutations use the existing audit service. Selected source and amount
are validated and saved together in SQLite transactions. Changed prices require renewed
confirmation; stable record IDs and linked-condition checks protect against duplicates.

Named plans, Latest Used, automatic insurance assignment and benefit/reimbursement
calculations are outside this PR. Provider price selection does not change the patient's
insurance text, verify coverage, or calculate insurer/patient responsibility.

Run `task format` and `task test`. The Svelte interaction tests mock service bindings;
Go integration tests exercise actual SQLite/services. The optional
`frontend/e2e/insurance.cjs` uses a built server in `LIBREDENTAL_E2E_BINARY`, external
Playwright/Chromium, and Node 22's SQLite module. It creates its own temporary synthetic
patient database, including legacy claim fixtures while the server is stopped.

Desktop runtime testing must use isolated synthetic configurations. Successful desktop
cross-compilation does not establish runtime or installer verification. Per-PR build,
GUI and test results are recorded separately in the split verification reports.
