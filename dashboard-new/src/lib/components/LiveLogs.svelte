<script lang="ts">
  export let cmdH2: string = '';
  export let cmdH3: string = '';
  export let logH2: string = '';
  export let logH3: string = '';
  export let phaseH2: string = '';
  export let phaseH3: string = '';
</script>

<div class="space-y-3">
  <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
    <div class="card p-3">
      <div class="text-xs muted mb-1">HTTP/2 Command</div>
      <pre class="text-[11px] md:text-xs font-mono whitespace-pre-wrap break-words">{cmdH2}</pre>
    </div>
    <div class="card p-3">
      <div class="text-xs muted mb-1">HTTP/3 Command</div>
      <pre class="text-[11px] md:text-xs font-mono whitespace-pre-wrap break-words">{cmdH3}</pre>
    </div>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
    <div class="card p-0">
      <div class="flex items-center justify-between px-3 py-2 border-b border-gray-200 dark:border-white/10">
        <div class="text-sm font-semibold">HTTP/2 Logs</div>
        <div class="text-xs muted">{phaseH2 || 'idle'}</div>
      </div>
      <pre class="h-64 p-3 text-[11px] md:text-xs font-mono whitespace-pre-wrap break-words overflow-auto">{logH2}</pre>
    </div>
    <div class="card p-0">
      <div class="flex items-center justify-between px-3 py-2 border-b border-gray-200 dark:border-white/10">
        <div class="text-sm font-semibold">HTTP/3 Logs</div>
        <div class="text-xs muted">{phaseH3 || 'waiting'}</div>
      </div>
      <pre class="h-64 p-3 text-[11px] md:text-xs font-mono whitespace-pre-wrap break-words overflow-auto">{logH3}</pre>
    </div>
  </div>
</div>

