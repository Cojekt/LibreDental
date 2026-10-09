<script lang="ts">
  import type { NotificationLog } from "@bindings/domain/models.js";
  import { NotificationService } from "@bindings/services/index.js";
  import { m } from "../paraglide/messages.js";
  import { auth } from "../stores/auth.svelte.js";
  import { handleError } from "$lib/error.js";

  // Shows messages sent to a patient, or for one appointment. Pass exactly one of the two.
  let { patientId = "", appointmentId = "" } = $props<{
    patientId?: string;
    appointmentId?: string;
  }>();

  let entries = $state<NotificationLog[]>([]);
  let loading = $state(false);
  let error = $state("");
  let requestGen = 0;

  async function load(patient: string, appointment: string) {
    const gen = ++requestGen;
    entries = [];
    error = "";
    if (!auth.token || (!patient && !appointment)) return;
    loading = true;
    try {
      const res = appointment
        ? await NotificationService.ListNotificationLogForAppointment(auth.token, appointment)
        : await NotificationService.ListNotificationLog(auth.token, patient, 20, 0);
      if (gen !== requestGen) return;
      entries = (res?.filter(Boolean) as NotificationLog[]) || [];
    } catch (e) {
      if (gen !== requestGen) return;
      console.error("Failed to load notification history:", e);
      error = handleError(e, m.notif_history_error());
    } finally {
      if (gen === requestGen) loading = false;
    }
  }

  $effect(() => {
    load(patientId, appointmentId);
  });

  function statusLabel(status: string): string {
    switch (status) {
      case "sent":
        return m.notif_status_sent();
      case "failed":
        return m.notif_status_failed();
      case "pending":
        return m.notif_status_pending();
      case "skipped":
        return m.notif_status_skipped();
      default:
        return m.notif_status_unknown();
    }
  }

  function statusClass(status: string): string {
    switch (status) {
      case "sent":
        return "text-emerald-400 bg-emerald-500/10 border-emerald-500/20";
      case "failed":
        return "text-rose-400 bg-rose-500/10 border-rose-500/20";
      case "skipped":
        return "text-slate-400 bg-slate-800 border-slate-700";
      default:
        return "text-amber-400 bg-amber-500/10 border-amber-500/20";
    }
  }

  function channelLabel(channel: string): string {
    switch (channel) {
      case "sms":
        return m.notif_channel_sms();
      case "email":
        return m.notif_channel_email();
      default:
        return m.notif_channel_voice();
    }
  }
</script>

<div class="space-y-2 text-xs">
  <p class="font-semibold text-slate-400 uppercase tracking-wider text-[11px]">
    {m.notif_history_title()}
  </p>
  {#if loading}
    <p class="text-slate-500">{m.notif_history_loading()}</p>
  {:else if error}
    <p class="text-rose-400" role="alert">{error}</p>
  {:else if entries.length === 0}
    <p class="text-slate-500">{m.notif_history_empty()}</p>
  {:else}
    <ul class="space-y-1.5">
      {#each entries as entry (entry.id)}
        <li class="rounded-lg border border-slate-700/60 bg-slate-900/40 p-2.5 space-y-1">
          <div class="flex items-center justify-between gap-2">
            <span class="text-slate-300">
              {channelLabel(entry.channel)}
              {#if entry.reminder_kind}
                <span class="text-slate-500">· {m.notif_history_automatic()}</span>
              {/if}
            </span>
            <span
              class="rounded border px-1.5 py-0.5 text-[11px] font-medium {statusClass(
                entry.status
              )}">{statusLabel(entry.status)}</span
            >
          </div>
          <div class="text-slate-500">
            {new Date(entry.sent_at).toLocaleString()} · {entry.recipient}
          </div>
          {#if entry.error_message}
            <div class="text-slate-400">{entry.error_message}</div>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
</div>
