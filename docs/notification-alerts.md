# Appointment Reminders (Email and SMS)

> **Status:** in progress (branch `sms-notif`). PR 1 (system actor) and PR 2 (email and SMS
> providers) are implemented; PRs 3 and 4 are planned. Last updated 2026-10-05.

LibreDental will send automatic appointment reminders to patients by email and text message.
This document records the design decisions, the research behind them, and how the feature
fits into the existing notification framework from #89.

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
   index on `(appointment_id, reminder_kind, channel)` makes a second claim fail, so only one
   process can send.
2. Call the vendor.
3. Update the row to `sent` or `failed`.

If the app crashes between steps 1 and 3, the row stays `pending`. On restart it isn't
retried automatically, because the message may or may not have gone out. It's shown to staff
as "status unknown". For reminders, a missed message is less harmful than a duplicate, and
duplicates also count against the TCPA limits below.

This needs an additive migration on `notification_log`: a nullable `reminder_kind` column
(NULL for manual sends) and the unique index. SQLite treats NULLs as distinct in unique
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
- Reminders are only sent inside configurable quiet hours (default 08:00 to 21:00 in the
  practice's timezone). A reminder that falls outside them moves to the next allowed time, as
  long as that's still before the appointment.

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
| 3 | Automatic reminders | Yes, additive | ~1,100 / ~1,000 lines |
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
| `aws sesv2 get-account --query SendQuota.SentLast24Hours` | The count went up. |

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

### PR 3: Automatic reminders

The largest PR. It contains the only schema changes.

**Schema** (edited declaratively, migration generated with `atlas migrate diff --env main`,
then `atlas migrate hash`). All changes are additive:

| Schema file | Change |
| --- | --- |
| `schema/notifications.sql` | `notification_log.reminder_kind TEXT` (nullable; NULL for manual sends). Partial unique index on `(appointment_id, reminder_kind, channel) WHERE reminder_kind IS NOT NULL`. |
| `schema/config.sql` | `practice_config.timezone TEXT DEFAULT ''`. New single-row `reminder_settings` table (enabled, quiet-hours start and end). New `reminder_rules` table (offset before the appointment, channel, provider, subject and body templates, enabled, timestamps). |

Existing installs get `timezone = ''`, and reminders stay off until someone sets a timezone.
That way no reminder is ever rendered in a guessed timezone.

**Modified**

| File | Change |
| --- | --- |
| `internal/domain/notification.go` | `NotificationStatusPending`; `ReminderKind` on `NotificationLog`. |
| `internal/domain/config.go` | `Timezone` on `PracticeConfig`. |
| `internal/storage/repository.go` | `NotificationLogRepository` gains `Claim` (insert a `pending` row; returns a conflict error if the reminder was already claimed), `UpdateResult`, and `CountReminderSends(patientID, channel, since)`. New `ReminderRepository` interface. |
| `internal/storage/sqlite/notification_repo.go` | Implement the new methods; read and write `reminder_kind`. |
| `internal/storage/sqlite/practice_config_repo.go` | Read and write `timezone`. |
| `internal/services/notification_service.go` | Move `SendNotification`'s body into an unexported, actor-aware `deliver`. Manual sends pass the session user; the scheduler passes `system:reminders`. Reminder sends claim the row before calling the provider. |
| `internal/services/config_service.go` | Validate the timezone with `time.LoadLocation`. |
| `main.go` | Import `time/tzdata` so timezones work on Windows, which has no system timezone database for Go. Build the reminder service and scheduler; register the service with Wails; start the scheduler in a goroutine and stop it when the app exits. |
| `frontend/src/views/ClinicView.svelte` | Add the Reminders section. |
| `frontend/src/views/clinic/ClinicProfileSection.svelte` | Timezone picker. |
| `frontend/src/components/PatientInfoPanel.svelte` | Notification history (uses the existing `ListNotificationLog`). |
| `frontend/src/components/AppointmentModal.svelte` | The appointment's reminder history (uses the existing `ListNotificationLogForAppointment`). |
| `frontend/messages/en.json` | Settings, history, statuses, and default templates. |
| `docs/lan-server-setup.md` | Reminders run on the server in LAN mode. |

**Added**

| File | Contents |
| --- | --- |
| `internal/storage/sqlite/migrations/<timestamp>_appointment_reminders.sql` | Generated by Atlas. |
| `internal/domain/reminder.go` | `ReminderRule`, `ReminderSettings`. |
| `internal/storage/sqlite/reminder_repo.go` + test | Reminder settings and rules storage. |
| `internal/services/reminder_service.go` + test | Wails-bound, token-gated get/save for settings and rules. Every change is audited under the staff member. |
| `internal/services/reminder_template.go` + test | Renders templates. Only the allowed placeholders exist (`{first_name}`, `{date}`, `{time}`, `{practice_name}`, `{practice_phone}`), so a template can't pull in chart data. Unknown placeholders are rejected when saved. |
| `internal/services/reminder_scheduler.go` + test | Not bound to Wails. A ticker loop with an injectable clock. Each pass loads settings and finds upcoming appointments. For each rule that's due it re-checks the appointment and patient, applies quiet hours and the TCPA limits, renders the message, and calls `deliver` as the system actor. |
| `frontend/src/views/clinic/RemindersSection.svelte` | On/off switch, quiet hours, rules, and template editor. Default templates come from Paraglide, so they're localized. Notes that desktop installs only send while the app is open. |

**Tests**

| Area | Checks |
| --- | --- |
| Migration upgrade (extend `TestMigrations_UpgradePreservesPatientData`) | Seed a `notification_log` row and a `practice_config` row on the previous schema, upgrade, and confirm both survive with `reminder_kind` NULL and `timezone` `''`. Roll back and confirm the down migration works. |
| Notification repo (extended) | `Claim` succeeds once and conflicts the second time; manual sends (NULL kind) never conflict; `UpdateResult`; `CountReminderSends` windows. |
| Reminder repo (new) | Settings and rules CRUD. |
| Practice config repo (extended) | `timezone` round-trips. |
| Template (new) | Every placeholder renders; unknown placeholders are rejected; date and time use the practice timezone and date format; SMS segment count is reported. |
| Scheduler (new), fake clock and dummy provider | Sends when due. Doesn't send twice across two passes. Skips cancelled, completed, and no-show appointments, rescheduled appointments, opted-out patients, and patients with no contact for the channel. Holds messages during quiet hours and sends them once quiet hours end. Enforces 1 per day and 3 per week across appointments. Renders correctly across a DST change. Catch-up at startup skips appointments that have already started. A provider failure is logged as `failed` and not retried. A row left `pending` (simulated crash) is not resent. Does nothing while disabled or without a timezone. |
| Scheduler concurrency (new) | Two schedulers on the same database run a pass at the same moment; exactly one message is sent. |
| Audit (new) | Every scheduler send, success or failure, writes an audit entry with `user_id = system:reminders`, the patient, and the `notification_log` ID. Settings changes are audited under the staff member. |
| Notification service (existing, kept passing) | `SendNotification` behaves as before after moving to `deliver`. |
| Reminder service (new) | Requires a session; validates rules; audits changes. |

**Manual:** with `task demo` data and the AWS sandbox, set the timezone, create a rule a few
minutes ahead of a demo appointment whose contact details are changed to your own number or a
simulator number, and watch it send once. Restart the app and confirm it doesn't resend.
Check the Audit view shows the system actor. Repeat in server mode (`task server`).

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

- Which reminder rules ship by default (for example 48 hours by SMS and email, 2 hours by
  SMS)?
- Should templates be per-language, using the patient's `preferred_language`?
- Should reminders respect `preferred_contact_method`, or send on every channel the practice
  enables?
- Two-way replies ("reply C to confirm") through SNS to SQS polling: in scope for a later
  phase?

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
