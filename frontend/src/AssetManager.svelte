<script lang="ts">
  import { onDestroy, tick } from 'svelte';
  import { ListAssets, AddAsset } from '../wailsjs/go/desktop/CashFlowAPI';
  let { onadded }: { onadded: (asset: Awaited<ReturnType<typeof ListAssets>>[number]) => void } = $props();
  let dialog: HTMLDialogElement;
  let input: HTMLInputElement;
  let opened = $state(false), loading = $state(false), busy = $state(false);
  let assets = $state<Awaited<ReturnType<typeof ListAssets>>>([]);
  let ticker = $state(''), error = $state(''), fieldError = $state(''), message = $state('');
  let request = 0, alive = true;
  let returnTo: HTMLElement | null;
  async function load() {
    const id = ++request; loading = true; error = '';
    try { const rows = await ListAssets(); if (alive && opened && id === request) assets = rows; }
    catch { if (alive && opened && id === request) error = 'Не удалось загрузить активы'; }
    finally { if (alive && id === request) loading = false; }
  }
  export async function openList(trigger?: HTMLElement) {
    returnTo = trigger ?? (document.activeElement instanceof HTMLElement ? document.activeElement : null);
    opened = true; ticker = ''; fieldError = ''; message = '';
    await tick(); dialog.showModal(); input.focus(); void load();
  }
  function close() { if (busy) return; opened = false; ++request; dialog.close(); returnTo?.focus(); }
  async function add(event: SubmitEvent) {
    event.preventDefault(); if (busy) return;
    busy = true; fieldError = ''; message = '';
    try {
      const result = await AddAsset(ticker);
      if (!alive || !opened) return;
      fieldError = result.errors.ticker ?? ''; message = result.message;
      if (!fieldError && !message) {
        ticker = ''; onadded(result.asset); await load();
        message = `Добавлен ${result.asset.ticker}`;
      }
    } catch { if (alive) fieldError = 'Не удалось добавить актив'; }
    finally { busy = false; }
    if (alive && opened) { await tick(); input.focus(); }
  }
  onDestroy(() => { alive = false; ++request; });
</script>
<dialog bind:this={dialog} aria-labelledby="assets-title" oncancel={e => { e.preventDefault(); close(); }}>
  <h2 id="assets-title">Активы</h2>
  {#if loading}<p role="status">Загрузка…</p>
  {:else if error}<p role="alert">{error}</p><button onclick={() => void load()}>Повторить загрузку</button>
  {:else if !assets.length}<p>Активов пока нет</p>
  {:else}<ul>{#each assets as asset (asset.id)}<li>{asset.ticker}</li>{/each}</ul>{/if}
  <form onsubmit={add} novalidate>
    <label for="asset-ticker">Новый тикер</label>
    <input id="asset-ticker" bind:this={input} bind:value={ticker} disabled={busy} aria-invalid={!!fieldError} aria-describedby={fieldError ? 'asset-error' : undefined} autocomplete="off" />
    {#if fieldError}<p id="asset-error" class="error" role="alert">{fieldError}</p>{/if}
    {#if message}<p class:error={!message.startsWith('Добавлен ')} role={message.startsWith('Добавлен ') ? 'status' : 'alert'}>{message}</p>{/if}
    <div class="actions"><button type="button" disabled={busy} onclick={close}>Закрыть</button><button type="submit" disabled={busy}>{busy ? 'Сохранение…' : 'Добавить'}</button></div>
  </form>
</dialog>
<style>
  button, input { font: inherit; color: inherit; padding: 10px 16px; border: 1px solid #c4cfd8; border-radius: 8px; background: white; }
  button { cursor: pointer; } button:disabled { opacity: .6; cursor: default; }
  input { width: 100%; } label { display: block; margin: 16px 0 8px; }
  dialog { width: min(520px, calc(100% - 32px)); max-height: calc(100% - 32px); padding: 28px; border: 1px solid #dce3e9; border-radius: 16px; color: inherit; }
  dialog::backdrop { background: #172a3a66; } ul { max-height: 240px; overflow-y: auto; }
  .actions { display: flex; justify-content: flex-end; gap: 12px; margin-top: 20px; }
  .error { color: #a12e2e; } [aria-invalid="true"] { border-color: #a12e2e; }
</style>
