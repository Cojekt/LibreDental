<script lang="ts">
  import { onMount } from "svelte";
  import type {
    PracticeConfig,
    ReminderJobStatus,
    ReminderPreview,
    ReminderRecipientCounts,
    ReminderRule,
    ReminderSettings,
  } from "@bindings/domain/models.js";
  import { NotificationChannel } from "@bindings/domain/models.js";
  import { ReminderService } from "@bindings/services/index.js";
  import { m } from "../../paraglide/messages.js";
  import { auth } from "../../stores/auth.svelte.js";
  import { handleError } from "$lib/error.js";
  import ConfirmModal from "../../components/ui/ConfirmModal.svelte";

  let {
    canEdit = false,
    practiceConfig = null,
    onopenprofile,
  } = $props<{
    canEdit: boolean;
    practiceConfig: PracticeConfig | null;
    onopenprofile?: () => void;
  }>();

  const TWO_DAYS = 2 * 24 * 60;
  const TWO_HOURS = 2 * 60;
  const MAX_SMS = 160;
  // Template placeholders; they're identifiers, so they aren't translated.
  const PLACEHOLDERS = ["first_name", "date", "time", "practice_name", "practice_phone"];
  const placeholderValues = Object.fromEntries(PLACEHOLDERS.map((p) => [p, `{${p}}`])) as {
    first_name: string;
    date: string;
    time: string;
    practice_name: string;
    practice_phone: string;
  };

  let settings = $state<ReminderSettings | null>(null);
  let rules = $state<ReminderRule[]>([]);
  let providers = $state<{ [channel: string]: string[] }>({});
  let jobStatus = $state<ReminderJobStatus | null>(null);
  let loadError = $state("");
  let message = $state<{ ok: boolean; text: string } | null>(null);
  let busy = $state(false);

  let hoursStart = $state("08:00");
  let hoursEnd = $state("20:00");

  let showEnableConfirm = $state(false);
  let counts = $state<ReminderRecipientCounts | null>(null);

  let previews = $state<{ [ruleId: string]: ReminderPreview | null }>({});
  let previewErrors = $state<{ [ruleId: string]: string }>({});
  const previewTimers: { [ruleId: string]: ReturnType<typeof setTimeout> } = {};

  const hasTimezone = $derived(!!practiceConfig?.timezone);

  async function load() {
    loadError = "";
    if (!auth.token) return;
    try {
      const [s, r, p, j] = await Promise.all([
        ReminderService.GetReminderSettings(auth.token),
        ReminderService.ListReminderRules(auth.token),
        ReminderService.ListReminderProviders(auth.token),
        ReminderService.GetReminderJobStatus(auth.token),
      ]);
      settings = s;
      hoursStart = s?.sending_hours_start || "08:00";
      hoursEnd = s?.sending_hours_end || "20:00";
      rules = (r?.filter(Boolean) as ReminderRule[]) || [];
      providers = Object.fromEntries(Object.entries(p || {}).map(([k, v]) => [k, v || []]));
      jobStatus = j;
      for (const rule of rules) schedulePreview(rule);
    } catch (e) {
      console.error("Failed to load reminder settings:", e);
      loadError = handleError(e, m.reminders_load_error());
    }
  }

  onMount(load);

  function show(ok: boolean, text: string) {
    message = { ok, text };
  }

  // The default rules' text comes from the translations; placeholders pass through unchanged.
  function defaultRules(): ReminderRule[] {
    const rule = (offset: number, channel: NotificationChannel, subject: string, body: string) =>
      ({
        id: "",
        offset_minutes: offset,
        channel,
        provider_name: "",
        subject_template: subject,
        body_template: body,
        enabled: true,
      }) as ReminderRule;
    return [
      rule(
        TWO_DAYS,
        NotificationChannel.NotificationChannelSMS,
        "",
        m.reminders_default_sms_2days(placeholderValues)
      ),
      rule(
        TWO_DAYS,
        NotificationChannel.NotificationChannelEmail,
        m.reminders_default_email_subject(placeholderValues),
        m.reminders_default_email_body(placeholderValues)
      ),
      rule(
        TWO_HOURS,
        NotificationChannel.NotificationChannelSMS,
        "",
        m.reminders_default_sms_2hours(placeholderValues)
      ),
    ];
  }

  async function requestEnable() {
    message = null;
    try {
      counts = await ReminderService.GetRecipientCounts(auth.token);
      showEnableConfirm = true;
    } catch (e) {
      console.error("Failed to count reminder recipients:", e);
      show(false, handleError(e, m.reminders_enable_error()));
    }
  }

  async function confirmEnable() {
    busy = true;
    try {
      settings = await ReminderService.EnableReminders(auth.token, defaultRules());
      show(true, m.reminders_enabled_success());
      await load();
    } catch (e) {
      console.error("Failed to turn on reminders:", e);
      show(false, handleError(e, m.reminders_enable_error()));
    } finally {
      busy = false;
    }
  }

  async function disable() {
    busy = true;
    message = null;
    try {
      settings = await ReminderService.DisableReminders(auth.token);
      show(true, m.reminders_disabled_success());
    } catch (e) {
      console.error("Failed to turn off reminders:", e);
      show(false, handleError(e, m.reminders_save_error()));
    } finally {
      busy = false;
    }
  }

  async function saveHours() {
    busy = true;
    message = null;
    try {
      settings = await ReminderService.SaveSendingHours(auth.token, hoursStart, hoursEnd);
      show(true, m.reminders_saved());
    } catch (e) {
      console.error("Failed to save sending hours:", e);
      show(false, handleError(e, m.reminders_save_error()));
    } finally {
      busy = false;
    }
  }

  async function saveRule(rule: ReminderRule) {
    busy = true;
    message = null;
    try {
      const saved = await ReminderService.SaveReminderRule(auth.token, rule);
      if (saved) rules = rules.map((r) => (r.id === saved.id ? saved : r));
      show(true, m.reminders_saved());
    } catch (e) {
      console.error("Failed to save reminder rule:", e);
      show(false, handleError(e, m.reminders_save_error()));
    } finally {
      busy = false;
    }
  }

  function updateRule(id: string, patch: Partial<ReminderRule>) {
    rules = rules.map((r) => (r.id === id ? ({ ...r, ...patch } as ReminderRule) : r));
    const rule = rules.find((r) => r.id === id);
    if (rule && ("body_template" in patch || "subject_template" in patch)) schedulePreview(rule);
  }

  function schedulePreview(rule: ReminderRule) {
    clearTimeout(previewTimers[rule.id]);
    previewTimers[rule.id] = setTimeout(async () => {
      try {
        previews[rule.id] = await ReminderService.PreviewReminderTemplate(
          auth.token,
          rule.channel,
          rule.subject_template,
          rule.body_template
        );
        previewErrors[rule.id] = "";
      } catch (e) {
        previews[rule.id] = null;
        previewErrors[rule.id] = handleError(e, m.reminders_preview_error());
      }
    }, 300);
  }

  function ruleTitle(rule: ReminderRule): string {
    const timing =
      rule.offset_minutes % (24 * 60) === 0
        ? m.reminders_days_before({ count: rule.offset_minutes / (24 * 60) })
        : rule.offset_minutes % 60 === 0
          ? m.reminders_hours_before({ count: rule.offset_minutes / 60 })
          : m.reminders_minutes_before({ count: rule.offset_minutes });
    const channel =
      rule.channel === NotificationChannel.NotificationChannelEmail
        ? m.notif_channel_email()
        : m.notif_channel_sms();
    return `${timing} · ${channel}`;
  }
</script>

<div class="space-y-6 animate-fadeIn">
  <div>
    <h3 class="text-lg font-bold text-slate-100 mb-1">{m.reminders_title()}</h3>
    <p class="text-sm text-slate-400">{m.reminders_subtitle()}</p>
  </div>

  {#if loadError}
    <p class="text-sm text-rose-400" role="alert">{loadError}</p>
  {/if}
  {#if message}
    <p
      class="text-xs font-semibold {message.ok ? 'text-emerald-400' : 'text-rose-400'}"
      role={message.ok ? "status" : "alert"}
    >
      {message.text}
    </p>
  {/if}

  {#if !hasTimezone}
    <div class="rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-amber-300">
      {m.reminders_timezone_missing()}
      {#if onopenprofile}
        <button type="button" class="ml-1 underline cursor-pointer" onclick={onopenprofile}
          >{m.reminders_open_profile()}</button
        >
      {/if}
    </div>
  {/if}

  <!-- On / off -->
  <div class="rounded-xl border border-slate-800 bg-slate-950/80 p-4 space-y-3">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <span class="text-sm font-semibold text-slate-100">{m.reminders_status_label()}</span>
        {#if settings?.enabled}
          <span
            class="ml-2 text-[11px] font-medium text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded border border-emerald-500/20"
            >{m.reminders_status_on()}</span
          >
        {:else}
          <span class="ml-2 text-[11px] font-medium text-slate-400 bg-slate-800 px-2 py-0.5 rounded"
            >{m.reminders_status_off()}</span
          >
        {/if}
      </div>
      {#if settings?.enabled}
        <button
          type="button"
          class="btn btn-secondary btn-sm cursor-pointer"
          disabled={!canEdit || busy}
          onclick={disable}>{m.reminders_btn_disable()}</button
        >
      {:else}
        <button
          type="button"
          class="btn btn-secondary btn-sm cursor-pointer"
          disabled={!canEdit || busy || !hasTimezone}
          onclick={requestEnable}>{m.reminders_btn_enable()}</button
        >
      {/if}
    </div>
    <p class="text-xs text-slate-500">{m.reminders_opt_in_note()}</p>
    <p class="text-xs text-slate-500">{m.reminders_runtime_note()}</p>
  </div>

  <!-- Sending hours -->
  <div class="rounded-xl border border-slate-800 bg-slate-950/80 p-4 space-y-3">
    <span class="block text-sm font-semibold text-slate-100">{m.reminders_hours_title()}</span>
    <p class="text-xs text-slate-500">{m.reminders_hours_hint()}</p>
    <div class="flex flex-wrap items-end gap-3">
      <label class="text-xs text-slate-400">
        {m.reminders_hours_from()}
        <input
          type="time"
          bind:value={hoursStart}
          disabled={!canEdit || busy}
          class="mt-1 block rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-sm text-slate-100 disabled:opacity-50"
        />
      </label>
      <label class="text-xs text-slate-400">
        {m.reminders_hours_to()}
        <input
          type="time"
          bind:value={hoursEnd}
          disabled={!canEdit || busy}
          class="mt-1 block rounded-lg border border-slate-700 bg-slate-900 px-3 py-1.5 text-sm text-slate-100 disabled:opacity-50"
        />
      </label>
      <button
        type="button"
        class="btn btn-secondary btn-sm cursor-pointer"
        disabled={!canEdit || busy}
        onclick={saveHours}>{m.reminders_btn_save()}</button
      >
    </div>
  </div>

  <!-- Rules -->
  {#if rules.length > 0}
    <div class="space-y-3">
      <span class="block text-sm font-semibold text-slate-100">{m.reminders_rules_title()}</span>
      <p class="text-xs text-slate-500">
        {m.reminders_placeholders_hint()}
        <span class="font-mono text-slate-400">{PLACEHOLDERS.map((p) => `{${p}}`).join(" ")}</span>
      </p>
      {#each rules as rule (rule.id)}
        {@const isEmail = rule.channel === NotificationChannel.NotificationChannelEmail}
        {@const preview = previews[rule.id]}
        <div class="rounded-xl border border-slate-800 bg-slate-950/80 p-4 space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <span class="text-sm font-semibold text-slate-200">{ruleTitle(rule)}</span>
            <label class="flex items-center gap-2 text-xs text-slate-300">
              <input
                type="checkbox"
                checked={rule.enabled}
                disabled={!canEdit || busy}
                onchange={(e) => updateRule(rule.id, { enabled: e.currentTarget.checked })}
              />
              {m.reminders_rule_enabled()}
            </label>
          </div>

          <label class="block text-xs text-slate-400">
            {m.reminders_rule_provider()}
            <select
              value={rule.provider_name}
              disabled={!canEdit || busy}
              onchange={(e) => updateRule(rule.id, { provider_name: e.currentTarget.value })}
              class="mt-1 w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 disabled:opacity-50"
            >
              {#each providers[rule.channel] ?? [] as name}
                <option value={name}>{name}</option>
              {/each}
            </select>
          </label>

          {#if isEmail}
            <label class="block text-xs text-slate-400">
              {m.reminders_rule_subject()}
              <input
                type="text"
                value={rule.subject_template}
                disabled={!canEdit || busy}
                oninput={(e) => updateRule(rule.id, { subject_template: e.currentTarget.value })}
                class="mt-1 w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 disabled:opacity-50"
              />
            </label>
          {/if}

          <label class="block text-xs text-slate-400">
            {m.reminders_rule_body()}
            <textarea
              rows={isEmail ? 6 : 3}
              value={rule.body_template}
              disabled={!canEdit || busy}
              oninput={(e) => updateRule(rule.id, { body_template: e.currentTarget.value })}
              class="mt-1 w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 disabled:opacity-50"
            ></textarea>
          </label>

          {#if previewErrors[rule.id]}
            <p class="text-xs text-rose-400" role="alert">{previewErrors[rule.id]}</p>
          {:else if preview}
            <div class="rounded-lg border border-slate-800 bg-slate-900/60 p-3 space-y-1">
              <span class="block text-[11px] uppercase tracking-wider text-slate-500"
                >{m.reminders_preview_title()}</span
              >
              {#if isEmail}
                <p class="text-xs font-semibold text-slate-300">{preview.subject}</p>
              {/if}
              <p class="text-xs text-slate-300 whitespace-pre-line">{preview.body}</p>
              {#if !isEmail}
                <p class="text-[11px] {preview.too_long ? 'text-rose-400' : 'text-slate-500'}">
                  {m.reminders_preview_length({ count: preview.characters, max: MAX_SMS })}
                  {#if preview.too_long}
                    · {m.reminders_preview_too_long()}
                  {/if}
                </p>
              {/if}
            </div>
          {/if}

          <div class="flex justify-end">
            <button
              type="button"
              class="btn btn-secondary btn-sm cursor-pointer"
              disabled={!canEdit || busy}
              onclick={() => saveRule(rule)}>{m.reminders_btn_save()}</button
            >
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <!-- Job status -->
  <div
    class="rounded-xl border border-slate-800 bg-slate-950/80 p-4 space-y-1 text-xs text-slate-400"
  >
    <span class="block text-sm font-semibold text-slate-100">{m.reminders_job_title()}</span>
    {#if jobStatus?.last_run_at}
      <p>
        {m.reminders_job_last_run({ time: new Date(jobStatus.last_run_at).toLocaleString() })}
      </p>
      <p>
        {m.reminders_job_counts({
          sent: jobStatus.sent,
          failed: jobStatus.failed,
          unknown: jobStatus.unknown,
          skipped: jobStatus.skipped,
        })}
      </p>
    {:else}
      <p>{m.reminders_job_not_run()}</p>
    {/if}
    {#if jobStatus?.last_error}
      <p class="text-rose-400" role="alert">
        {m.reminders_job_error({ error: jobStatus.last_error })}
      </p>
    {/if}
    <button type="button" class="underline cursor-pointer" onclick={load}
      >{m.reminders_job_refresh()}</button
    >
  </div>
</div>

<ConfirmModal
  bind:showModal={showEnableConfirm}
  title={m.reminders_confirm_title()}
  message={m.reminders_confirm_counts({
    optedIn: counts?.opted_in ?? 0,
    mobile: counts?.with_mobile ?? 0,
    email: counts?.with_email ?? 0,
  })}
  confirmText={m.reminders_btn_enable()}
  onConfirm={confirmEnable}
>
  <p class="text-slate-400">{m.reminders_confirm_note()}</p>
</ConfirmModal>
