<script lang="ts">
  import { tick, onDestroy } from 'svelte';
  import { ListBenchmarks, SaveBenchmarkSources } from '../wailsjs/go/desktop/BenchmarksAPI';
  let dialog: HTMLDialogElement;
  let sourceInput = $state<HTMLTextAreaElement>();
  let opened = $state(false), loading = $state(false), busy = $state(false);
  let rows = $state<Awaited<ReturnType<typeof ListBenchmarks>>>([]);
  let editing = $state(''), sources = $state(''), error = $state(''), fieldError = $state(''), message = $state('');
  let returnTo: HTMLElement | undefined;
  let editTrigger: HTMLElement | undefined;
  let alive = true, request = 0;
  async function load() {
    const id = ++request; loading = true; error = '';
    try { const result = await ListBenchmarks(); if (alive && opened && id === request) rows = result; }
    catch { if (alive && opened && id === request) error = 'Не удалось загрузить бенчмарки. Повторите попытку.'; }
    finally { if (alive && id === request) loading = false; }
  }
  export async function openList(trigger?: HTMLElement) {
    returnTo = trigger; opened = true; editing = ''; fieldError = ''; message = ''; rows = [];
    await tick(); dialog.showModal(); void load();
  }
  function close() { if (busy) return; opened = false; ++request; dialog.close(); returnTo?.focus(); }
  async function edit(row: (typeof rows)[number], trigger: HTMLElement) {
    editing = row.id; sources = row.sources; fieldError = ''; message = ''; editTrigger = trigger;
    await tick(); sourceInput?.focus();
  }
  async function cancel() { if (busy) return; editing = ''; fieldError = ''; await tick(); editTrigger?.focus(); }
  async function save(event: SubmitEvent) {
    event.preventDefault(); if (busy) return; busy = true; fieldError = ''; message = '';
    try {
      const result = await SaveBenchmarkSources(editing, sources);
      if (!alive || !opened) return;
      fieldError = result.errors.sources || result.errors.benchmarkID || ''; message = result.message;
      if (!fieldError && !message) {
        rows = rows.map(row => row.id === editing ? {...row, sources: sources.split('\n').map(s => s.trim()).filter(Boolean).join('\n')} : row);
        editing = ''; message = 'Источники сохранены';
      }
    } catch { if (alive && opened) fieldError = 'Не удалось сохранить источники. Повторите попытку.'; }
    finally { if (alive) { busy = false; if (opened && !editing) { await tick(); editTrigger?.focus(); } } }
  }
  onDestroy(() => { alive = false; ++request; });
</script>
<dialog bind:this={dialog} aria-labelledby="benchmark-directory-title" oncancel={e => { e.preventDefault(); if (editing) void cancel(); else close(); }}>
  <h2 id="benchmark-directory-title">Бенчмарки</h2>
  {#if loading}<p role="status">Загрузка…</p>
  {:else if error}<p role="alert" class="error">{error}</p><button onclick={() => void load()}>Повторить загрузку</button>
  {:else if !rows.length}<p>Бенчмарков пока нет. Добавьте первый на вкладке «Бенчмарки».</p>
  {:else}
    <ul>{#each rows as row (row.id)}
      <li><h3>{row.name} <small>({row.currencyCode || 'индекс'})</small></h3>
        <p class="label">Источники</p>
        {#if row.sources}<ul class="sources">{#each row.sources.split('\n') as url}<li><a href={url} target="_blank" rel="noopener noreferrer">{url}</a></li>{/each}</ul>{:else}<p>Не указаны</p>{/if}
        <button disabled={busy || !!editing} onclick={e => void edit(row, e.currentTarget)} aria-label={`Изменить источники: ${row.name}`}>Изменить источники</button>
        {#if editing === row.id}
          <form onsubmit={save}>
            <label>Источники <textarea bind:this={sourceInput} bind:value={sources} rows="4" disabled={busy} aria-invalid={!!fieldError} aria-describedby="benchmark-sources-hint benchmark-sources-error"></textarea></label>
            <p id="benchmark-sources-hint">Ссылки http:// или https://, по одной на строку. Поле можно оставить пустым.</p>
            <p id="benchmark-sources-error" class="error" role="alert">{fieldError}</p>
            <div class="actions"><button type="button" disabled={busy} onclick={() => void cancel()}>Отмена</button><button type="submit" disabled={busy}>{busy ? 'Сохранение…' : 'Сохранить'}</button></div>
          </form>
        {/if}
      </li>
    {/each}</ul>
  {/if}
  {#if message}<p role="status">{message}</p>{/if}
  <div class="actions"><button disabled={busy} onclick={close}>Закрыть</button></div>
</dialog>
<style>
  dialog { width: min(760px, calc(100% - 32px)); max-height: calc(100% - 32px); padding: 24px; border: 1px solid #dce3e9; border-radius: 16px; color: inherit; }
  dialog::backdrop { background: #172a3a66; }
  h2 { margin-top: 0; } h3 { overflow-wrap: anywhere; } small { font-weight: normal; }
  ul { list-style: none; padding: 0; } li { border-bottom: 1px solid #dce3e9; padding: 12px 0; }
  .sources li { border: 0; padding: 4px 0; } a { overflow-wrap: anywhere; color: #176045; }
  .label { font-weight: bold; } form { margin-top: 16px; } label { display: grid; gap: 8px; }
  button,textarea { font: inherit; color: inherit; border: 1px solid #c4cfd8; border-radius: 8px; padding: 10px; background: white; }
  textarea { width: 100%; box-sizing: border-box; resize: vertical; } button { cursor: pointer; } button:disabled { opacity: .6; cursor: default; }
  .actions { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 12px; margin-top: 16px; }
  .error { color: #a12e2e; overflow-wrap: anywhere; }
</style>
