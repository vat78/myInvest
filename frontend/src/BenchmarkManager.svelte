<script lang="ts">
  import { onMount } from 'svelte';
  import ReturnComparison from './ReturnComparison.svelte';
  let { revision = 0 }: { revision?: number } = $props();
  let historyRevision = $state(0);
  import { ListBenchmarks, CreateBenchmark, BenchmarkHistory, SaveBenchmarkPoint, PreviewBenchmarkPoint } from '../wailsjs/go/desktop/BenchmarksAPI';
  import { ListCurrencies } from '../wailsjs/go/desktop/API';
  import { localDate } from './calendar';
  import { formatDecimal } from './decimal-format.mjs';
  let benchmarks = $state<Awaited<ReturnType<typeof ListBenchmarks>>>([]);
  let points = $state<Awaited<ReturnType<typeof BenchmarkHistory>>>([]);
  let currencies = $state<string[]>([]);
  let benchmarkID = $state(''), date = $state(localDate()), mode = $state('value'), input = $state('');
  let sources = $state('');
  let name = $state(''), currencyCode = $state(''), adding = $state(false), editing = $state(false);
  let busy = $state(false), loading = $state(false), loadError = $state('');
  let listLoading = $state(true), listError = $state('');
  let errors = $state<Record<string,string>>({}), message = $state(''), feedback = $state('');
  let createErrors = $state<Record<string,string>>({}), createMessage = $state('');
  let preview = $state<{value: string; growth: string} | null>(null), previewError = $state('');
  let alive = true, request = 0, listRequest = 0, previewRequest = 0;
  const selected = $derived(benchmarks.find(b => b.id === benchmarkID));
  async function refreshList() {
    const id = ++listRequest; listLoading = true; listError = '';
    try { const [rows, codes] = await Promise.all([ListBenchmarks(), ListCurrencies()]);
      if (!alive || id !== listRequest) return;
      benchmarks = rows; currencies = codes;
    } catch { if (alive && id === listRequest) listError = 'Не удалось загрузить бенчмарки. Повторите попытку.'; }
    finally { if (alive && id === listRequest) listLoading = false; }
  }
  async function refreshHistory() {
    const id = ++request, selectedID = benchmarkID; loading = true; loadError = ''; points = [];
    try { const rows = selectedID ? await BenchmarkHistory(selectedID) : [];
      if (alive && id === request && selectedID === benchmarkID) points = rows;
    } catch { if (alive && id === request) loadError = 'Не удалось обновить историю бенчмарка. Повторите попытку.'; }
    finally { if (alive && id === request) loading = false; }
  }
  onMount(() => { void refreshList(); return () => { alive = false; ++request; ++listRequest; ++previewRequest; }; });
  $effect(() => { benchmarkID; void refreshHistory(); });
  $effect(() => {
    points; // Refresh the preview when saved history changes.
    const p = {benchmarkID, date, mode, input, value: '', growth: ''};
    const id = ++previewRequest; preview = null; previewError = '';
    if (!p.benchmarkID || !p.date || !p.input) return;
    const timer = setTimeout(async () => {
      try { const result = await PreviewBenchmarkPoint(p);
        if (!alive || id !== previewRequest) return;
        previewError = result.message || Object.values(result.errors).join('; ');
        if (!previewError) preview = result.point;
      } catch { if (alive && id === previewRequest) previewError = 'Не удалось рассчитать предварительное значение.'; }
    }, 200);
    return () => clearTimeout(timer);
  });
  function reset() { if (busy) return; editing = false; date = localDate(); mode = 'value'; input = ''; errors = {}; message = ''; feedback = ''; }
  function selectBenchmark() { reset(); }
  function edit(p: {date: string; mode: string; input: string}) {
    if (busy) return; editing = true; date = p.date; mode = p.mode; input = p.input;
    errors = {}; message = ''; feedback = ''; document.getElementById('benchmark-input')?.focus();
  }
  async function create(event: SubmitEvent) {
    event.preventDefault(); if (busy) return; busy = true; createErrors = {}; createMessage = '';
    try { const result = await CreateBenchmark({id: '', name, currencyCode, sources}); if (!alive) return;
      createErrors = result.errors; createMessage = result.message;
      if (!createMessage && !Object.keys(createErrors).length) {
        // Keep the newly created selection even if the directory refresh fails.
        benchmarks = [...benchmarks, result.benchmark]; benchmarkID = result.benchmark.id;
        editing = false; date = localDate(); mode = 'value'; input = ''; errors = {}; message = ''; feedback = '';
        adding = false; name = ''; currencyCode = ''; sources = ''; await refreshList();
      }
    } catch { if (alive) createMessage = 'Не удалось создать бенчмарк. Повторите попытку.'; }
    finally { if (alive) busy = false; }
  }
  async function save(event: SubmitEvent) {
    event.preventDefault(); if (busy) return; busy = true; errors = {}; message = ''; feedback = '';
    try { const result = await SaveBenchmarkPoint({benchmarkID, date, mode, input, value: '', growth: ''}); if (!alive) return;
      errors = result.errors; message = result.message;
      if (!message && !Object.keys(errors).length) { feedback = 'Исторические данные сохранены'; historyRevision++; await refreshHistory(); }
    } catch { if (alive) message = 'Не удалось сохранить данные. Повторите попытку.'; }
    finally { if (alive) busy = false; }
  }
</script>
<ReturnComparison {benchmarks} revision={revision + historyRevision} />
<section aria-labelledby="benchmarks-title">
  <h2 id="benchmarks-title">Бенчмарки</h2>
  <p class="hint">Инфляция, депозиты, недвижимость и другие ориентиры. Прирост применяется к ближайшей предыдущей дате, без пересчёта в годовую ставку. Для индекса можно начать со стоимости 100.</p>
  <div class="actions">
    <label>Бенчмарк <select bind:value={benchmarkID} onchange={selectBenchmark} disabled={busy}>
      <option value="">Выберите бенчмарк</option>
      {#each benchmarks as b (b.id)}<option value={b.id}>{b.name} ({b.currencyCode || 'индекс'})</option>{/each}
    </select></label>
    <button disabled={busy} onclick={() => { adding = !adding; createErrors = {}; createMessage = ''; }}>{adding ? 'Закрыть добавление' : 'Добавить бенчмарк'}</button>
  </div>
  {#if adding}
    <form onsubmit={create}>
      <h3>Новый бенчмарк</h3>
      <fieldset disabled={busy}>
        <div class="fields">
          <label>Название <input bind:value={name} required maxlength="120" aria-invalid={!!createErrors.name} aria-describedby="benchmark-name-error" />
            <span class="error" id="benchmark-name-error">{createErrors.name ?? ''}</span></label>
          <label>Единица стоимости <select bind:value={currencyCode} aria-invalid={!!createErrors.currencyCode} aria-describedby="benchmark-currency-error">
            <option value="">Индекс (без валюты)</option>{#each currencies as c}<option value={c}>{c}</option>{/each}
          </select><span class="error" id="benchmark-currency-error">{createErrors.currencyCode ?? ''}</span></label>
        </div>
        <label>Источники <textarea bind:value={sources} rows="3" placeholder="https://example.com/statistics" aria-invalid={!!createErrors.sources} aria-describedby="benchmark-create-sources-hint benchmark-create-sources-error"></textarea><span id="benchmark-create-sources-hint">Ссылки http:// или https://, по одной на строку.</span><span class="error" id="benchmark-create-sources-error">{createErrors.sources ?? ''}</span></label>
        <button class="primary" type="submit">{busy ? 'Сохранение…' : 'Создать'}</button>
      </fieldset>
      {#if createMessage}<p class="error" role="alert">{createMessage}</p>{/if}
    </form>
  {/if}
  {#if listLoading}<p role="status">Загрузка бенчмарков…</p>{:else if !listError && !benchmarks.length}<p>Бенчмарков пока нет. Добавьте первый бенчмарк.</p>{/if}
  {#if listError}<p class="error" role="alert">{listError} <button disabled={busy} onclick={() => void refreshList()}>Повторить</button></p>{/if}
  {#if loadError}<p class="error" role="alert">{loadError} <button disabled={busy} onclick={async () => { await refreshHistory(); await refreshList(); }}>Повторить</button></p>{/if}
  {#if selected}
    <form onsubmit={save}>
      <h3>{editing ? 'Изменить исторические данные' : 'Новая историческая запись'}</h3>
      <fieldset disabled={busy}>
        <div class="fields">
          <label>Дата <input type="date" bind:value={date} disabled={editing} required aria-invalid={!!errors.date} aria-describedby="benchmark-date-error" />
            <span class="error" id="benchmark-date-error">{errors.date ?? ''}</span></label>
          <label>Способ ввода <select bind:value={mode}><option value="value">Новая стоимость</option><option value="growth">Процент прироста</option></select></label>
          <label>{mode === 'value' ? `Стоимость (${selected.currencyCode || 'индекс'})` : 'Прирост, %'}
            <input id="benchmark-input" type="text" inputmode="decimal" bind:value={input} required aria-invalid={!!errors.input} aria-describedby="benchmark-input-error" />
            <span class="error" id="benchmark-input-error">{errors.input ?? ''}</span></label>
        </div>
        <div class="actions"><button class="primary" type="submit" disabled={loading}>{busy ? 'Сохранение…' : 'Сохранить'}</button><button type="button" onclick={reset}>{editing ? 'Отмена изменения' : 'Очистить'}</button></div>
      </fieldset>
      {#if preview}<p role="status">Стоимость: {formatDecimal(preview.value)} {selected.currencyCode || '(индекс)'}; прирост: {preview.growth === '' ? '— (начальное значение)' : `${formatDecimal(preview.growth)}%`}.</p>{/if}
      {#if previewError}<p class="error">{previewError}</p>{/if}
      {#if message}<p class="error" role="alert">{message}</p>{/if}
      {#if errors.benchmarkID || errors.mode}<p class="error" role="alert">{errors.benchmarkID || errors.mode}</p>{/if}
      {#if feedback}<p role="status">{feedback}</p>{/if}
      <p class="hint">Первая запись требует стоимости больше нуля. Сохранение той же даты заменяет запись. Последующие значения, введённые процентом, пересчитываются; явно введённые стоимости сохраняются, их проценты обновляются.</p>
    </form>
    {#if loading}<p role="status">Загрузка истории…</p>{:else if !loadError && !points.length}<p>Исторических данных пока нет.</p>{/if}
    {#if points.length}<div class="table-wrap"><table><caption>История: {selected.name}</caption><thead><tr><th>Дата</th><th>Стоимость ({selected.currencyCode || 'индекс'})</th><th>Прирост, %</th><th>Введено</th><th>Действия</th></tr></thead>
      <tbody>{#each [...points].reverse() as p (p.date)}<tr><td>{p.date.split('-').reverse().join('.')}</td><td class="decimal">{formatDecimal(p.value)}</td><td class="decimal">{p.growth === '' ? '—' : formatDecimal(p.growth)}</td><td>{p.mode === 'value' ? 'Стоимость' : 'Прирост'}</td><td><button disabled={busy} onclick={() => edit(p)} aria-label={`Изменить ${selected.name} от ${p.date}`}>Изменить</button></td></tr>{/each}</tbody>
    </table></div>{/if}
  {/if}
</section>
<style>
  section, form { background: white; border: 1px solid #dce3e9; border-radius: 12px; padding: 20px; margin-bottom: 20px; min-width: 0; }
  form { background: #f5f7fa; margin-top: 20px; } h2,h3 { margin-top: 0; }
  .hint { font-size: 14px; color: #506477; } .error { color: #a12e2e; overflow-wrap: anywhere; font-size: 14px; }
  fieldset { border: 0; padding: 0; margin: 0; min-width: 0; }
  .fields { display: grid; grid-template-columns: repeat(auto-fit,minmax(min(100%,180px),1fr)); gap: 16px; }
  label { display: flex; flex-direction: column; gap: 8px; min-width: 0; font-size: 14px; }
  button,input,select,textarea { font: inherit; border: 1px solid #c4cfd8; border-radius: 8px; padding: 10px; color: inherit; background: white; min-width: 0; }
  button { cursor: pointer; } button:disabled { opacity: .6; cursor: default; }
  .primary { background: #207653; color: white; border-color: #207653; }
  .actions { display: flex; flex-wrap: wrap; gap: 12px; align-items: end; margin-top: 16px; }
  .table-wrap { overflow-x: auto; } table { width: 100%; border-collapse: collapse; }
  caption { text-align: left; font-size: 14px; color: #506477; padding: 12px 0; }
  th,td { text-align: left; padding: 12px 10px; border-bottom: 1px solid #dce3e9; }
  .decimal { overflow-wrap: anywhere; min-width: 140px; max-width: 320px; font-variant-numeric: tabular-nums; }
</style>
