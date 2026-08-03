<script lang="ts">
  import type { Patient, CountryConfig } from "../../bindings/github.com/LibreDental/libredental/pkg/domain/models.js";

  let {
    patients,
    loading,
    statusFilter,
    countryMeta,
    onaddpatient,
    oneditpatient,
    onarchivepatient,
  } = $props<{
    patients: Patient[];
    loading: boolean;
    statusFilter: string;
    countryMeta?: CountryConfig | null;
    onaddpatient: () => void;
    oneditpatient: (p: Patient) => void;
    onarchivepatient: (p: Patient) => void;
  }>();

  const idHeader = $derived(countryMeta ? countryMeta.national_id_name : "National ID");
</script>

<div class="overflow-hidden rounded-xl border border-slate-700 bg-slate-800">
  {#if loading}
    <div class="p-12 text-center text-slate-400">Loading local patient database...</div>
  {:else if patients.length === 0}
    <div class="p-12 text-center text-slate-400">
      <p class="mb-2 text-lg font-semibold text-slate-50">No patients found</p>
      <p class="mb-4 text-sm">No patients match your current filters.</p>
      {#if statusFilter === "active"}
        <button class="btn btn-secondary" onclick={onaddpatient}>Add First Patient</button>
      {/if}
    </div>
  {:else}
    <table class="w-full border-collapse text-left">
      <thead>
        <tr>
          <th class="border-b border-slate-700 bg-slate-900 px-5 py-3.5 text-xs font-semibold uppercase tracking-wider text-slate-400">Patient Name</th>
          <th class="border-b border-slate-700 bg-slate-900 px-5 py-3.5 text-xs font-semibold uppercase tracking-wider text-slate-400">{idHeader}</th>
          <th class="border-b border-slate-700 bg-slate-900 px-5 py-3.5 text-xs font-semibold uppercase tracking-wider text-slate-400">Contact Info</th>
          <th class="border-b border-slate-700 bg-slate-900 px-5 py-3.5 text-xs font-semibold uppercase tracking-wider text-slate-400">Date of Birth</th>
          <th class="border-b border-slate-700 bg-slate-900 px-5 py-3.5 text-xs font-semibold uppercase tracking-wider text-slate-400">Medical Alerts</th>
          <th class="border-b border-slate-700 bg-slate-900 px-5 py-3.5 text-xs font-semibold uppercase tracking-wider text-slate-400">Actions</th>
        </tr>
      </thead>
      <tbody>
        {#each patients as p}
          <tr>
            <td class="border-b border-slate-700 px-5 py-4 text-sm">
              <div class="font-semibold text-slate-50">{p.first_name} {p.last_name}</div>
              <div class="mt-0.5 text-[11px] text-slate-500">{p.id}</div>
            </td>
            <td class="border-b border-slate-700 px-5 py-4 text-sm">
              {#if p.national_id}
                <span class="font-mono text-xs text-blue-400 bg-blue-500/10 px-2 py-1 rounded border border-blue-500/20">{p.national_id}</span>
              {:else}
                <span class="text-slate-500 text-xs italic">Unassigned</span>
              {/if}
            </td>
            <td class="border-b border-slate-700 px-5 py-4 text-sm">
              <div>{p.phone_primary || "No phone"}</div>
              <div class="text-xs text-slate-400">{p.email || "No email"}</div>
            </td>
            <td class="border-b border-slate-700 px-5 py-4 text-sm">{p.date_of_birth ? new Date(p.date_of_birth).toLocaleDateString() : "N/A"}</td>
            <td class="border-b border-slate-700 px-5 py-4 text-sm">
              {#if p.medical_alerts && p.medical_alerts.length > 0}
                <div class="flex flex-wrap gap-1.5">
                  {#each p.medical_alerts as alert}
                    <span class="rounded-md border border-amber-400/30 bg-amber-500/15 px-2 py-1 text-xs font-medium text-amber-400">⚠️ {alert}</span>
                  {/each}
                </div>
              {:else}
                <span class="rounded-md bg-emerald-500/15 px-2 py-1 text-xs font-medium text-emerald-400">Clean Record</span>
              {/if}
            </td>
            <td class="border-b border-slate-700 px-5 py-4 text-sm">
              <div class="flex items-center gap-2">
                <button class="btn btn-ghost btn-sm" onclick={() => oneditpatient(p)}>Edit</button>
                {#if p.status !== "archived"}
                  <button class="btn btn-ghost btn-danger btn-sm" onclick={() => onarchivepatient(p)}>Archive</button>
                {/if}
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>
