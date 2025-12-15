<script lang="ts">
  export let tone: "good" | "bad" | "warn" | "neutral" = "neutral";
  const map = {
    good: "bg-emerald-100 text-emerald-800 border-emerald-200",
    bad: "bg-rose-100 text-rose-800 border-rose-200",
    warn: "bg-amber-100 text-amber-800 border-amber-200",
    neutral: "bg-gray-100 text-gray-800 border-gray-200",
  } as const;
</script>

<span
  class={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium border ${map[tone]}`}
>
  <slot />
</span>
