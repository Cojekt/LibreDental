# Appointment Reminders (Email and SMS)

> **Status:** in progress (branch `sms-notif`). PRs 1 to 3 (system actor, email and SMS
> providers, automatic reminders) are implemented; PR 3's manual testing and PR 4 remain. Last updated 2026-10-05.

LibreDental will send automatic appointment reminders to patients by email and text message.
This document records the design decisions, the research behind them, and how the feature
fits into the existing notification framework from #89. For the steps a practice follows to
set reminders up, see [Setting Up Appointment Reminders](appointment-reminders-setup.md).

## Design decisions

**No project-hosted infrastructure.** LibreDental promises that installs never depend on
third-party cloud servers run by the project. Each practice brings its own email and SMS
vendor accounts. The app calls the vendor APIs directly from the practice's machine, with
credentials stored in the OS keychain (`SecretsService`). This also keeps the project out of
the path of patient data, so no Business Associate Agreement (BAA) is needed between
LibreDental and each practice.

**First vendors:**

| Channel | Vendor | Why |
| --- | --- | --- |
| Email | Any SMTP server (Amazon SES SMTP, Google Workspace, Microsoft 365, Mailgun, ...) | One provider covers most hosts; the Go standard library is enough. |
| SMS | AWS End User Messaging SMS | AWS signs a BAA self-serve and at no cost through AWS Artifact, and the service is on AWS's HIPAA-eligible list. Twilio only offers a BAA on its paid Security and Enterprise Editions. |

The provider registry supports several vendors per channel, so a Twilio provider can be
added later for practices that prefer it.

**Optional, practice-owned IaC.** A Terraform module (planned, see [PR 4](#pr-4-optional-terraform-module)) will set up
the AWS side in the *practice's own* account: SES domain identity and DKIM, SMS resources, and
a least-privilege IAM user. The project runs none of it.

## What exists today

From #89:

| Piece | Location |
| --- | --- |
| `NotificationProvider` interface (email, SMS, voice) | `internal/domain/notification.go` |
| Provider registry, keychain config, manual `SendNotification` | `internal/services/notification_service.go` |
| Delivery history table `notification_log` | `internal/storage/sqlite/schema/notifications.sql` |
| Patient opt-in flag `reminder_opt_in` | `internal/domain/patient.go` |
| Provider config panel | `frontend/src/views/clinic/IntegrationsSection.svelte` |

`SendNotification` already checks the patient's opt-in, checks that the appointment belongs
to the patient, and records every attempt (success or failure) in both `notification_log`
and the audit trail.

**Missing:** no provider is registered in `main.go`, nothing sends reminders automatically,
there are no reminder settings or templates, and `practice_config` has no timezone.

## The system actor

### Why it's needed

Every audit entry today is tied to a logged-in staff member. `AuditService.LogAction` and
`LogPatientAction` look up the session token and return `ErrUnauthorized` without one.
Automatic reminders are sent by a background job with nobody logged in, so they can't be
audited under the current API. Skipping the audit entry is not an option (see `AGENTS.md`),
and attributing the send to whichever staff member last logged in would be false.

The audit trail needs a **system actor**: a reserved, non-human identity that records actions
the software takes on its own.

### What the standards say

- **HIPAA Security Rule, 45 CFR 164.312(b) (audit controls):** systems holding ePHI must
  record and examine activity. Sending a patient's contact details and appointment time to a
  vendor is activity on ePHI, whoever starts it.
- **45 CFR 164.312(a)(2)(i) (unique user identification):** each identity must be unique and
  traceable. A system actor must never share an ID with a staff account, and it must never be
  a shared login that people could use.
- **NIST SP 800-53 AU-3 (content of audit records):** a record must show what happened, when,
  where, the source, the outcome, and the identity of the individuals, *subjects, or
  processes* involved. That explicitly covers non-human actors, and it means failures must be
  logged as well as successes.
- **HL7 FHIR `AuditEvent`:** the healthcare-standard audit model lets an agent be a person,
  device, or *software*, and marks which agent was the `requestor` (the initiator). LibreDental
  doesn't use FHIR, but it's a useful model: an automatic reminder's requestor is the
  software, and the staff member who turned reminders on appears in a separate entry for that
  configuration change.

### Proposed design

1. **Reserved IDs with a prefix.** System actors use IDs of the form `system:<job>`, for
   example `system:reminders`. One ID per background job means the audit log shows *which*
   process acted, as AU-3 asks. The constants live in `internal/domain/audit.go`.
2. **No audit schema change.** `audit_logs.user_id` is free text in a separate database, with
   no foreign key to `providers`, so system IDs fit the existing columns and need no migration.
3. **Not callable from the frontend.** Wails binds every exported method on a registered
   service, and in LAN server mode that means any client on the network can call it. A method
   like `AuditService.LogSystemAction` would let any client write audit entries that look like
   they came from the system. The system-logging entry point must be unexported, or a
   package-level function, following the `RegisterNotificationProvider` pattern.
4. **Staff can't take a system ID.** `PracticeConfigService.SaveProvider` and
   `CreateInitialProvider` accept a client-supplied provider ID, and the repository saves with
   `ON CONFLICT(id) DO UPDATE`. Today a client could create a staff account with ID
   `system:reminders`, log in with its PIN, and have its actions look automatic. Both methods
   must reject IDs starting with `system:`. Generated IDs are always `prov_<nanoseconds>`, so
   this should only ever reject a deliberately crafted ID. Before shipping, check that no
   existing install could have such an ID.
5. **Tie automatic actions to a human decision.** Turning reminders on or changing their
   settings is audited under the staff member who did it. Each system entry's details name the
   rule it followed (for example "48-hour SMS reminder"), so an auditor can trace a send back
   to the configuration change behind it.
6. **Localized display.** The audit view (`frontend/src/views/AuditingSubtab.svelte`) shows
   `user_name` as stored. For `system:` IDs it shows a Paraglide string ("LibreDental
   (automatic)") rather than English text stored in the database. The ID next to it, such as
   `system:reminders`, says which job acted.

A system send would produce an audit entry like this:

| Field | Value |
| --- | --- |
| `user_id` | `system:reminders` |
| `user_name` | `LibreDental` |
| `patient_id` | the patient's ID |
| `action` | `CREATE` |
| `resource` | `notification` |
| `resource_id` | the `notification_log` entry's ID (currently unused by notifications) |
| `details` | the rule, channel, provider, and outcome, for example "Sent 48-hour SMS reminder via aws_sms" or the failure reason |

### Sharing the send path

`SendNotification` is token-gated and stays that way for manual sends. Its body (opt-in
check, recipient lookup, provider call, log entry, audit entry) moves into an unexported
method that takes an *actor*: either the session user or a system actor. The manual send and
the background job then go through the same checks and the same audit logic. That stops the
two paths from drifting apart.

## Sending reminders safely

### Never send a duplicate

A reminder must go out at most once per appointment, rule, and channel, even if the app
crashes, restarts, or two processes run at once. Vendor-side deduplication shouldn't be
relied on.

The plan is **claim before send**, a variant of the transactional outbox pattern:

1. Insert a `notification_log` row with status `pending` before calling the vendor. A unique
   index on `(appointment_id, appointment_start, reminder_kind, channel)` makes a second claim
   fail, so only one process can send. The appointment's start time is part of the key so
   that a rescheduled appointment gets reminders for its new time.
2. Call the vendor.
3. Update the row to `sent` or `failed`.

If the app crashes between steps 1 and 3, the row stays `pending`. On restart it isn't
retried automatically, because the message may or may not have gone out. It's shown to staff
as "status unknown". For reminders, a missed message is less harmful than a duplicate, and
duplicates also count against the TCPA limits below.

This needs an additive migration on `notification_log`: nullable `reminder_kind` and
`appointment_start` columns (NULL for manual sends) and the unique index. SQLite treats NULLs as distinct in unique
indexes, so existing rows and manual sends are unaffected. The migration is generated with
Atlas, per `AGENTS.md`.

### Check again right before sending

The job re-reads the appointment and patient just before each send, and skips the reminder
if:

- the appointment is cancelled, completed, a no-show, or has been rescheduled outside the
  reminder window;
- the patient has turned off `reminder_opt_in`;
- the patient has no email address or phone number for that channel.

### Message content (HIPAA minimum necessary)

HHS treats appointment reminders as part of treatment, so they don't need the patient's
written authorization. They must still disclose only the minimum necessary. Templates are
limited to:

- the patient's first name;
- the appointment date and time, in the practice's timezone;
- the practice's name and phone number.

Templates must **not** include the procedure, the appointment reason, the provider's
specialty, or anything else from the chart. Unencrypted SMS and email are acceptable when the
patient has been told about the risk and chose that channel. If a patient asks for
confidential communications, the practice must offer another channel.

### TCPA (US text messages)

The FCC's healthcare exemption lets providers text appointment reminders to a patient's
mobile number without prior express written consent, if the messages:

- go only to the number the patient gave the practice;
- are not marketing and are free to the patient;
- include the practice's name and contact information;
- include a way to opt out (for example "Reply STOP to opt out");
- are limited to **one per day and three per week** per patient.

The scheduler enforces the frequency limit across all of a patient's appointments. Default
reminder templates include the practice's name, phone number, and opt-out text. This is
engineering guidance, not legal advice. Practices should confirm their obligations,
including state rules.

AWS End User Messaging SMS adds a number to its opt-out list automatically when someone
replies STOP. That list is not synced back to the patient's `reminder_opt_in` flag yet. The
app can't receive webhooks, so syncing would mean polling (for example SNS to SQS) in a later
phase.

### Time and quiet hours

- `practice_config` gets a timezone column (IANA name, for example `America/Los_Angeles`).
  Appointment times are rendered in it, not in the server machine's local timezone.
- Reminders are only sent during configurable sending hours (default 08:00 to 20:00 in the
  practice's timezone). A reminder that comes due outside them waits for the next allowed
  time, unless that's past its latest send. See
  [How reminders will work](#how-reminders-will-work).

### When the job runs

The job runs inside the Go process on a ticker (for example every 5 minutes):

- **LAN server mode:** the server is always on, so reminders go out on time.
- **Single desktop install:** reminders only go out while the app is open. At startup the
  job catches up on reminders that are still due and skips any whose appointment has already
  started. The settings screen should say this.

The unique index from [Never send a duplicate](#never-send-a-duplicate) keeps this safe even
if a desktop app and a server ever point at the same database.

## Implementation plan

Four PRs, each reviewable on its own and each leaving `main` working. Every PR ends with
`task format` and `task test` clean, and CI's `gofmt -l`, `go vet`, `go test ./internal/...`,
`format:check`, and `svelte-check` passing.

Sizes are rough estimates to help reviewers plan, not commitments.

| PR | Scope | Schema change | Est. size (code / tests) |
| --- | --- | --- | --- |
| 1 | System actor | None | ~80 / ~150 lines |
| 2 | Email and SMS providers, send test | None | ~550 / ~750 lines |
| 3 | Automatic reminders | Yes, additive | ~1,300 / ~1,300 lines |
| 4 | Optional Terraform module | None | ~250 HCL / ~80 lines |

### PR 1: System actor (implemented)

Audit-only. No migration, no new dependencies, no change for existing callers.

**Modified**

| File | Change |
| --- | --- |
| `internal/domain/audit.go` | Add `SystemActorPrefix = "system:"`, `SystemActorReminders = "system:reminders"`, and `IsSystemActorID(id string) bool`. |
| `internal/services/audit_service.go` | Add an unexported `auditActor` (ID and name) and `logPatientActionAs(actor, action, patientID, resource, resourceID, details)`. `LogPatientAction` resolves the session user and delegates to it, so existing behavior is unchanged. Add unexported `logSystemPatientAction`, which only accepts `system:` IDs. Nothing new is exported, so Wails binds nothing new. |
| `internal/services/config_service.go` | `SaveProvider` and `CreateInitialProvider` reject IDs starting with `system:` with `storage.ErrInvalidInput`. The check ignores case and surrounding spaces, so look-alikes such as `SYSTEM:reminders` are refused too. |
| `frontend/src/views/AuditingSubtab.svelte` | For `system:` user IDs, show a localized actor name instead of the stored `user_name`. |
| `frontend/messages/en.json` | New key `audit_system_actor`. |
| `docs/notification-alerts.md` | Mark PR 1 done. |

**Added:** `internal/services/audit_service_system_test.go`. It uses the internal `services`
package because `logSystemPatientAction` is unexported; the existing `audit_service_test.go`
uses the external `services_test` package.

**Tests**

| Test | Checks |
| --- | --- |
| `TestAuditService_LogSystemPatientAction` (new) | Writes an entry with no session; `user_id`, `user_name`, `patient_id`, `resource_id`, and details are stored; `""`, `prov_1`, a bare `system:`, `System:reminders`, and ` system:reminders` are rejected and write nothing. |
| `TestAuditService_BoundMethods` (new) | Uses reflection to compare `AuditService`'s exported methods against a fixed list. It fails if anyone exports a method later and accidentally makes system logging callable from the frontend. |
| `TestPracticeConfigService_RejectsReservedProviderIDs` (new) | `SaveProvider` and `CreateInitialProvider` refuse `system:` IDs, including case and spacing look-alikes, and no provider is saved; a failed onboarding attempt leaves onboarding open; IDs that only contain "system" (`prov_system`, `systemadmin`) and generated IDs still save. |
| `TestAuditService`, `TestAuditService_LogPatientActionUsesSessionUser` (existing) | Regression: session logging is unchanged. |

**Results:** `task format` and `task test` are clean, as are CI's `gofmt -l` and
`go vet ./internal/...`. Both new guard tests were checked by disabling the check they cover:
each failed, then passed again once the check was restored.

**Manual (done):** with `task desktop`, the Audit view shows staff entries the same as before.
There are no system entries to see until PR 3.

### PR 2: Email and SMS providers (implemented)

This section starts with the research behind the providers and then gives the plan. The
research changed the plan in several places; those changes are called out.

#### Research: email over SMTP

**Library: Go's standard `net/smtp`, with the connection managed by us.** `net/smtp` is
frozen (it gets no new features) and has no context support, but a plain-text reminder needs
nothing it lacks. To get timeouts and enforce TLS, the provider dials the connection itself,
sets a deadline from the context, and hands it to `smtp.NewClient`. Its `PlainAuth` refuses to
send a password over a connection without TLS (fixed after CVE-2017-15042). A third-party
library such as `wneessen/go-mail` would add OAuth2 and context support; it can be revisited
if OAuth2 becomes necessary (see Microsoft 365 below).

**TLS for the whole session, not just authentication.** `smtp.SendMail` only upgrades to TLS
if the server offers it, so a man-in-the-middle could strip STARTTLS and receive patient
data in plain text. The provider instead:

- in `starttls` mode (ports 587, 2587), fails if the server doesn't offer STARTTLS;
- in `implicit` mode (ports 465, 2465), connects with TLS from the start;
- requires TLS 1.2 or newer and verifies the server certificate against the configured host;
- has no "no TLS" mode at all.

**What the email hosts require:**

| Host | Works with this provider? | Notes |
| --- | --- | --- |
| Amazon SES | Yes | Endpoint `email-smtp.<region>.amazonaws.com`. STARTTLS on 25, 587, 2587; implicit TLS on 465, 2465. All connections must use TLS. **SMTP credentials are not the IAM access key:** they're generated in the SES console (or derived from an IAM user's secret key), are specific to one region, and need the `ses:SendRawEmail` permission. Don't derive them from temporary credentials. |
| Google Workspace / Gmail | Yes, with an app password | Google ended plain-password SMTP sign-in in May 2025. App passwords still work; Google prefers OAuth2. |
| Microsoft 365 | **Not targeted** | Microsoft disables basic authentication for SMTP AUTH at the end of December 2026. Tenants can turn it back on temporarily, and a final removal date comes in the second half of 2027. Supporting Microsoft 365 properly means OAuth2, which is future work. |
| Other hosts (Mailgun, Postmark, a practice's own server) | Yes, if they offer TLS and `AUTH PLAIN` | |

**Message format.** Plain text only: `Content-Type: text/plain; charset=UTF-8` with
quoted-printable encoding. The subject is encoded with `mime.QEncoding`, and addresses are
parsed and formatted with `net/mail`. Headers: `From`, `To`, `Subject`, `Date`, `Message-ID`,
`MIME-Version`. Any CR or LF in the subject, sender name, or addresses is rejected, which
blocks header injection even though the encoders would also catch it.

**Message ID.** `net/smtp` doesn't expose the server's reply to the end of the message, so
SES's own message ID can't be read. The provider generates the `Message-ID` header itself
(random, at the sender's domain) and records that as the external message ID. It also
appears in bounce messages, so bounces can be matched to the log later.

**"Sent" means "accepted by the server".** SMTP has no delivery confirmation; bounces arrive
later. The log status reflects acceptance only.

**Deliverability.** Gmail, Yahoo, and Outlook require every sender to pass SPF or DKIM and to
use TLS. Senders of more than 5,000 messages a day to their users must also have DMARC and
one-click unsubscribe. A dental practice is far below that, but setup docs will still
recommend SPF, DKIM, and DMARC for the sending domain (for SES: a verified domain identity
with Easy DKIM and a custom MAIL FROM domain).

**Encryption beyond SES.** TLS to SES protects the first hop. SES then delivers to the
patient's mail server with TLS when that server supports it. An SES configuration set with
TLS policy "Require" bounces mail rather than sending it unencrypted; making that the
identity's default configuration set applies it without any app change. That belongs in the
setup docs and the PR 4 Terraform module.

#### Research: SMS through AWS End User Messaging

**SDK modules.** `aws-sdk-go-v2` with only `aws`, `credentials`, and
`service/pinpointsmsvoicev2`. The client is built directly with `pinpointsmsvoicev2.New` and
static credentials from the keychain. The SDK's `config` package is not used, so the app can
never pick up `~/.aws` or environment credentials.

**No automatic retries.** `SendTextMessage` has no idempotency token: AWS's API model lists no
client token parameter. The SDK's default retryer makes up to 3 attempts, including after
`InternalServerException` (HTTP 500) and network timeouts, when the first attempt may already
have sent the text. So the client uses `aws.NopRetryer`, which makes exactly one attempt. Any
retry is decided later by the reminder job (PR 3), which knows whether a retry is safe.

**Request fields used:**

| Field | Value |
| --- | --- |
| `DestinationPhoneNumber` | The patient's number in E.164 (`+` and up to 15 digits). |
| `OriginationIdentity` | From config: phone number, phone number ID or ARN, or pool ID or ARN. |
| `MessageBody` | The rendered message. |
| `MessageType` | `TRANSACTIONAL` (time-sensitive), never `PROMOTIONAL`. |
| `ConfigurationSetName` | Optional, from config. Enables delivery events later. |
| `TimeToLive` | Not set in PR 2 (AWS default is 72 hours). PR 3 sets it so a reminder that can't be handed to the carrier in time expires instead of arriving after the appointment. |
| `Context` | Not set in PR 2. PR 3 can pass the `notification_log` ID (no patient data) so delivery events can be matched to the log. |
| `DryRun` | Used by the opt-in live test. AWS validates the request without sending it, at no charge. |

**Message length.** A message part holds 160 characters in GSM-7 or 70 in UCS-2 (153 or 67
in multipart messages). One character outside GSM-7, such as a curly apostrophe `’` from a
word processor, switches the whole message to UCS-2. `^ { } \ [ ] ~ | €` count as two
characters. AWS's maximum is 1,530 GSM-7 or 630 UCS-2 characters, and each part is billed.
The provider counts parts before sending and rejects messages over the maximum with a clear
error. PR 3's template editor reuses the same counter.

**Errors.** Every `SendTextMessage` error is HTTP 400 except `InternalServerException` (500).
The provider maps the reason codes from AWS's API model into readable failures:

| AWS error | Reason | Meaning for the practice |
| --- | --- | --- |
| `ConflictException` | `DESTINATION_PHONE_NUMBER_OPTED_OUT` | The patient replied STOP. |
| `ConflictException` | `DESTINATION_PHONE_NUMBER_NOT_VERIFIED` | Account is in the SMS sandbox and the number isn't verified. |
| `ServiceQuotaExceededException` | `MONTHLY_SPEND_LIMIT_REACHED_FOR_TEXT` | The account's SMS spend limit is used up. |
| `AccessDeniedException` | `ACCOUNT_DISABLED`, `INSUFFICIENT_ACCOUNT_REPUTATION` | AWS has restricted the account. |
| `ValidationException` | for example `DESTINATION_COUNTRY_BLOCKED`, `INVALID_IDENTITY_FOR_DESTINATION_COUNTRY`, `CANNOT_PARSE` | Configuration or number problem; the reason is shown. |
| `ResourceNotFoundException` | | The origination identity or configuration set doesn't exist in that region. |
| `ThrottlingException` | | Too many requests; not sent. |
| `InternalServerException` or a network error after the request was sent | | **Outcome unknown**; the text may have been sent. |

**"Not sent" versus "unknown".** Duplicate prevention in PR 3 needs to know whether a failed
attempt might have reached the patient. Both providers therefore classify failures:

- **Not sent:** rejected before the message could go out. For SMS, any 400 error; for SMTP,
  any failure before the end of the message data (connection, TLS, authentication, rejected
  sender or recipient).
- **Unknown:** the request may have been delivered. For SMS, a 500 error or a timeout after
  sending; for SMTP, no reply after the message data was sent.

A new sentinel error, `domain.ErrDeliveryUnknown`, marks the second case. In PR 2 both cases
are logged as `failed`, with the reason. PR 3 uses the distinction to leave unknown sends for
staff to review instead of retrying them.

#### Research: phone numbers

`patients.phone_primary` is free text, and AWS needs E.164. LibreDental supports six
countries: US and CA (both +1, no trunk prefix), and GB, AU, DE, and FR (national numbers
start with a trunk `0` that's dropped after the country code). Numbers already starting with
`+`, or with an international prefix (`00`, or `011` from US and CA), are taken as
international.

Two ways to normalize:

| Option | For | Against |
| --- | --- | --- |
| `nyaruka/phonenumbers` (Go port of Google's libphonenumber) | Uses the same per-country rules as Android and most messaging platforms. Validates real number ranges. In GB, AU, DE, and FR it can tell mobile numbers from landlines, so the app can skip texting a landline. | A new dependency with embedded metadata, which makes the binary bigger (to be measured). US and CA numbers can't be classified as mobile or landline. |
| Small hand-written function for the six countries | No dependency; easy to read and review. | Only checks length and prefixes, so it accepts numbers that don't exist; no landline detection. Each new country needs new rules. |

Either way the function **refuses to guess**: anything it can't normalize is rejected with a
clear message rather than sent to a possibly wrong number. **Chosen:** `nyaruka/phonenumbers`
(see [Outcome](#outcome) for its measured cost).

#### Plan

**New dependencies:** the three AWS SDK modules above, and possibly `nyaruka/phonenumbers`.

**Modified**

| File | Change |
| --- | --- |
| `internal/domain/notification.go` | Add `ErrDeliveryUnknown`. |
| `internal/services/secrets_service.go` | Redact every secret field, not only `api_key`: also `password` (SMTP) and `secret_access_key` (AWS). Restore each from the keychain when the frontend sends back the placeholder. Without this, an SMTP password or AWS secret key would go back to the frontend in plain text. |
| `internal/services/notification_service.go` | Add `SendTestMessage(token, providerName, to, subject, body)`: requires a session, sends to an address staff type in (not a patient), and is audited with `LogAction`. It writes no `notification_log` row, because that table requires a patient. For SMS, `recipientFor` normalizes the patient's phone to E.164 using the practice's country, so the constructor also takes the practice config repository. |
| `internal/services/notification_service_test.go` | Constructor change; new tests below. |
| `main.go` | Register the two providers. Pass the practice config repository to `NewNotificationService`. |
| `frontend/src/views/clinic/IntegrationsSection.svelte` | The notification panel shows each provider's own fields instead of a single API key field. The claims panel is unchanged. Secret fields use password inputs and keep the redaction placeholder. Add a "Send test" form. |
| `frontend/messages/en.json` | Field labels, help text, test-send messages. |
| `go.mod`, `go.sum` | New dependencies. |

**Added**

| File | Contents |
| --- | --- |
| `internal/services/notification_provider_smtp.go` | `smtp_email` provider. Config: `host`, `port`, `username`, `password`, `from_address`, `from_name`, `tls_mode` (`starttls` or `implicit`). |
| `internal/services/notification_provider_aws_sms.go` | `aws_sms` provider. Config: `access_key_id`, `secret_access_key`, `region`, `origination_identity`, optional `configuration_set`. |
| `internal/services/notification_sms.go` | SMS part counter (GSM-7 or UCS-2, number of parts) used by the AWS provider and later by PR 3. |
| `internal/services/notification_phone.go` | `toE164(raw, country)`. |
| A `_test.go` for each of the four files above | See below. |

**Tests**

| Test | Checks |
| --- | --- |
| SMTP message building | Headers present and well-formed; non-ASCII subject and sender name encoded; quoted-printable body; CRLF line endings; a body line starting with `.`; CR or LF in subject, name, or address rejected. |
| SMTP transport, in-process fake server with a test TLS certificate | STARTTLS and implicit TLS both succeed; the message arrives intact; credentials are sent only after TLS. Failures: server doesn't offer STARTTLS (refused before sending credentials), wrong certificate, authentication rejected, recipient rejected (all **not sent**), connection dropped after the message data (**unknown**). The context deadline stops a server that never replies. |
| SMTP config | Missing or invalid host, port, from address, or TLS mode is reported before connecting. |
| SMS part counter | GSM-7 at 160 and 161 characters, double-counted extended characters, a curly apostrophe switching to UCS-2, multipart boundaries (153 and 67), the 1,530 and 630 maximums. |
| AWS request shape, `httptest` server as the SDK endpoint | `X-Amz-Target` is `PinpointSMSVoiceV2.SendTextMessage`; a SigV4 `Authorization` header is present; JSON has the expected destination, origination identity, `TRANSACTIONAL`, and body, and the configuration set only when configured. |
| AWS responses | The message ID is stored on success. Each error in the table above maps to the right message and to **not sent** or **unknown**. A 500 is attempted exactly once (no SDK retry). AWS's API reference has no example error bodies, so the fake responses are written in the AWS JSON 1.0 error format with the exception names and reason codes from AWS's API model. |
| AWS config | Missing keys, region, or origination identity fail before any request. With `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_PROFILE` set in the test, the provider still uses only its own config. |
| `toE164` table test | For each of the six countries: national and international formats, spaces, dashes, dots, parentheses, `00` and `011` prefixes, extensions, too short, too long, letters, empty. |
| Secrets redaction (extended) | Each secret field is redacted on read and restored on save; non-secret fields pass through; a placeholder for a field with no stored value is not saved as the literal placeholder. |
| `SendTestMessage` | Requires a session; audited on success and on failure; unknown provider rejected. |
| `TestAWSSMSLive` (opt-in) | Skipped unless `AWS_SMS_LIVE_TEST=1` and credentials are set. Sends a `DryRun` request (free, nothing delivered), then, if a simulator destination number is given, a real send from a simulator origination number. CI never runs it. |
| `TestSMTPLive` (opt-in) | Skipped unless SMTP settings are in the environment. Sends to `success@simulator.amazonses.com` when pointed at SES. |

**Manual:** in the app, configure both providers against the AWS sandbox. Send a test SMS to
a simulator number and a test email to `success@simulator.amazonses.com`. Confirm the password
and secret key show as redacted after saving and reloading, and that the audit log records
each test send.

**Estimated size:** about 550 lines of code and 750 lines of tests (revised up from the first
estimate because of the error classification, part counter, and TLS enforcement).

#### Decisions

1. **Phone normalization:** `nyaruka/phonenumbers`. It may be revisited because of its size
   (see below).
2. **Microsoft 365:** not supported in this PR. Its password-based SMTP is being retired, and
   proper support needs OAuth2.

#### Outcome

Implemented as planned, with these differences:

- **Size:** about 600 lines in the four new files plus about 280 changed lines elsewhere; about
  1,080 lines of tests.
- **A bug the tests caught:** when the connection drops after a request is sent, the AWS SDK
  still reports a `ResponseError`, with HTTP status 0. The first version treated anything
  under 500 as "AWS rejected it, not sent", which would have let PR 3 retry a text that may
  already have been delivered. Status 0 is now treated as "no response", and
  `TestAWSSMSProvider_SendWithoutResponse` covers it.
- **Phone numbers are now validated for real.** `+1 555 555 0100` (area code 555 doesn't
  exist) and the UK's fictional `07700 900xxx` range are rejected. The existing
  `TestNotificationService_SendNotification` used `+15555550100`, so it now uses
  `+1 (202) 555-0123` and also checks that the number is normalized to E.164. Every demo
  patient's phone number uses area code 555 (for example `(555) 111-2233`), so SMS to demo
  patients is rejected as an invalid number. To test SMS with demo data, edit a patient's
  number first.
- **Binary size.** The server binary (`-tags server`, stripped) grew from 22.2 MB to 28.3 MB
  (+6.1 MB, about 27%). Measured by building with each part removed:

  | Part | Added |
  | --- | --- |
  | `nyaruka/phonenumbers` and the `google.golang.org/protobuf` runtime it requires | 4.2 MB |
  | AWS SDK (`pinpointsmsvoicev2`, `smithy-go`, credentials) | 1.2 MB |
  | SMTP provider and the rest of PR 2 | 0.65 MB |

  `phonenumbers` embeds carrier and geocoding data (0.9 MB) that LibreDental doesn't use and
  can't strip, and protobuf brings its reflection runtime. If that cost is too high, the hand-
  written alternative can replace `toE164` and `toSMSNumber` without touching anything else.

**Results:** `task format` and `task test` are clean, as are CI's `gofmt -l`,
`go vet ./internal/...`, and Prettier's check. Three guard tests were checked by disabling
the guard they cover: re-enabling SDK retries fails `TestAWSSMSProvider_SendErrors` (3
attempts instead of 1); removing the STARTTLS requirement fails
`TestSMTPEmailProvider_Send/STARTTLS_not_offered`; and removing `password` from the redacted
fields fails `TestSecretsService_RedactsEverySecretField`. Each passed again once restored.

**Manual (done):** the [manual test guide](#manual-test-guide) below was followed against the
AWS sandbox, with SES SMTP for email and a simulator number for SMS.

#### Manual test guide

Nothing in the app sends to patients until PR 3, so these steps exercise the providers
through **My Clinic > Integrations > Send a test message**. They use the AWS sandbox and
simulators only, so no real patient is contacted and almost nothing is spent. Never enter a
real patient's contact details. Commands assume `us-east-1`.

**1. AWS setup (once).** The CLI can run as your own admin login; the app gets separate,
narrow credentials.

- *Email:* in the SES console, under **Identities**, verify an email address you own (click
  the link SES emails you). While the account is in the SES sandbox, mail can only be sent
  from and to verified addresses, plus SES's mailbox simulator addresses. Then, under
  **SMTP settings**, choose **Create SMTP credentials** and save the username and password
  (shown once). Note the endpoint, `email-smtp.us-east-1.amazonaws.com`.
- *SMS:* request a simulator origination number and note its `PhoneNumber`:

  ```bash
  aws pinpoint-sms-voice-v2 request-phone-number --iso-country-code US \
    --message-type TRANSACTIONAL --number-capabilities SMS --number-type SIMULATOR
  ```

  Create an IAM user that can only send texts, and an access key for it (the secret is shown
  once):

  ```bash
  aws iam create-user --user-name libredental-sms-dev
  aws iam put-user-policy --user-name libredental-sms-dev --policy-name SendTextOnly \
    --policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"sms-voice:SendTextMessage","Resource":"*"}]}'
  aws iam create-access-key --user-name libredental-sms-dev
  ```

  `Resource: "*"` is acceptable for a sandbox; PR 4's Terraform module narrows it to the
  practice's number.

**2. Start the app.** Run `task desktop` (or `task server` and open `http://localhost:4242`),
log in, and make sure the practice country is the US, so numbers typed without `+1` are read
as US numbers.

**3. Email.** In Integrations, choose `smtp_email` and enter: SMTP server
`email-smtp.us-east-1.amazonaws.com`, Encryption STARTTLS, Port blank (587), the SMTP
username and password, From address = your verified address, and a From name. Save.

| Check | Expected |
| --- | --- |
| Send a test to `success@simulator.amazonses.com` | "Test message sent." |
| Send a test to your verified address | The email arrives (check spam). Its headers include LibreDental's `Message-ID`. |
| Switch to another provider and back | The password field shows 8 dots. An SES SMTP password is 44 characters, so 8 means the frontend got the redacted placeholder, not the secret. |
| Save again without retyping the password, then send a test | Still works: the stored password was kept. |
| Change the From address to an unverified one and send | Error: SES rejects the unverified address. Restore it afterwards. |
| Enter a wrong password and send | Error: SMTP sign-in failed. |
| Set Encryption to "TLS from the start" with port blank (465) | Works. |
| `aws sesv2 get-account --query SendQuota.SentLast24Hours` | The count went up for sends to your verified address. Sends to the mailbox simulator don't count toward it. |

**4. SMS.** Choose `aws_sms` and enter the IAM user's access key ID and secret, region
`us-east-1`, and the simulator number as the sending number. Leave the configuration set
blank for now. Save.

| Check | Expected |
| --- | --- |
| Send a test to `+14254147755` (US success simulator) | "Test message sent." |
| Send a test to `(425) 414-7755` | Same result: the number is converted to `+14254147755` before sending. |
| Send a test to `555-0123` or `+1 555 555 0100` | Rejected by the app as an invalid number; nothing reaches AWS. |
| Send a test to your own mobile number | AWS rejects it: a simulator number can only text simulator numbers. The error shows AWS's reason. |
| Send a test to `+14254147167` (US failure simulator) | Most likely "sent". The simulated failure is reported later as a delivery event, not as an error from `SendTextMessage`. |
| Switch providers and back | The secret access key field shows 8 dots, not the 40-character secret. |

To see delivery events, optionally create a configuration set in the End User Messaging SMS
console with a CloudWatch Logs event destination, enter its name as the configuration set,
save, and send to both simulator numbers. The success and failure events then appear in the
log group.

**5. Audit trail.** In the Audit tab, each test send appears with resource
`notification_test`, your name, the provider, the recipient, and, for failures, the reason.
None of them are attached to a patient.

**6. Secrets stay out of the database (optional).** The SMTP password and AWS secret appear
under "LibreDental" in the OS keychain (on Ubuntu: the *Passwords and Keys* app), and
`grep -c '<the SMTP password>' ~/.config/LibreDental/libredental.db` prints `0`.

**7. Clean up.** Release the simulator number when you're done with SMS testing
(`aws pinpoint-sms-voice-v2 release-phone-number --phone-number-id <id>`), and delete the
access key if you won't need it for PR 3
(`aws iam delete-access-key --user-name libredental-sms-dev --access-key-id <id>`).

### PR 3: Automatic reminders (implemented)

The largest PR. It contains the only schema changes.

This section starts with the research done before implementation (2026-10-06), then the
decisions made from it, and the plan that follows from both.

#### Research: how appointments and times work today

1. **Appointments are stored as UTC instants.** `appointment_repo.go` writes `start_time` and
   `end_time` as RFC 3339 UTC strings (`2026-10-07T21:00:00Z`) and filters date ranges by
   string comparison, which works because every row has the same format.

2. **Rescheduling changes `start_time` in place.** `UpdateAppointment` edits the same row
   (with optimistic locking on `version`). A duplicate-prevention key of
   `(appointment_id, reminder_kind, channel)` would therefore block the reminder for the new
   time after a reschedule. **The key must include the appointment's start time.**

3. **Appointments can be hard-deleted.** `DeleteAppointment` removes the row, and
   `notification_log.appointment_id` is set to NULL (`ON DELETE SET NULL`). The history row
   survives without its link, and the partial unique index ignores it. The pre-send re-check
   must treat "appointment not found" as "skip".

4. **The practice's timezone today is whatever timezone the staff computers use.** The
   appointment form (`App.svelte`, `handleSaveAppt`) builds
   `new Date("YYYY-MM-DDTHH:MM:00").toISOString()`, which reads the typed time in the
   *client's* timezone (the desktop app, or a LAN browser) and stores the UTC result. Every
   view displays times in the client's timezone too. Go code never uses a timezone, except
   the demo seeder (`America/Los_Angeles`). So:
   - the reminder job must never use the server process's `time.Local`; in LAN mode the
     server PC's timezone has nothing to do with how appointments were entered;
   - the configured practice timezone must match the staff computers, or every reminder
     shows the wrong time. The settings screen can default it from the browser
     (`Intl.DateTimeFormat().resolvedOptions().timeZone`) and warn when the two differ.

5. **Times are always shown in 12-hour format.** `AppointmentsView.svelte` hard-codes
   `hour12: true`. `practice_config.date_format` exists per country (`MM/DD/YYYY` for the US,
   `DD/MM/YYYY` for GB, AU, and FR, `DD.MM.YYYY` for DE, `YYYY-MM-DD` for CA), but there is no
   time-format setting. Reminder text needs both. Go has no locale data for month or weekday
   names, so numeric dates in the practice's `date_format` avoid translating them in Go.

6. **Everyone is opted in by default, and that flag has never done anything.** The schema
   default is `reminder_opt_in INTEGER NOT NULL DEFAULT 1`, the patient form treats a missing
   value as opted in (`reminderOptIn = p.reminder_opt_in !== false`), and every demo patient
   is opted in. Until now nothing sent to patients, so the checkbox had no effect. Once
   automatic reminders exist, it decides who gets texts and emails. **This changes what
   existing stored data means**, which `AGENTS.md` says must be called out. See
   [Open decisions](#open-decisions-for-pr-3).

7. **Preferred contact method defaults to "phone".** The choices are `phone`, `sms`, and
   `email`, and the form defaults to `phone`, meaning a voice call, for which there is no
   provider. If reminders follow this preference, most patients would get none.

8. **Preferred language defaults to "en" in the form but `''` in the database**, and the
   frontend only has an English locale today.

9. **Patients can be archived** (`status = 'archived'`). Archived patients must not get
   reminders.

10. **No screen shows notification history yet.** `ListNotificationLog` and
    `ListNotificationLogForAppointment` exist in Go but nothing in the frontend calls them.

11. **The demo data never triggers a reminder.** Scheduled demo appointments are pushed a
    year ahead, and every demo phone number uses area code 555, which PR 2 rejects as
    invalid. Manual testing needs an edited patient and appointment. Changing the demo
    fixtures is out of scope.

#### Research: database and migrations

12. **Timestamps written by the driver are not SQL-friendly.** With no `_time_format`
    setting, `modernc.org/sqlite` stores Go `time.Time` values in Go's own format
    (`2026-10-05 09:30:00 +0000 UTC`). That applies to `notification_log.sent_at` and every
    `created_at`. SQLite's date functions can't parse it (`julianday(sent_at)` is NULL).
    String comparison still sorts correctly, but only when every value is UTC. New columns
    that are compared should store RFC 3339 UTC strings, like `appointments.start_time`, and
    queries must always pass UTC times.

13. **Two schedulers on one database is a supported setup.** `lan-server-setup.md` supports
    running the desktop app on the server PC alongside the server process, both using
    `libredental.db`, and `SingleInstance` isn't configured, so two desktop windows also each
    run a job. A unique index stops exact duplicates, but not two processes each deciding a
    patient is under the daily limit for *different* appointments. The frequency check and
    the claim must therefore be one atomic statement: a conditional
    `INSERT ... SELECT ... WHERE (count of recent sends) < limit`. SQLite runs each statement
    atomically and serializes writers. (The driver's `_txlock=immediate` option would also
    work, but it applies to every transaction on the connection.)

14. **Concurrency settings are already suitable.** The database uses WAL with a 5-second
    busy timeout, so a background writer waits rather than failing when the UI is writing.

15. **IDs based on the clock can collide.** `SendNotification` uses
    `notif_<nanoseconds>`. Two schedulers could, rarely, generate the same ID, which would
    fail on the primary key rather than on the duplicate-prevention index and be misreported.
    New rows should use UUIDs (`github.com/google/uuid` is already a dependency).

16. **Migration rules confirmed.** goose runs each migration in a transaction, and the
    previous migration was hand-edited because Atlas's table rebuild (copy, drop, rename)
    breaks on populated databases in a transaction. The planned changes avoid rebuilds:
    `ADD COLUMN ... NULL`, `CREATE TABLE`, and `CREATE UNIQUE INDEX ... WHERE` are all done in
    place. SQLite doesn't allow `ADD COLUMN` with `UNIQUE`, so uniqueness has to come from the
    separate index anyway. In the down migration, the index must be dropped before its
    columns, because SQLite can't drop an indexed column. `TestMigrations_UpgradePreservesPatientData`
    already upgrades and rolls back a populated database and can be extended.

17. **Atlas isn't installed on this machine.** `migrate diff` is available in both the free
    official build and the Apache-licensed Community Edition, and Atlas supports SQLite
    partial indexes. Install with `curl -sSf https://atlasgo.sh | sh` before starting.

#### Research: running a background job in this app

18. **Wails provides the lifecycle.** A service can implement
    `ServiceStartup(ctx, options)`, whose context stays valid while the app runs and is
    cancelled just before shutdown, and `ServiceShutdown()`, which runs before `App.Run`
    returns, so before `main.go`'s deferred `db.Close()`. Wails leaves `ServiceStartup`,
    `ServiceShutdown`, `ServiceName`, and `ServeHTTP` out of the frontend bindings (both when
    generating bindings and at runtime), so the bound reminder settings service can own the
    job safely. A bound-methods test like PR 1's should pin its exported methods.

19. **Timers pause while the computer sleeps.** Go's timers run on a monotonic clock that
    doesn't advance during suspend on Linux and Windows. A short ticker (5 minutes) still
    fires soon after waking; what matters is deciding what's due from the wall clock
    (`time.Now()` against stored times), never from elapsed ticks. Catch-up after waking
    then works the same as catch-up at startup.

20. **Failures need to be visible.** A Linux server run as a system service has no keyring
    (`lan-server-setup.md`), so the job can fail to read provider credentials on every pass.
    Those failures aren't patient actions, so they don't belong in the audit trail on every
    tick. The job should keep its last run time, counts, and last error where the Reminders
    screen can show them.

21. **Windows needs bundled timezone data.** `time.LoadLocation` reads
    `$GOROOT/lib/time/zoneinfo.zip`, which end-user Windows machines don't have. Importing
    `time/tzdata` embeds it (about 450 KB).

22. **A browser timezone list may not be available.** `resolvedOptions().timeZone` is widely
    supported, but I couldn't confirm `Intl.supportedValuesOf("timeZone")` in WebKitGTK and
    WebView2. A curated list of the zones in the six supported countries (for example the
    US's six main zones, Canada's, Australia's, `Europe/London`, `Europe/Berlin`,
    `Europe/Paris`) avoids depending on it and is easy to review.

#### Research: rules for the messages themselves

23. **The FCC healthcare exemption limits reminder texts to 160 characters.** Its full
    conditions for appointment reminders: sent only to the number the patient gave; the
    practice's name and contact information in the message; no marketing, solicitation, or
    billing content; concise (160 characters or less for a text); an opt-out mechanism; and
    at most one message per day and three per week per practice. A text holding the
    practice's name, phone number, the date and time, the patient's first name, and the
    opt-out text is tight at 160 characters, so the limit must be checked on the *rendered*
    message, including long names, not only on the template.

24. **Quiet hours.** Federal rules limit *telemarketing* to 8 a.m. to 9 p.m. in the
    recipient's time zone, and Florida's law narrows that to 8 p.m. Appointment reminders
    are informational and generally outside those limits, but a conservative default of
    08:00 to 20:00 costs nothing. The recipient's timezone is assumed to be the practice's.

25. **Reminder timing.** Industry sources (mostly reminder vendors, so weak evidence)
    recommend more than one reminder, for example 2 to 3 days before plus one on the day.
    Combined with one text per day and three per week, a sensible default is two rules:
    2 days before (SMS and email) and 2 hours before (SMS only). Email isn't covered by the
    TCPA limits, but the same rules keep it reasonable.

#### What the research changes

Compared with the first plan (all now part of the plan below):

- **Duplicate key:** `(appointment_id, appointment_start, reminder_kind, channel)`, with a new
  `notification_log.appointment_start` column (RFC 3339 UTC), so a rescheduled appointment
  gets reminders for its new time. (Findings 2 and 12.)
- **Atomic claim:** the frequency limit and the claim become a single conditional `INSERT`,
  safe with two schedulers. (Finding 13.)
- **UUIDs** for new `notification_log` rows. (Finding 15.)
- **SMS length:** rendered text messages over 160 characters are not sent; they're logged as
  failed with the reason, and the template editor previews the length with long sample
  values. (Finding 23.)
- **Timezone setting:** defaults from the browser, warns on mismatch, uses a curated zone
  list, and the job never uses `time.Local`. (Findings 4, 21, 22.)
- **Date and time format:** numeric date in `date_format`, plus a 12- or 24-hour setting.
  (Finding 5.)
- **Job status** (last run, counts, last error) shown on the Reminders screen. (Finding 20.)
- **Lifecycle:** the job starts in the reminder service's `ServiceStartup` and stops in
  `ServiceShutdown`. (Finding 18.)
- **Archived patients** are skipped, and **deleted appointments** are treated as "skip".
  (Findings 3 and 9.)
- **Before starting:** install Atlas. (Finding 17.)

#### Decisions for PR 3

Made on 2026-10-06:

1. **Reminders ship off; option A below.** Staff turn them on with a confirmation that shows
   how many patients would receive them, and new patients default to not opted in.
2. **Each rule names its channel** and goes to every opted-in patient with contact details
   for it; `preferred_contact_method` isn't used.
3. **English only for now.** The practice using LibreDental is in the US.
4. **Default rules:** 2 days before (SMS and email) and 2 hours before (SMS).

The options that were considered for the first decision:

1. **What `reminder_opt_in` means now.** (Finding 6.) Options:
   - *A (recommended):* keep the existing flag as the patient's preference, ship reminders
     **off**, and when staff turn them on, show how many patients would receive them and
     require an explicit confirmation. New patients default to **not** opted in from now on
     (a form change only; the database default is unused because the form always sends a
     value).
   - *B:* add a new consent column that starts empty for everyone, so nobody receives
     reminders until staff record consent per patient. Safest, but a lot of clerical work.
   - *C:* keep everything as is. Not recommended: patients who never chose reminders would
     start receiving them.
2. **Which channel each patient gets.** (Finding 7.) Either each rule names its channel and
   is sent to every opted-in patient with contact details for it, or reminders follow
   `preferred_contact_method` (with `phone` meaning no reminder, or falling back to SMS).
3. **Language and format.** English-only templates for now (the frontend has only English),
   with numeric dates and a 12/24-hour setting; or per-language templates keyed by
   `preferred_language`.
4. **Default rules.** 2 days before (SMS and email) and 2 hours before (SMS), or something
   else.

#### Research sources

- [FCC healthcare exemption conditions (Bass, Berry & Sims)](https://bassberry.com/news/tcpa-exemptions-for-healthcare-companies/)
- [FCC 2015 ruling for healthcare calls (Kutak Rock)](https://www.kutakrock.com/newspublications/publications/2015/08/fcc-clarifies-tcpa-robocalls-rules-for-health-care)
- [State texting time rules, including Florida (ActiveProspect)](https://activeprospect.com/blog/tcpa-state-regulations/)
- [SQLite: ALTER TABLE](https://www.sqlite.org/lang_altertable.html)
- [SQLite: partial indexes](https://sqlite.org/partialindex.html)
- [Atlas Community Edition](https://www.atlasgo.io/community-edition)
- [Go issue 38453: embedding timezone data](https://golang.org/issue/38453)
- [Go issue 35012: timers and system sleep](https://golang.org/issue/35012)
- [Solutionreach: reminder timing analysis](https://www.businesswire.com/news/home/20190306005043/en/Solutionreach-Data-Analysis-Uncovers-Optimal-Patient-Reminder-Timing-to-Maximize-Appointments)
- [Curogram: best time to send an appointment reminder](https://curogram.com/blog/best-practices/appointment-management/best-time-send-appointment-reminder)

#### How reminders will work

**Turning reminders on.** Reminders ship **off**. When staff turn them on, the Reminders
screen shows how many active, opted-in patients have a mobile number and an email address,
and asks for confirmation. Turning them on or off is audited under the staff member, with
those counts. From this PR on, the patient form defaults new patients to **not** opted in
(`App.svelte`, the new-patient reset at line 426). Existing patients keep their stored
value, and the database default is unchanged, because the form always sends a value.

**Rules.** Turning reminders on for the first time creates three rules, with English
templates from Paraglide:

| Rule | Channel | Sent | Latest send |
| --- | --- | --- | --- |
| 2 days before | SMS | 48 hours before the appointment | 12 hours before |
| 2 days before | Email | 48 hours before the appointment | 12 hours before |
| 2 hours before | SMS | 2 hours before the appointment | 30 minutes before |

Each rule names its channel and provider, and goes to every active, opted-in patient who has
contact details for that channel. `preferred_contact_method` is not used. Staff can turn
each rule on or off and edit its templates; adding rules with other timings is left for
later. A rule's "latest send" is the appointment start minus a quarter of its offset: past
that, a late reminder (after the app was closed, or held back by sending hours) is skipped
rather than sent too close to the appointment.

**Sending hours.** Reminders are sent only between 08:00 and 20:00 in the practice's
timezone (configurable). A reminder that comes due outside those hours waits for the next
allowed time, unless that is past its latest send. For example, the 2-hour text for a 9:00
appointment comes due at 7:00 and its latest send is 8:30, so it waits and goes out at 8:00.
The 2-hour text for an 8:15 appointment comes due at 6:15 and its latest send is 7:45, before
sending hours open, so it is skipped; that patient still received the 2-day reminder.

**Message text.** English only for now. Numeric dates in the practice's `date_format`
(`MM/DD/YYYY` for the US), and the time in 12-hour format for US, CA, and AU practices and
24-hour for GB, DE, and FR. Placeholders: `{first_name}`, `{date}`, `{time}`,
`{practice_name}`, `{practice_phone}`; nothing else from the chart can be used. Default SMS
templates include the practice's name and phone and "Reply STOP to opt out."

**Each pass of the job** (every 5 minutes, and once at startup):

1. Do nothing if reminders are off, or the practice has no timezone set.
2. Load scheduled and confirmed appointments starting between now and 48 hours from now.
3. For each enabled rule and appointment: skip unless the rule is due now (past its due
   time, within sending hours, not past its latest send).
4. Re-read the appointment and patient. Skip silently, writing nothing, if the appointment
   is gone or no longer scheduled or confirmed, or the patient is archived, opted out, or
   has no contact details for the channel. The reminder can still go out later if that
   changes in time.
5. Render the message. If the phone number is invalid or the text is over 160 characters,
   record the reminder as **skipped** with the reason, so staff can see it and it isn't
   retried every pass.
6. **Claim** it with one atomic `INSERT`: a `pending` row keyed by appointment, appointment
   start, reminder kind, and channel, inserted only if, for SMS, the patient has had no text
   reminder today (in the practice's timezone) and fewer than three in the last 7 days. If
   the key already exists, another pass or process has handled it. If the limit stops it,
   record it as **skipped** with the reason.
7. Send through the provider, then update the row to **sent**, **failed** (definitely not
   sent), or **unknown** (`domain.ErrDeliveryUnknown`). Audit the attempt as
   `system:reminders`, with the patient and the log row's ID.
8. Remember the pass's time, counts, and any error (for example, credentials that can't be
   read) for the Reminders screen.

Rows left `pending` by a crash are shown as "status unknown" and never retried.

#### Plan

Delivered as one PR in separately reviewable commits: schema; storage; sending path and job;
reminder service and lifecycle; frontend; docs.

**Before starting:** install Atlas (`curl -sSf https://atlasgo.sh | sh`).

**Schema.** Edited declaratively, migration generated with `atlas migrate diff --env main`,
then `atlas migrate hash`. Every change is additive and done in place (no table rebuild):

| Schema file | Change |
| --- | --- |
| `schema/notifications.sql` | `notification_log`: nullable `reminder_kind` (for example `2880m`; NULL for manual sends) and `appointment_start` (RFC 3339 UTC). Partial unique index on `(appointment_id, appointment_start, reminder_kind, channel) WHERE reminder_kind IS NOT NULL`. New `reminder_settings` table (single row: enabled, sending-hours start and end, when and by whom enabled, timestamps). New `reminder_rules` table (ID, offset in minutes, channel, provider, subject and body templates, enabled, timestamps; unique on offset and channel). |
| `schema/config.sql` | `practice_config.timezone TEXT NULL DEFAULT ''`. |

New statuses (stored as text, no schema change): `pending`, `unknown`, `skipped`. Existing
rows only use `sent` and `failed`, so their meaning doesn't change. Existing installs get
`timezone = ''` and no `reminder_settings` row, which means reminders are off.

**Modified**

| File | Change |
| --- | --- |
| `internal/domain/notification.go` | Statuses `pending`, `unknown`, `skipped`; `ReminderKind` and `AppointmentStart` on `NotificationLog`. |
| `internal/domain/config.go` | `Timezone` on `PracticeConfig`. |
| `internal/storage/repository.go` | `NotificationLogRepository` gains `ClaimReminder` (the atomic conditional insert; reports claimed, already claimed, or over the limit), `RecordSkipped`, and `UpdateResult`. New `ReminderRepository`. |
| `internal/storage/sqlite/notification_repo.go` | Implement the above; read and write the new columns. Compare `sent_at` only against UTC `time.Time` values. |
| `internal/storage/sqlite/practice_config_repo.go` | Read and write `timezone`. |
| `internal/services/notification_service.go` | Move `SendNotification`'s body into an unexported `deliver` that takes an actor: the session user for manual sends, `system:reminders` for the job. New log rows get UUIDs. |
| `internal/services/config_service.go` | Validate `timezone` with `time.LoadLocation`. |
| `main.go` | Import `time/tzdata`. Create the reminder service and register it with Wails. |
| `frontend/src/App.svelte` | New patients default to not opted in. |
| `frontend/src/views/ClinicView.svelte` | Add a Reminders tab. |
| `frontend/src/views/clinic/ClinicProfileSection.svelte` | Timezone picker: curated zones for the six countries, defaulting to the browser's zone, with a warning when it differs from this computer's. |
| `frontend/src/components/PatientInfoPanel.svelte` | Notification history (existing `ListNotificationLog`). |
| `frontend/src/components/AppointmentModal.svelte` | The appointment's reminder history (existing `ListNotificationLogForAppointment`). |
| `frontend/messages/en.json` | Screen text, statuses, reasons, and default templates. |
| `docs/lan-server-setup.md` | Reminders run wherever LibreDental runs; the server needs keyring access; the desktop app on the server PC is safe to run alongside it. |

**Added**

| File | Contents |
| --- | --- |
| `internal/storage/sqlite/migrations/<timestamp>_appointment_reminders.sql` | Generated by Atlas. Down migration drops the index before its columns. |
| `internal/domain/reminder.go` | `ReminderRule`, `ReminderSettings`, and the default rule timings. |
| `internal/storage/sqlite/reminder_repo.go` + test | Settings and rules storage. |
| `internal/services/reminder_template.go` + test | Placeholder rendering, date and time formatting by country, length checks. |
| `internal/services/reminder_schedule.go` + test | Pure functions: is a rule due, next allowed sending time, latest send, the practice's "today" in UTC. Kept free of I/O so the timing rules are easy to test exhaustively. |
| `internal/services/reminder_scheduler.go` + test | The pass described above, with an injectable clock. Not bound to Wails. |
| `internal/services/reminder_service.go` + test | Wails-bound, token-gated: get and save settings and rules, preview who would receive reminders, enable and disable (with confirmation counts), and get the job's status. Starts the job in `ServiceStartup` and stops it in `ServiceShutdown`. |
| `frontend/src/views/clinic/RemindersSection.svelte` | On/off with confirmation, sending hours, rules and template editor with a live 160-character check, job status, and a note that desktop installs only send while open. |

**Tests**

| Area | Checks |
| --- | --- |
| Migration upgrade (extend the existing test) | Seed `notification_log` and `practice_config` rows on the previous schema; upgrade; both survive with the new columns NULL or empty. Roll back and re-apply on the populated database. |
| Notification repo | `ClaimReminder`: succeeds once; second claim for the same key reports "already claimed"; a reschedule (new `appointment_start`) can be claimed again; the daily and weekly SMS limits; manual sends never conflict. `UpdateResult` and `RecordSkipped`. |
| Atomic limit | Two goroutines claim different appointments for the same patient at once: exactly one SMS claim succeeds. |
| Schedule functions | Due and latest-send boundaries for both rules; sending hours, including the 9:00 and 8:15 examples above; DST changes in spring and fall; "today" across midnight in the practice's timezone. |
| Templates | Every placeholder; US date and 12-hour time; 24-hour countries; unknown placeholders rejected; rendered text over 160 characters rejected, including with a long first name. |
| Scheduler, fake clock and providers | Sends when due and only once across passes and restarts. A rescheduled appointment gets reminders for its new time. Skips cancelled, completed, no-show, deleted, archived, opted-out, and no-contact cases without writing anything. Records invalid numbers, over-long texts, and limit hits as skipped. Provider failures are `failed`; `ErrDeliveryUnknown` is `unknown`; a `pending` row is never resent. Does nothing while off or with no timezone. Catch-up after the app was closed honours latest send. |
| Two schedulers | Two schedulers on one database run passes at the same time: each reminder is sent exactly once. |
| Audit | Every send, failure, and unknown outcome is audited as `system:reminders` with the patient and log row ID; skips are not audited (nothing left the machine); enabling and disabling are audited under the staff member. |
| Reminder service | Requires a session; validates rules and sending hours; enabling reports the counts; `TestReminderService_BoundMethods` pins the exported methods. |
| Lifecycle | The job stops when the startup context is cancelled, and `ServiceShutdown` waits for a pass in progress. |
| Existing tests | `SendNotification` behaves the same after moving to `deliver`. |

**Manual:** in the AWS sandbox, set the timezone, turn reminders on, and point a test
patient's mobile number at the SMS success simulator and email at
`success@simulator.amazonses.com`. Create appointments about 48 hours and 2 hours ahead,
and confirm one of each reminder per appointment. Restart the app and confirm nothing is
resent. Reschedule an appointment and confirm new reminders. Check the Audit tab shows
`system:reminders`. Repeat with `task server`.

**Estimated size:** about 1,300 lines of code and 1,300 lines of tests.

#### Outcome

Implemented as planned, with these differences and findings:

- **A limit hit defers instead of being recorded as skipped.** The plan recorded it as
  skipped, which is final. But a 2-day reminder held back today by another appointment's text
  can still go out tomorrow, within its window. Now nothing is recorded and later passes try
  again until the window closes. Reliably reaching the patient was judged more important than
  showing the deferral.
- **Interrupted sends become "unknown" after 10 minutes.** Each pass marks reminders still
  `pending` after 10 minutes as `unknown` (a send is bounded by a 30-second timeout), so the
  "status unknown" described above is visible in the history.
- **Manual sends** now record `ErrDeliveryUnknown` as `unknown` rather than `failed`, and get
  UUID-based IDs. Otherwise `SendNotification` is unchanged; it now shares
  `providerConfig`/`send`/`deliveryStatus` with test sends and the reminder job.
- **Profile saves would have cleared the timezone.** `ClinicView` saves the practice config
  by sending a hand-built object of the fields it knows, and the backend replaces the whole
  row. The timezone is now part of that object.
- **Changing the practice's country resets the timezone** (`SetConfig` rebuilds the config
  from country defaults, as it already did for the NPI). This fails safe: reminders stop and
  the Reminders screen says the timezone isn't set.
- **The lifecycle adapter lives in package `main`** (`reminder_lifecycle.go`). `services`
  imports `internal/app`, so the adapter couldn't go there.
- **Driver time format.** The SQLite driver writes `time.Time` values with
  `time.Time.String`, which includes a `m=+...` suffix if the value still has a monotonic
  reading. Every time bound in a query goes through `.UTC()`, which strips it; the code
  comments say why.
- **A bug the tests caught:** `createRules` saved each rule as it validated it, so a bad
  second rule left the first saved, and the defaults were then never created (they're only
  created when no rules exist). All rules are now validated before any is saved.
- **Default text.** Paraglide treats `{name}` as its own parameter, so the default templates
  are produced by passing each placeholder as its literal value (`{ first_name: "{first_name}" }`).
  The 2-day text renders at 138 characters with an 11-letter first name, leaving room for a
  practice name of about 34 characters; the editor's preview flags anything longer.
- **New patients default to not opted in** (`App.svelte`); existing patients keep their value.
- **Opt-in changes are now audited explicitly** (found during manual testing). Patient saves
  were audited only as "Updated patient record", so the audit trail couldn't show who opted
  a patient in or out of reminders, or when, even though reminders are now sent on that flag
  alone. `UpdatePatient` reads the stored record first and appends "opted in to automated
  reminders" or "opted out of automated reminders" when the flag changes; `CreatePatient`
  records whether the new patient is opted in. Entries written before this change stay
  generic and can't be backfilled. Covered by `TestPatientService_AuditsReminderConsent`,
  which fails if the comparison is removed.
- **Size:** about 1,900 new and 440 changed lines of code (about 600 of them Svelte) and
  about 1,540 lines of tests, above the estimate mainly because of the frontend.

**Results:** `task format` and `task test` are clean, as are CI's `gofmt -l`, `go vet ./...`,
and Prettier's check, and the `-tags server` build compiles. The migration was generated by
`atlas migrate diff` and Atlas reports no drift. Four guards were checked by breaking them and
confirming a test fails:

| Guard broken | Caught by |
| --- | --- |
| Limit check and insert as separate statements | `TestNotificationRepository_ClaimReminderIsAtomic` (8 of 8 simultaneous claims succeeded) |
| Appointment start left out of the reminder key | `TestReminderScheduler_RescheduledAppointment` |
| No latest-send cutoff | `TestReminderDueNow`, `TestReminderScheduler_CatchUpRespectsLatestSend` |
| Rules saved one at a time | `TestReminderService_EnableRequiresTimezoneAndValidRules` |

Each passed again once restored.

**Manual (partly done, 2026-10-07):** the timezone, turning reminders on (with the confirmation
counts and audit entry), 2-day reminders, rescheduling, and skips were tested in the desktop
app against the AWS sandbox. A rescheduled appointment's 2-day reminder email reached a real
Gmail inbox through SES; texts went to the SMS simulator; a patient with `111-111-1111` was
skipped as an invalid number; and every send was audited as `system:reminders`. Still to do:
the 2-hour reminder, turning reminders off, and server mode (steps 6, 10, and 11 of the
[PR 3 manual test guide](#pr-3-manual-test-guide)).

Manual testing also showed that **a failed send is final even when nothing about the
recipient was wrong**. The first attempt failed with SES rejecting the saved SMTP password
(`535 Authentication Credentials Invalid`), left over from the PR 2 wrong-password test.
Correctly, nothing was sent and the failure was recorded and audited, but the reminder
wasn't retried after the password was fixed; rescheduling the appointment was needed. See
[Open questions](#open-questions).

#### PR 3 manual test guide

Uses the AWS sandbox setup from the [PR 2 guide](#manual-test-guide): SES SMTP for email and
a simulator number for SMS, already configured in Integrations. Use only test patients.

1. **Timezone.** In **My Clinic > Practice Profile**, edit, choose this computer's timezone
   (the button offers it), and save. Save the profile again and confirm the timezone is still
   set. Pick a different zone and check the mismatch warning appears; set it back.
2. **New patients start unchecked.** Create a patient and confirm "Opt-in for Automated
   Reminders" is unchecked. Then create a test patient with it **checked**, mobile number
   `(425) 414-7755` (the SMS success simulator), and email `success@simulator.amazonses.com`.
3. **Turn reminders on.** In **My Clinic > Reminders**, choose Turn On Reminders. The
   confirmation shows the opted-in counts. Confirm; three rules appear with previews. In the
   Audit tab, the "Turned on automatic reminders for N opted-in patients..." entry is there.
4. **2-day reminders.** Book an appointment for the test patient between 36 and 48 hours from
   now (so the 2-day reminders are already due). Reminders only go out during sending hours
   (08:00 to 20:00 in the practice timezone); to test outside them, temporarily set sending
   hours to `00:00`–`23:59` and restore them afterwards. Within 5 minutes (or restart the app
   to run a pass at once), **Reminder activity** shows "Sent 1" or more, and the patient's
   panel and the appointment show a text and an email as Sent.

   Emails to SES's mailbox simulator **don't count** toward
   `aws sesv2 get-account --query SendQuota.SentLast24Hours`. To see the count go up (and the
   email arrive), give a test patient your own SES-verified address instead; in the SES
   sandbox, recipients must be verified.
5. **No repeats.** Restart the app; nothing new is sent for that appointment.
6. **2-hour reminder.** Book an appointment about 1 hour 45 minutes ahead; one text is sent.
   It's the patient's second text today, so it's held back by the one-per-day limit unless you
   use a second test patient (for example with the failure simulator, `(425) 414-7167`).
7. **Reschedule.** Move the appointment from step 4 to another time 36 to 48 hours ahead; new
   reminders go out for the new time.
8. **Skips.** Give a test patient the number `555-0123` and book them; the text shows as
   Skipped with "not a valid phone number", and the email is still sent.
9. **Audit.** Each send appears in the Audit tab as **LibreDental (automatic)** with
   `system:reminders`.
10. **Turn off.** Turn reminders off and book another appointment in the window; nothing is
    sent. The audit log records turning them off.
11. **Server mode.** Repeat steps 3 to 5 with `task server` at `http://localhost:4242`.

### PR 4: Optional Terraform module

Only after the maintainer agrees to adding Terraform to the repo.

**Added**

| File | Contents |
| --- | --- |
| `deploy/aws/notifications/versions.tf`, `variables.tf`, `main.tf`, `outputs.tf` | SES domain identity with DKIM and a configuration set; End User Messaging SMS configuration set; an IAM user whose policy only allows sending SMS from the practice's origination identity and sending email from its SES identity. No access key resource, so no secret ends up in Terraform state. Exact resource types will be checked against the current AWS provider. |
| `deploy/aws/notifications/README.md` | Prerequisites (BAA through AWS Artifact, 10DLC registration, leaving the sandboxes), `terraform apply`, creating the access key by hand, and entering values in the app. |
| `deploy/aws/notifications/tests/*.tftest.hcl` | `terraform test` with a mocked AWS provider: the IAM policy contains only send actions, and resources are scoped to the declared identities. |

**Modified**

| File | Change |
| --- | --- |
| `.github/workflows/ci.yml` | New job: `terraform fmt -check`, `init -backend=false`, `validate`, `tflint`, `terraform test`. No AWS credentials in CI. |
| `Taskfile.yml` | `task iac:check` runs the same checks locally. |
| `.gitignore` | `.terraform/`, `*.tfstate*`. |
| `docs/notification-alerts.md` | Link to the module. |

**Manual:** `terraform apply` and `terraform destroy` against the developer sandbox account
(not root credentials).

### What is not covered by automated tests

- **Frontend behavior.** The repo has no frontend test framework. Svelte changes are checked
  by `svelte-check` (types) and by hand. Adding one is out of scope.
- **Real carrier and inbox delivery.** CI only uses fakes and recorded responses. Real AWS
  calls are opt-in, and only reach simulator numbers and SES's mailbox simulator.
- **Vendor API drift.** The recorded AWS responses can go stale, as with Stedi. The opt-in
  live test is the check for that.

## Developer testing with AWS

Use a personal AWS account for development, with these rules:

- **Never put real patient data in it.** Use `task demo` data, your own phone and email, and
  AWS's simulator numbers and addresses. A personal account has no BAA.
- **Stay in the sandboxes.** New accounts start with SMS in sandbox (verified destinations
  only) and SES in sandbox (verified addresses only, 200 emails per day).
- **Use the SMS simulator** for most testing. Simulator origination numbers (US only) can only
  text simulator destination numbers. Messages never reach a carrier, but realistic delivery
  events still come back, and no 10DLC registration is needed. Verify your own phone only to
  check how a real message looks.
- **Use the SES mailbox simulator** (`success@simulator.amazonses.com`,
  `bounce@simulator.amazonses.com`, and so on) to test bounces and complaints.
- **Keep the spend limit.** New accounts have a $1/month SMS limit. Leave it, and add an AWS
  Budgets alert as a backstop.
- **Separate credentials.** Your CLI login is for you. The app gets its own IAM user that can
  only send SMS and email, and its key goes into the app's settings, never into `~/.aws`.
  Don't use root credentials. Don't let Terraform create access keys, because the secret
  would be stored in the state file.
- **Live tests are opt-in.** Following the Stedi pattern, tests that call AWS only run when
  an environment variable is set, and CI never has AWS credentials.

## Open questions

Default rules, language, and channel choice were decided on 2026-10-06 (see
[Decisions for PR 3](#decisions-for-pr-3)). Still open:

- Two-way replies ("reply C to confirm") through SNS to SQS polling: in scope for a later
  phase?
- Per-language templates using `preferred_language`, once the frontend has more locales.
- Rules with other timings, beyond turning the defaults on and off.
- Retrying failures caused by configuration rather than the recipient (provider sign-in
  rejected, server unreachable) on later passes until the reminder's window closes, while
  keeping recipient failures (address or number rejected) final. Today any failure is final,
  so a mistyped password drops every reminder that comes due until it's fixed. Retries would
  need to avoid flooding the audit log with repeated attempts.
- Whether sending hours should apply only to texts (they come from US texting rules), so
  emails can go out whenever the app is running.

## References

- [HHS: Does HIPAA permit providers to use e-mail with patients?](https://www.hhs.gov/hipaa/for-professionals/faq/570/does-hipaa-permit-health-care-providers-to-use-email-to-discuss-health-issues-with-patients/index.html)
- [HIPAA Security Rule technical safeguards, 45 CFR 164.312](https://www.ecfr.gov/current/title-45/subtitle-A/subchapter-C/part-164/subpart-C/section-164.312)
- [NIST SP 800-53 AU-3: Content of Audit Records](https://www.upguard.com/compliance/nist-sp-800-53/au/au-3)
- [HL7 FHIR R4 AuditEvent definitions](https://hl7.org/fhir/R4/auditevent-definitions.html)
- [K&L Gates: FCC clarifies TCPA rules for health care communications (2015)](https://klgates.com/Health-Care-Entities-Get-Clarity-from-FCC-on-Telephone-Communications-08-07-2015)
- [Feldesman: Sending patient appointment reminders? Don't forget the FCC](https://www.feldesman.com/compliance-corner-sending-patient-appointment-reminders-dont-forget-the-fcc/)
- [Transactional outbox with at-least-once delivery](https://oneuptime.com/blog/post/2026-07-22-transactional-outbox-duplicate-events/markdown)
- [AWS HIPAA eligible services](https://aws.amazon.com/compliance/hipaa-eligible-services-reference/)
- [Twilio HIPAA accounts](https://www.twilio.com/docs/iam/twilio-editions/hippa)
- [AWS End User Messaging SMS: simulator phone numbers](https://docs.aws.amazon.com/sms-voice/latest/userguide/test-phone-numbers.html)
- [AWS End User Messaging SMS: quotas](https://docs.aws.amazon.com/sms-voice/latest/userguide/quotas.html)
- [AWS End User Messaging SMS: spend limits](https://docs.aws.amazon.com/sms-voice/latest/userguide/spend-limit.md)
- [AWS End User Messaging SMS: SendTextMessage API](https://docs.aws.amazon.com/pinpoint/latest/apireference_smsvoicev2/API_SendTextMessage.html)
- [AWS End User Messaging SMS: character limits](https://docs.aws.amazon.com/sms-voice/latest/userguide/sms-limitations-character.html)
- [AWS SDK for Go v2: retries and timeouts](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-retries-timeouts.html)
- [Amazon SES: connecting to an SMTP endpoint](https://docs.aws.amazon.com/ses/latest/dg/smtp-connect.html)
- [Amazon SES: obtaining SMTP credentials](https://docs.aws.amazon.com/ses/latest/dg/smtp-credentials.html)
- [Go CVE-2017-15042: net/smtp PlainAuth sent credentials without TLS](https://go.googlesource.com/vulndb/+/a2c41878c4ef/data/reports/GO-2021-0178.yaml)
- [Office 365 for IT Pros: SMTP AUTH basic authentication retirement delayed](https://office365itpros.com/2026/01/29/smtp-auth-basic-retirement/)
- [Google Workspace SMTP: app passwords and OAuth](https://developer.nylas.com/docs/cookbook/email/gmail-smtp-settings/)
- [Google and Yahoo sender requirements](https://www.twilio.com/en-us/blog/insights/new-sending-requirements-for-gmail-yahoo)
- [nyaruka/phonenumbers](https://github.com/nyaruka/phonenumbers)
