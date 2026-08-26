<script lang="ts">
  import { onMount } from "svelte";
  import { m } from "../paraglide/messages.js";
  import { AuditService } from "@bindings/services/index.js";
  import type { Patient, AuditLogEntry } from "@bindings/domain/index.js";

  let { patients = [] } = $props<{
    patients: Patient[];
  }>();

  let logs = $state<AuditLogEntry[]>([]);
  let loading = $state(true);
  let selectedPatient = $state("");
  let page = $state(0);
  let limit = $state(50);

  function getPatientName(id: string): string {
    const p = patients.find((p: Patient) => p.id === id);
    return p ? `${p.last_name}, ${p.first_name}` : id;
  }

  async function fetchLogs() {
    loading = true;
    try {
      const res = await AuditService.GetAuditLogs(selectedPatient, limit, page * limit);
      logs = (res?.filter(Boolean) as AuditLogEntry[]) || [];
    } catch (e) {
      console.error("Failed to fetch audit logs", e);
      logs = [];
    } finally {
      loading = false;
    }
  }

  function handlePrev() {
    if (page > 0) {
      page--;
      fetchLogs();
    }
  }

  function handleNext() {
    page++;
    fetchLogs();
  }

  $effect(() => {
    // When selectedPatient changes, reset page and fetch
    if (selectedPatient !== undefined) {
      page = 0;
      fetchLogs();
    }
  });

  onMount(() => {
    fetchLogs();
  });
</script>

<div class="flex h-full flex-col">
  <!-- Toolbar -->
  <div class="mb-4 flex items-center justify-between">
    <div class="flex items-center gap-3">
      <h2 class="text-lg font-bold text-slate-100">{m.audit_tab_auditing()}</h2>

      <select
        bind:value={selectedPatient}
        class="input input-sm w-64 bg-slate-800 text-slate-200 border-slate-700"
      >
        <option value="">All Patients</option>
        {#each patients as p}
          <option value={p.id}>{p.last_name}, {p.first_name}</option>
        {/each}
      </select>
    </div>

    <div class="flex items-center gap-2">
      <button class="btn btn-secondary btn-sm" onclick={fetchLogs} title="Refresh">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="h-4 w-4">
          <path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.59-9.21l5.67-5.67" />
        </svg>
      </button>
      <button class="btn btn-secondary btn-sm" onclick={handlePrev} disabled={page === 0}
        >Prev</button
      >
      <span class="text-sm text-slate-400">Page {page + 1}</span>
      <button class="btn btn-secondary btn-sm" onclick={handleNext} disabled={logs.length < limit}
        >Next</button
      >
    </div>
  </div>

  <!-- Table -->
  <div class="flex-1 overflow-auto rounded-lg border border-slate-800 bg-slate-900/50">
    <table class="w-full text-left text-sm text-slate-300">
      <thead class="sticky top-0 bg-slate-800/90 uppercase text-slate-400 backdrop-blur">
        <tr>
          <th class="px-4 py-3 font-medium">Timestamp</th>
          <th class="px-4 py-3 font-medium">User</th>
          <th class="px-4 py-3 font-medium">Patient</th>
          <th class="px-4 py-3 font-medium">Action</th>
          <th class="px-4 py-3 font-medium">Resource</th>
          <th class="px-4 py-3 font-medium">Details</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-800">
        {#if loading}
          <tr>
            <td colspan="6" class="p-8 text-center text-slate-500">Loading audit logs...</td>
          </tr>
        {:else if logs.length === 0}
          <tr>
            <td colspan="6" class="p-8 text-center text-slate-500">No audit logs found.</td>
          </tr>
        {:else}
          {#each logs as log}
            <tr class="hover:bg-slate-800/40">
              <td class="whitespace-nowrap px-4 py-3 font-mono text-xs"
                >{new Date(log.timestamp).toLocaleString()}</td
              >
              <td class="px-4 py-3"
                >{log.user_name} <span class="text-xs text-slate-500">({log.user_id})</span></td
              >
              <td class="px-4 py-3">{log.patient_id ? getPatientName(log.patient_id) : "-"}</td>
              <td class="px-4 py-3">
                <span
                  class="inline-flex rounded-full bg-slate-800 px-2.5 py-0.5 text-xs font-medium text-slate-300"
                >
                  {log.action}
                </span>
              </td>
              <td class="px-4 py-3">{log.resource}</td>
              <td class="px-4 py-3 text-xs text-slate-400">{log.details || "-"}</td>
            </tr>
          {/each}
        {/if}
      </tbody>
    </table>
  </div>
</div>
