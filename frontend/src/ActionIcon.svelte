<script lang="ts">
  let { kind }: { kind: 'edit' | 'delete' } = $props();
</script>
<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">
  {#if kind === 'edit'}
    <path d="M16 3a2.12 2.12 0 0 1 3 3L8 17l-4 1 1-4Z" />
    <path d="m14 5 3 3M4 21h16" />
  {:else}
    <path d="M3 6h18M9 6V3h6v3M5 6l1 15h12l1-15M10 10v7M14 10v7" />
  {/if}
</svg>
