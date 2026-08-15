<script lang="ts">
  let {
    variant = "scheduled",
    label = "",
    size = "md",
    customClass = "",
  } = $props<{
    variant?: string;
    label?: string;
    size?: "sm" | "md" | "lg";
    customClass?: string;
  }>();

  const variantStyles: Record<string, string> = {
    // Appointment Statuses
    scheduled: "bg-blue-500/15 text-blue-400 border-blue-500/30",
    confirmed: "bg-sky-500/15 text-sky-400 border-sky-500/30",
    arrived: "bg-amber-500/15 text-amber-400 border-amber-500/30",
    in_chair: "bg-purple-500/15 text-purple-400 border-purple-500/30",
    completed: "bg-emerald-500/15 text-emerald-400 border-emerald-500/30",
    cancelled: "bg-rose-500/15 text-rose-400 border-rose-500/30",
    no_show: "bg-slate-500/15 text-slate-400 border-slate-500/30",

    // Claim Statuses
    draft: "bg-slate-700/80 text-slate-300 border-slate-600/50",
    submitted: "bg-blue-900/60 text-blue-300 border-blue-700/50",
    accepted: "bg-emerald-900/60 text-emerald-300 border-emerald-700/50",
    rejected: "bg-red-900/60 text-red-300 border-red-700/50",
    paid: "bg-emerald-500/15 text-emerald-400 border-emerald-500/30",

    // Timecard Statuses
    manual: "bg-amber-500/15 text-amber-400 border-amber-500/30",
    punched: "bg-slate-500/15 text-slate-400 border-slate-500/30",
    unpaid: "bg-rose-500/15 text-rose-400 border-rose-500/30",

    // Tooth Condition Statuses
    missing: "bg-rose-950/80 text-rose-300 border-rose-800",
    treatment_planned: "bg-amber-950/80 text-amber-300 border-amber-800",
    existing: "bg-blue-950/80 text-blue-300 border-blue-800",

    // Payment Methods
    cash: "bg-emerald-900/60 text-emerald-300 border-emerald-700/50",
    check: "bg-sky-900/60 text-sky-300 border-sky-700/50",
    credit_card: "bg-violet-900/60 text-violet-300 border-violet-700/50",
    insurance: "bg-blue-900/60 text-blue-300 border-blue-700/50",
    write_off: "bg-slate-700/80 text-slate-400 border-slate-600/50",
  };

  const sizeStyles: Record<string, string> = {
    sm: "px-1.5 py-0.2 text-[9px]",
    md: "px-2 py-0.5 text-[10px]",
    lg: "px-2.5 py-1 text-xs",
  };

  let activeClass = $derived(
    variantStyles[variant] || "bg-slate-800 text-slate-300 border-slate-700"
  );
  let activeSize = $derived(sizeStyles[size] || sizeStyles.md);
</script>

<span
  class={`inline-flex items-center justify-center rounded-full border font-semibold uppercase tracking-wider ${activeSize} ${activeClass} ${customClass}`}
>
  {label || variant}
</span>
