<script lang="ts">
  let { patientCount } = $props<{ patientCount: number }>();
</script>

<div class="mb-6 grid grid-cols-[repeat(auto-fit,minmax(280px,1fr))] gap-4">
  <div class="flex items-center gap-4 rounded-xl border border-slate-700 bg-slate-800 p-5">
    <div class="text-[28px] text-indigo-400">👥</div>
    <div>
      <div class="text-[13px] text-slate-400">Total Patients</div>
      <div class="mt-0.5 text-xl font-bold text-slate-50">{patientCount}</div>
    </div>
  </div>

  <div class="flex items-center gap-4 rounded-xl border border-slate-700 bg-slate-800 p-5">
    <div class="text-[28px] text-teal-400">📅</div>
    <div>
      <div class="text-[13px] text-slate-400">Today's Schedule</div>
      <div class="mt-0.5 text-xl font-bold text-slate-50">0 Appointments</div>
    </div>
  </div>

  <div class="flex items-center gap-4 rounded-xl border border-slate-700 bg-slate-800 p-5">
    <div class="text-[28px] text-emerald-400">🛡️</div>
    <div>
      <div class="text-[13px] text-slate-400">HIPAA Data Protection</div>
      <div class="mt-0.5 text-xl font-bold text-emerald-400">Compliant & Secure</div>
    </div>
  </div>
</div>
