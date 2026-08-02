<script lang="ts">
  let {
    searchQuery = $bindable(),
    statusFilter = $bindable(),
    onloadpatients,
  } = $props<{
    searchQuery: string;
    statusFilter: string;
    onloadpatients: () => void;
  }>();
</script>

<div class="mb-4 flex items-center gap-3">
  <div class="relative w-full max-w-[480px] flex-1">
    <svg
      class="absolute left-3 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-slate-500"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
    >
      <circle cx="11" cy="11" r="8" />
      <line x1="21" y1="21" x2="16.65" y2="16.65" />
    </svg>
    <input
      type="text"
      placeholder="Search patients by name, email, or phone..."
      class="box-border w-full rounded-lg border border-slate-700 bg-slate-800 py-2.5 pl-10 pr-3 text-sm text-white focus:border-blue-500 focus:outline-none"
      bind:value={searchQuery}
      oninput={onloadpatients}
    />
  </div>
  <div>
    <select
      bind:value={statusFilter}
      onchange={onloadpatients}
      class="cursor-pointer rounded-lg border border-slate-700 bg-slate-800 px-3 py-2.5 text-sm text-white focus:border-blue-500 focus:outline-none"
    >
      <option value="active">Active Patients</option>
      <option value="archived">Archived Patients</option>
    </select>
  </div>
</div>
