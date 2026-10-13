<script lang="ts">
  import { onDestroy, tick } from 'svelte';
  import { ListAssets, ListAssetTypes, AddAsset, UpdateAssetType } from '../wailsjs/go/desktop/CashFlowAPI';
  let { onadded, onchanged = () => {} }: { onadded: (asset: Awaited<ReturnType<typeof ListAssets>>[number]) => void; onchanged?: () => void } = $props();
  let dialog: HTMLDialogElement;
  let input: HTMLInputElement;
  let typeSelect: HTMLSelectElement;
  let opened = $state(false), loading = $state(false), busy = $state(false);
  let assets = $state<Awaited<ReturnType<typeof ListAssets>>>([]);
  let types = $state<Awaited<ReturnType<typeof ListAssetTypes>>>([]);
  let ticker = $state(''), typeCode = $state(''), error = $state(''), fieldError = $state(''), typeError = $state(''), message = $state('');
  let editing = $state<Awaited<ReturnType<typeof ListAssets>>[number] | null>(null);
  let success = $state(false);
  let request = 0, alive = true;
  let returnTo: HTMLElement | null, editTrigger: HTMLElement | null;
  async function load() {
    const id = ++request; loading = true; error = '';
    try {
      const [rows, catalog] = await Promise.all([ListAssets(), ListAssetTypes()]);
      if (alive && opened && id === request) { assets = rows; types = catalog; }
    }
    catch { if (alive && opened && id === request) error = 'Не удалось загрузить активы'; }
    finally { if (alive && id === request) loading = false; }
  }
  export async function openList(trigger?: HTMLElement) {
    returnTo = trigger ?? (document.activeElement instanceof HTMLElement ? document.activeElement : null);
    opened = true; editing = null; ticker = ''; typeCode = ''; fieldError = ''; typeError = ''; message = ''; success = false;
    await tick(); dialog.showModal(); input.focus(); void load();
  }
  function close() { if (busy) return; opened = false; ++request; dialog.close(); returnTo?.focus(); }
  async function edit(asset: Awaited<ReturnType<typeof ListAssets>>[number], trigger: HTMLElement) {
    if (busy || loading) return;
    editTrigger = trigger; editing = asset; ticker = asset.ticker; typeCode = asset.typeCode;
    fieldError = ''; typeError = ''; message = ''; success = false;
    await tick(); typeSelect.focus();
  }
  async function cancelEdit() {
    if (busy) return;
    editing = null; ticker = ''; typeCode = ''; fieldError = ''; typeError = ''; message = ''; success = false;
    await tick(); editTrigger?.focus();
  }
  async function add(event: SubmitEvent) {
    event.preventDefault(); if (busy || loading || error) return;
    busy = true; fieldError = ''; typeError = ''; message = ''; success = false;
    const old = editing;
    try {
      const result = old ? await UpdateAssetType({ id: old.id, typeCode }) : await AddAsset({ ticker, typeCode });
      if (!alive || !opened) return;
      fieldError = result.errors.ticker ?? ''; typeError = result.errors.typeCode ?? ''; message = result.message;
      if (!fieldError && !typeError && !message) {
        ticker = ''; typeCode = ''; editing = null;
        if (old) onchanged(); else onadded(result.asset);
        await load();
        if (!alive || !opened) return;
        success = true; message = old ? `Тип ${result.asset.ticker} изменён` : `Добавлен ${result.asset.ticker}`;
      }
    } catch { if (alive && opened) message = 'Не удалось сохранить актив. Повторите попытку'; }
    finally { busy = false; }
    if (alive && opened) {
      await tick();
      if (typeError || editing) typeSelect.focus(); else if (old && success) dialog.querySelector<HTMLButtonElement>(`[data-asset-id="${old.id}"]`)?.focus(); else input.focus();
    }
  }
  const typeName = (code: string) => types.find(v => v.code === code)?.name ?? 'Тип не указан';
  onDestroy(() => { alive = false; ++request; });
</script>
<dialog bind:this={dialog} aria-labelledby="assets-title" oncancel={e => { e.preventDefault(); if (editing) void cancelEdit(); else close(); }}>
  <h2 id="assets-title">Активы</h2>
  {#if loading}<p role="status">Загрузка…</p>
  {:else if error}<p role="alert">{error}</p><button disabled={busy} onclick={() => void load()}>Повторить загрузку</button>
  {:else if !assets.length}<p>Активов пока нет</p>
  {:else}<ul>{#each assets as asset (asset.id)}<li><span>{asset.ticker} — {typeName(asset.typeCode)}</span><button data-asset-id={asset.id} disabled={busy || !!editing} aria-label={`Изменить тип ${asset.ticker}`} onclick={e => void edit(asset, e.currentTarget)}>Изменить тип</button></li>{/each}</ul>{/if}
  <form onsubmit={add} novalidate>
    <label for="asset-ticker">{editing ? 'Тикер' : 'Новый тикер'}</label>
    <input id="asset-ticker" bind:this={input} bind:value={ticker} readonly={!!editing} disabled={busy} aria-invalid={!!fieldError} aria-describedby={fieldError ? 'asset-error' : undefined} autocomplete="off" />
    {#if fieldError}<p id="asset-error" class="error" role="alert">{fieldError}</p>{/if}
    <label for="asset-type">Тип актива</label>
    <select id="asset-type" bind:this={typeSelect} bind:value={typeCode} disabled={busy || loading || !!error} aria-invalid={!!typeError} aria-describedby={typeError ? 'asset-type-error' : undefined}>
      <option value="">Выберите тип</option>
      {#each types as type (type.code)}<option value={type.code}>{type.name}</option>{/each}
    </select>
    {#if typeError}<p id="asset-type-error" class="error" role="alert">{typeError}</p>{/if}
    {#if message}<p class:error={!success} role={success ? 'status' : 'alert'}>{message}</p>{/if}
    <div class="actions">
      <button type="button" disabled={busy} onclick={close}>Закрыть</button>
      {#if editing}<button type="button" disabled={busy} onclick={() => void cancelEdit()}>Отмена</button>{/if}
      <button type="submit" disabled={busy || loading || !!error}>{busy ? 'Сохранение…' : editing ? 'Сохранить' : 'Добавить'}</button>
    </div>
  </form>
</dialog>
<style>
  button, input, select { font: inherit; color: inherit; padding: 10px 16px; border: 1px solid #c4cfd8; border-radius: 8px; background: white; }
  button { cursor: pointer; } button:disabled { opacity: .6; cursor: default; }
  input, select { width: 100%; } label { display: block; margin: 16px 0 8px; }
  dialog { width: min(620px, calc(100% - 32px)); max-height: calc(100% - 32px); padding: 28px; border: 1px solid #dce3e9; border-radius: 16px; color: inherit; }
  dialog::backdrop { background: #172a3a66; } ul { max-height: 240px; overflow-y: auto; padding: 0; }
  li { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin: 8px 0; } li span { overflow-wrap: anywhere; min-width: 0; } li button { flex-shrink: 0; }
  .actions { display: flex; justify-content: flex-end; flex-wrap: wrap; gap: 12px; margin-top: 20px; }
  .error { color: #a12e2e; } [aria-invalid="true"] { border-color: #a12e2e; }
</style>
