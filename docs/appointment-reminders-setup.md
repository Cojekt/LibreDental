# Setting Up Appointment Reminders

> **Audience:** practice managers and IT staff setting up LibreDental's automatic text and
> email reminders. No programming needed.
> **Applies to:** practices in the United States. Texting rules differ in other countries.

LibreDental can send patients automatic reminders before their appointments:

- **2 days before**, by text message and email;
- **2 hours before**, by text message.

Reminders only go to patients who have agreed to receive them. LibreDental doesn't run a
messaging service of its own: your practice sends through **its own Amazon Web Services
(AWS) account**. That keeps your patient data between you and AWS, but it means a one-time
setup, described below.

**Plan for:** about 2 hours of setup, then **1 to 3 weeks of waiting** while US phone carriers
approve your texting number. Email can be working within a day.

**Costs** (AWS prices at the time of writing; check AWS's pricing pages):

| Item | Approximate cost |
| --- | --- |
| Emails | About $0.10 per 1,000 |
| Toll-free texting number | About $2 per month |
| Text messages | About 1 cent each, including carrier fees |

A practice sending a few hundred reminders a month typically pays a few dollars.

## Before you start

You'll need:

- [ ] A computer with LibreDental installed, and a LibreDental login.
- [ ] Access to your practice's **website domain settings** (DNS), or the person who manages
      them. This is how AWS confirms your practice owns its email address.
- [ ] Your practice's **legal business name, address, EIN, and website**, for the texting
      number registration.
- [ ] A way patients agree to reminders, such as a line on your intake form (see
      [Patient consent](#patient-consent)).

## Step 1: Create your AWS account

1. Sign up at [aws.amazon.com](https://aws.amazon.com) using a practice email address (not a
   personal one), so the account stays with the practice if staff change.
2. Turn on **multi-factor authentication (MFA)** for the account's main ("root") login, and
   keep that login for emergencies only.
3. In **IAM Identity Center**, create a login for whoever manages the account day to day.
4. **Choose one region and use it for everything below.** For example, `US East (N. Virginia)`,
   shown as `us-east-1`. Settings made in one region aren't visible in another.

## Step 2: Accept the Business Associate Agreement

HIPAA requires a **Business Associate Agreement (BAA)** with any company that handles patient
information for you. AWS offers one at no cost.

1. In the AWS console, open **AWS Artifact > Agreements**.
2. Find the **AWS Business Associate Addendum**, review it (with your compliance advisor if
   you have one), and accept it for your account.

Do this **before** sending any reminders to real patients. The email and text services used
here (Amazon SES and AWS End User Messaging SMS) are covered by the BAA.

## Step 3: Set up email (Amazon SES)

1. **Verify your domain.** In the **Amazon SES** console, go to **Identities > Create
   identity**, choose **Domain**, and enter your practice's domain (for example
   `smiledental.com`). Keep **Easy DKIM** selected. SES shows three DNS records; add them to
   your domain's DNS settings. Verification usually completes within an hour.
2. **Add a DMARC record** to your DNS, which helps your emails avoid spam folders. A simple
   one is a TXT record named `_dmarc` with the value `v=DMARC1; p=none;`.
3. **Request production access.** New SES accounts can only send to addresses you've verified
   (the "sandbox"). In SES, open **Account dashboard > Request production access**. Describe
   your use: "Appointment reminders to our dental patients who have opted in. About N emails
   per month." AWS usually replies within a day.
4. **Create SMTP credentials.** In SES, open **SMTP settings**, note the **SMTP endpoint**
   (for example `email-smtp.us-east-1.amazonaws.com`), and choose **Create SMTP credentials**.
   Download or copy the **SMTP user name** and **SMTP password** right away; the password is
   shown only once.

   > The SMTP password is **not** your AWS login password or an "access key". It's a separate
   > password, usually 44 characters long and starting with `B`.

## Step 4: Set up text messages (AWS End User Messaging SMS)

US phone carriers only deliver business texts from **registered** numbers. This step starts
that registration, which takes the longest, so start it early.

1. **Get a number.** In the **AWS End User Messaging SMS** console, go to **Phone numbers >
   Request originator**. Choose **United States**, **Toll-free**, capability **SMS**, and
   message type **Transactional**.

   > A toll-free number is the simplest choice for a single practice. "10DLC" numbers are an
   > alternative with extra monthly fees.

2. **Register the number.** Go to **Registrations > Create registration** and choose **US
   toll-free number registration**. You'll be asked for:
   - your business's legal name, address, website, and contact person;
   - the use case: **Appointment reminders**;
   - **sample messages**, for example:
     `Smile Dental: Hi Jane, reminder of your visit on 10/15/2026 at 10:00 AM. Call (206) 555-0100 with questions. Reply STOP to opt out.`;
   - **how patients agree** to receive texts, for example: "Patients give consent on our
     new-patient intake form, which states message frequency and how to opt out. Staff record
     consent in our practice management software."; you may be asked for a link to, or image
     of, the form;
   - expected monthly volume.

   Submit it. Approval can take **up to 15 business days**. AWS emails you when it's done, or
   if they need changes.
3. **Leave the SMS sandbox and set a budget.** New accounts can only text verified numbers
   and spend $1 a month. In the SMS console, request **production access** (if you don't see
   the option, open a case in **AWS Support**), and ask for a monthly spend limit that fits
   your volume, for example $20.
4. **Set the HELP reply.** Under **Phone numbers**, open your number's **Keywords** and set the
   reply to **HELP** to your practice's name and phone number. **STOP** is handled
   automatically: anyone who replies STOP stops receiving texts.
5. **Create a login for LibreDental.** In **IAM > Users**, create a user (for example
   `libredental-sms`) with **no console access**. Give it a policy that only allows sending
   texts:

   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       { "Effect": "Allow", "Action": "sms-voice:SendTextMessage", "Resource": "*" }
     ]
   }
   ```

   Then open the user's **Security credentials** and choose **Create access key** ("Application
   running outside AWS"). Copy the **access key ID** and **secret access key**; the secret is
   shown only once.

## Step 5: Connect LibreDental

1. **Email.** In LibreDental, open **My Clinic > Integrations**, choose `smtp_email`, and enter:
   - **SMTP server:** the endpoint from Step 3, for example `email-smtp.us-east-1.amazonaws.com`
   - **Encryption:** STARTTLS (leave **Port** blank)
   - **Username / Password:** the SMTP credentials from Step 3
   - **From address:** an address on your verified domain, for example `reminders@smiledental.com`
   - **From name:** your practice's name

   Save, then use **Send a test message** to send to your own email address.
2. **Text messages.** Choose `aws_sms` and enter the **access key ID** and **secret access
   key** from Step 4, the **region** (for example `us-east-1`), and your registered number as
   the **sending number** (for example `+18885550100`). Save, then send a test text to your
   own mobile.

   > Until your number's registration is approved and you've left the sandbox, test texts to
   > real phones will be refused. Email can go live before texting.

3. **Practice details.** In **My Clinic > Practice Profile & Standards**, check that the practice name and
   phone number are right: they appear in every reminder. Set the **Timezone** to the one your
   front-desk computers use (the screen offers this computer's timezone).

   > If the timezone doesn't match the computers used to book appointments, reminders will
   > show the wrong appointment time. LibreDental warns you when they differ.

4. **Patients.** For each patient who should get reminders:
   - their **Phone Primary** should be a **mobile** number (landlines can't receive texts);
   - their **email** should be filled in, if they want email reminders;
   - tick **Opt-in for Automated Reminders** only if they've agreed.

   New patients start unticked. **Review existing patients before turning reminders on:**
   records created before reminders existed may have the box ticked by default.
5. **Turn reminders on.** Open **My Clinic > Reminders**:
   - review the message text for each reminder; text messages must stay under 160
     characters, and the preview shows the length;
   - check the **sending hours** (8:00 AM to 8:00 PM by default);
   - choose **Turn On Reminders**. LibreDental shows how many patients will receive reminders
     and asks you to confirm.

## Step 6: Test with your own details

Before relying on reminders, create a test patient using a staff member's mobile number and
email, tick the opt-in box, and book an appointment **between 36 and 48 hours from now**.
Within a few minutes (during sending hours), the staff member should get the 2-day text and
email. The **Reminder activity** box on the Reminders screen, and the patient's **Messages
sent** history, show the result.

Archive the test patient afterwards.

## Keeping it running

- **LibreDental must be running to send reminders.** If you use the single-computer desktop
  app, reminders only go out while it's open. Keep it open during the day, or run LibreDental
  as a LAN server (see [LAN Server Setup](lan-server-setup.md)), which runs all the time.
- Each reminder can go out at any point in a window (for example, the 2-day reminder from
  48 hours before the appointment until 12 hours before), so the app doesn't need to be open
  at an exact moment. Appointments on Monday mornings may miss their 2-day reminder if
  LibreDental isn't running over the weekend.
- Check **My Clinic > Reminders > Reminder activity** now and then. Problems, such as a wrong
  password, appear there.
- Every reminder sent is recorded in the patient's history and in the **Audit** tab as
  "LibreDental (automatic)".

## Patient consent

Appointment reminders are allowed under HIPAA and US texting rules when they stick to the
basics: the patient's first name, the appointment date and time, and the practice's name and
phone. LibreDental's reminders never include the reason for the visit or other health
information. You should still:

- **Ask patients first** and record their answer with the opt-in box. A line on your intake
  form works, for example:

  > I agree to receive appointment reminders from [Practice name] by text message and email at
  > the number and address I provided. Up to 3 texts per week. Message and data rates may
  > apply. Reply STOP to opt out or HELP for help. Texts and emails may not be encrypted.

- **Tell patients that texts and email aren't encrypted**, and offer another way to reach them
  if they'd rather not.
- **Stop when asked.** Untick the opt-in box when a patient asks. If they reply STOP to a text,
  AWS stops texting them automatically; untick the box too, so LibreDental's records match.

This guide isn't legal advice. If you're unsure about your obligations, including state
rules, ask your compliance advisor.

## Troubleshooting

| What you see | What it means | What to do |
| --- | --- | --- |
| `SMTP sign-in failed: 535 Authentication Credentials Invalid` | The SMTP user name or password is wrong. | Re-enter the **SMTP** credentials from Step 3 (not an access key or your AWS password), or create new ones. They only work in the region they were created in. |
| Test emails arrive, but only for some addresses | SES is still in its sandbox. | Finish **Request production access** in Step 3. |
| Emails land in spam | Your domain's email checks aren't set up. | Check the DKIM records (Step 3.1) and add the DMARC record (Step 3.2). |
| `this number is not a verified destination` | The texting service is still in its sandbox. | Finish Step 4.3. |
| `the recipient has opted out of text messages` | The patient replied STOP. | Respect it, and untick their opt-in box. Only if the patient asks to receive texts again can the opt-out be removed in the SMS console (**Opt-out lists**). |
| `monthly SMS spend limit has been reached` | Your monthly text budget is used up. | Raise the spend limit (Step 4.3). |
| `is not a valid phone number` or `is a landline` | The patient's number can't receive texts. | Correct their **Phone Primary**. |
| `over the 160-character limit` | The text was too long, often because of a long name. | Shorten the reminder text on the Reminders screen. |
| Reminder activity says `the practice timezone is not set` | Reminders need the timezone. | Set it in **My Clinic > Practice Profile & Standards**. |
| Nothing is being sent | Reminders only go out while LibreDental is running, during sending hours, and to opted-in patients with a scheduled or confirmed appointment. | Check each of those, and the **Reminder activity** box. |
| `dbus-launch` or keyring errors (Linux server) | The server can't reach the system keyring where passwords are stored. | See "Credential Storage" in [LAN Server Setup](lan-server-setup.md). |

A reminder that failed (for example because of a wrong password) isn't sent again
automatically. Fix the problem; reminders that come due afterwards go out normally. Contact
any patient who missed a reminder directly.
