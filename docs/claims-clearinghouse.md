# Electronic Claims (Clearinghouse)

LibreDental submits US dental claims electronically (X12 837D) through a clearinghouse.
Clearinghouses are implemented as `domain.ClaimProvider`s and registered in `main.go`; the
first one is [Stedi](https://www.stedi.com), via its
[Dental Claims JSON API](https://www.stedi.com/docs/healthcare/api-reference/post-healthcare-dental-claims)
(`internal/services/claim_provider_stedi.go`).

## Data a claim needs

Before a claim can be submitted, these must be filled in. Submission fails with a list of
everything that's missing, before any data leaves the machine.

| Where | Field |
| --- | --- |
| Clinic profile | Name, phone, street address, city, state, **ZIP+4** (all 9 digits), tax ID (EIN), **billing NPI** |
| Provider on the claim | **NPI** and **taxonomy code** (only sent when the NPI differs from the practice's) |
| Patient | Date of birth, address, insurance carrier, member/policy ID, **payer ID** |
| Patient, if not the policyholder | Policyholder name, date of birth, and relationship to the patient |

The payer ID is the clearinghouse's identifier for the insurer (for example `52133` for
United HealthCare Dental). Look it up in [Stedi's payer network](https://www.stedi.com/healthcare/network).

The patient control number sent with each claim is derived from the claim ID
(`Claim.PatientControlNumber()`), so payer responses (277CA, 835) can be matched back to a
claim by recomputing it.

## Setup

In **Clinic > Integrations**, choose `stedi`, enter an API key, and save. The key is stored
in the OS keychain, not the database.

**Test mode is on by default.** Test claims run through Stedi's claim edits and return a
test acknowledgment, but are never sent to a payer. Turn test mode off only when you are
ready to bill real claims (this needs a production API key).

## Testing

Compatibility with Stedi's API is tested offline, with no API key or network access:

- `testdata/stedi_dental_claim_fields.txt` lists every request property in Stedi's API
  reference, with its type and whether it's required. The tests check that each request
  LibreDental builds uses only documented properties, with the right JSON types, and
  includes every required one.
- The checker is itself verified against Stedi's documented example request.
- A fake Stedi server replays the documented responses: success, claim-edit rejections, and
  HTTP errors.

To also submit a test claim to the real API:

```bash
STEDI_API_KEY=... go test ./internal/services -run TestStediLiveTestMode -v
```

Stedi's free sandbox accounts can't submit claims, not even test claims. A pay-as-you-go
account ($25 starting balance) with a test API key is needed.

To refresh the field list after Stedi changes its API, regenerate it from
`https://www.stedi.com/docs/healthcare/api-reference/post-healthcare-dental-claims.md`.
