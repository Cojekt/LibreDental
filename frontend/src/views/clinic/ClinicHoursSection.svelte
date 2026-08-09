<script lang="ts">
  import type { BusinessHourDay } from "@bindings/domain/models.js";

  let {
    businessHours = $bindable([]),
    isEditingProfile = false,
    formatTime12,
    addSlot,
    removeSlot,
    addBreak,
    removeBreak,
    syncDayBounds,
  } = $props<{
    businessHours: BusinessHourDay[];
    isEditingProfile: boolean;
    formatTime12: (time24: string) => string;
    addSlot: (hour: BusinessHourDay) => void;
    removeSlot: (hour: BusinessHourDay, index: number) => void;
    addBreak: (hour: BusinessHourDay) => void;
    removeBreak: (hour: BusinessHourDay, index: number) => void;
    syncDayBounds: (hour: BusinessHourDay) => void;
  }>();
</script>

<div class="rounded-xl border border-slate-800 bg-slate-900/70 p-6 space-y-6">
  <div class="flex items-center justify-between border-b border-slate-800 pb-4">
    <div>
      <h3 class="text-lg font-bold text-slate-100">⏰ Practice Operating Schedule</h3>
      <p class="text-xs text-slate-400 mt-0.5">
        Configure your weekly operating hours, split shifts, and scheduled breaks or closure gaps
        for appointment scheduling.
      </p>
    </div>
  </div>

  <div class="space-y-4">
    {#each businessHours as hour, idx}
      <div
        class={`p-4 rounded-xl border transition-all space-y-3 ${
          hour.is_closed
            ? "border-slate-800/60 bg-slate-950/40 opacity-70"
            : "border-slate-800 bg-slate-950/80 shadow-sm"
        }`}
      >
        <div class="flex flex-wrap items-center justify-between gap-4">
          <div class="w-36 flex items-center gap-3">
            <input
              type="checkbox"
              id={`open-${idx}`}
              checked={!hour.is_closed}
              onchange={(e) => (hour.is_closed = !(e.target as HTMLInputElement).checked)}
              disabled={!isEditingProfile}
            />
            <label
              for={`open-${idx}`}
              class="text-sm font-semibold text-slate-200 cursor-pointer select-none"
            >
              {hour.day}
            </label>
          </div>

          {#if hour.is_closed}
            <span
              class="text-xs font-semibold text-rose-400 bg-rose-500/10 border border-rose-500/20 px-3 py-1 rounded-full"
            >
              CLOSED
            </span>
          {:else if !isEditingProfile}
            <div class="flex flex-wrap items-center gap-3 text-xs font-medium text-slate-300">
              {#if hour.slots && hour.slots.length > 1}
                <div class="flex flex-wrap items-center gap-2">
                  {#each hour.slots as slot, sIdx}
                    <span
                      class="bg-sky-500/10 border border-sky-500/20 text-sky-300 px-2.5 py-1 rounded-lg"
                    >
                      Shift {sIdx + 1}: {formatTime12(slot.open_time)} – {formatTime12(
                        slot.close_time
                      )}
                    </span>
                  {/each}
                </div>
              {:else}
                <div class="flex items-center gap-2">
                  <span>Opens:</span>
                  <span class="font-semibold text-slate-100">{formatTime12(hour.open_time)}</span>
                  <span class="ml-2">Closes:</span>
                  <span class="font-semibold text-slate-100">{formatTime12(hour.close_time)}</span>
                </div>
              {/if}

              {#if hour.breaks && hour.breaks.length > 0}
                <div class="flex flex-wrap items-center gap-2">
                  {#each hour.breaks as brk}
                    <span
                      class="bg-amber-500/10 border border-amber-500/20 text-amber-300 px-2.5 py-1 rounded-lg flex items-center gap-1"
                    >
                      <span>☕ {brk.name || "Break"}:</span>
                      <span class="font-semibold"
                        >{formatTime12(brk.start_time)} – {formatTime12(brk.end_time)}</span
                      >
                    </span>
                  {/each}
                </div>
              {/if}
            </div>
          {/if}
        </div>

        {#if !hour.is_closed && isEditingProfile}
          <div class="pl-4 border-l-2 border-sky-500/30 space-y-3 pt-2">
            <div class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-xs font-semibold text-slate-400">Working Hours / Shifts</span>
                <button
                  type="button"
                  onclick={() => addSlot(hour)}
                  class="text-xs text-sky-400 hover:text-sky-300 font-semibold flex items-center gap-1 transition-colors"
                >
                  <span>+ Add Split Shift / Time Slot</span>
                </button>
              </div>

              <div class="space-y-2">
                {#each hour.slots || [] as slot, sIdx}
                  <div
                    class="flex flex-wrap items-center gap-3 text-xs bg-slate-900/60 p-2.5 rounded-lg border border-slate-800"
                  >
                    {#if (hour.slots || []).length > 1}
                      <span class="font-semibold text-slate-400 text-[11px]">Shift {sIdx + 1}:</span
                      >
                    {/if}

                    <div class="flex items-center gap-2">
                      <span class="text-slate-400">Opens:</span>
                      <input
                        type="time"
                        bind:value={slot.open_time}
                        onchange={() => syncDayBounds(hour)}
                        class="rounded-lg border border-slate-700 bg-slate-950 px-2.5 py-1 text-xs text-slate-100 focus:border-sky-500 focus:outline-none"
                      />
                    </div>

                    <div class="flex items-center gap-2">
                      <span class="text-slate-400">Closes:</span>
                      <input
                        type="time"
                        bind:value={slot.close_time}
                        onchange={() => syncDayBounds(hour)}
                        class="rounded-lg border border-slate-700 bg-slate-950 px-2.5 py-1 text-xs text-slate-100 focus:border-sky-500 focus:outline-none"
                      />
                    </div>

                    {#if (hour.slots || []).length > 1}
                      <button
                        type="button"
                        onclick={() => removeSlot(hour, sIdx)}
                        class="text-rose-400 hover:text-rose-300 ml-auto text-xs px-2 py-0.5 rounded hover:bg-rose-500/10 transition-colors"
                        title="Remove Shift"
                      >
                        ✕ Remove
                      </button>
                    {/if}
                  </div>
                {/each}
              </div>
            </div>

            <div class="pt-2 border-t border-slate-800/60 space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-xs font-semibold text-amber-400/90 flex items-center gap-1.5">
                  <span>☕ Scheduled Breaks & Gaps</span>
                </span>
                <button
                  type="button"
                  onclick={() => addBreak(hour)}
                  class="text-xs text-amber-400 hover:text-amber-300 font-semibold flex items-center gap-1 transition-colors"
                >
                  <span>+ Add Break / Schedule Gap</span>
                </button>
              </div>

              {#if (hour.breaks || []).length === 0}
                <p class="text-[11px] text-slate-500 italic">
                  No scheduled breaks or closure gaps configured for this day.
                </p>
              {:else}
                <div class="space-y-2">
                  {#each hour.breaks || [] as brk, bIdx}
                    <div
                      class="flex flex-wrap items-center gap-3 text-xs bg-amber-500/5 p-2.5 rounded-lg border border-amber-500/20"
                    >
                      <div class="flex items-center gap-2 flex-1 min-w-[160px]">
                        <span class="text-amber-400 font-semibold text-[11px]">Label:</span>
                        <input
                          type="text"
                          bind:value={brk.name}
                          placeholder="e.g. Lunch Break, Staff Meeting"
                          class="w-full rounded border border-amber-500/30 bg-slate-950 px-2.5 py-1 text-xs text-slate-100 focus:border-amber-400 focus:outline-none"
                        />
                      </div>

                      <div class="flex items-center gap-2">
                        <span class="text-amber-400 text-[11px]">Start:</span>
                        <input
                          type="time"
                          bind:value={brk.start_time}
                          class="rounded border border-amber-500/30 bg-slate-950 px-2 py-1 text-xs text-slate-100 focus:border-amber-400 focus:outline-none"
                        />
                      </div>

                      <div class="flex items-center gap-2">
                        <span class="text-amber-400 text-[11px]">End:</span>
                        <input
                          type="time"
                          bind:value={brk.end_time}
                          class="rounded border border-amber-500/30 bg-slate-950 px-2 py-1 text-xs text-slate-100 focus:border-amber-400 focus:outline-none"
                        />
                      </div>

                      <button
                        type="button"
                        onclick={() => removeBreak(hour, bIdx)}
                        class="text-rose-400 hover:text-rose-300 text-xs px-2 py-1 rounded hover:bg-rose-500/10 transition-colors ml-auto"
                        title="Remove Break"
                      >
                        ✕ Remove
                      </button>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          </div>
        {/if}
      </div>
    {/each}
  </div>
</div>
