<script lang="ts">
  import type { Patient } from "@bindings/domain/index.js";
  import AuditingSubtab from "./AuditingSubtab.svelte";

  let { patients = [] } = $props<{
    patients: Patient[];
  }>();
</script>

<div class="flex h-full w-full flex-col gap-6">
  <div
    class="flex-1 overflow-auto rounded-xl border border-slate-800/50 bg-slate-900/40 p-6 shadow-sm"
  >
    <AuditingSubtab {patients} />
  </div>
</div>
